package inventory

// This service handles:
// 1. OITM (Item Master) CRUD
// 2. OITB (Item Group) CRUD
// 3. UoM Master / Group CRUD
// 4. Goods Receipt (OIGN) and Goods Issue (OIGE)
// 5. OWHS (Warehouse Master) CRUD
// 6. OITW (Per-Warehouse Stock) queries
// 7. OIVL (Inventory Ledger) recording
// 8. ITM1 (Item Price List), ITM2 (Preferred Vendors), ITM12 (UoM per Item), OSPP (Special Prices)

import (
	"errors"
	"fmt"

	"ricemill/app/db"
	"ricemill/app/models"
	"ricemill/app/services/docnumber"

	"gorm.io/gorm"
)

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

// nullDate returns nil for an empty date string so MySQL date columns receive NULL instead of ''.
func nullDate(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// uomMap loads all UoMMaster records keyed by UomEntry for quick lookup.
func uomMap() map[uint]models.UoMMaster {
	var uoms []models.UoMMaster
	db.DB.Find(&uoms)
	m := make(map[uint]models.UoMMaster, len(uoms))
	for _, u := range uoms {
		m[u.UomEntry] = u
	}
	return m
}

// convertToInventoryQty converts a transaction quantity from the given UoM to the
// item's inventory UoM using UGP1 conversion factors. Returns qty unchanged if:
//   - transUomEntry is 0 (unspecified)
//   - transUomEntry == item.IUomEntry (already in inventory UoM)
//   - item has no UgpEntry (no conversion group)
//   - the UoM is not found in the group (best-effort: no conversion)
// ConvertToInventoryQty is the exported wrapper for use by other packages (purchasing, sales).
func ConvertToInventoryQty(tx *gorm.DB, item *models.OITM, transUomEntry uint, qty float64) float64 {
	return convertToInventoryQty(tx, item, transUomEntry, qty)
}

func convertToInventoryQty(tx *gorm.DB, item *models.OITM, transUomEntry uint, qty float64) float64 {
	if transUomEntry == 0 || transUomEntry == item.IUoMEntry || item.UgpEntry == 0 {
		return qty
	}
	var lines []models.UoMGroupLine
	if tx.Where("ugp_entry = ?", item.UgpEntry).Find(&lines).Error != nil {
		return qty
	}
	for _, l := range lines {
		if l.UomEntry == transUomEntry && l.AltQty > 0 {
			return qty * (l.BaseQty / l.AltQty)
		}
	}
	return qty // UoM not found in group — skip conversion
}

// upsertOITWOnHand atomically adds delta to OITW.on_hand for the given item-warehouse pair.
// Fallback chain for whsCode: lineWhs → itemDfltWh → "WH01".
// Uses INSERT … ON DUPLICATE KEY UPDATE so missing OITW records are auto-created.
func upsertOITWOnHand(tx *gorm.DB, itemCode, lineWhs, itemDfltWh string, delta float64) error {
	whs := lineWhs
	if whs == "" {
		whs = itemDfltWh
	}
	if whs == "" {
		whs = "WH01"
	}
	return tx.Exec(`
		INSERT INTO oitw (item_code, whs_code, on_hand)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE on_hand = on_hand + ?`,
		itemCode, whs, delta, delta).Error
}

// appendOIVL inserts one immutable audit row into the inventory value ledger.
// Must be called inside the same DB transaction as the stock update.
// whsCode fallback mirrors upsertOITWOnHand: lineWhs → itemDfltWh → "WH01".
func appendOIVL(tx *gorm.DB, itemCode, itemName, lineWhs, itemDfltWh, transType, docDate string, docNum int, inQty, outQty, price float64, createdByID uint) error {
	whs := lineWhs
	if whs == "" {
		whs = itemDfltWh
	}
	if whs == "" {
		whs = "WH01"
	}
	return tx.Create(&models.OIVL{
		ItemCode:    itemCode,
		ItemName:    itemName,
		Warehouse:   whs,
		DocDate:     docDate,
		TransType:   transType,
		DocNum:      docNum,
		InQty:       inQty,
		OutQty:      outQty,
		Price:       price,
		Value:       (inQty + outQty) * price,
		CreatedByID: createdByID,
	}).Error
}

// UpsertOITWOnHand is the exported wrapper around upsertOITWOnHand for use by other packages.
func UpsertOITWOnHand(tx *gorm.DB, itemCode, lineWhs, itemDfltWh string, delta float64) error {
	return upsertOITWOnHand(tx, itemCode, lineWhs, itemDfltWh, delta)
}

// RecalcMovingAvgPrice computes the new moving-average unit cost after receiving invQty units
// at invUnitPrice (expressed in inventory UoM), then persists it to oitm.avg_price and the
// matching oitw.avg_price row.
//
// Must be called BEFORE the on_hand increment so oitm.OnHand still reflects the pre-receipt
// stock level.
//
// Cancellations do NOT call this function — reversing a moving average is not possible;
// the avg_price stays at its current value when stock is removed.
//
// Returns the new average price.
func RecalcMovingAvgPrice(tx *gorm.DB, oitm *models.OITM, invQty, invUnitPrice float64) float64 {
	oldOnHand := oitm.OnHand
	if oldOnHand < 0 {
		oldOnHand = 0 // guard: negative stock shouldn't drag the average down
	}
	denom := oldOnHand + invQty
	if denom <= 0 {
		return oitm.AvgPrice
	}
	newAvg := (oldOnHand*oitm.AvgPrice + invQty*invUnitPrice) / denom

	// Persist to oitm
	tx.Model(&models.OITM{}).Where("item_code = ?", oitm.ItemCode).
		Update("avg_price", newAvg)

	// Mirror to oitw row(s) for this item so per-warehouse cost is consistent
	tx.Exec(`UPDATE oitw SET avg_price = ? WHERE item_code = ?`, newAvg, oitm.ItemCode)

	return newAvg
}

// AppendOIVL is the exported wrapper around appendOIVL for use by other packages.
func AppendOIVL(tx *gorm.DB, itemCode, itemName, lineWhs, itemDfltWh, transType, docDate string, docNum int, inQty, outQty, price float64, createdByID uint) error {
	return appendOIVL(tx, itemCode, itemName, lineWhs, itemDfltWh, transType, docDate, docNum, inQty, outQty, price, createdByID)
}

// attachUoMFields fills the in-memory BaseUoM, SalesUoM, and PurchUoM fields for each item.
func attachUoMFields(items []models.OITM, m map[uint]models.UoMMaster) {
	for i := range items {
		if items[i].IUoMEntry > 0 {
			if u, ok := m[items[i].IUoMEntry]; ok {
				items[i].BaseUoM = &u
			}
		}
		if items[i].SUoMEntry > 0 {
			if u, ok := m[items[i].SUoMEntry]; ok {
				items[i].SalesUoM = &u
			}
		}
		if items[i].PUoMEntry > 0 {
			if u, ok := m[items[i].PUoMEntry]; ok {
				items[i].PurchUoM = &u
			}
		}
	}
}

// attachCategoryNames populates the virtual CategoryName field for each item
// using a batch lookup from oitb. Used by production service for snapshot strings.
func attachCategoryNames(items []models.OITM) {
	// Collect unique group codes
	grpCodes := make([]int, 0)
	seen := make(map[int]bool)
	for _, it := range items {
		if !seen[it.ItmsGrpCod] {
			grpCodes = append(grpCodes, it.ItmsGrpCod)
			seen[it.ItmsGrpCod] = true
		}
	}
	var cats []models.OITB
	db.DB.Where("itms_grp_cod IN ?", grpCodes).Find(&cats)
	catMap := make(map[int]string, len(cats))
	for _, c := range cats {
		catMap[c.ItmsGrpCod] = c.ItmsGrpNam
	}
	for i := range items {
		items[i].CategoryName = catMap[items[i].ItmsGrpCod]
	}
}

// ─────────────────────────────────────────────
// OITM — Item Master CRUD
// ─────────────────────────────────────────────

// ListProductionItems returns active items flagged as usable in production/milling,
// restricted to categories that are enabled for production.
func ListProductionItems() ([]models.OITM, error) {
	var items []models.OITM
	if err := db.DB.Where("valid_for = 'Y' AND mak_item = 'Y' AND itms_grp_cod IN (SELECT itms_grp_cod FROM oitb WHERE is_active = 1 AND for_production = 1)").
		Order("itms_grp_cod, item_name").Find(&items).Error; err != nil {
		return nil, err
	}
	attachUoMFields(items, uomMap())
	attachCategoryNames(items)
	return items, nil
}

// ListItemMasters returns item masters. Pass activeOnly=true to filter inactive.
func ListItemMasters(activeOnly bool) ([]models.OITM, error) {
	var items []models.OITM
	q := db.DB.Preload("UoMPrices.Unit").Order("itms_grp_cod, item_name")
	if activeOnly {
		q = q.Where("valid_for = 'Y'")
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	attachUoMFields(items, uomMap())
	attachCategoryNames(items)
	return items, nil
}

// ListSalesItems returns active items whose category is enabled for sales.
func ListSalesItems() ([]models.OITM, error) {
	var items []models.OITM
	if err := db.DB.Preload("UoMPrices.Unit").
		Where("valid_for = 'Y' AND itms_grp_cod IN (SELECT itms_grp_cod FROM oitb WHERE is_active = 1 AND for_sales = 1)").
		Order("itms_grp_cod, item_name").Find(&items).Error; err != nil {
		return nil, err
	}
	attachUoMFields(items, uomMap())
	attachCategoryNames(items)
	return items, nil
}

// ListPurchasingItems returns active items whose category is enabled for purchasing.
func ListPurchasingItems() ([]models.OITM, error) {
	var items []models.OITM
	if err := db.DB.Preload("UoMPrices.Unit").
		Where("valid_for = 'Y' AND itms_grp_cod IN (SELECT itms_grp_cod FROM oitb WHERE is_active = 1 AND for_purchasing = 1)").
		Order("itms_grp_cod, item_name").Find(&items).Error; err != nil {
		return nil, err
	}
	attachUoMFields(items, uomMap())
	attachCategoryNames(items)
	return items, nil
}

// ListInventoryItems returns active items whose category is enabled for inventory.
func ListInventoryItems() ([]models.OITM, error) {
	var items []models.OITM
	if err := db.DB.Preload("UoMPrices.Unit").
		Where("valid_for = 'Y' AND itms_grp_cod IN (SELECT itms_grp_cod FROM oitb WHERE is_active = 1 AND for_inventory = 1)").
		Order("itms_grp_cod, item_name").Find(&items).Error; err != nil {
		return nil, err
	}
	attachUoMFields(items, uomMap())
	attachCategoryNames(items)
	return items, nil
}

// GetItemMaster fetches a single OITM by primary key.
func GetItemMaster(id uint) (*models.OITM, error) {
	var item models.OITM
	if err := db.DB.Preload("UoMPrices.Unit").First(&item, id).Error; err != nil {
		return nil, err
	}
	m := uomMap()
	if item.IUoMEntry > 0 {
		if u, ok := m[item.IUoMEntry]; ok {
			item.BaseUoM = &u
		}
	}
	if item.SUoMEntry > 0 {
		if u, ok := m[item.SUoMEntry]; ok {
			item.SalesUoM = &u
		}
	}
	if item.PUoMEntry > 0 {
		if u, ok := m[item.PUoMEntry]; ok {
			item.PurchUoM = &u
		}
	}
	// Populate virtual category name
	items := []models.OITM{item}
	attachCategoryNames(items)
	item.CategoryName = items[0].CategoryName
	return &item, nil
}

// upsertITM12ForItem syncs ITM12 rows (I/S/P) from the item's three UoM entry fields.
func upsertITM12ForItem(itemCode string, iUomEntry, sUomEntry, pUomEntry uint) {
	type row struct {
		entry   uint
		uomType string
	}
	for _, r := range []row{{iUomEntry, "I"}, {sUomEntry, "S"}, {pUomEntry, "P"}} {
		if r.entry > 0 {
			db.DB.Save(&models.ITM12{ItemCode: itemCode, UomEntry: r.entry, UomType: r.uomType})
		}
	}
}

// toUint extracts a uint from a map[string]interface{} value (handles float64/int/uint from JSON).
func toUint(m map[string]interface{}, key string) uint {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return uint(n)
	case int:
		return uint(n)
	case uint:
		return n
	}
	return 0
}

// CreateItemMaster inserts a new OITM and syncs its ITM12 UoM mapping rows.
func CreateItemMaster(item *models.OITM) error {
	item.ValidFor = "Y"
	if err := db.DB.Create(item).Error; err != nil {
		return err
	}
	upsertITM12ForItem(item.ItemCode, item.IUoMEntry, item.SUoMEntry, item.PUoMEntry)
	return nil
}

// UpdateItemMaster applies a partial update to an OITM and syncs ITM12 if UoM entries changed.
func UpdateItemMaster(id uint, updates map[string]interface{}) error {
	if err := db.DB.Model(&models.OITM{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	if itemCode, _ := updates["item_code"].(string); itemCode != "" {
		upsertITM12ForItem(itemCode, toUint(updates, "i_uom_entry"), toUint(updates, "s_uom_entry"), toUint(updates, "p_uom_entry"))
	}
	return nil
}

// DeleteItemMaster soft-deletes an OITM by setting valid_for='N'.
func DeleteItemMaster(id uint) error {
	return db.DB.Model(&models.OITM{}).Where("id = ?", id).
		Update("valid_for", "N").Error
}

// ─────────────────────────────────────────────
// ItemUoMPrice CRUD (ITM9)
// PK changed from item_id (uint) to item_code (string)
// ─────────────────────────────────────────────

// GetItemUoMPrices returns all UoM prices for an item by item_code.
func GetItemUoMPrices(itemCode string) ([]models.ItemUoMPrice, error) {
	var prices []models.ItemUoMPrice
	err := db.DB.Preload("Unit").Where("item_code = ?", itemCode).Find(&prices).Error
	return prices, err
}

// UpsertItemUoMPrice creates or updates a UoM price record for an item.
func UpsertItemUoMPrice(itemCode string, uomEntry uint, price, factor float64) error {
	p := models.ItemUoMPrice{ItemCode: itemCode, UomEntry: uomEntry, Price: price, Factor: factor}
	return db.DB.Save(&p).Error
}

// DeleteItemUoMPrice removes a UoM price record by item_code and uom_entry.
func DeleteItemUoMPrice(itemCode string, uomEntry uint) error {
	return db.DB.Delete(&models.ItemUoMPrice{}, "item_code = ? AND uom_entry = ?", itemCode, uomEntry).Error
}

// ─────────────────────────────────────────────
// OITB — Item Group CRUD
// ─────────────────────────────────────────────

// ListItemCategories returns all item groups (OITB) ordered by name.
// The name "ListItemCategories" is retained for backward compatibility.
func ListItemCategories(activeOnly bool) ([]models.OITB, error) {
	var cats []models.OITB
	q := db.DB.Order("itms_grp_nam")
	if activeOnly {
		q = q.Where("is_active = 1")
	}
	return cats, q.Find(&cats).Error
}

// ListItemCategoriesForModule returns active categories visible in the given module.
func ListItemCategoriesForModule(module string) ([]models.OITB, error) {
	var cats []models.OITB
	q := db.DB.Where("is_active = 1").Order("itms_grp_nam")
	switch module {
	case "sales":
		q = q.Where("for_sales = 1")
	case "purchasing":
		q = q.Where("for_purchasing = 1")
	case "inventory":
		q = q.Where("for_inventory = 1")
	case "production":
		q = q.Where("for_production = 1")
	}
	return cats, q.Find(&cats).Error
}

// GetItemCategory fetches a single OITB by ItmsGrpCod (int PK).
func GetItemCategory(id int) (*models.OITB, error) {
	var cat models.OITB
	err := db.DB.First(&cat, id).Error
	return &cat, err
}

// CreateItemCategory inserts a new OITB.
func CreateItemCategory(cat *models.OITB) error {
	cat.IsActive = true
	return db.DB.Create(cat).Error
}

// UpdateItemCategory applies a partial update to an OITB.
func UpdateItemCategory(id int, updates map[string]interface{}) error {
	return db.DB.Model(&models.OITB{}).Where("itms_grp_cod = ?", id).Updates(updates).Error
}

// DeleteItemCategory soft-deletes an OITB (sets is_active=false).
func DeleteItemCategory(id int) error {
	return db.DB.Model(&models.OITB{}).Where("itms_grp_cod = ?", id).
		Update("is_active", false).Error
}

// GetItemCategoryName returns the group name string for a given ItmsGrpCod.
// Used to populate the CategoryName virtual field for backward compatibility
// with production service snapshot fields.
func GetItemCategoryName(grpCod int) string {
	var cat models.OITB
	if err := db.DB.Select("itms_grp_nam").First(&cat, grpCod).Error; err != nil {
		return ""
	}
	return cat.ItmsGrpNam
}

// ─────────────────────────────────────────────
// UoM Master (OUOM)
// ─────────────────────────────────────────────

func ListUoMMasters() ([]models.UoMMaster, error) {
	var items []models.UoMMaster
	err := db.DB.Order("uom_code").Find(&items).Error
	return items, err
}

func GetUoMMaster(entry uint) (*models.UoMMaster, error) {
	var item models.UoMMaster
	err := db.DB.First(&item, entry).Error
	return &item, err
}

func CreateUoMMaster(item *models.UoMMaster) error {
	return db.DB.Create(item).Error
}

func UpdateUoMMaster(entry uint, updates map[string]interface{}) error {
	return db.DB.Model(&models.UoMMaster{}).Where("uom_entry = ?", entry).Updates(updates).Error
}

func DeleteUoMMaster(entry uint) error {
	return db.DB.Delete(&models.UoMMaster{}, entry).Error
}

// ─────────────────────────────────────────────
// UoM Groups (OUGP)
// ─────────────────────────────────────────────

func ListUoMGroups() ([]models.UoMGroup, error) {
	var groups []models.UoMGroup
	err := db.DB.Preload("Lines.Unit").Preload("BaseUnit").Order("ugp_code").Find(&groups).Error
	return groups, err
}

func GetUoMGroup(entry uint) (*models.UoMGroup, error) {
	var g models.UoMGroup
	err := db.DB.Preload("Lines.Unit").Preload("BaseUnit").First(&g, entry).Error
	return &g, err
}

type CreateUoMGroupRequest struct {
	UgpCode string              `json:"ugp_code"`
	UgpName string              `json:"ugp_name"`
	BaseUom int                 `json:"base_uom"`
	Lines   []UoMGroupLineInput `json:"lines"`
}

type UoMGroupLineInput struct {
	UomEntry uint    `json:"uom_entry"`
	AltQty   float64 `json:"alt_qty"`
	BaseQty  float64 `json:"base_qty"`
}

func CreateUoMGroup(req CreateUoMGroupRequest) (*models.UoMGroup, error) {
	g := models.UoMGroup{
		UgpCode:    req.UgpCode,
		UgpName:    req.UgpName,
		BaseUom:    req.BaseUom,
		DataSource: "M",
	}
	if err := db.DB.Create(&g).Error; err != nil {
		return nil, err
	}
	for i, l := range req.Lines {
		line := models.UoMGroupLine{
			UgpEntry: g.UgpEntry,
			LineNum:  i,
			UomEntry: l.UomEntry,
			AltQty:   l.AltQty,
			BaseQty:  l.BaseQty,
			ObjType:  "173",
		}
		if err := db.DB.Create(&line).Error; err != nil {
			return nil, err
		}
	}
	return GetUoMGroup(g.UgpEntry)
}

func UpdateUoMGroup(entry uint, ugpCode, ugpName string, baseUom int, lines []UoMGroupLineInput) error {
	if err := db.DB.Model(&models.UoMGroup{}).Where("ugp_entry = ?", entry).Updates(map[string]interface{}{
		"ugp_code": ugpCode,
		"ugp_name": ugpName,
		"base_uom": baseUom,
	}).Error; err != nil {
		return err
	}
	// Replace lines
	db.DB.Where("ugp_entry = ?", entry).Delete(&models.UoMGroupLine{})
	for i, l := range lines {
		line := models.UoMGroupLine{
			UgpEntry: entry,
			LineNum:  i,
			UomEntry: l.UomEntry,
			AltQty:   l.AltQty,
			BaseQty:  l.BaseQty,
			ObjType:  "173",
		}
		if err := db.DB.Create(&line).Error; err != nil {
			return err
		}
	}
	return nil
}

func DeleteUoMGroup(entry uint) error {
	db.DB.Where("ugp_entry = ?", entry).Delete(&models.UoMGroupLine{})
	return db.DB.Delete(&models.UoMGroup{}, entry).Error
}

// ─────────────────────────────────────────────
// Goods Receipt (OIGN / IGN1)
// ─────────────────────────────────────────────

// GRLineInput is the DTO for a single line in a CreateGoodsReceiptRequest.
type GRLineInput struct {
	ItemID        uint    `json:"item_id"`
	UomEntry      uint    `json:"uom_entry"`
	Quantity      float64 `json:"quantity"`
	Price         float64 `json:"price"`
	WarehouseCode string  `json:"warehouse_code"`
	AccountCode   string  `json:"account_code"`
	Project       string  `json:"project"`
}

// CreateGoodsReceiptRequest is the DTO used to create a new GoodsReceipt.
type CreateGoodsReceiptRequest struct {
	PostingDate string        `json:"posting_date"`
	DocDueDate  string        `json:"doc_due_date"`
	Remarks     string        `json:"remarks"`
	Lines       []GRLineInput `json:"lines"`
}

// ListGoodsReceipts returns all goods receipts ordered newest-first.
func ListGoodsReceipts() ([]models.GoodsReceipt, error) {
	var list []models.GoodsReceipt
	err := db.DB.Order("created_at DESC").Find(&list).Error
	return list, err
}

// GetGoodsReceipt fetches a single goods receipt with its lines.
func GetGoodsReceipt(id uint) (*models.GoodsReceipt, error) {
	var gr models.GoodsReceipt
	if err := db.DB.First(&gr, id).Error; err != nil {
		return nil, err
	}
	db.DB.Where("goods_receipt_id = ?", id).Order("line_num").Find(&gr.Lines)
	return &gr, nil
}

// CreateGoodsReceipt inserts a new GoodsReceipt and updates on_hand for each item.
func CreateGoodsReceipt(req CreateGoodsReceiptRequest, userID *uint) (*models.GoodsReceipt, error) {
	if len(req.Lines) == 0 {
		return nil, errors.New("at least one line is required")
	}
	for _, l := range req.Lines {
		if l.Quantity <= 0 {
			return nil, errors.New("all line quantities must be greater than zero")
		}
		if l.ItemID == 0 {
			return nil, errors.New("all lines must have a valid item")
		}
	}

	grNum, err := docnumber.Svc.NextGRNumber()
	if err != nil {
		return nil, fmt.Errorf("generate GR number: %w", err)
	}

	// Pre-fetch item masters for all line items
	itemIDs := make([]uint, len(req.Lines))
	for i, l := range req.Lines {
		itemIDs[i] = l.ItemID
	}
	var items []models.OITM
	if err := db.DB.Where("id IN ?", itemIDs).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("fetch items: %w", err)
	}
	itemMap := make(map[uint]models.OITM, len(items))
	for _, it := range items {
		itemMap[it.ID] = it
	}

	// Build lines and compute doc total
	var docTotal float64
	lines := make([]models.GoodsReceiptItem, 0, len(req.Lines))
	for i, l := range req.Lines {
		it, ok := itemMap[l.ItemID]
		if !ok {
			return nil, fmt.Errorf("item ID %d not found", l.ItemID)
		}
		uomEntry := l.UomEntry
		if uomEntry == 0 {
			uomEntry = it.IUoMEntry
		}
		lineTotal := l.Quantity * l.Price
		docTotal += lineTotal
		lines = append(lines, models.GoodsReceiptItem{
			LineNum:       i,
			ItemID:        l.ItemID,
			ItemCode:      it.ItemCode,
			Description:   it.ItemName,
			Quantity:      l.Quantity,
			Unit:          it.InvntryUom,
			UomEntry:      uomEntry,
			Price:         l.Price,
			LineTotal:     lineTotal,
			WarehouseCode: l.WarehouseCode,
			AccountCode:   l.AccountCode,
			Project:       l.Project,
		})
	}

	var createdByID uint
	if userID != nil {
		createdByID = *userID
	}
	header := models.GoodsReceipt{
		GRNumber:    grNum,
		PostingDate: req.PostingDate,
		DocDueDate:  nullDate(req.DocDueDate),
		Status:      "Open",
		Remarks:     req.Remarks,
		DocTotal:    docTotal,
		CreatedByID: createdByID,
	}

	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&header).Error; err != nil {
			return err
		}
		for i := range lines {
			lines[i].GoodsReceiptID = header.ID
			if err := tx.Create(&lines[i]).Error; err != nil {
				return err
			}
			// Increase on_hand (converted to inventory UoM)
			it := itemMap[lines[i].ItemID]
			invQty := convertToInventoryQty(tx, &it, lines[i].UomEntry, lines[i].Quantity)

			// Convert line price to per-inventory-unit cost, then recalc moving average
			invUnitPrice := lines[i].Price
			if invQty > 0 && lines[i].Quantity > 0 {
				invUnitPrice = (lines[i].Quantity * lines[i].Price) / invQty
			}
			newAvg := RecalcMovingAvgPrice(tx, &it, invQty, invUnitPrice)

			if err := tx.Model(&models.OITM{}).Where("id = ?", lines[i].ItemID).
				UpdateColumn("on_hand", gorm.Expr("on_hand + ?", invQty)).Error; err != nil {
				return err
			}
			// Mirror update to OITW (per-warehouse stock)
			if err := upsertOITWOnHand(tx, it.ItemCode, lines[i].WarehouseCode, it.DfltWh, invQty); err != nil {
				return err
			}
			// Append OIVL audit row — use new avg as the post-receipt unit cost
			if err := appendOIVL(tx, it.ItemCode, it.ItemName, lines[i].WarehouseCode, it.DfltWh, "GR", header.PostingDate, int(header.ID), invQty, 0, newAvg, createdByID); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	header.Lines = lines
	return &header, nil
}

// CancelGoodsReceipt sets status to Cancelled and reverses on_hand for each line.
func CancelGoodsReceipt(id uint, userID *uint) error {
	gr, err := GetGoodsReceipt(id)
	if err != nil {
		return err
	}
	if gr.Status == "Cancelled" {
		return errors.New("goods receipt is already cancelled")
	}

	var cancelledByID uint
	if userID != nil {
		cancelledByID = *userID
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, line := range gr.Lines {
			var item models.OITM
			if err := tx.First(&item, line.ItemID).Error; err != nil {
				return err
			}
			invQty := convertToInventoryQty(tx, &item, line.UomEntry, line.Quantity)
			if err := tx.Model(&models.OITM{}).Where("id = ?", line.ItemID).
				UpdateColumn("on_hand", gorm.Expr("on_hand - ?", invQty)).Error; err != nil {
				return err
			}
			// Mirror reversal to OITW (per-warehouse stock)
			if err := upsertOITWOnHand(tx, item.ItemCode, line.WarehouseCode, item.DfltWh, -invQty); err != nil {
				return err
			}
			// Append OIVL audit row (offsetting entry)
			if err := appendOIVL(tx, item.ItemCode, item.ItemName, line.WarehouseCode, item.DfltWh, "GR_CANCEL", gr.PostingDate, int(gr.ID), 0, invQty, item.AvgPrice, cancelledByID); err != nil {
				return err
			}
		}
		return tx.Model(&models.GoodsReceipt{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":        "Cancelled",
				"updated_by_id": cancelledByID,
			}).Error
	})
}

// ─────────────────────────────────────────────
// Goods Issue (OIGE / IGE1)
// ─────────────────────────────────────────────

// GILineInput is the DTO for a single line in a CreateGoodsIssueRequest.
type GILineInput struct {
	ItemID        uint    `json:"item_id"`
	UomEntry      uint    `json:"uom_entry"`
	Quantity      float64 `json:"quantity"`
	Price         float64 `json:"price"`
	WarehouseCode string  `json:"warehouse_code"`
	AccountCode   string  `json:"account_code"`
	Project       string  `json:"project"`
}

// CreateGoodsIssueRequest is the DTO used to create a new GoodsIssue.
type CreateGoodsIssueRequest struct {
	PostingDate string        `json:"posting_date"`
	DocDueDate  string        `json:"doc_due_date"`
	Remarks     string        `json:"remarks"`
	Lines       []GILineInput `json:"lines"`
}

// ListGoodsIssues returns all goods issues ordered newest-first.
func ListGoodsIssues() ([]models.GoodsIssue, error) {
	var list []models.GoodsIssue
	err := db.DB.Order("created_at DESC").Find(&list).Error
	return list, err
}

// GetGoodsIssue fetches a single goods issue with its lines.
func GetGoodsIssue(id uint) (*models.GoodsIssue, error) {
	var gi models.GoodsIssue
	if err := db.DB.First(&gi, id).Error; err != nil {
		return nil, err
	}
	db.DB.Where("goods_issue_id = ?", id).Order("line_num").Find(&gi.Lines)
	return &gi, nil
}

// CreateGoodsIssue inserts a new GoodsIssue and decrements on_hand for each item.
func CreateGoodsIssue(req CreateGoodsIssueRequest, userID *uint) (*models.GoodsIssue, error) {
	if len(req.Lines) == 0 {
		return nil, errors.New("at least one line is required")
	}
	for _, l := range req.Lines {
		if l.Quantity <= 0 {
			return nil, errors.New("all line quantities must be greater than zero")
		}
		if l.ItemID == 0 {
			return nil, errors.New("all lines must have a valid item")
		}
	}

	// Pre-fetch item masters for all line items
	itemIDs := make([]uint, len(req.Lines))
	for i, l := range req.Lines {
		itemIDs[i] = l.ItemID
	}
	var items []models.OITM
	if err := db.DB.Where("id IN ?", itemIDs).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("fetch items: %w", err)
	}
	itemMap := make(map[uint]models.OITM, len(items))
	for _, it := range items {
		itemMap[it.ID] = it
	}

	// Validate sufficient stock before issuing (compare in inventory UoM)
	for _, l := range req.Lines {
		it, ok := itemMap[l.ItemID]
		if !ok {
			return nil, fmt.Errorf("item ID %d not found", l.ItemID)
		}
		uomEntry := l.UomEntry
		if uomEntry == 0 {
			uomEntry = it.IUoMEntry
		}
		invQty := convertToInventoryQty(db.DB, &it, uomEntry, l.Quantity)
		if it.OnHand < invQty {
			return nil, fmt.Errorf("insufficient stock for item %s: on hand %.3f, requested %.3f", it.ItemCode, it.OnHand, invQty)
		}
	}

	giNum, err := docnumber.Svc.NextGINumber()
	if err != nil {
		return nil, fmt.Errorf("generate GI number: %w", err)
	}

	// Build lines and compute doc total
	var docTotal float64
	lines := make([]models.GoodsIssueItem, 0, len(req.Lines))
	for i, l := range req.Lines {
		it := itemMap[l.ItemID]
		uomEntry := l.UomEntry
		if uomEntry == 0 {
			uomEntry = it.IUoMEntry
		}
		lineTotal := l.Quantity * l.Price
		docTotal += lineTotal
		lines = append(lines, models.GoodsIssueItem{
			LineNum:       i,
			ItemID:        l.ItemID,
			ItemCode:      it.ItemCode,
			Description:   it.ItemName,
			Quantity:      l.Quantity,
			Unit:          it.InvntryUom,
			UomEntry:      uomEntry,
			Price:         l.Price,
			LineTotal:     lineTotal,
			WarehouseCode: l.WarehouseCode,
			AccountCode:   l.AccountCode,
			Project:       l.Project,
		})
	}

	var createdByID uint
	if userID != nil {
		createdByID = *userID
	}
	header := models.GoodsIssue{
		GINumber:    giNum,
		PostingDate: req.PostingDate,
		DocDueDate:  nullDate(req.DocDueDate),
		Status:      "Open",
		Remarks:     req.Remarks,
		DocTotal:    docTotal,
		CreatedByID: createdByID,
	}

	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&header).Error; err != nil {
			return err
		}
		for i := range lines {
			lines[i].GoodsIssueID = header.ID
			if err := tx.Create(&lines[i]).Error; err != nil {
				return err
			}
			// Decrease on_hand (converted to inventory UoM)
			it := itemMap[lines[i].ItemID]
			invQty := convertToInventoryQty(tx, &it, lines[i].UomEntry, lines[i].Quantity)
			if err := tx.Model(&models.OITM{}).Where("id = ?", lines[i].ItemID).
				UpdateColumn("on_hand", gorm.Expr("on_hand - ?", invQty)).Error; err != nil {
				return err
			}
			// Mirror update to OITW (per-warehouse stock)
			if err := upsertOITWOnHand(tx, it.ItemCode, lines[i].WarehouseCode, it.DfltWh, -invQty); err != nil {
				return err
			}
			// Append OIVL audit row
			if err := appendOIVL(tx, it.ItemCode, it.ItemName, lines[i].WarehouseCode, it.DfltWh, "GI", header.PostingDate, int(header.ID), 0, invQty, it.AvgPrice, createdByID); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	header.Lines = lines
	return &header, nil
}

