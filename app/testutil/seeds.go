package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ricemill/app/db"
	"ricemill/app/models"
)

// ─────────────────────────────────────────────────────────────────────────────
// Low-level seed helpers
// ─────────────────────────────────────────────────────────────────────────────

// GLAcct inserts a POSTING-type GL account and returns its auto-assigned ID.
func GLAcct(t *testing.T, code, name, section, normalBal string) uint {
	t.Helper()
	a := models.GLAccount{
		Code:          code,
		Name:          name,
		Section:       section,
		AccountType:   "POSTING",
		NormalBalance: normalBal,
		IsActive:      true,
	}
	require.NoError(t, db.DB.Create(&a).Error)
	return a.ID
}

// AcctDet inserts one account determination rule.
// Pass nil for category to create a catch-all (item_category IS NULL) rule.
func AcctDet(t *testing.T, module, event string, category *string, glID uint) {
	t.Helper()
	require.NoError(t, db.DB.Create(&models.AccountDetermination{
		Module:       module,
		PostingEvent: event,
		ItemCategory: category,
		GLAccountID:  glID,
		IsActive:     true,
	}).Error)
}

// SeedOITM inserts a minimal OITM (Item Master) row for tests that exercise
// inventory and COGS paths (on_hand decrement, COGS JE, etc.).
// Raw SQL is used so callers don't need to populate every GORM model field.
// Returns itemCode for convenient chaining in test setup.
func SeedOITM(t *testing.T, itemCode, itemName string, avgPrice float64) string {
	t.Helper()
	require.NoError(t, db.DB.Exec(
		`INSERT INTO oitm (item_code, item_name, avg_price, on_hand, sell_item, valid_for)
		 VALUES (?, ?, ?, 1000, 'Y', 'Y')`,
		itemCode, itemName, avgPrice,
	).Error)
	return itemCode
}

// SeedOITB inserts an item category (OITB) row and returns its auto-assigned
// itms_grp_cod.  forSales controls whether items in this category may appear
// on Sales Orders (CreateSalesOrder validates this before saving).
// Raw SQL is used so callers don't need every GORM model field in the DDL.
func SeedOITB(t *testing.T, name string, forSales bool) int {
	t.Helper()
	forSalesInt := 0
	if forSales {
		forSalesInt = 1
	}
	require.NoError(t, db.DB.Exec(
		`INSERT INTO oitb (itms_grp_nam, is_active, for_sales, for_purchasing, for_inventory)
		 VALUES (?, 1, ?, 1, 1)`,
		name, forSalesInt,
	).Error)
	var id int
	require.NoError(t, db.DB.Raw(
		"SELECT itms_grp_cod FROM oitb WHERE itms_grp_nam = ?", name,
	).Scan(&id).Error)
	return id
}

// SeedOITMInGroup inserts a minimal OITM row belonging to the specified
// itms_grp_cod category.  Use alongside SeedOITB when a test needs to verify
// category-visibility guards (e.g. for_sales = false blocks SO creation).
// Returns itemCode for convenient chaining.
func SeedOITMInGroup(t *testing.T, itemCode, itemName string, avgPrice float64, grpCod int) string {
	t.Helper()
	require.NoError(t, db.DB.Exec(
		`INSERT INTO oitm (item_code, item_name, avg_price, on_hand, itms_grp_cod, sell_item, valid_for)
		 VALUES (?, ?, ?, 1000, ?, 'Y', 'Y')`,
		itemCode, itemName, avgPrice, grpCod,
	).Error)
	return itemCode
}

// ─────────────────────────────────────────────────────────────────────────────
// Module-level seed bundles
// ─────────────────────────────────────────────────────────────────────────────

// SeedSalesAccounting seeds the minimum GL accounts and determination rules
// required for the full O2C cycle: SO → DO → AR Invoice → Collection.
func SeedSalesAccounting(t *testing.T) {
	t.Helper()

	ar   := GLAcct(t, "1-1200", "Accounts Receivable",    "ASSET",     "DEBIT")
	inv  := GLAcct(t, "1-1330", "Inventory — Milled Rice", "ASSET",    "DEBIT")
	rev  := GLAcct(t, "4-1000", "Rice Sales Revenue",      "REVENUE",  "CREDIT")
	cogs := GLAcct(t, "5-1000", "Cost of Goods Sold",      "EXPENSE",  "DEBIT")
	cash := GLAcct(t, "1-1100", "Cash on Hand",            "ASSET",    "DEBIT")
	vat  := GLAcct(t, "2-1200", "Output VAT Payable",      "LIABILITY", "CREDIT")

	AcctDet(t, "SALES",   "AR_RECEIVABLE",      nil, ar)
	AcctDet(t, "SALES",   "SALES_REVENUE",       nil, rev)
	AcctDet(t, "SALES",   "COGS",                nil, cogs)
	AcctDet(t, "SALES",   "INVENTORY_SOLD",      nil, inv)
	AcctDet(t, "SALES",   "OUTPUT_VAT",          nil, vat)
	AcctDet(t, "SALES",   "COLLECTION_CLEARING", nil, ar)
	AcctDet(t, "SALES",   "SALES_DISCOUNT",      nil, rev)
	AcctDet(t, "BANKING", "CASH_INFLOW",         nil, cash)

	require.NoError(t, db.DB.Create(&models.PaymentMethodAccount{
		PaymentMethod: "Cash",
		Direction:     "INFLOW",
		GLAccountID:   cash,
		IsActive:      true,
	}).Error)
}

