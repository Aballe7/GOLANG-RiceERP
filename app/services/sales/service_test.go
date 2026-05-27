package sales

// Integration tests for the O2C (Order-to-Cash) sales cycle.
//
// DB strategy: each test calls testutil.SetupDB, which opens a fresh in-memory
// SQLite instance and runs AutoMigrateAll.  No OITM records are seeded, so
// ConfirmDeliveryOrder's OITM lookup silently hits `continue` — bypassing the
// MySQL-specific ON DUPLICATE KEY / GREATEST() SQL in the inventory layer.  The
// financial and status-transition logic is still fully exercised.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ricemill/app/db"
	"ricemill/app/models"
	"ricemill/app/testutil"
)

// ─────────────────────────────────────────────
// Customer
// ─────────────────────────────────────────────

func TestCreateCustomer_PersistsAndIsActive(t *testing.T) {
	testutil.SetupDB(t)

	cust := &models.Customer{Name: "Agro Traders", CustomerType: "Account"}
	require.NoError(t, CreateCustomer(cust))
	require.NotZero(t, cust.ID)
	require.True(t, cust.IsActive)
}

func TestUpdateCustomer_ChangesName(t *testing.T) {
	testutil.SetupDB(t)

	cust := &models.Customer{Name: "Old Name"}
	require.NoError(t, CreateCustomer(cust))

	require.NoError(t, UpdateCustomer(cust.ID, map[string]interface{}{"name": "New Name"}))

	updated, err := GetCustomer(cust.ID)
	require.NoError(t, err)
	require.Equal(t, "New Name", updated.Name)
}

func TestDeleteCustomer_SoftDeletesIsActive(t *testing.T) {
	testutil.SetupDB(t)

	cust := &models.Customer{Name: "Temp Customer"}
	require.NoError(t, CreateCustomer(cust))

	require.NoError(t, DeleteCustomer(cust.ID))

	updated, err := GetCustomer(cust.ID)
	require.NoError(t, err)
	require.False(t, updated.IsActive, "soft-delete must set is_active = false")
}

// ─────────────────────────────────────────────
// PriceGroup CRUD
// ─────────────────────────────────────────────

// TestCreatePriceGroup_PersistsAndIsActive verifies that CreatePriceGroup
// assigns an ID, forces IsActive=true, and stores the name.
func TestCreatePriceGroup_PersistsAndIsActive(t *testing.T) {
	testutil.SetupDB(t)

	pg := &models.PriceGroup{Name: "Premium Buyers", Description: "Top-tier pricing"}
	pg.IsActive = false // service must override this to true

	require.NoError(t, CreatePriceGroup(pg))
	require.NotZero(t, pg.ID)

	loaded, err := GetPriceGroup(pg.ID)
	require.NoError(t, err)
	require.Equal(t, "Premium Buyers", loaded.Name)
	require.True(t, loaded.IsActive, "CreatePriceGroup must force IsActive = true")
}

// TestUpdatePriceGroup_ChangesFieldsAndPersists verifies that UpdatePriceGroup
// writes name, description, and is_active changes through correctly.
func TestUpdatePriceGroup_ChangesFieldsAndPersists(t *testing.T) {
	testutil.SetupDB(t)

	pg := &models.PriceGroup{Name: "Old Name"}
	require.NoError(t, CreatePriceGroup(pg))

	require.NoError(t, UpdatePriceGroup(pg.ID, map[string]interface{}{
		"name":        "New Name",
		"description": "Updated description",
	}))

	loaded, err := GetPriceGroup(pg.ID)
	require.NoError(t, err)
	require.Equal(t, "New Name", loaded.Name)
	require.Equal(t, "Updated description", loaded.Description)
}

// TestDeletePriceGroup_SoftDeletesIsActive verifies that DeletePriceGroup sets
// is_active=false without removing the record from the DB.
func TestDeletePriceGroup_SoftDeletesIsActive(t *testing.T) {
	testutil.SetupDB(t)

	pg := &models.PriceGroup{Name: "Temp Group"}
	require.NoError(t, CreatePriceGroup(pg))

	require.NoError(t, DeletePriceGroup(pg.ID))

	loaded, err := GetPriceGroup(pg.ID)
	require.NoError(t, err)
	require.False(t, loaded.IsActive, "soft-delete must set is_active = false, not remove the record")
}

// TestUpsertPriceGroupItem_CreatesThenUpdatesOnSameKey verifies the full
// upsert behaviour: first call creates a new PriceGroupItem; a second call
// with the same (pgID, eggSize, unit) key updates the price in place rather
// than inserting a duplicate row.
func TestUpsertPriceGroupItem_CreatesThenUpdatesOnSameKey(t *testing.T) {
	testutil.SetupDB(t)

	pg := &models.PriceGroup{Name: "Agri Buyers"}
	require.NoError(t, CreatePriceGroup(pg))

	// First call — must create a new item
	require.NoError(t, UpsertPriceGroupItem(pg.ID, "Large", "Tray", 120.0))

	loaded, _ := GetPriceGroup(pg.ID)
	require.Len(t, loaded.Items, 1, "first upsert must create exactly one item")
	require.Equal(t, 120.0, loaded.Items[0].Price)

	// Second call — same key, different price; must update, not insert
	require.NoError(t, UpsertPriceGroupItem(pg.ID, "Large", "Tray", 135.0))

	loaded, _ = GetPriceGroup(pg.ID)
	require.Len(t, loaded.Items, 1, "second upsert must update in place — no duplicate row")
	require.Equal(t, 135.0, loaded.Items[0].Price, "price must reflect the updated value")
}

// TestUpsertPriceGroupItem_DifferentKeysCreateSeparateItems verifies that
// upserts with different (eggSize, unit) combinations each produce their own row.
func TestUpsertPriceGroupItem_DifferentKeysCreateSeparateItems(t *testing.T) {
	testutil.SetupDB(t)

	pg := &models.PriceGroup{Name: "Multi-SKU Buyers"}
	require.NoError(t, CreatePriceGroup(pg))

	require.NoError(t, UpsertPriceGroupItem(pg.ID, "Large", "Tray", 120.0))
	require.NoError(t, UpsertPriceGroupItem(pg.ID, "Medium", "Tray", 100.0))
	require.NoError(t, UpsertPriceGroupItem(pg.ID, "Large", "Piece", 10.0))

	loaded, _ := GetPriceGroup(pg.ID)
	require.Len(t, loaded.Items, 3, "distinct (eggSize, unit) keys must produce separate rows")
}

// TestListPriceGroups_ActiveOnlyFilter verifies the activeOnly flag:
// false returns all groups; true returns only is_active = 1 rows.
func TestListPriceGroups_ActiveOnlyFilter(t *testing.T) {
	testutil.SetupDB(t)

	pg1 := &models.PriceGroup{Name: "Active Group"}
	pg2 := &models.PriceGroup{Name: "Inactive Group"}
	require.NoError(t, CreatePriceGroup(pg1))
	require.NoError(t, CreatePriceGroup(pg2))
	require.NoError(t, DeletePriceGroup(pg2.ID)) // soft-delete → is_active = false

	all, err := ListPriceGroups(false)
	require.NoError(t, err)
	require.Len(t, all, 2, "activeOnly=false must return both groups")

	active, err := ListPriceGroups(true)
	require.NoError(t, err)
	require.Len(t, active, 1, "activeOnly=true must return only the active group")
	require.Equal(t, "Active Group", active[0].Name)
}

