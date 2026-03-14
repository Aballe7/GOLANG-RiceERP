package inventory

// This service handles:
// 1. Egg inventory calculation (three-way deduction: DO + AR + legacy SO)
// 2. Supply inventory (Inventory model) CRUD
// 3. ItemMaster CRUD
// 4. Feed/Vaccine stock management

import (
	"fmt"

	"egglayererp/app/constants"
	"egglayererp/app/db"
	"egglayererp/app/models"

	"gorm.io/gorm"
)

// ─────────────────────────────────────────────
// Egg Inventory (three-way deduction mirror of Python egg_inventory())
// ─────────────────────────────────────────────

// EggInventory computes the current on-hand egg count per size.
//
// Production  = SUM of each DailyLog egg-size column across all flocks.
// Deduction 1 = DeliveryOrderItems whose parent DeliveryOrder.status = 'Delivered'
//
//	(quantity is stored in trays → convert to pieces with TRAY_SIZE=30)
//
// Deduction 2 = ARInvoiceItems where ARInvoice.delivery_order_id IS NULL
//
//	AND ARInvoice.status != 'Cancelled'
//
// Deduction 3 = SalesOrderItems for SalesOrders that have
//
//	NO associated DeliveryOrders AND NO ARInvoices AND payment_status != 'Void'
//
// Result: map[eggSize] = max(0, produced - deducted)
func EggInventory() (map[string]int, error) {
	produced := map[string]int{}
	deducted := map[string]int{}

	// Initialise all known sizes to 0
	for _, sz := range constants.EGGSizes {
		produced[sz] = 0
		deducted[sz] = 0
	}

	// ── Production: aggregate DailyLog egg columns ────────────────────────
	type prodRow struct {
		Pewee        int
		Pullet       int
		Small        int
		Medium       int
		Large        int
		ExtraLarge   int
		Jumbo        int
		DoubleYolk   int
		CrackedDirty int
	}
	var prod prodRow
	if err := db.DB.Model(&models.DailyLog{}).
		Select(`
			SUM(pewee)        AS pewee,
			SUM(pullet)       AS pullet,
			SUM(small)        AS small,
			SUM(medium)       AS medium,
			SUM(large)        AS large,
			SUM(extra_large)  AS extra_large,
			SUM(jumbo)        AS jumbo,
			SUM(double_yolk)  AS double_yolk,
			SUM(cracked_dirty) AS cracked_dirty
		`).
		Scan(&prod).Error; err != nil {
		return nil, fmt.Errorf("egg_inventory: aggregate production: %w", err)
	}
	produced["Pewee"] = prod.Pewee
	produced["Pullet"] = prod.Pullet
	produced["Small"] = prod.Small
	produced["Medium"] = prod.Medium
	produced["Large"] = prod.Large
	produced["Extra Large"] = prod.ExtraLarge
	produced["Jumbo"] = prod.Jumbo
	produced["Double Yolk"] = prod.DoubleYolk
	produced["Cracked/Dirty"] = prod.CrackedDirty

	// ── Deduction 1: DeliveryOrderItems (status='Delivered') ──────────────
	// quantity_delivered is in trays; multiply by TraySize to get pieces.
	type doRow struct {
		SKU              string
		Unit             string
		QuantityDelivered float64
	}
	var doItems []doRow
	if err := db.DB.Table("do_line doi").
		Select("doi.sku, doi.unit, doi.quantity_delivered").
		Joins("JOIN delivery_order do ON do.id = doi.delivery_order_id").
		Where("do.status = ?", "Delivered").
		Scan(&doItems).Error; err != nil {
		return nil, fmt.Errorf("egg_inventory: deduction1 (DO): %w", err)
	}
	for _, row := range doItems {
		pieces := toPieces(row.SKU, row.Unit, row.QuantityDelivered)
		deducted[row.SKU] += pieces
	}

	// ── Deduction 2: ARInvoiceItems (standalone AR – no linked DO) ────────
	type arRow struct {
		SKU      string
		Unit     string
		Quantity float64
	}
	var arItems []arRow
	if err := db.DB.Table("ar_invoice_line aii").
		Select("aii.sku, aii.unit, aii.quantity").
		Joins("JOIN ar_invoice ai ON ai.id = aii.ar_invoice_id").
		Where("ai.delivery_order_id IS NULL AND ai.status != ?", "Cancelled").
		Scan(&arItems).Error; err != nil {
		return nil, fmt.Errorf("egg_inventory: deduction2 (AR): %w", err)
	}
	for _, row := range arItems {
		pieces := toPieces(row.SKU, row.Unit, row.Quantity)
		deducted[row.SKU] += pieces
	}

	// ── Deduction 3: SalesOrderItems (legacy SO – no DO, no AR, not Void) ─
	// Find SO IDs that have neither a DeliveryOrder nor an ARInvoice.
	type soIDRow struct {
		ID uint
	}
	var legacySOs []soIDRow
	subDO := db.DB.Table("delivery_order").Select("sales_order_id").
		Where("sales_order_id IS NOT NULL")
	subAR := db.DB.Table("ar_invoice").Select("sales_order_id").
		Where("sales_order_id IS NOT NULL")
	if err := db.DB.Table("sales_order").Select("id").
		Where("payment_status != ?", "Void").
		Where("id NOT IN (?)", subDO).
		Where("id NOT IN (?)", subAR).
		Scan(&legacySOs).Error; err != nil {
		return nil, fmt.Errorf("egg_inventory: deduction3 (legacy SO ids): %w", err)
	}

	if len(legacySOs) > 0 {
		soIDs := make([]uint, len(legacySOs))
		for i, s := range legacySOs {
			soIDs[i] = s.ID
		}
		type soItemRow struct {
			SKU      string
			Unit     string
			Quantity float64
		}
		var soItems []soItemRow
		if err := db.DB.Table("so_line").
			Select("sku, unit, quantity").
			Where("order_id IN ?", soIDs).
			Scan(&soItems).Error; err != nil {
			return nil, fmt.Errorf("egg_inventory: deduction3 (legacy SO items): %w", err)
		}
		for _, row := range soItems {
			pieces := toPieces(row.SKU, row.Unit, row.Quantity)
			deducted[row.SKU] += pieces
		}
	}

	// ── Result ─────────────────────────────────────────────────────────────
	result := make(map[string]int, len(constants.EGGSizes))
	for _, sz := range constants.EGGSizes {
		val := produced[sz] - deducted[sz]
		if val < 0 {
			val = 0
		}
		result[sz] = val
	}
	return result, nil
}

