package purchasing

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"ricemill/app/db"
	"ricemill/app/models"
)

// helper to set up an in-memory database and migrate all tables.
func setupTestDB(t *testing.T) {
	var err error
	dsn := fmt.Sprintf("file:db_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db.DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

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
		 wht_category varchar(30) default 'NONE',
		 is_vat_registered boolean default 0,
		 created_at datetime
		)`,
		// SAP B1-aligned PO header (OPOR)
		`CREATE TABLE purchase_header (
		 id integer primary key autoincrement,
		 po_number varchar(50) unique,
		 doc_num integer default 0,
		 status varchar(20) default 'Open',
		 supplier_id integer,
		 supplier_code varchar(15),
		 supplier_name varchar(100),
		 posting_date date not null default '2000-01-01',
		 delivery_date date,
		 tax_date date,
		 ref_number varchar(100),
		 currency varchar(3) default 'PHP',
		 payment_method varchar(30),
		 doc_total numeric default 0,
		 vat_sum numeric default 0,
		 comments varchar(254),
		 created_by_id integer
		)`,
		// SAP B1-aligned PO line (POR1) — FK is header_id (GORM naming for HeaderID)
		`CREATE TABLE purchase_line (
		 id integer primary key autoincrement,
		 header_id integer not null,
		 line_num integer default 0,
		 item_code varchar(30),
		 description varchar(100),
		 unit varchar(20),
		 i_uom_entry integer not null default 0,
		 quantity numeric default 0,
		 open_qty numeric default 0,
		 price numeric default 0,
		 line_total numeric default 0,
		 warehouse_code varchar(8),
		 account_code varchar(15),
		 tax_code varchar(8),
		 project_code varchar(20),
		 cost_center varchar(8),
		 line_status varchar(1) default 'O',
		 base_doc_entry integer,
		 base_doc_num integer,
		 base_line_num integer
		)`,
		// SAP B1-aligned GRPO header (OPDN) — purchase_header_id replaces purchase_id
		`CREATE TABLE delivery_receipt (
		 id integer primary key autoincrement,
		 dr_number varchar(30) unique not null,
		 doc_num integer default 0,
		 status varchar(20) default 'Draft',
		 supplier_id integer,
		 supplier_code varchar(15),
		 supplier_name varchar(100),
		 date date not null default '2000-01-01',
		 delivery_date date,
		 tax_date date,
		 supplier_dr_ref varchar(50),
		 currency varchar(3) default 'PHP',
		 amount_invoiced numeric default 0,
		 vat_sum numeric default 0,
		 notes text,
		 received_by varchar(100),
		 purchase_header_id integer,
		 created_at datetime,
		 created_by_id integer
		)`,
		// SAP B1-aligned GRPO line (PDN1)
		`CREATE TABLE dr_line (
		 id integer primary key autoincrement,
		 delivery_receipt_id integer not null,
		 line_num integer default 0,
		 item_code varchar(30),
		 item_name varchar(100) not null,
		 unit varchar(20) not null,
		 category varchar(50),
		 quantity_ordered numeric default 0,
		 quantity_received numeric default 0,
		 open_qty numeric default 0,
		 uom_entry integer not null default 0,
		 unit_price numeric default 0,
		 line_total numeric default 0,
		 warehouse_code varchar(8),
		 account_code varchar(15),
		 tax_code varchar(8),
		 project_code varchar(20),
		 cost_center varchar(8),
		 line_status varchar(1) default 'O',
		 base_doc_entry integer,
		 base_line_num integer
		)`,
		// SAP B1-aligned AP Invoice (OPCH) — purchase_header_id replaces purchase_id
		`CREATE TABLE ap_invoice (
		 id integer primary key autoincrement,
		 invoice_number varchar(30) unique not null,
		 doc_num integer default 0,
		 status varchar(20) default 'Open',
		 supplier_id integer,
		 supplier_code varchar(15),
		 supplier_name varchar(150),
		 date date not null default '2000-01-01',
		 due_date date,
		 tax_date date,
		 terms varchar(30),
		 supplier_invoice_ref varchar(50),
		 currency varchar(3) default 'PHP',
		 total_amount numeric default 0,
		 vat_sum numeric default 0,
		 notes text,
		 base_doc_entry integer,
		 base_doc_num integer,
		 purchase_header_id integer,
		 delivery_receipt_id integer,
		 amount_paid_stored numeric default 0,
		 vat_exclusive_amount numeric default 0,
		 wht_rate numeric default 0,
		 wht_amount numeric default 0,
		 wht_atc_code varchar(10),
		 net_payable numeric default 0,
		 created_at datetime,
		 created_by_id integer
		)`,
		`CREATE TABLE ap_invoice_line (
		 id integer primary key autoincrement,
		 apinvoice_id integer not null,
		 line_num integer default 0,
		 item_code varchar(30),
		 item_name varchar(100) not null,
		 unit varchar(20) not null,
		 category varchar(50),
		 quantity numeric not null,
		 open_qty numeric default 0,
		 uom_entry integer not null default 0,
		 unit_price numeric not null,
		 line_total numeric default 0,
		 warehouse_code varchar(8),
		 account_code varchar(15),
		 tax_code varchar(8),
		 project_code varchar(20),
		 cost_center varchar(8),
		 line_status varchar(1) default 'O',
		 base_doc_entry integer,
		 base_line_num integer,
		 dr_line_id integer
		)`,
		`CREATE TABLE ap_payment (
		 id integer primary key autoincrement,
		 payment_number varchar(30) unique not null,
		 status varchar(20) default 'Posted',
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
			Module:       "PURCHASING",
			PostingEvent: nm,
			GLAccountID:  a.ID,
		}
		require.NoError(t, db.DB.Create(&rule).Error)
	}
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