// CancelGoodsIssue sets status to Cancelled and restores on_hand for each line.
func CancelGoodsIssue(id uint, userID *uint) error {
	gi, err := GetGoodsIssue(id)
	if err != nil {
		return err
	}
	if gi.Status == "Cancelled" {
		return errors.New("goods issue is already cancelled")
	}

	var cancelledByID uint
	if userID != nil {
		cancelledByID = *userID
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, line := range gi.Lines {
			var item models.OITM
			if err := tx.First(&item, line.ItemID).Error; err != nil {
				return err
			}
			invQty := convertToInventoryQty(tx, &item, line.UomEntry, line.Quantity)
			if err := tx.Model(&models.OITM{}).Where("id = ?", line.ItemID).
				UpdateColumn("on_hand", gorm.Expr("on_hand + ?", invQty)).Error; err != nil {
				return err
			}
			// Mirror reversal to OITW (per-warehouse stock)
			if err := upsertOITWOnHand(tx, item.ItemCode, line.WarehouseCode, item.DfltWh, invQty); err != nil {
				return err
			}
			// Append OIVL audit row (offsetting entry)
			if err := appendOIVL(tx, item.ItemCode, item.ItemName, line.WarehouseCode, item.DfltWh, "GI_CANCEL", gi.PostingDate, int(gi.ID), invQty, 0, item.AvgPrice, cancelledByID); err != nil {
				return err
			}
		}
		return tx.Model(&models.GoodsIssue{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":        "Cancelled",
				"updated_by_id": cancelledByID,
			}).Error
	})
}