// SeedProductionAccounting seeds the GL accounts and determination rules needed by
// the Milling / Work-Order lifecycle: WIP, input/output inventory, and the
// conversion-cost absorption account.
func SeedProductionAccounting(t *testing.T) {
	t.Helper()

	wip    := GLAcct(t, "1-1350", "Work in Process",           "ASSET",   "DEBIT")
	inInv  := GLAcct(t, "1-1310", "Inventory — Palay",         "ASSET",   "DEBIT")
	outInv := GLAcct(t, "1-1330", "Inventory — Milled Rice",   "ASSET",   "DEBIT")
	ovh    := GLAcct(t, "5-2100", "Milling Overhead Absorbed", "EXPENSE", "CREDIT")

	AcctDet(t, "PRODUCTION", "MILLING_WIP",      nil, wip)
	AcctDet(t, "PRODUCTION", "MILLING_INPUT",    nil, inInv)
	AcctDet(t, "PRODUCTION", "MILLING_OUTPUT",   nil, outInv)
	AcctDet(t, "PRODUCTION", "MILLING_OVERHEAD", nil, ovh)
}

// SeedFarmSetting upserts one farm_settings key/value pair (recovery band,
// conversion-cost standard, …).
func SeedFarmSetting(t *testing.T, key, value string) {
	t.Helper()
	require.NoError(t, db.DB.Exec(
		`INSERT INTO farm_settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = ?`,
		key, value, value,
	).Error)
}

// SeedProdItem inserts an OITM tuned for production tests: caller controls on_hand,
// avg_price, the mak_item flag, and the category. Returns the row's auto-assigned ID
// (milling/work-order services address items by ID).
func SeedProdItem(t *testing.T, itemCode, itemName string, avgPrice, onHand float64, makItem bool, grpCod int) uint {
	t.Helper()
	mak := "N"
	if makItem {
		mak = "Y"
	}
	require.NoError(t, db.DB.Exec(
		`INSERT INTO oitm (item_code, item_name, avg_price, on_hand, mak_item, itms_grp_cod, invntry_uom, sell_item, valid_for)
		 VALUES (?, ?, ?, ?, ?, ?, 'kg', 'Y', 'Y')`,
		itemCode, itemName, avgPrice, onHand, mak, grpCod,
	).Error)
	var id uint
	require.NoError(t, db.DB.Raw("SELECT id FROM oitm WHERE item_code = ?", itemCode).Scan(&id).Error)
	return id
}

// SeedOITW seeds a per-warehouse stock row so ReleaseWorkOrder's per-warehouse guard
// has a balance to check against.
func SeedOITW(t *testing.T, itemCode, whsCode string, onHand float64) {
	t.Helper()
	require.NoError(t, db.DB.Exec(
		`INSERT INTO oitw (item_code, whs_code, on_hand) VALUES (?, ?, ?)`,
		itemCode, whsCode, onHand,
	).Error)
}

// SeedUoM inserts a UoM master (ouom) row and returns its uom_entry.
func SeedUoM(t *testing.T, code, name string) uint {
	t.Helper()
	require.NoError(t, db.DB.Exec(
		`INSERT INTO ouom (uom_code, uom_name) VALUES (?, ?)`, code, name).Error)
	var id uint
	require.NoError(t, db.DB.Raw(`SELECT uom_entry FROM ouom WHERE uom_code = ?`, code).Scan(&id).Error)
	return id
}

// SeedUoMGroup inserts a UoM group (ougp) row and returns its ugp_entry.
func SeedUoMGroup(t *testing.T, code, name string, baseUom uint) uint {
	t.Helper()
	require.NoError(t, db.DB.Exec(
		`INSERT INTO ougp (ugp_code, ugp_name, base_uom) VALUES (?, ?, ?)`, code, name, baseUom).Error)
	var id uint
	require.NoError(t, db.DB.Raw(`SELECT ugp_entry FROM ougp WHERE ugp_code = ?`, code).Scan(&id).Error)
	return id
}

// SeedItemPrice seeds an ITM1 price-list row (used by relative-sales-value tests).
func SeedItemPrice(t *testing.T, itemCode string, priceList int, price float64) {
	t.Helper()
	require.NoError(t, db.DB.Exec(
		`INSERT INTO itm1 (item_code, price_list, price) VALUES (?, ?, ?)`,
		itemCode, priceList, price,
	).Error)
}

// SeedPurchasingAccounting seeds the minimum accounting tables for a P2P cycle.
// It is a drop-in replacement for the raw seedAccounts() helper in
// purchasing/service_test.go; new purchasing tests should prefer this.
func SeedPurchasingAccounting(t *testing.T) {
	t.Helper()

	grni   := GLAcct(t, "1-1400", "Goods Received Not Invoiced", "ASSET",     "DEBIT")
	invRec := GLAcct(t, "1-1310", "Inventory — Palay",           "ASSET",     "DEBIT")
	ap     := GLAcct(t, "2-1100", "Accounts Payable",            "LIABILITY", "CREDIT")
	apClr  := GLAcct(t, "2-1101", "AP Clearing",                 "LIABILITY", "CREDIT")
	cash   := GLAcct(t, "1-1100", "Cash on Hand",                "ASSET",     "DEBIT")

	AcctDet(t, "PURCHASING", "GRNI",               nil, grni)
	AcctDet(t, "PURCHASING", "INVENTORY_RECEIVED", nil, invRec)
	AcctDet(t, "PURCHASING", "AP_PAYABLE",         nil, ap)
	AcctDet(t, "PURCHASING", "AP_CLEARING",        nil, apClr)
	AcctDet(t, "BANKING",    "CASH_OUTFLOW",       nil, cash)

	require.NoError(t, db.DB.Create(&models.PaymentMethodAccount{
		PaymentMethod: "Cash",
		Direction:     "OUTFLOW",
		GLAccountID:   cash,
		IsActive:      true,
	}).Error)
}