// createSamplePurchaseHeader creates a PO with one line (100 kg Layer Mash @ ₱20).
func createSamplePurchaseHeader(t *testing.T) (*models.PurchaseHeader, float64) {
	ph, err := CreatePurchaseHeader(CreatePurchaseParams{
		Date:         "2025-01-01",
		SupplierName: "Test Supplier",
		Lines: []POLineInput{
			{Category: "Feed", ItemName: "Layer Mash", Unit: "kg", Quantity: 100, UnitPrice: 20.0},
		},
	}, nil)
	require.NoError(t, err)
	total := 100 * 20.0
	return ph, total
}

// TestFullPurchaseCycle: PO → confirm DR → AP Invoice → AP Payment → Paid
func TestFullPurchaseCycle(t *testing.T) {
	setupTestDB(t)
	seedAccounts(t)

	// 1. Create PurchaseHeader with one line
	ph, total := createSamplePurchaseHeader(t)
	require.NotZero(t, ph.ID)
	require.Len(t, ph.Lines, 1)
	require.Equal(t, 100.0, ph.Lines[0].OpenQty)

	// 2. Create and confirm delivery receipt for full quantity
	phLineID := int(ph.Lines[0].ID)
	drItems := []models.DeliveryReceiptItem{{
		Description:     "Layer Mash",
		Category:        "Feed",
		Unit:            "kg",
		QuantityOrdered: 100,
		Quantity:        100,
		Price:           20.0,
		BaseLineNum:     &phLineID, // direct PurchaseLine.ID link for open_qty decrement
	}}
	dr, err := CreateDeliveryReceipt(ph.ID, "2025-01-01", "Receiver", "SUP-DR-123", "note", drItems, nil)
	require.NoError(t, err)
	require.Equal(t, "Draft", dr.Status)

	err = ConfirmDeliveryReceipt(dr.ID, nil)
	require.NoError(t, err)

	// After confirming: PurchaseLine.open_qty should be 0 (fully received)
	var updatedLine models.PurchaseLine
	require.NoError(t, db.DB.First(&updatedLine, ph.Lines[0].ID).Error)
	require.Equal(t, 0.0, updatedLine.OpenQty)

	// 3. Create AP invoice linked to DR and PO header
	invItems := []models.APInvoiceItem{{
		Description: "Layer Mash",
		Category:    "Feed",
		Unit:        "kg",
		Quantity:    100,
		Price:       20.0,
	}}
	inv, err := CreateAPInvoice(ph.ID, &dr.ID, "2025-01-02", "30d", "Test Supplier", "INV-001", "notes", nil, invItems, nil, 0, 0, 0, "")
	require.NoError(t, err)
	require.Equal(t, total, inv.DocTotal)
	require.Equal(t, "Open", inv.Status)

	// 4. Make payment covering entire invoice
	lines := []APPaymentLineInput{{APInvoiceID: inv.ID, AmountApplied: total}}
	pay, err := CreateAPPayment("2025-01-03", nil, "Test Supplier", "Cash", "REF123", "notes", lines, nil)
	require.NoError(t, err)
	require.Equal(t, total, pay.TotalAmount)

	// Invoice should now be Paid
	inv2, err := GetAPInvoice(inv.ID)
	require.NoError(t, err)
	require.Equal(t, "Paid", inv2.Status)

	// ListDeliveryReceipts should preload PurchaseHeader
	drs, err := ListDeliveryReceipts(0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(drs), 1)
	require.NotNil(t, drs[0].PurchaseHeader)
}