// ─────────────────────────────────────────────
// OWHS — Warehouse Master CRUD
// ─────────────────────────────────────────────

// ListWarehouses returns all warehouses ordered by code.
func ListWarehouses() ([]models.OWHS, error) {
	var list []models.OWHS
	err := db.DB.Order("whs_code").Find(&list).Error
	return list, err
}

// GetWarehouse fetches a single warehouse by WhsCode.
func GetWarehouse(whsCode string) (*models.OWHS, error) {
	var whs models.OWHS
	err := db.DB.First(&whs, "whs_code = ?", whsCode).Error
	return &whs, err
}

// CreateWarehouse inserts a new OWHS record.
func CreateWarehouse(whs *models.OWHS) error {
	return db.DB.Create(whs).Error
}

// UpdateWarehouse applies a partial update to an OWHS record.
func UpdateWarehouse(whsCode string, updates map[string]interface{}) error {
	return db.DB.Model(&models.OWHS{}).Where("whs_code = ?", whsCode).Updates(updates).Error
}

// ─────────────────────────────────────────────
// OITW — Per-Warehouse Stock (read-only queries)
// Written by GR/GI/PO/SO transactions, not directly by user input.
// ─────────────────────────────────────────────

// GetItemWarehouseStock returns all warehouse stock records for a given item.
func GetItemWarehouseStock(itemCode string) ([]models.OITW, error) {
	var list []models.OITW
	err := db.DB.Where("item_code = ?", itemCode).Find(&list).Error
	return list, err
}