// TestGetPriceGroup_LoadsItemsAndReturnsCorrectPrice verifies that GetPriceGroup
// eagerly loads PriceGroupItems and that the in-memory GetPrice helper finds the
// right line by (eggSize, unit) key.
func TestGetPriceGroup_LoadsItemsAndReturnsCorrectPrice(t *testing.T) {
	testutil.SetupDB(t)

	pg := &models.PriceGroup{Name: "Egg Buyers"}
	require.NoError(t, CreatePriceGroup(pg))
	require.NoError(t, UpsertPriceGroupItem(pg.ID, "Large", "Tray", 120.0))
	require.NoError(t, UpsertPriceGroupItem(pg.ID, "Medium", "Tray", 100.0))

	loaded, err := GetPriceGroup(pg.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Items, 2)

	// GetPrice is a pure in-memory helper on the loaded struct
	p := loaded.GetPrice("Large", "Tray")
	require.NotNil(t, p, "GetPrice must find the Large/Tray line")
	require.Equal(t, 120.0, *p)

	require.Nil(t, loaded.GetPrice("Jumbo", "Tray"), "GetPrice must return nil for unknown key")
}

// ─────────────────────────────────────────────
// Sales Order — creation
// ─────────────────────────────────────────────

func TestCreateSalesOrder_ComputesGrandTotal(t *testing.T) {
	testutil.SetupDB(t)

	items := []models.SalesOrderItem{
		{SKU: "RICE-001", Unit: "bag", Quantity: 100, PricePerUnit: 100, LineTotal: 10_000},
		{SKU: "BRAN-001", Unit: "sack", Quantity: 50, PricePerUnit: 80, LineTotal: 4_000},
	}
	so, err := CreateSalesOrder(
		"2025-01-01",
		nil, "Test Customer", "", "", "Cash", "30d", nil, "",
		items, nil,
	)
	require.NoError(t, err)
	require.NotZero(t, so.ID)
	require.Equal(t, 14_000.0, so.GrandTotal)
	require.Equal(t, "Draft", so.DocStatus)
	require.Contains(t, so.SalesOrderNumber, "SO-", "doc number must carry the SO prefix")
}

func TestCreateSalesOrder_OpenQtyMatchesQuantity(t *testing.T) {
	testutil.SetupDB(t)

	so, err := CreateSalesOrder(
		"2025-01-01", nil, "Customer A", "", "", "Cash", "", nil, "",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 60, PricePerUnit: 100, LineTotal: 6_000},
		},
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, 60.0, so.Items[0].OpenQty, "open_qty must equal quantity at creation")
}

func TestCreateSalesOrder_DocNumberIsUnique(t *testing.T) {
	testutil.SetupDB(t)

	so1, _ := createSampleSO(t)
	so2, _ := createSampleSO(t)
	require.NotEqual(t, so1.SalesOrderNumber, so2.SalesOrderNumber, "every SO must get a distinct number")
}

// TestCreateSalesOrder_ItemInNonSalesCategoryIsBlocked verifies that an item
// whose OITB category has for_sales = false is rejected at SO creation time.
func TestCreateSalesOrder_ItemInNonSalesCategoryIsBlocked(t *testing.T) {
	testutil.SetupDB(t)

	grp := testutil.SeedOITB(t, "Industrial Products", false) // for_sales = false
	testutil.SeedOITMInGroup(t, "MACHINE-001", "Milling Machine", 0.0, grp)

	_, err := CreateSalesOrder(
		"2025-01-01",
		nil, "Test Customer", "", "", "Cash", "", nil, "",
		[]models.SalesOrderItem{
			{SKU: "MACHINE-001", Unit: "unit", Quantity: 1, PricePerUnit: 100_000, LineTotal: 100_000},
		},
		nil,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Industrial Products", "error must name the blocked category")
}

// ─────────────────────────────────────────────
// Sales Order — status transitions
// ─────────────────────────────────────────────

func TestSubmitSalesOrder_DraftToOpen(t *testing.T) {
	testutil.SetupDB(t)

	so, err := createSampleSO(t)
	require.NoError(t, err)
	require.Equal(t, "Draft", so.DocStatus)

	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	reloaded, err := GetSalesOrder(so.ID)
	require.NoError(t, err)
	require.Equal(t, "Open", reloaded.DocStatus)
}

func TestSubmitSalesOrder_CannotSubmitTwice(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	err := SubmitSalesOrder(so.ID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Draft", "error should mention required status")
}

// TestSubmitSalesOrder_IncrementsIsCommited verifies that submitting a Draft SO
// increments OITM.is_commited by the ordered quantity (1:1, no UoM conversion
// when uom_entry = 0 — ConvertToInventoryQty returns qty unchanged).
func TestSubmitSalesOrder_IncrementsIsCommited(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedOITM(t, "RICE-001", "Milled Rice", 80.0) // is_commited = 0 initially

	so, err := createSampleSO(t) // 100 bags × ₱100
	require.NoError(t, err)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	var oitm models.OITM
	require.NoError(t, db.DB.Where("item_code = ?", "RICE-001").First(&oitm).Error)
	require.Equal(t, 100.0, oitm.IsCommited, "is_commited must equal the ordered quantity after Submit")
}

func TestCancelSalesOrder_FromDraft(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	require.NoError(t, CancelSalesOrder(so.ID, nil))

	reloaded, _ := GetSalesOrder(so.ID)
	require.Equal(t, "Cancelled", reloaded.DocStatus)
}

func TestCancelSalesOrder_CannotCancelTwice(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	require.NoError(t, CancelSalesOrder(so.ID, nil))

	err := CancelSalesOrder(so.ID, nil)
	require.Error(t, err)
}

// TestCancelSalesOrder_BlockedWhenClosed verifies that a Closed SO (fully
// delivered) cannot be cancelled — the service must return a "closed" error.
func TestCancelSalesOrder_BlockedWhenClosed(t *testing.T) {
	testutil.SetupDB(t)

	// confirmSampleDO delivers all 100 bags → auto-closes the SO
	_, so := confirmSampleDO(t)
	require.Equal(t, "Closed", so.DocStatus)

	err := CancelSalesOrder(so.ID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed", "error must explain that Closed SOs cannot be cancelled")
}

// TestCancelSalesOrder_ReleasesIsCommitedOnOpenCancel verifies the full
// commit/release round-trip: Submit increments is_commited; cancelling the
// Open SO decrements it back to zero.
func TestCancelSalesOrder_ReleasesIsCommitedOnOpenCancel(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedOITM(t, "RICE-001", "Milled Rice", 80.0) // is_commited = 0

	so, _ := createSampleSO(t) // 100 bags
	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	// Verify increment
	var oitm models.OITM
	require.NoError(t, db.DB.Where("item_code = ?", "RICE-001").First(&oitm).Error)
	require.Equal(t, 100.0, oitm.IsCommited, "is_commited must be 100 after Submit")

	// Cancel — must release the commitment
	require.NoError(t, CancelSalesOrder(so.ID, nil))

	require.NoError(t, db.DB.Where("item_code = ?", "RICE-001").First(&oitm).Error)
	require.Equal(t, 0.0, oitm.IsCommited, "is_commited must return to 0 when Open SO is cancelled")
}

func TestCancelSalesOrder_BlockedWhenDeliveryOrderExists(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID
	_, err := CreateDeliveryOrder(
		so.ID, "2025-01-02", "Driver A", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID:  &soItemID,
			SKU:               "RICE-001",
			Unit:              "bag",
			QuantityOrdered:   100,
			QuantityDelivered: 100,
			PricePerUnit:      100,
		}},
		nil,
	)
	require.NoError(t, err)

	// Cancel must be blocked because a child DO exists
	err = CancelSalesOrder(so.ID, nil)
	require.Error(t, err)
}

// ─────────────────────────────────────────────
// Sales Order — update
// ─────────────────────────────────────────────

// TestUpdateSalesOrder_ReplacesItemsAndRecalculatesGrandTotal verifies that
// updating a Draft SO removes all existing line items and replaces them with
// the new set, recomputing grand_total from the new lines.
func TestUpdateSalesOrder_ReplacesItemsAndRecalculatesGrandTotal(t *testing.T) {
	testutil.SetupDB(t)

	so, err := createSampleSO(t) // 100 bags × ₱100 = ₱10,000
	require.NoError(t, err)
	require.Equal(t, 10_000.0, so.GrandTotal)

	// Reload to get DB-assigned version (default 1)
	so, _ = GetSalesOrder(so.ID)

	updated, err := UpdateSalesOrder(
		so.ID,
		so.Version,
		nil, "Updated Customer", "", "",
		"2025-02-01", "Cash", "15d", nil, "",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag",  Quantity: 50, PricePerUnit: 120, LineTotal: 6_000},
			{SKU: "BRAN-001", Unit: "sack", Quantity: 20, PricePerUnit: 80,  LineTotal: 1_600},
		},
		nil,
	)
	require.NoError(t, err)
	require.InDelta(t, 7_600.0, updated.GrandTotal, 0.005, "grand_total must equal sum of new lines")
	require.Len(t, updated.Items, 2, "old single item must be replaced by 2 new items")
	// open_qty is reset to quantity for each new line
	for _, item := range updated.Items {
		require.Equal(t, item.Quantity, item.OpenQty, "open_qty must match quantity on each new line")
	}
}