// TestPartialDeliveryAndPayment: two partial DRs, partial invoice, partial payment.
func TestPartialDeliveryAndPayment(t *testing.T) {
	setupTestDB(t)
	seedAccounts(t)

	ph, total := createSamplePurchaseHeader(t)
	phLineID := int(ph.Lines[0].ID)

	// First partial delivery (50%)
	dr1Items := []models.DeliveryReceiptItem{{
		Description:     "Layer Mash",
		Category:        "Feed",
		Unit:            "kg",
		QuantityOrdered: 100,
		Quantity:        50,
		Price:           20.0,
		BaseLineNum:     &phLineID,
	}}
	dr1, err := CreateDeliveryReceipt(ph.ID, "2025-01-01", "Receiver", "SUP-DR-001", "", dr1Items, nil)
	require.NoError(t, err)
	require.NoError(t, ConfirmDeliveryReceipt(dr1.ID, nil))

	// OpenQty should be 50 remaining
	var line models.PurchaseLine
	require.NoError(t, db.DB.First(&line, ph.Lines[0].ID).Error)
	require.Equal(t, 50.0, line.OpenQty)

	// Second partial delivery (remaining 50%)
	dr2Items := []models.DeliveryReceiptItem{{
		Description:     "Layer Mash",
		Category:        "Feed",
		Unit:            "kg",
		QuantityOrdered: 100,
		Quantity:        50,
		Price:           20.0,
		BaseLineNum:     &phLineID,
	}}
	dr2, err := CreateDeliveryReceipt(ph.ID, "2025-01-05", "Receiver", "SUP-DR-002", "", dr2Items, nil)
	require.NoError(t, err)
	require.NoError(t, ConfirmDeliveryReceipt(dr2.ID, nil))

	// OpenQty should now be 0
	require.NoError(t, db.DB.First(&line, ph.Lines[0].ID).Error)
	require.Equal(t, 0.0, line.OpenQty)

	// Invoice for only the first 50 units (DR1)
	invItems := []models.APInvoiceItem{{
		Description: "Layer Mash",
		Category:    "Feed",
		Unit:        "kg",
		Quantity:    50,
		Price:       20.0,
	}}
	inv, err := CreateAPInvoice(ph.ID, &dr1.ID, "2025-01-06", "30d", "Test Supplier", "INV-PART", "", nil, invItems, nil, 0, 0, 0, "")
	require.NoError(t, err)
	require.Equal(t, total/2, inv.DocTotal)

	// Pay only 25% of the invoice
	lines := []APPaymentLineInput{{APInvoiceID: inv.ID, AmountApplied: total / 4}}
	pay, err := CreateAPPayment("2025-01-07", nil, "Test Supplier", "Cash", "REF-P", "", lines, nil)
	require.NoError(t, err)
	require.Equal(t, total/4, pay.TotalAmount)

	inv2, err := GetAPInvoice(inv.ID)
	require.NoError(t, err)
	require.Equal(t, "Partial", inv2.Status)
}
