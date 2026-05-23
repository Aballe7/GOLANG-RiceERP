package handlers

import (
	audit "ricemill/app/services/audit"
	"ricemill/app/middleware"
	"ricemill/app/models"
	"ricemill/app/services/inventory"
)

func checkInventory() (Response, bool) {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized(), false
	}
	if !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory"), false
	}
	return Response{}, true
}

// ─────────────────────────────────────────────
// ItemCategory
// ─────────────────────────────────────────────

// ListItemCategories returns item categories.
func ListItemCategories(activeOnly bool) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	cats, err := inventory.ListItemCategories(activeOnly)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", cats)
}

// GetItemCategory fetches a single OITB (Item Group) by ItmsGrpCod (int).
func GetItemCategory(id int) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	cat, err := inventory.GetItemCategory(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", cat)
}

// CreateItemCategory creates a new OITB (Item Group).
func CreateItemCategory(cat models.OITB) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.CreateItemCategory(&cat); err != nil {
		return errResponse(err)
	}
	return okResponse("Category created", cat)
}

// UpdateItemCategory applies a partial update to an OITB (Item Group).
func UpdateItemCategory(id int, updates map[string]interface{}) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.UpdateItemCategory(id, updates); err != nil {
		return errResponse(err)
	}
	return okResponse("Category updated", nil)
}

// DeleteItemCategory soft-deletes an OITB (Item Group).
func DeleteItemCategory(id int) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() {
		return forbidden("Admin")
	}
	if err := inventory.DeleteItemCategory(id); err != nil {
		return errResponse(err)
	}
	return okResponse("Category deactivated", nil)
}

// ─────────────────────────────────────────────
// ItemMaster
// ─────────────────────────────────────────────

// ListProductionItems returns active items flagged as usable in production/milling.
func ListProductionItems() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	items, err := inventory.ListProductionItems()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", items)
}

// ListItemMasters returns item master records.
// Pass activeOnly=true to filter to active items only.
func ListItemMasters(activeOnly bool) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	items, err := inventory.ListItemMasters(activeOnly)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", items)
}

// ListSalesItems returns active items whose category is enabled for sales.
func ListSalesItems() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	items, err := inventory.ListSalesItems()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", items)
}

// ListPurchasingItems returns active items whose category is enabled for purchasing.
func ListPurchasingItems() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	items, err := inventory.ListPurchasingItems()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", items)
}

// ListInventoryItems returns active items whose category is enabled for inventory.
func ListInventoryItems() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	items, err := inventory.ListInventoryItems()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", items)
}

// ListItemCategoriesForModule returns active categories visible in a specific module.
func ListItemCategoriesForModule(module string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	cats, err := inventory.ListItemCategoriesForModule(module)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", cats)
}

// GetItemMaster fetches a single ItemMaster by ID.
func GetItemMaster(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	item, err := inventory.GetItemMaster(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", item)
}

// CreateItemMaster creates a new ItemMaster.
// Requires Admin or Inventory module access.
func CreateItemMaster(item models.ItemMaster) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.CreateItemMaster(&item); err != nil {
		return errResponse(err)
	}
	return okResponse("Item master created", item)
}

// UpdateItemMaster applies a partial update to an ItemMaster.
// Requires Admin or Inventory module access.
func UpdateItemMaster(id uint, updates map[string]interface{}) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.UpdateItemMaster(id, updates); err != nil {
		return errResponse(err)
	}
	return okResponse("Item master updated", nil)
}

// DeleteItemMaster soft-deletes an ItemMaster (sets is_active=false).
// Requires Admin role.
func DeleteItemMaster(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() {
		return forbidden("Admin")
	}
	if err := inventory.DeleteItemMaster(id); err != nil {
		return errResponse(err)
	}
	return okResponse("Item master deactivated", nil)
}

// ─────────────────────────────────────────────
// ItemUoMPrice (ITM9 equivalent)
// ─────────────────────────────────────────────

// UpsertItemUoMPriceRequest — ITM9 DTO. Uses item_code (string) not item_id (uint).
type UpsertItemUoMPriceRequest struct {
	ItemCode string  `json:"item_code"`
	UomEntry uint    `json:"uom_entry"`
	Price    float64 `json:"price"`
	Factor   float64 `json:"factor"`
}