// TestUpdateSalesOrder_BlockedForNonDraft verifies that an Open (submitted) SO
// cannot be edited — the service must return an error mentioning "cannot be edited".
func TestUpdateSalesOrder_BlockedForNonDraft(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	require.NoError(t, SubmitSalesOrder(so.ID, nil)) // Draft → Open
	so, _ = GetSalesOrder(so.ID)

	_, err := UpdateSalesOrder(
		so.ID,
		so.Version,
		nil, "Should Not Save", "", "",
		"2025-02-01", "Cash", "", nil, "",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 50, PricePerUnit: 100, LineTotal: 5_000},
		},
		nil,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be edited", "error must explain why the edit is blocked")
}

// TestUpdateSalesOrder_VersionConflict verifies optimistic-lock semantics:
// after a successful update bumps the version, a second call with the original
// (now stale) version must be rejected with a "conflict" error.
func TestUpdateSalesOrder_VersionConflict(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	so, _ = GetSalesOrder(so.ID) // version = 1 from DB default

	// First update — passes with correct version; bumps version to 2
	_, err := UpdateSalesOrder(
		so.ID,
		so.Version, // 1
		nil, "First Update", "", "",
		"2025-02-01", "Cash", "", nil, "",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 80, PricePerUnit: 100, LineTotal: 8_000},
		},
		nil,
	)
	require.NoError(t, err)

	// Second update uses the original stale version — must be rejected
	_, err = UpdateSalesOrder(
		so.ID,
		so.Version, // still 1 — stale after first update
		nil, "Stale Update", "", "",
		"2025-02-01", "Cash", "", nil, "",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 60, PricePerUnit: 100, LineTotal: 6_000},
		},
		nil,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "conflict", "stale version must return a conflict error")
}

// TestUpdateSalesOrder_HeaderFieldsPersisted verifies that customer snapshot,
// payment method, terms, and notes are all written through correctly.
func TestUpdateSalesOrder_HeaderFieldsPersisted(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	so, _ = GetSalesOrder(so.ID)

	updated, err := UpdateSalesOrder(
		so.ID,
		so.Version,
		nil, "New Customer Name", "123 New Street", "09171234567",
		"2025-06-01", "Bank Transfer", "45d", nil, "urgent order",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 100, PricePerUnit: 100, LineTotal: 10_000},
		},
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, "New Customer Name",  updated.CustomerNameSnapshot)
	require.Equal(t, "123 New Street",     updated.CustomerAddressSnapshot)
	require.Equal(t, "09171234567",        updated.CustomerContactSnapshot)
	require.Equal(t, "Bank Transfer",      updated.PaymentMethod)
	require.Equal(t, "45d",               updated.Terms)
	require.Equal(t, "urgent order",       updated.Notes)
}

// ─────────────────────────────────────────────
// Delivery Order
// ─────────────────────────────────────────────

func TestCreateDeliveryOrder_RequiresOpenSO(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	// SO is still Draft — CreateDeliveryOrder must reject it
	_, err := CreateDeliveryOrder(
		so.ID, "2025-01-02", "Driver", "",
		[]models.DeliveryOrderItem{{SKU: "RICE-001", Unit: "bag", QuantityDelivered: 10, PricePerUnit: 100}},
		nil,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Open")
}

func TestCreateDeliveryOrder_RejectsDuplicateDraft(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID

	doItem := func() models.DeliveryOrderItem {
		return models.DeliveryOrderItem{
			SalesOrderItemID:  &soItemID,
			SKU:               "RICE-001",
			Unit:              "bag",
			QuantityOrdered:   50,
			QuantityDelivered: 50,
			PricePerUnit:      100,
		}
	}

	// First draft DO — should succeed
	_, err := CreateDeliveryOrder(so.ID, "2025-01-02", "Driver", "", []models.DeliveryOrderItem{doItem()}, nil)
	require.NoError(t, err)

	// Second draft DO while the first is still Draft — must fail
	_, err = CreateDeliveryOrder(so.ID, "2025-01-03", "Driver", "", []models.DeliveryOrderItem{doItem()}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "draft delivery order")
}

func TestCreateDeliveryOrder_RejectsOverQty(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t) // 100 bags
	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID

	_, err := CreateDeliveryOrder(
		so.ID, "2025-01-02", "Driver", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID:  &soItemID,
			SKU:               "RICE-001",
			Unit:              "bag",
			QuantityOrdered:   100,
			QuantityDelivered: 999, // exceeds open qty of 100
			PricePerUnit:      100,
		}},
		nil,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds")
}

// ─────────────────────────────────────────────
// Full O2C cycle
// ─────────────────────────────────────────────