// GetWarehouseStock returns all item stock records for a given warehouse.
func GetWarehouseStock(whsCode string) ([]models.OITW, error) {
	var list []models.OITW
	err := db.DB.Where("whs_code = ?", whsCode).Find(&list).Error
	return list, err
}

// ListAllWarehouseStock returns all OITW records across every warehouse.
func ListAllWarehouseStock() ([]models.OITW, error) {
	var list []models.OITW
	err := db.DB.Order("whs_code, item_code").Find(&list).Error
	return list, err
}

// ListOIVL returns ledger entries ordered newest-first.
// Pass itemCode="" to return all items (capped at 500 rows).
func ListOIVL(itemCode string) ([]models.OIVL, error) {
	q := db.DB.Order("doc_entry desc").Limit(500)
	if itemCode != "" {
		q = q.Where("item_code = ?", itemCode)
	}
	var list []models.OIVL
	return list, q.Find(&list).Error
}

// UpdateOITWMinMax sets the min_stock and max_stock thresholds for a specific item-warehouse record.
func UpdateOITWMinMax(itemCode, whsCode string, minStock, maxStock float64) error {
	return db.DB.Model(&models.OITW{}).
		Where("item_code = ? AND whs_code = ?", itemCode, whsCode).
		Updates(map[string]interface{}{
			"min_stock": minStock,
			"max_stock": maxStock,
		}).Error
}