func GetItemUoMPrices(itemCode string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	prices, err := inventory.GetItemUoMPrices(itemCode)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", prices)
}

func UpsertItemUoMPrice(req UpsertItemUoMPriceRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.UpsertItemUoMPrice(req.ItemCode, req.UomEntry, req.Price, req.Factor); err != nil {
		return errResponse(err)
	}
	return okResponse("UoM price saved", nil)
}

func DeleteItemUoMPrice(itemCode string, uomEntry uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.DeleteItemUoMPrice(itemCode, uomEntry); err != nil {
		return errResponse(err)
	}
	return okResponse("UoM price deleted", nil)
}

// ─────────────────────────────────────────────
// ITM12 — UoM per Item Mapping
// ─────────────────────────────────────────────

// GetItemUoMMapping returns the ITM12 rows for an item (I/S/P roles).
func GetItemUoMMapping(itemCode string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	mappings, err := inventory.ListItemUoMMapping(itemCode)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", mappings)
}

// ─────────────────────────────────────────────
// OWHS — Warehouse Master
// ─────────────────────────────────────────────

func ListWarehouses() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	list, err := inventory.ListWarehouses()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

func GetWarehouse(whsCode string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	whs, err := inventory.GetWarehouse(whsCode)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", whs)
}

func CreateWarehouse(whs models.OWHS) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.CreateWarehouse(&whs); err != nil {
		return errResponse(err)
	}
	return okResponse("Warehouse created", whs)
}

func UpdateWarehouse(whsCode string, updates map[string]interface{}) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.UpdateWarehouse(whsCode, updates); err != nil {
		return errResponse(err)
	}
	return okResponse("Warehouse updated", nil)
}

// ─────────────────────────────────────────────
// OITW — Per-Warehouse Stock Queries
// ─────────────────────────────────────────────