// TestFullO2CCycle exercises the complete Order-to-Cash flow:
//
//	SO (Draft→Open) → DO (Draft→Delivered) → AR Invoice (Open) → Collection → Paid
//
// OITM records are intentionally absent so ConfirmDeliveryOrder skips the
// MySQL-specific inventory SQL while still exercising all financial state transitions.
func TestFullO2CCycle(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	const qty   = 100.0
	const price = 120.0
	const total = qty * price // 12,000

	// 1 ── Create and submit Sales Order ─────────────────────────────────────
	so, err := CreateSalesOrder(
		"2025-02-01",
		nil, "Sunshine Rice Store", "", "", "Cash", "30d", nil, "",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: qty, PricePerUnit: price, LineTotal: total},
		},
		nil,
	)
	require.NoError(t, err, "CreateSalesOrder")
	require.NoError(t, SubmitSalesOrder(so.ID, nil), "SubmitSalesOrder")

	so, _ = GetSalesOrder(so.ID)
	require.Equal(t, "Open", so.DocStatus)

	// 2 ── Create and confirm Delivery Order ─────────────────────────────────
	soItemID := so.Items[0].ID
	do, err := CreateDeliveryOrder(
		so.ID, "2025-02-02", "Driver B", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID:  &soItemID,
			SKU:               "RICE-001",
			Unit:              "bag",
			QuantityOrdered:   qty,
			QuantityDelivered: qty,
			PricePerUnit:      price,
		}},
		nil,
	)
	require.NoError(t, err, "CreateDeliveryOrder")
	require.NoError(t, ConfirmDeliveryOrder(do.ID, nil), "ConfirmDeliveryOrder")

	do, _ = GetDeliveryOrder(do.ID)
	require.Equal(t, "Delivered", do.Status)

	// All SO lines fulfilled → SO auto-closes
	so, _ = GetSalesOrder(so.ID)
	require.Equal(t, "Closed", so.DocStatus)
	require.Equal(t, 0.0, so.Items[0].OpenQty, "open_qty must be 0 after full delivery")

	// 3 ── Create AR Invoice (triggers accounting JE) ─────────────────────────
	inv, err := CreateARInvoice(
		[]uint{do.ID},
		nil, "Sunshine Rice Store", "", "",
		"2025-02-03", "30d", "", nil,
		[]models.ARInvoiceItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: qty, PricePerUnit: price, DeliveryOrderID: &do.ID},
		},
		nil,
	)
	require.NoError(t, err, "CreateARInvoice")
	require.Equal(t, total, inv.TotalAmount)
	require.Equal(t, "Open", inv.Status)

	// 4 ── Create Collection (full payment) ───────────────────────────────────
	col, err := CreateCollection(
		"2025-02-04",
		nil, "Sunshine Rice Store", "Cash", "REF-2025-001", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: total}},
		nil,
	)
	require.NoError(t, err, "CreateCollection")
	require.Equal(t, total, col.TotalAmount)
	require.Equal(t, "Posted", col.Status)

	// AR Invoice must be Paid
	inv2, err := GetARInvoice(inv.ID)
	require.NoError(t, err)
	require.Equal(t, "Paid", inv2.Status)
	require.Equal(t, total, inv2.AmountCollected)

	// SO payment_status must be Paid
	so2, _ := GetSalesOrder(so.ID)
	require.Equal(t, "Paid", so2.PaymentStatus)
}

// TestPartialCollectionSetsPartialStatus verifies that paying less than the invoice
// total yields status = "Partial" and a correct amount_collected.
func TestPartialCollectionSetsPartialStatus(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	const total = 10_000.0

	so, _ := createSampleSO(t) // 100 bags × ₱100 = ₱10,000
	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID

	do, _ := CreateDeliveryOrder(
		so.ID, "2025-01-02", "Driver", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID:  &soItemID,
			SKU:               "RICE-001",
			Unit:              "bag",
			QuantityOrdered:   100,
			QuantityDelivered: 100,
			PricePerUnit:      100,
		}},
		nil,
	)
	require.NoError(t, ConfirmDeliveryOrder(do.ID, nil))

	inv, err := CreateARInvoice(
		[]uint{do.ID},
		nil, "Test Customer", "", "",
		"2025-01-03", "30d", "", nil,
		[]models.ARInvoiceItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 100, PricePerUnit: 100, DeliveryOrderID: &do.ID},
		},
		nil,
	)
	require.NoError(t, err)

	// Pay only 40 %
	_, err = CreateCollection(
		"2025-01-04",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: total * 0.40}},
		nil,
	)
	require.NoError(t, err)

	inv2, _ := GetARInvoice(inv.ID)
	require.Equal(t, "Partial", inv2.Status)
	require.InDelta(t, total*0.40, inv2.AmountCollected, 0.005)
}

// TestTwoPartialCollectionsThenPaid verifies that two payments summing to the total
// move the invoice from Partial → Paid.
func TestTwoPartialCollectionsThenPaid(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	const total = 10_000.0

	so, _ := createSampleSO(t)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))
	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID

	do, _ := CreateDeliveryOrder(
		so.ID, "2025-01-02", "Driver", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID: &soItemID, SKU: "RICE-001", Unit: "bag",
			QuantityOrdered: 100, QuantityDelivered: 100, PricePerUnit: 100,
		}},
		nil,
	)
	require.NoError(t, ConfirmDeliveryOrder(do.ID, nil))

	inv, _ := CreateARInvoice(
		[]uint{do.ID}, nil, "Test Customer", "", "",
		"2025-01-03", "30d", "", nil,
		[]models.ARInvoiceItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 100, PricePerUnit: 100, DeliveryOrderID: &do.ID},
		},
		nil,
	)

	// First partial payment
	_, err := CreateCollection("2025-01-04", nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 6_000}}, nil)
	require.NoError(t, err)

	inv2, _ := GetARInvoice(inv.ID)
	require.Equal(t, "Partial", inv2.Status)

	// Second payment clears the balance
	_, err = CreateCollection("2025-01-05", nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 4_000}}, nil)
	require.NoError(t, err)

	inv3, _ := GetARInvoice(inv.ID)
	require.Equal(t, "Paid", inv3.Status)
	require.InDelta(t, total, inv3.AmountCollected, 0.005)
}

// ─────────────────────────────────────────────
// Confirm Delivery Order — additional gap tests
// ─────────────────────────────────────────────

// TestConfirmDeliveryOrder_CannotConfirmTwice verifies that a Delivered DO
// rejects a second ConfirmDeliveryOrder call with an "already Delivered" error.
func TestConfirmDeliveryOrder_CannotConfirmTwice(t *testing.T) {
	testutil.SetupDB(t)

	do, _ := confirmSampleDO(t) // status = Delivered

	err := ConfirmDeliveryOrder(do.ID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already Delivered")
}

// ─────────────────────────────────────────────
// Create AR Invoice — additional gap tests
// ─────────────────────────────────────────────

// TestCreateARInvoice_RequiresDeliveredDO verifies that CreateARInvoice rejects
// a Draft DO — only "Delivered" DOs may be invoiced.
func TestCreateARInvoice_RequiresDeliveredDO(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	so, _ := createSampleSO(t)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))
	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID

	do, err := CreateDeliveryOrder(
		so.ID, "2025-01-02", "Driver", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID: &soItemID, SKU: "RICE-001", Unit: "bag",
			QuantityOrdered: 100, QuantityDelivered: 100, PricePerUnit: 100,
		}},
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, "Draft", do.Status)

	// DO is still Draft — AR Invoice must be rejected
	_, err = CreateARInvoice(
		[]uint{do.ID},
		nil, "Test Customer", "", "",
		"2025-01-03", "30d", "", nil,
		[]models.ARInvoiceItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 100, PricePerUnit: 100},
		},
		nil,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not Delivered", "error must state the DO status requirement")
}

// TestCreateARInvoice_AmountInvoicedRollup verifies that CreateARInvoice
// increments both SO.amount_invoiced and DO.amount_invoiced by the invoice total.
func TestCreateARInvoice_AmountInvoicedRollup(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, do, so := invoiceSampleDO(t) // ₱10,000 invoice; helper reloads DO and SO

	require.InDelta(t, inv.TotalAmount, so.AmountInvoiced, 0.005,
		"SO.amount_invoiced must equal the invoice total")
	require.InDelta(t, inv.TotalAmount, do.AmountInvoiced, 0.005,
		"DO.amount_invoiced must equal the invoice total")
}