// toPieces converts a quantity (trays or pieces) to pieces.
// SKU is used only for future extensibility; conversion is unit-based.
func toPieces(sku, unit string, qty float64) int {
	_ = sku
	switch unit {
	case "Tray", "tray", "Trays", "trays":
		return int(qty * float64(constants.TraySize))
	default:
		return int(qty)
	}
}

// ─────────────────────────────────────────────
// Supply Inventory (Inventory model) CRUD
// ─────────────────────────────────────────────

// ListInventory returns all supply inventory records.
func ListInventory() ([]models.Inventory, error) {
	var items []models.Inventory
	err := db.DB.Order("category, item_name").Find(&items).Error
	return items, err
}

// UpdateInventory upserts an Inventory record (mirrors Python update_inventory).
// If a record with the same category+item_name exists it is updated; otherwise created.
func UpdateInventory(category, itemName string, quantity float64, unit string) error {
	var inv models.Inventory
	err := db.DB.
		Where("category = ? AND item_name = ?", category, itemName).
		First(&inv).Error

	if err == gorm.ErrRecordNotFound {
		inv = models.Inventory{
			Category: category,
			ItemName: itemName,
			Quantity: quantity,
			Unit:     unit,
		}
		return db.DB.Create(&inv).Error
	}
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"quantity": quantity,
	}
	if unit != "" {
		updates["unit"] = unit
	}
	return db.DB.Model(&inv).Updates(updates).Error
}

// ─────────────────────────────────────────────
// ItemMaster CRUD
// ─────────────────────────────────────────────

// ListItemMasters returns item masters. Pass activeOnly=true to filter inactive.
func ListItemMasters(activeOnly bool) ([]models.ItemMaster, error) {
	var items []models.ItemMaster
	q := db.DB.Order("category, name")
	if activeOnly {
		q = q.Where("is_active = 1")
	}
	return items, q.Find(&items).Error
}