func GetItemWarehouseStock(itemCode string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	list, err := inventory.GetItemWarehouseStock(itemCode)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

func GetWarehouseStock(whsCode string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	list, err := inventory.GetWarehouseStock(whsCode)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

func ListAllWarehouseStock() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	list, err := inventory.ListAllWarehouseStock()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

func ListOIVL(itemCode string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	list, err := inventory.ListOIVL(itemCode)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

func UpdateOITWMinMax(itemCode, whsCode string, minStock, maxStock float64) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.UpdateOITWMinMax(itemCode, whsCode, minStock, maxStock); err != nil {
		return errResponse(err)
	}
	return okResponse("Stock thresholds updated", nil)
}

// ─────────────────────────────────────────────
// ITM1 — Item Prices per Price List
// ─────────────────────────────────────────────

type UpsertItemPriceRequest struct {
	ItemCode  string  `json:"item_code"`
	PriceList int     `json:"price_list"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
}

func ListItemPrices(itemCode string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	list, err := inventory.ListItemPrices(itemCode)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

func UpsertItemPrice(req UpsertItemPriceRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.UpsertItemPrice(req.ItemCode, req.PriceList, req.Price, req.Currency); err != nil {
		return errResponse(err)
	}
	return okResponse("Item price saved", nil)
}

// ─────────────────────────────────────────────
// OSPP — Special Prices per Business Partner
// ─────────────────────────────────────────────

func ListSpecialPrices(itemCode string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	list, err := inventory.ListSpecialPrices(itemCode)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

func UpsertSpecialPrice(sp models.OSPP) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.UpsertSpecialPrice(&sp); err != nil {
		return errResponse(err)
	}
	return okResponse("Special price saved", nil)
}

func DeleteSpecialPrice(cardCode, itemCode string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.DeleteSpecialPrice(cardCode, itemCode); err != nil {
		return errResponse(err)
	}
	return okResponse("Special price deleted", nil)
}

// ─────────────────────────────────────────────
// UoM Master (OUOM)
// ─────────────────────────────────────────────

func ListUoMMasters() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	items, err := inventory.ListUoMMasters()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", items)
}

func GetUoMMaster(entry uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	item, err := inventory.GetUoMMaster(entry)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", item)
}

func CreateUoMMaster(item models.UoMMaster) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.CreateUoMMaster(&item); err != nil {
		return errResponse(err)
	}
	return okResponse("Unit of measure created", item)
}

func UpdateUoMMaster(entry uint, updates map[string]interface{}) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.UpdateUoMMaster(entry, updates); err != nil {
		return errResponse(err)
	}
	return okResponse("Unit of measure updated", nil)
}

func DeleteUoMMaster(entry uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() {
		return forbidden("Admin")
	}
	if err := inventory.DeleteUoMMaster(entry); err != nil {
		return errResponse(err)
	}
	return okResponse("Unit of measure deleted", nil)
}

// ─────────────────────────────────────────────
// UoM Groups (OUGP)
// ─────────────────────────────────────────────

func ListUoMGroups() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	groups, err := inventory.ListUoMGroups()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", groups)
}

func GetUoMGroup(entry uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	g, err := inventory.GetUoMGroup(entry)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", g)
}

func CreateUoMGroup(req inventory.CreateUoMGroupRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	g, err := inventory.CreateUoMGroup(req)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("UoM group created", g)
}

func UpdateUoMGroup(entry uint, ugpCode, ugpName string, baseUom int, lines []inventory.UoMGroupLineInput) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() && !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	if err := inventory.UpdateUoMGroup(entry, ugpCode, ugpName, baseUom, lines); err != nil {
		return errResponse(err)
	}
	return okResponse("UoM group updated", nil)
}

func DeleteUoMGroup(entry uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.IsAdmin() {
		return forbidden("Admin")
	}
	if err := inventory.DeleteUoMGroup(entry); err != nil {
		return errResponse(err)
	}
	return okResponse("UoM group deleted", nil)
}

// ─────────────────────────────────────────────
// Goods Receipt (OIGN / IGN1)
// ─────────────────────────────────────────────

// ListGoodsReceipts returns all goods receipts.
func ListGoodsReceipts() Response {
	if r, ok := checkInventory(); !ok {
		return r
	}
	list, err := inventory.ListGoodsReceipts()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

// GetGoodsReceipt fetches a single goods receipt with its lines.
func GetGoodsReceipt(id uint) Response {
	if r, ok := checkInventory(); !ok {
		return r
	}
	gr, err := inventory.GetGoodsReceipt(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", gr)
}

// CreateGoodsReceipt creates a new goods receipt and increases stock.
func CreateGoodsReceipt(req inventory.CreateGoodsReceiptRequest) Response {
	if r, ok := checkInventory(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	gr, err := inventory.CreateGoodsReceipt(req, userID)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Inventory", "GoodsReceipt", gr.GRNumber, "Goods receipt created", gr.ID)
	return okResponse("Goods receipt created", gr)
}

// CancelGoodsReceipt cancels an open goods receipt and reverses the stock.
func CancelGoodsReceipt(id uint) Response {
	if r, ok := checkInventory(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := inventory.CancelGoodsReceipt(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Inventory", "GoodsReceipt", "", "Goods receipt cancelled", id)
	return okResponse("Goods receipt cancelled", nil)
}

// ─────────────────────────────────────────────
// Goods Issue (OIGE / IGE1)
// ─────────────────────────────────────────────

// ListGoodsIssues returns all goods issues.
func ListGoodsIssues() Response {
	if r, ok := checkInventory(); !ok {
		return r
	}
	list, err := inventory.ListGoodsIssues()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

// GetGoodsIssue fetches a single goods issue with its lines.
func GetGoodsIssue(id uint) Response {
	if r, ok := checkInventory(); !ok {
		return r
	}
	gi, err := inventory.GetGoodsIssue(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", gi)
}

// CreateGoodsIssue creates a new goods issue and decreases stock.
func CreateGoodsIssue(req inventory.CreateGoodsIssueRequest) Response {
	if r, ok := checkInventory(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	gi, err := inventory.CreateGoodsIssue(req, userID)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Inventory", "GoodsIssue", gi.GINumber, "Goods issue created", gi.ID)
	return okResponse("Goods issue created", gi)
}

// CancelGoodsIssue cancels an open goods issue and restores the stock.
func CancelGoodsIssue(id uint) Response {
	if r, ok := checkInventory(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := inventory.CancelGoodsIssue(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Inventory", "GoodsIssue", "", "Goods issue cancelled", id)
	return okResponse("Goods issue cancelled", nil)
}