// ─────────────────────────────────────────────
// COGS journal entry
// ─────────────────────────────────────────────

// TestConfirmDeliveryOrder_PostsCOGSJournalEntry verifies that confirming a DO
// posts a balanced JE: DR Cost of Goods Sold / CR Inventory — Milled Rice.
// The amounts are qty × avg_price (moving-average cost), not the selling price.
func TestConfirmDeliveryOrder_PostsCOGSJournalEntry(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)
	testutil.SeedOITM(t, "RICE-001", "Milled Rice Premium", 80.0) // avg cost ₱80/bag

	const qty   = 100.0
	const price = 120.0        // selling price — must not appear in COGS JE
	const cost  = 80.0         // avg cost  — must appear in COGS JE
	const expectedCOGS = qty * cost // ₱8,000

	so, err := CreateSalesOrder(
		"2025-03-01",
		nil, "Agri Buyer", "", "", "Cash", "", nil, "",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: qty, PricePerUnit: price, LineTotal: qty * price},
		},
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID

	do, err := CreateDeliveryOrder(
		so.ID, "2025-03-02", "Driver", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID:  &soItemID,
			SKU:               "RICE-001",
			Unit:              "bag",
			QuantityOrdered:   qty,
			QuantityDelivered: qty,
			PricePerUnit:      price,
		}},
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, ConfirmDeliveryOrder(do.ID, nil))

	// A COGS JE must exist, sourced from this Delivery Order
	var je models.JournalEntry
	err = db.DB.
		Where("source_type = 'DELIVERY_ORDER' AND source_id = ?", do.ID).
		First(&je).Error
	require.NoError(t, err, "COGS JE must be posted when DO is confirmed")
	require.Equal(t, "SALES", je.Module)
	require.Equal(t, "POSTED", je.Status)

	// Load lines and verify the entry balances at the cost amount, not the selling price
	var lines []models.JournalEntryLine
	require.NoError(t, db.DB.Where("journal_entry_id = ?", je.ID).Find(&lines).Error)
	require.Len(t, lines, 2, "COGS JE must have exactly 2 lines (Dr + Cr)")

	var totalDr, totalCr float64
	for _, l := range lines {
		if l.Debit != nil {
			totalDr += *l.Debit
		}
		if l.Credit != nil {
			totalCr += *l.Credit
		}
	}
	require.InDelta(t, expectedCOGS, totalDr, 0.005, "debit must equal qty × avg_price")
	require.InDelta(t, expectedCOGS, totalCr, 0.005, "credit must equal qty × avg_price")
	require.InDelta(t, totalDr, totalCr, 0.005, "JE must balance")
}

// TestConfirmDeliveryOrder_COGSAggregatesMultipleItems verifies that a DO with
// two line items posts a single COGS JE whose amounts sum all item costs.
func TestConfirmDeliveryOrder_COGSAggregatesMultipleItems(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)
	testutil.SeedOITM(t, "RICE-001", "Premium Rice", 80.0) // 50 bags  × ₱80 = ₱4,000
	testutil.SeedOITM(t, "BRAN-001", "Rice Bran",    20.0) // 200 bags × ₱20 = ₱4,000

	so, err := CreateSalesOrder(
		"2025-03-01",
		nil, "Multi Buyer", "", "", "Cash", "", nil, "",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 50,  PricePerUnit: 120, LineTotal: 6_000},
			{SKU: "BRAN-001", Unit: "bag", Quantity: 200, PricePerUnit: 30,  LineTotal: 6_000},
		},
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	so, _ = GetSalesOrder(so.ID)
	soItem1ID := so.Items[0].ID
	soItem2ID := so.Items[1].ID

	do, err := CreateDeliveryOrder(
		so.ID, "2025-03-02", "Driver", "",
		[]models.DeliveryOrderItem{
			{SalesOrderItemID: &soItem1ID, SKU: "RICE-001", Unit: "bag",
				QuantityOrdered: 50, QuantityDelivered: 50, PricePerUnit: 120},
			{SalesOrderItemID: &soItem2ID, SKU: "BRAN-001", Unit: "bag",
				QuantityOrdered: 200, QuantityDelivered: 200, PricePerUnit: 30},
		},
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, ConfirmDeliveryOrder(do.ID, nil))

	// Single aggregated COGS JE — not one per item
	var count int64
	db.DB.Model(&models.JournalEntry{}).
		Where("source_type = 'DELIVERY_ORDER' AND source_id = ?", do.ID).
		Count(&count)
	require.Equal(t, int64(1), count, "exactly one COGS JE per DO")

	var je models.JournalEntry
	db.DB.Where("source_type = 'DELIVERY_ORDER' AND source_id = ?", do.ID).First(&je)

	var lines []models.JournalEntryLine
	require.NoError(t, db.DB.Where("journal_entry_id = ?", je.ID).Find(&lines).Error)
	require.Len(t, lines, 2)

	// Total cost: (50 × ₱80) + (200 × ₱20) = ₱4,000 + ₱4,000 = ₱8,000
	expectedCOGS := 50.0*80.0 + 200.0*20.0

	var totalDr float64
	for _, l := range lines {
		if l.Debit != nil {
			totalDr += *l.Debit
		}
	}
	require.InDelta(t, expectedCOGS, totalDr, 0.005,
		"aggregated COGS must equal sum of (invQty × avgPrice) across all items")
}

// TestConfirmDeliveryOrder_NoCOGSWhenAvgPriceIsZero verifies that no COGS JE
// is posted when all delivered items carry zero avg_price (cost not yet known).
// This guards the case where items exist in OITM but haven't been costed.
func TestConfirmDeliveryOrder_NoCOGSWhenAvgPriceIsZero(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)
	testutil.SeedOITM(t, "RICE-001", "Uncosted Rice", 0.0) // avg_price intentionally zero

	so, err := CreateSalesOrder(
		"2025-03-01",
		nil, "Zero Cost Buyer", "", "", "Cash", "", nil, "",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 50, PricePerUnit: 120, LineTotal: 6_000},
		},
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))

	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID

	do, err := CreateDeliveryOrder(
		so.ID, "2025-03-02", "Driver", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID:  &soItemID,
			SKU:               "RICE-001",
			Unit:              "bag",
			QuantityOrdered:   50,
			QuantityDelivered: 50,
			PricePerUnit:      120,
		}},
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, ConfirmDeliveryOrder(do.ID, nil))

	// No COGS JE — zero-cost guard must hold
	var count int64
	db.DB.Model(&models.JournalEntry{}).
		Where("source_type = 'DELIVERY_ORDER' AND source_id = ?", do.ID).
		Count(&count)
	require.Zero(t, count, "no COGS JE should be posted when avg_price is 0")
}

// ─────────────────────────────────────────────
// Cancel Delivery Order
// ─────────────────────────────────────────────

// TestCancelDeliveryOrder_DraftBecomeCancelled verifies that cancelling a Draft DO
// sets its status to Cancelled and leaves the parent SO Open (no stock was moved).
func TestCancelDeliveryOrder_DraftBecomeCancelled(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))
	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID

	do, err := CreateDeliveryOrder(
		so.ID, "2025-04-02", "Driver", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID: &soItemID, SKU: "RICE-001", Unit: "bag",
			QuantityOrdered: 100, QuantityDelivered: 100, PricePerUnit: 100,
		}},
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, "Draft", do.Status)

	require.NoError(t, CancelDeliveryOrder(do.ID, nil))

	do, _ = GetDeliveryOrder(do.ID)
	require.Equal(t, "Cancelled", do.Status)

	// SO stays Open — no stock was confirmed out
	so, _ = GetSalesOrder(so.ID)
	require.Equal(t, "Open", so.DocStatus)
}