// ─────────────────────────────────────────────
// ITM1 — Item Prices per Price List
// ─────────────────────────────────────────────

// ListItemPrices returns all price list entries for a given item.
func ListItemPrices(itemCode string) ([]models.ITM1, error) {
	var list []models.ITM1
	err := db.DB.Where("item_code = ?", itemCode).Find(&list).Error
	return list, err
}

// UpsertItemPrice creates or updates an ITM1 price record.
func UpsertItemPrice(itemCode string, priceList int, price float64, currency string) error {
	p := models.ITM1{
		ItemCode:  itemCode,
		PriceList: priceList,
		Price:     price,
		Currency:  currency,
	}
	return db.DB.Save(&p).Error
}

// ─────────────────────────────────────────────
// ITM2 — Multiple Preferred Vendors per Item
// ─────────────────────────────────────────────

// ListPreferredVendors returns all preferred vendor entries for a given item.
func ListPreferredVendors(itemCode string) ([]models.ITM2, error) {
	var list []models.ITM2
	err := db.DB.Where("item_code = ?", itemCode).Order("line_num").Find(&list).Error
	return list, err
}

// UpsertPreferredVendor creates or updates an ITM2 vendor record.
func UpsertPreferredVendor(vendor *models.ITM2) error {
	return db.DB.Save(vendor).Error
}