// GetItemMaster fetches a single ItemMaster by primary key.
func GetItemMaster(id uint) (*models.ItemMaster, error) {
	var item models.ItemMaster
	err := db.DB.First(&item, id).Error
	return &item, err
}

// CreateItemMaster inserts a new ItemMaster.
func CreateItemMaster(item *models.ItemMaster) error {
	item.IsActive = true
	return db.DB.Create(item).Error
}

// UpdateItemMaster applies a partial update to an ItemMaster.
func UpdateItemMaster(id uint, updates map[string]interface{}) error {
	return db.DB.Model(&models.ItemMaster{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteItemMaster soft-deletes an ItemMaster by setting is_active=false.
func DeleteItemMaster(id uint) error {
	return db.DB.Model(&models.ItemMaster{}).Where("id = ?", id).
		Update("is_active", false).Error
}

// ─────────────────────────────────────────────
// ItemCategory CRUD
// ─────────────────────────────────────────────

// ListItemCategories returns all active item categories ordered by name.
func ListItemCategories(activeOnly bool) ([]models.ItemCategory, error) {
	var cats []models.ItemCategory
	q := db.DB.Order("name")
	if activeOnly {
		q = q.Where("is_active = 1")
	}
	return cats, q.Find(&cats).Error
}

// GetItemCategory fetches a single ItemCategory by ID.
func GetItemCategory(id uint) (*models.ItemCategory, error) {
	var cat models.ItemCategory
	err := db.DB.First(&cat, id).Error
	return &cat, err
}

// CreateItemCategory inserts a new ItemCategory.
func CreateItemCategory(cat *models.ItemCategory) error {
	cat.IsActive = true
	return db.DB.Create(cat).Error
}

// UpdateItemCategory applies a partial update.
func UpdateItemCategory(id uint, updates map[string]interface{}) error {
	return db.DB.Model(&models.ItemCategory{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteItemCategory soft-deletes an ItemCategory (sets is_active=false).
func DeleteItemCategory(id uint) error {
	return db.DB.Model(&models.ItemCategory{}).Where("id = ?", id).
		Update("is_active", false).Error
}

// ─────────────────────────────────────────────
// Feed / Vaccine stock helpers
// ─────────────────────────────────────────────

// ListFeedStocks returns all FeedStock records.
func ListFeedStocks() ([]models.FeedStock, error) {
	var stocks []models.FeedStock
	err := db.DB.Find(&stocks).Error
	return stocks, err
}

// UpdateFeedStock adjusts the sack count for a named feed (upsert).
func UpdateFeedStock(name string, totalSacks, kgPerSack float64) error {
	var fs models.FeedStock
	err := db.DB.Where("name = ?", name).First(&fs).Error
	if err == gorm.ErrRecordNotFound {
		fs = models.FeedStock{Name: name, TotalSacks: totalSacks, KgPerSack: kgPerSack}
		return db.DB.Create(&fs).Error
	}
	if err != nil {
		return err
	}
	return db.DB.Model(&fs).Updates(map[string]interface{}{
		"total_sacks": totalSacks,
		"kg_per_sack": kgPerSack,
	}).Error
}

// ListVaccineStocks returns all VaccineStock records.
func ListVaccineStocks() ([]models.VaccineStock, error) {
	var stocks []models.VaccineStock
	err := db.DB.Find(&stocks).Error
	return stocks, err
}

// UpdateVaccineStock adjusts the quantity for a named vaccine (upsert).
func UpdateVaccineStock(name string, quantity int) error {
	var vs models.VaccineStock
	err := db.DB.Where("name = ?", name).First(&vs).Error
	if err == gorm.ErrRecordNotFound {
		vs = models.VaccineStock{Name: name, Quantity: quantity}
		return db.DB.Create(&vs).Error
	}
	if err != nil {
		return err
	}
	return db.DB.Model(&vs).Update("quantity", quantity).Error
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
	UgpCode string               `json:"ugp_code"`
	UgpName string               `json:"ugp_name"`
	BaseUom int                  `json:"base_uom"`
	Lines   []UoMGroupLineInput  `json:"lines"`
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