// TestCancelDeliveryOrder_CannotCancelTwice verifies that cancelling an already-cancelled
// DO returns an error.
func TestCancelDeliveryOrder_CannotCancelTwice(t *testing.T) {
	testutil.SetupDB(t)

	so, _ := createSampleSO(t)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))
	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID
	do, _ := CreateDeliveryOrder(
		so.ID, "2025-04-02", "Driver", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID: &soItemID, SKU: "RICE-001", Unit: "bag",
			QuantityOrdered: 100, QuantityDelivered: 100, PricePerUnit: 100,
		}},
		nil,
	)
	require.NoError(t, CancelDeliveryOrder(do.ID, nil))
	require.Error(t, CancelDeliveryOrder(do.ID, nil))
}

// TestCancelDeliveryOrder_DeliveredRestoresOpenQtyAndReopensesSO verifies that
// cancelling a Delivered DO restores the SO item open_qty and reopens a Closed SO.
func TestCancelDeliveryOrder_DeliveredRestoresOpenQtyAndReopensesSO(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	do, so := confirmSampleDO(t) // 100 bags; SO is now Closed

	require.Equal(t, "Closed", so.DocStatus)
	require.Equal(t, 0.0, so.Items[0].OpenQty)

	require.NoError(t, CancelDeliveryOrder(do.ID, nil))

	do, _ = GetDeliveryOrder(do.ID)
	require.Equal(t, "Cancelled", do.Status)

	so, _ = GetSalesOrder(so.ID)
	require.Equal(t, "Open", so.DocStatus, "SO must be reopened after DO cancel")
	require.Equal(t, 100.0, so.Items[0].OpenQty, "open_qty must be restored to original quantity")
}

// TestCancelDeliveryOrder_BlockedWhenARInvoiceExists verifies that a Delivered DO
// cannot be cancelled once an AR Invoice has been raised against it.
func TestCancelDeliveryOrder_BlockedWhenARInvoiceExists(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	_, do, _ := invoiceSampleDO(t)

	err := CancelDeliveryOrder(do.ID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "AR invoice")
}

// TestCancelDeliveryOrder_ReversesCogsJournalEntry verifies that cancelling a Delivered
// DO (which had a COGS JE posted) marks the original JE as REVERSED and creates a
// reversal JE with swapped debits/credits.
func TestCancelDeliveryOrder_ReversesCogsJournalEntry(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)
	testutil.SeedOITM(t, "RICE-001", "Milled Rice", 80.0) // triggers COGS JE at confirm

	do, _ := confirmSampleDO(t)

	// COGS JE must have been posted by ConfirmDeliveryOrder
	var cogsJE models.JournalEntry
	require.NoError(t, db.DB.
		Where("source_type = 'DELIVERY_ORDER' AND source_id = ?", do.ID).
		First(&cogsJE).Error, "COGS JE must exist before cancel")
	require.Equal(t, "POSTED", cogsJE.Status)

	require.NoError(t, CancelDeliveryOrder(do.ID, nil))

	// Original JE must now be REVERSED
	require.NoError(t, db.DB.First(&cogsJE, cogsJE.ID).Error)
	require.Equal(t, "REVERSED", cogsJE.Status, "COGS JE must be reversed on DO cancel")

	// A reversal JE must exist and be balanced
	var revJE models.JournalEntry
	require.NoError(t, db.DB.
		Where("is_reversal = 1 AND reversed_entry_id = ?", cogsJE.ID).
		First(&revJE).Error, "reversal JE must be created")

	var lines []models.JournalEntryLine
	require.NoError(t, db.DB.Where("journal_entry_id = ?", revJE.ID).Find(&lines).Error)
	require.Len(t, lines, 2)

	var totalDr, totalCr float64
	for _, l := range lines {
		if l.Debit != nil {
			totalDr += *l.Debit
		}
		if l.Credit != nil {
			totalCr += *l.Credit
		}
	}
	require.InDelta(t, totalDr, totalCr, 0.005, "reversal JE must balance")
	require.InDelta(t, 100.0*80.0, totalDr, 0.005, "reversal amount must equal original COGS")
}

// ─────────────────────────────────────────────
// Cancel AR Invoice
// ─────────────────────────────────────────────

// TestCancelARInvoice_SetsStatusAndReversesJE verifies that cancelling an Open AR Invoice
// sets its status to Cancelled and posts a reversal of the original AR JE.
func TestCancelARInvoice_SetsStatusAndReversesJE(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, _ := invoiceSampleDO(t)
	require.Equal(t, "Open", inv.Status)

	// Find the AR JE posted by CreateARInvoice
	var arJE models.JournalEntry
	require.NoError(t, db.DB.
		Where("source_type = 'AR_INVOICE' AND source_id = ?", inv.ID).
		First(&arJE).Error)
	require.Equal(t, "POSTED", arJE.Status)

	require.NoError(t, CancelARInvoice(inv.ID, nil))

	// Invoice is Cancelled
	inv2, _ := GetARInvoice(inv.ID)
	require.Equal(t, "Cancelled", inv2.Status)

	// Original AR JE must be REVERSED
	require.NoError(t, db.DB.First(&arJE, arJE.ID).Error)
	require.Equal(t, "REVERSED", arJE.Status)
}

// TestCancelARInvoice_CannotCancelTwice verifies that cancelling an already-cancelled
// AR Invoice returns an error.
func TestCancelARInvoice_CannotCancelTwice(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, _ := invoiceSampleDO(t)
	require.NoError(t, CancelARInvoice(inv.ID, nil))
	require.Error(t, CancelARInvoice(inv.ID, nil))
}

// TestCancelARInvoice_BlockedWhenCollectionApplied verifies that an AR Invoice with any
// amount_collected > 0 cannot be cancelled directly — the collection must be reversed first.
func TestCancelARInvoice_BlockedWhenCollectionApplied(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, _ := invoiceSampleDO(t)

	// Apply a partial payment
	_, err := CreateCollection(
		"2025-04-04",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 3_000}},
		nil,
	)
	require.NoError(t, err)

	err = CancelARInvoice(inv.ID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "collections applied")
}

// TestCancelARInvoice_ReversesAmountInvoicedOnSO verifies that cancelling an AR Invoice
// decrements SO.amount_invoiced back to zero.
func TestCancelARInvoice_ReversesAmountInvoicedOnSO(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	const total = 10_000.0
	inv, _, so := invoiceSampleDO(t)

	// SO.amount_invoiced must reflect the invoice
	so, _ = GetSalesOrder(so.ID)
	require.InDelta(t, total, so.AmountInvoiced, 0.005, "amount_invoiced must be set after CreateARInvoice")

	require.NoError(t, CancelARInvoice(inv.ID, nil))

	so, _ = GetSalesOrder(so.ID)
	require.InDelta(t, 0.0, so.AmountInvoiced, 0.005, "amount_invoiced must return to 0 after cancel")
}

// ─────────────────────────────────────────────
// Cancel Collection
// ─────────────────────────────────────────────

