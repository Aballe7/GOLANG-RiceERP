package purchasing

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"egglayererp/app/db"
	"egglayererp/app/models"
)

// helper to set up an in-memory database and migrate all tables.
func setupTestDB(t *testing.T) {
	// each invocation uses its own in-memory database name so tests don't
	// collide when run in the same process.
	var err error
	dsn := fmt.Sprintf("file:db_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db.DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Instead of relying on GORM's migration (which generates MySQL-specific
	// syntax such as CURRENT_TIMESTAMP(3)), create the minimal tables ourselves
	// using plain SQL.  The columns include everything that the purchasing
	// service and accounting helpers reference during tests; defaults and
	// constraints are intentionally kept simple.
	stmts := []string{
		`CREATE TABLE supplier (
		 id integer primary key autoincrement,
		 name varchar(150) not null,
		 supplier_type varchar(30) default 'Regular',
		 contact_person varchar(100),
		 contact_number varchar(20),
		 email varchar(100),
		 address varchar(200),
		 tin_number varchar(30),
		 payment_terms varchar(30) default 'COD',
		 bank_details text,
		 categories varchar(200),
		 notes text,
		 is_active boolean default 1,
		 created_at datetime
		)`,
		`CREATE TABLE purchase (
		 id integer primary key autoincrement,
		 date date not null,
		 category varchar(50),
		 item_name varchar(100),
		 quantity numeric,
		 unit varchar(20),
		 unit_price numeric default 0,
		 total_cost numeric,
		 supplier varchar(100),
		 supplier_id integer,
		 po_number varchar(50),
		 invoice_number varchar(50),
		 received_by varchar(100),
		 payment_status varchar(20) default 'Undelivered',
		 amount_paid numeric default 0,
		 payment_method varchar(20) default 'Cash',
		 remarks text,
		 created_by_id integer,
		 amount_received numeric default 0,
		 amount_invoiced numeric default 0,
		 amount_settled numeric default 0,
		 is_active boolean default 1
		)`,
		`CREATE TABLE delivery_receipt (
		 id integer primary key autoincrement,
		 dr_number varchar(30) unique not null,
		 date date not null,
		 purchase_id integer not null,
		 status varchar(20) default 'Draft',
		 received_by varchar(100),
		 supplier_dr_ref varchar(50),
		 notes text,
		 created_at datetime,
		 created_by_id integer,
		 amount_invoiced numeric default 0
		)`,
		`CREATE TABLE dr_line (
		 id integer primary key autoincrement,
		 delivery_receipt_id integer not null,
		 category varchar(50),
		 item_name varchar(100) not null,
		 unit varchar(20) not null,
		 quantity_ordered numeric default 0,
		 quantity_received numeric default 0,
		 unit_price numeric default 0
		)`,
		`CREATE TABLE ap_invoice (
		 id integer primary key autoincrement,
		 invoice_number varchar(30) unique not null,
		 date date not null,
		 due_date date,
		 terms varchar(30),
		 purchase_id integer not null,
		 delivery_receipt_id integer,
		 supplier_name varchar(150),
		 supplier_invoice_ref varchar(50),
		 total_amount numeric default 0,
		 status varchar(20) default 'Open',
		 notes text,
		 created_at datetime,
		 created_by_id integer,
		 amount_paid_stored numeric default 0
		)`,
		`CREATE TABLE ap_invoice_line (
		 id integer primary key autoincrement,
		 apinvoice_id integer not null,
		 category varchar(50),
		 item_name varchar(100) not null,
		 unit varchar(20) not null,
		 quantity numeric not null,
		 unit_price numeric not null
		)`,
		`CREATE TABLE ap_payment (
		 id integer primary key autoincrement,
		 payment_number varchar(30) unique not null,
		 date date not null,
		 supplier_id integer,
		 supplier_name_snapshot varchar(150),
		 total_amount numeric default 0,
		 payment_method varchar(20) default 'Cash',
		 reference_number varchar(50),
		 notes text,
		 created_at datetime,
		 created_by_id integer
		)`,
		`CREATE TABLE ap_payment_line (
		 id integer primary key autoincrement,
		 payment_id integer not null,
		 apinvoice_id integer not null,
		 amount_applied numeric not null
		)`,
		`CREATE TABLE gl_account (
		 id integer primary key autoincrement,
		 code varchar(20) unique not null,
		 name varchar(150) not null,
		 section varchar(20) not null,
		 account_type varchar(10) not null default 'POSTING',
		 normal_balance varchar(6) not null,
		 parent_id integer,
		 is_active boolean default 1,
		 is_system boolean default 0,
		 description text,
		 created_at datetime
		)`,
		`CREATE TABLE account_determination (
		 id integer primary key autoincrement,
		 module varchar(20) not null,
		 posting_event varchar(40) not null,
		 item_category varchar(50),
		 gl_account_id integer not null,
		 is_active boolean default 1,
		 notes text
		)`,
		`CREATE TABLE payment_method_account (
		 id integer primary key autoincrement,
		 payment_method varchar(30) not null,
		 bank_name varchar(100),
		 gl_account_id integer not null,
		 direction varchar(10) not null default 'BOTH',
		 is_active boolean default 1
		)`,
		`CREATE TABLE journal_entry (
		 id integer primary key autoincrement,
		 entry_number varchar(30) unique not null,
		 date date not null,
		 module varchar(20) not null,
		 source_type varchar(30) not null,
		 source_id integer,
		 source_ref varchar(50),
		 narration text,
		 status varchar(10) not null default 'POSTED',
		 is_reversal boolean default 0,
		 reversed_entry_id integer,
		 created_by_id integer,
		 created_at datetime
		)`,
		`CREATE TABLE je_line (
		 id integer primary key autoincrement,
		 journal_entry_id integer not null,
		 line_number integer not null,
		 gl_account_id integer not null,
		 debit numeric,
		 credit numeric,
		 description varchar(200)
		)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.DB.Exec(s).Error)
	}
}

// seed minimal accounts and determination records that purchasing service relies on.
func seedAccounts(t *testing.T) {
	// create a catch-all GL account for each event
	names := []string{"GRNI", "INVENTORY_RECEIVED", "AP_PAYABLE", "AP_CLEARING"}
	for _, nm := range names {
		a := models.GLAccount{
			Code:          nm,
			Name:          nm,
			Section:       "ASSET",
			AccountType:   "POSTING",
			NormalBalance: "DEBIT",
		}
		require.NoError(t, db.DB.Create(&a).Error)
		rule := models.AccountDetermination{
			Module:      "PURCHASING",
			PostingEvent: nm,
			GLAccountID: a.ID,
		}
		require.NoError(t, db.DB.Create(&rule).Error)
	}
	// cash account for payments
	cash := models.GLAccount{
		Code:          "CASH",
		Name:          "Cash",
		Section:       "ASSET",
		AccountType:   "POSTING",
		NormalBalance: "DEBIT",
	}
	require.NoError(t, db.DB.Create(&cash).Error)
	pma := models.PaymentMethodAccount{
		PaymentMethod: "Cash",
		Direction:     "OUTFLOW",
		GLAccountID:   cash.ID,
	}
	require.NoError(t, db.DB.Create(&pma).Error)
}

// create a simple purchase record and return its ID along with total cost
func createSamplePurchase(t *testing.T) (uint, float64) {
	p := &models.Purchase{
		Date:      time.Now(),
		Category:  "Feed",
		ItemName:  "Layer Mash",
		Quantity:  100,
		Unit:      "kg",
		UnitPrice: 20.0,
		TotalCost: 2000.0,
		Supplier:  "Test Supplier",
	}
	require.NoError(t, CreatePurchase(p))
	return p.ID, p.TotalCost
}

// scenario: complete purchase -> full delivery -> invoice -> full payment
func TestFullPurchaseCycle(t *testing.T) {
	setupTestDB(t)
	seedAccounts(t)

	// 1. create purchase
	pid, total := createSamplePurchase(t)

	// 2. create and confirm delivery receipt for full quantity
	drItems := []models.DeliveryReceiptItem{{
		Category:         "Feed",
		ItemName:         "Layer Mash",
		Unit:             "kg",
		QuantityOrdered:  100,
		QuantityReceived: 100,
		UnitPrice:        20.0,
	}}
	dr, err := CreateDeliveryReceipt(pid, "2025-01-01", "Receiver", "SUP-DR-123", "note", drItems, nil)
	require.NoError(t, err)
	require.Equal(t, "Draft", dr.Status)

	err = ConfirmDeliveryReceipt(dr.ID, nil)
	require.NoError(t, err)

	// purchase.amount_received must equal total
	p, err := GetPurchase(pid)
	require.NoError(t, err)
	require.Equal(t, total, p.AmountReceived)
	// after receipt the purchase should be marked Delivered
	require.Equal(t, "Delivered", p.PaymentStatus)

	// 3. create AP invoice linked to DR
	invItems := []models.APInvoiceItem{{
		Category:  "Feed",
		ItemName:  "Layer Mash",
		Unit:      "kg",
		Quantity:  100,
		UnitPrice: 20.0,
	}}
	inv, err := CreateAPInvoice(pid, &dr.ID, "2025-01-02", "30d", "Test Supplier", "INV-001", "notes", nil, invItems, nil)
	require.NoError(t, err)
	require.Equal(t, total, inv.TotalAmount)

	// verify purchase.amount_invoiced was updated
	p, _ = GetPurchase(pid)
	require.Equal(t, total, p.AmountInvoiced)

	// 4. make payment covering entire invoice
	lines := []APPaymentLineInput{{APInvoiceID: inv.ID, AmountApplied: total}}
	pay, err := CreateAPPayment("2025-01-03", nil, "Test Supplier", "Cash", "REF123", "notes", lines, nil)
	require.NoError(t, err)
	require.Equal(t, total, pay.TotalAmount)

	// verify invoice status changed to Paid
	inv2, _ := GetAPInvoice(inv.ID)
	require.Equal(t, "Paid", inv2.Status)

	// purchase.amount_settled updated
	p, _ = GetPurchase(pid)
	require.Equal(t, total, p.AmountSettled)

	// ensure delivery receipts list preloads purchase data
	drs, err := ListDeliveryReceipts(0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(drs), 1)
	require.NotNil(t, drs[0].Purchase)
	require.Equal(t, "Test Supplier", drs[0].Purchase.Supplier)
}

// scenario: partial deliveries and partial payment
func TestPartialDeliveryAndPayment(t *testing.T) {
	setupTestDB(t)
	seedAccounts(t)

	pid, total := createSamplePurchase(t)

	// first partial delivery (50%)
	dr1Items := []models.DeliveryReceiptItem{{
		Category:         "Feed",
		ItemName:         "Layer Mash",
		Unit:             "kg",
		QuantityOrdered:  100,
		QuantityReceived: 50,
		UnitPrice:        20.0,
	}}
	dr1, err := CreateDeliveryReceipt(pid, "2025-01-01", "Receiver", "SUP-DR-001", "", dr1Items, nil)
	require.NoError(t, err)
	require.NoError(t, ConfirmDeliveryReceipt(dr1.ID, nil))

	p, _ := GetPurchase(pid)
	require.Equal(t, total/2, p.AmountReceived)
	// first partial delivery -> status Partial
	require.Equal(t, "Partial", p.PaymentStatus)

	// second partial delivery (remaining 50%)
	dr2Items := []models.DeliveryReceiptItem{{
		Category:         "Feed",
		ItemName:         "Layer Mash",
		Unit:             "kg",
		QuantityOrdered:  100,
		QuantityReceived: 50,
		UnitPrice:        20.0,
	}}
	dr2, err := CreateDeliveryReceipt(pid, "2025-01-05", "Receiver", "SUP-DR-002", "", dr2Items, nil)
	require.NoError(t, err)
	require.NoError(t, ConfirmDeliveryReceipt(dr2.ID, nil))

	p, _ = GetPurchase(pid)
	require.Equal(t, total, p.AmountReceived)
	// second delivery completes the purchase
	require.Equal(t, "Delivered", p.PaymentStatus)

	// create invoice only for first delivery (50%)
	invItems := []models.APInvoiceItem{{
		Category:  "Feed",
		ItemName:  "Layer Mash",
		Unit:      "kg",
		Quantity:  50,
		UnitPrice: 20.0,
	}}
	inv, err := CreateAPInvoice(pid, &dr1.ID, "2025-01-06", "30d", "Test Supplier", "INV-PART", "", nil, invItems, nil)
	require.NoError(t, err)
	require.Equal(t, total/2, inv.TotalAmount)

	// pay only half of that invoice
	lines := []APPaymentLineInput{{APInvoiceID: inv.ID, AmountApplied: total / 4}}
	pay, err := CreateAPPayment("2025-01-07", nil, "Test Supplier", "Cash", "REF-P", "", lines, nil)
	require.NoError(t, err)
	require.Equal(t, total/4, pay.TotalAmount)
	inv2, _ := GetAPInvoice(inv.ID)
	require.Equal(t, "Partial", inv2.Status)
	p, _ = GetPurchase(pid)
	require.Equal(t, total/4, p.AmountSettled)
}
