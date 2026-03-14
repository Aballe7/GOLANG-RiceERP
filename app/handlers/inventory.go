package handlers

import (
	"egglayererp/app/middleware"
	"egglayererp/app/models"
	"egglayererp/app/services/inventory"
)

// ─────────────────────────────────────────────
// Egg Inventory
// ─────────────────────────────────────────────

// GetEggInventory returns the computed egg on-hand map (per size).
// Requires CanAccess("Inventory").
func GetEggInventory() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Inventory") {
		return forbidden("Inventory")
	}
	result, err := inventory.EggInventory()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", result)
}

// ─────────────────────────────────────────────
// Supply Inventory
// ─────────────────────────────────────────────

// ListInventory returns all supply inventory records.
func ListInventory() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	items, err := inventory.ListInventory()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", items)
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

// GetItemCategory fetches a single ItemCategory by ID.
func GetItemCategory(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	cat, err := inventory.GetItemCategory(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", cat)
}

// CreateItemCategory creates a new ItemCategory.
func CreateItemCategory(cat models.ItemCategory) Response {
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

// UpdateItemCategory applies a partial update to an ItemCategory.
func UpdateItemCategory(id uint, updates map[string]interface{}) Response {
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

// DeleteItemCategory soft-deletes an ItemCategory.
func DeleteItemCategory(id uint) Response {
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