// TestCancelCollection_RestoresARInvoiceToOpen verifies that cancelling a full-payment
// collection sets the AR Invoice back to Open with amount_collected = 0.
func TestCancelCollection_RestoresARInvoiceToOpen(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, _ := invoiceSampleDO(t)

	col, err := CreateCollection(
		"2025-04-04",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 10_000}},
		nil,
	)
	require.NoError(t, err)

	inv2, _ := GetARInvoice(inv.ID)
	require.Equal(t, "Paid", inv2.Status)

	require.NoError(t, CancelCollection(col.ID, nil))

	inv3, _ := GetARInvoice(inv.ID)
	require.Equal(t, "Open", inv3.Status, "AR Invoice must return to Open after collection cancel")
	require.InDelta(t, 0.0, inv3.AmountCollected, 0.005, "amount_collected must be zeroed")
}

// TestCancelCollection_ReversesJournalEntry verifies that cancelling a Posted collection
// marks its JE as REVERSED and creates a balancing reversal JE.
func TestCancelCollection_ReversesJournalEntry(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, _ := invoiceSampleDO(t)

	col, _ := CreateCollection(
		"2025-04-04",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 10_000}},
		nil,
	)

	var colJE models.JournalEntry
	require.NoError(t, db.DB.
		Where("source_type = 'COLLECTION' AND source_id = ?", col.ID).
		First(&colJE).Error)
	require.Equal(t, "POSTED", colJE.Status)

	require.NoError(t, CancelCollection(col.ID, nil))

	require.NoError(t, db.DB.First(&colJE, colJE.ID).Error)
	require.Equal(t, "REVERSED", colJE.Status, "collection JE must be reversed on cancel")

	// Reversal JE must exist
	var revCount int64
	db.DB.Model(&models.JournalEntry{}).
		Where("is_reversal = 1 AND reversed_entry_id = ?", colJE.ID).
		Count(&revCount)
	require.Equal(t, int64(1), revCount)
}

// TestCancelCollection_RestoresSalesOrderPaymentStatus verifies that cancelling a full
// collection restores SO.payment_status to Unpaid and SO.amount_paid to 0.
func TestCancelCollection_RestoresSalesOrderPaymentStatus(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, so := invoiceSampleDO(t)

	col, err := CreateCollection(
		"2025-04-04",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 10_000}},
		nil,
	)
	require.NoError(t, err)

	so, _ = GetSalesOrder(so.ID)
	require.Equal(t, "Paid", so.PaymentStatus)

	require.NoError(t, CancelCollection(col.ID, nil))

	so, _ = GetSalesOrder(so.ID)
	require.Equal(t, "Unpaid", so.PaymentStatus, "payment_status must revert to Unpaid")
	require.InDelta(t, 0.0, so.AmountPaid, 0.005, "amount_paid must revert to 0")
}

// TestCancelCollection_CannotCancelTwice verifies that cancelling an already-cancelled
// collection returns an error.
func TestCancelCollection_CannotCancelTwice(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, _ := invoiceSampleDO(t)
	col, _ := CreateCollection(
		"2025-04-04",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 10_000}},
		nil,
	)
	require.NoError(t, CancelCollection(col.ID, nil))
	require.Error(t, CancelCollection(col.ID, nil))
}

// TestCancelCollection_PartialCancelLeavesInvoicePartial verifies that cancelling one
// of two partial payments leaves the AR Invoice in Partial status (not Open) because
// the other payment still stands.
func TestCancelCollection_PartialCancelLeavesInvoicePartial(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, _ := invoiceSampleDO(t) // ₱10,000 invoice

	// First payment: ₱6,000
	col1, err := CreateCollection(
		"2025-04-04",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 6_000}},
		nil,
	)
	require.NoError(t, err)

	// Second payment: ₱4,000 — tips invoice to Paid
	_, err = CreateCollection(
		"2025-04-05",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 4_000}},
		nil,
	)
	require.NoError(t, err)

	inv2, _ := GetARInvoice(inv.ID)
	require.Equal(t, "Paid", inv2.Status)

	// Cancel only the first collection (₱6,000)
	require.NoError(t, CancelCollection(col1.ID, nil))

	// ₱4,000 still applied — invoice should be Partial, not Open
	inv3, _ := GetARInvoice(inv.ID)
	require.Equal(t, "Partial", inv3.Status, "cancelling one of two payments must leave invoice Partial")
	require.InDelta(t, 4_000.0, inv3.AmountCollected, 0.005)
}

// ─────────────────────────────────────────────
// List and Get — read-path coverage
// ─────────────────────────────────────────────

// TestListCustomers_ActiveOnlyFilter verifies the activeOnly flag mirrors the
// same pattern as ListPriceGroups: false returns all rows; true returns only
// is_active = 1.
func TestListCustomers_ActiveOnlyFilter(t *testing.T) {
	testutil.SetupDB(t)

	c1 := &models.Customer{Name: "Buyer Alpha"}
	c2 := &models.Customer{Name: "Buyer Beta"}
	require.NoError(t, CreateCustomer(c1))
	require.NoError(t, CreateCustomer(c2))
	require.NoError(t, DeleteCustomer(c2.ID)) // soft-delete

	all, err := ListCustomers(false)
	require.NoError(t, err)
	require.Len(t, all, 2, "activeOnly=false must return both customers")

	active, err := ListCustomers(true)
	require.NoError(t, err)
	require.Len(t, active, 1, "activeOnly=true must return only the active customer")
	require.Equal(t, "Buyer Alpha", active[0].Name)
}

// TestListCustomers_OrderedByName verifies that results are sorted
// alphabetically by name regardless of insertion order.
func TestListCustomers_OrderedByName(t *testing.T) {
	testutil.SetupDB(t)

	require.NoError(t, CreateCustomer(&models.Customer{Name: "Zorro Farms"}))
	require.NoError(t, CreateCustomer(&models.Customer{Name: "Alpha Agri"}))
	require.NoError(t, CreateCustomer(&models.Customer{Name: "Midland Rice"}))

	list, err := ListCustomers(false)
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.Equal(t, "Alpha Agri",   list[0].Name)
	require.Equal(t, "Midland Rice", list[1].Name)
	require.Equal(t, "Zorro Farms",  list[2].Name)
}

// TestListSalesOrders_ReturnsNewestFirst verifies ORDER BY id DESC so the most
// recently created SO is always first in the slice.
func TestListSalesOrders_ReturnsNewestFirst(t *testing.T) {
	testutil.SetupDB(t)

	so1, _ := createSampleSO(t)
	so2, _ := createSampleSO(t)

	list, err := ListSalesOrders()
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, so2.ID, list[0].ID, "newest SO must be first in the list")
	require.Equal(t, so1.ID, list[1].ID)
}