// DeletePreferredVendor removes an ITM2 record.
func DeletePreferredVendor(itemCode string, lineNum int) error {
	return db.DB.Delete(&models.ITM2{}, "item_code = ? AND line_num = ?", itemCode, lineNum).Error
}

// ─────────────────────────────────────────────
// ITM12 — UoM per Item Mapping
// ─────────────────────────────────────────────

// ListItemUoMMapping returns all UoM mappings for a given item.
func ListItemUoMMapping(itemCode string) ([]models.ITM12, error) {
	var list []models.ITM12
	err := db.DB.Where("item_code = ?", itemCode).Find(&list).Error
	return list, err
}

// UpsertItemUoMMapping creates or updates an ITM12 mapping record.
func UpsertItemUoMMapping(mapping *models.ITM12) error {
	return db.DB.Save(mapping).Error
}

// DeleteItemUoMMapping removes an ITM12 record.
func DeleteItemUoMMapping(itemCode string, uomEntry uint) error {
	return db.DB.Delete(&models.ITM12{}, "item_code = ? AND uom_entry = ?", itemCode, uomEntry).Error
}

// ─────────────────────────────────────────────
// OSPP — Special Prices per Business Partner
// ─────────────────────────────────────────────

// ListSpecialPrices returns all special price records for a given item.
func ListSpecialPrices(itemCode string) ([]models.OSPP, error) {
	var list []models.OSPP
	err := db.DB.Where("item_code = ?", itemCode).Find(&list).Error
	return list, err
}

// UpsertSpecialPrice creates or updates an OSPP special price record.
func UpsertSpecialPrice(sp *models.OSPP) error {
	return db.DB.Save(sp).Error
}

// DeleteSpecialPrice removes an OSPP record by CardCode and ItemCode.
func DeleteSpecialPrice(cardCode, itemCode string) error {
	return db.DB.Delete(&models.OSPP{}, "card_code = ? AND item_code = ?", cardCode, itemCode).Error
}
