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
// Test helpers
// ─────────────────────────────────────────────

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