// TestListDeliveryOrders_FilterBySalesOrderID verifies that passing soID > 0
// returns only DOs for that SO, while soID = 0 returns all DOs.
func TestListDeliveryOrders_FilterBySalesOrderID(t *testing.T) {
	testutil.SetupDB(t)

	// SO 1 with one Draft DO
	so1, _ := createSampleSO(t)
	require.NoError(t, SubmitSalesOrder(so1.ID, nil))
	so1, _ = GetSalesOrder(so1.ID)
	item1ID := so1.Items[0].ID
	_, err := CreateDeliveryOrder(
		so1.ID, "2025-01-02", "Driver A", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID: &item1ID, SKU: "RICE-001", Unit: "bag",
			QuantityOrdered: 50, QuantityDelivered: 50, PricePerUnit: 100,
		}},
		nil,
	)
	require.NoError(t, err)

	// SO 2 with one Draft DO
	so2, _ := createSampleSO(t)
	require.NoError(t, SubmitSalesOrder(so2.ID, nil))
	so2, _ = GetSalesOrder(so2.ID)
	item2ID := so2.Items[0].ID
	_, err = CreateDeliveryOrder(
		so2.ID, "2025-01-03", "Driver B", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID: &item2ID, SKU: "RICE-001", Unit: "bag",
			QuantityOrdered: 60, QuantityDelivered: 60, PricePerUnit: 100,
		}},
		nil,
	)
	require.NoError(t, err)

	all, err := ListDeliveryOrders(0)
	require.NoError(t, err)
	require.Len(t, all, 2, "soID=0 must return all DOs")

	filtered, err := ListDeliveryOrders(so1.ID)
	require.NoError(t, err)
	require.Len(t, filtered, 1, "soID filter must return only the DO for that SO")
	require.Equal(t, so1.ID, filtered[0].SalesOrderID)
}

// TestListARInvoices_FilterByStatus verifies that the Status filter narrows
// results to invoices in the matching state.
func TestListARInvoices_FilterByStatus(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, _ := invoiceSampleDO(t) // status = Open
	require.NoError(t, CancelARInvoice(inv.ID, nil))

	open, err := ListARInvoices(ARInvoiceFilters{Status: "Open"})
	require.NoError(t, err)
	require.Empty(t, open, "no Open invoices after cancel")

	cancelled, err := ListARInvoices(ARInvoiceFilters{Status: "Cancelled"})
	require.NoError(t, err)
	require.Len(t, cancelled, 1)
	require.Equal(t, inv.ID, cancelled[0].ID)
}

// TestListARInvoices_FilterBySalesOrderID verifies that invoices can be
// narrowed to those linked to a specific SO.
func TestListARInvoices_FilterBySalesOrderID(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv1, _, so := invoiceSampleDO(t) // linked to `so`

	// Second invoice with no SO link (standalone — no delivery orders)
	inv2, err := CreateARInvoice(
		nil,
		nil, "Other Buyer", "", "",
		"2025-05-01", "30d", "", nil,
		[]models.ARInvoiceItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 50, PricePerUnit: 100},
		},
		nil,
	)
	require.NoError(t, err)
	_ = inv2

	// No filter → both
	all, err := ListARInvoices(ARInvoiceFilters{})
	require.NoError(t, err)
	require.Len(t, all, 2)

	// SO filter → only inv1
	bySOID, err := ListARInvoices(ARInvoiceFilters{SalesOrderID: &so.ID})
	require.NoError(t, err)
	require.Len(t, bySOID, 1)
	require.Equal(t, inv1.ID, bySOID[0].ID)
}

// TestGetCollection_LoadsLinesAndARInvoice verifies that GetCollection eagerly
// loads CollectionLines and each line's associated ARInvoice.
func TestGetCollection_LoadsLinesAndARInvoice(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, _ := invoiceSampleDO(t)

	col, err := CreateCollection(
		"2025-04-04",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 10_000}},
		nil,
	)
	require.NoError(t, err)

	loaded, err := GetCollection(col.ID)
	require.NoError(t, err)
	require.Equal(t, col.ID, loaded.ID)
	require.Equal(t, "Posted", loaded.Status)
	require.Len(t, loaded.Lines, 1, "must have one settlement line")
	require.NotNil(t, loaded.Lines[0].ARInvoice, "settlement line must preload the AR Invoice")
	require.Equal(t, inv.ID, loaded.Lines[0].ARInvoice.ID)
	require.InDelta(t, 10_000.0, loaded.Lines[0].AmountApplied, 0.005)
}

// TestListCollections_ReturnsNewestFirst verifies ORDER BY id DESC so the most
// recently created Collection is first in the returned slice.
func TestListCollections_ReturnsNewestFirst(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedSalesAccounting(t)

	inv, _, _ := invoiceSampleDO(t) // ₱10,000 invoice

	col1, err := CreateCollection(
		"2025-04-04",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 6_000}},
		nil,
	)
	require.NoError(t, err)

	col2, err := CreateCollection(
		"2025-04-05",
		nil, "Test Customer", "Cash", "", "",
		[]CollectionLineInput{{ARInvoiceID: inv.ID, AmountApplied: 4_000}},
		nil,
	)
	require.NoError(t, err)

	list, err := ListCollections()
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, col2.ID, list[0].ID, "newest collection must be first")
	require.Equal(t, col1.ID, list[1].ID)
}

// ─────────────────────────────────────────────
// Test helpers
// ─────────────────────────────────────────────

// confirmSampleDO submits the sample SO and confirms a full delivery of all 100 bags.
// Returns the confirmed DeliveryOrder and a freshly-loaded SalesOrder.
// Callers must call testutil.SetupDB before this; SeedSalesAccounting is optional
// (no COGS JE is posted unless SeedOITM is also called).
func confirmSampleDO(t *testing.T) (*models.DeliveryOrder, *models.SalesOrder) {
	t.Helper()
	so, err := createSampleSO(t)
	require.NoError(t, err)
	require.NoError(t, SubmitSalesOrder(so.ID, nil))
	so, _ = GetSalesOrder(so.ID)
	soItemID := so.Items[0].ID

	do, err := CreateDeliveryOrder(
		so.ID, "2025-04-02", "Driver", "",
		[]models.DeliveryOrderItem{{
			SalesOrderItemID:  &soItemID,
			SKU:               "RICE-001",
			Unit:              "bag",
			QuantityOrdered:   100,
			QuantityDelivered: 100,
			PricePerUnit:      100,
		}},
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, ConfirmDeliveryOrder(do.ID, nil))

	do, _ = GetDeliveryOrder(do.ID)
	so, _ = GetSalesOrder(so.ID)
	return do, so
}

// invoiceSampleDO confirms a full delivery then raises an AR Invoice against it.
// Returns the invoice, the confirmed DO, and the SO.
// Requires SeedSalesAccounting to be called before this (AR Invoice posts a JE).
func invoiceSampleDO(t *testing.T) (*models.ARInvoice, *models.DeliveryOrder, *models.SalesOrder) {
	t.Helper()
	do, so := confirmSampleDO(t)

	inv, err := CreateARInvoice(
		[]uint{do.ID},
		nil, "Test Customer", "", "",
		"2025-04-03", "30d", "", nil,
		[]models.ARInvoiceItem{{
			SKU: "RICE-001", Unit: "bag",
			Quantity: 100, PricePerUnit: 100,
			DeliveryOrderID: &do.ID,
		}},
		nil,
	)
	require.NoError(t, err)

	do, _ = GetDeliveryOrder(do.ID)
	so, _ = GetSalesOrder(so.ID)
	return inv, do, so
}

// createSampleSO creates a Draft Sales Order with one line: 100 bags × ₱100 = ₱10,000.
// The SO is NOT submitted; call SubmitSalesOrder separately when needed.
func createSampleSO(t *testing.T) (*models.SalesOrder, error) {
	t.Helper()
	return CreateSalesOrder(
		"2025-01-01",
		nil, "Test Customer", "", "", "Cash", "", nil, "",
		[]models.SalesOrderItem{
			{SKU: "RICE-001", Unit: "bag", Quantity: 100, PricePerUnit: 100, LineTotal: 10_000},
		},
		nil,
	)
}
