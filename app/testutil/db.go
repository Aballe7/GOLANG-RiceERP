// Package testutil provides shared helpers for ricemill service integration tests.
//
// # Why raw DDL instead of AutoMigrateAll?
//
// AutoMigrateAll uses the GORM model tags, which emit MySQL-specific SQL such as
// DEFAULT CURRENT_TIMESTAMP(3).  SQLite rejects the precision specifier "(3)",
// so AutoMigrateAll fails on the very first table.  Raw DDL lets us use
// CURRENT_TIMESTAMP (no parentheses), which both SQLite and MySQL accept.
//
// # Why in-memory SQLite?
//
// Each call to SetupDB opens a unique DSN (nanosecond timestamp), so tests are
// fully isolated from each other and from the production MySQL database.
// The GORM API is identical across both drivers, meaning the service layer is
// exercised without any code changes.
//
// # MySQL-specific SQL bypass
//
// A small number of code paths use MySQL-only expressions:
//   - upsertOITWOnHand    → ON DUPLICATE KEY UPDATE
//   - ConfirmDeliveryOrder → GREATEST(0, is_commited - ?)
//
// Both are only reached after a successful OITM lookup.  Integration tests that
// omit OITM rows naturally trigger the `continue` guard, skipping those paths
// while still exercising all financial and status-transition logic.
package testutil

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

// SetupDB opens a fresh in-memory SQLite instance, wires it to db.DB, and
// creates all tables required by the sales O2C and purchasing P2P cycles.
//
// Call at the top of every integration test function.
func SetupDB(t *testing.T) {
	t.Helper()
	dsn := fmt.Sprintf("file:test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	var err error
	db.DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "open in-memory SQLite")
	createAllTables(t)
}

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

// ─────────────────────────────────────────────────────────────────────────────
// SQLite-compatible DDL (no CURRENT_TIMESTAMP(3), no MySQL-only syntax)
// ─────────────────────────────────────────────────────────────────────────────

func createAllTables(t *testing.T) {
	t.Helper()
	for _, stmt := range allTablesDDL() {
		require.NoError(t, db.DB.Exec(stmt).Error, "createAllTables DDL failed:\n%s", stmt)
	}
}

func allTablesDDL() []string {
	return []string{
		// ── Accounting ────────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS gl_account (
			id integer primary key autoincrement,
			code varchar(20) unique not null,
			name varchar(150) not null,
			section varchar(20) not null,
			account_type varchar(10) not null default 'POSTING',
			normal_balance varchar(6) not null,
			cash_flow_class varchar(30) not null default '',
			parent_id integer,
			is_active tinyint(1) default 1,
			is_system tinyint(1) default 0,
			description text,
			created_at datetime default CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS account_determination (
			id integer primary key autoincrement,
			module varchar(20) not null,
			posting_event varchar(40) not null,
			item_category varchar(50),
			gl_account_id integer not null,
			is_active tinyint(1) default 1,
			notes text
		)`,
		`CREATE TABLE IF NOT EXISTS payment_method_account (
			id integer primary key autoincrement,
			payment_method varchar(30) not null,
			bank_name varchar(100),
			gl_account_id integer not null,
			direction varchar(10) not null default 'BOTH',
			is_active tinyint(1) default 1
		)`,
		`CREATE TABLE IF NOT EXISTS journal_entry (
			id integer primary key autoincrement,
			entry_number varchar(30) unique not null,
			date date not null,
			module varchar(20) not null,
			source_type varchar(30) not null,
			source_id integer,
			source_ref varchar(50),
			narration text,
			status varchar(10) not null default 'POSTED',
			is_reversal tinyint(1) default 0,
			reversed_entry_id integer,
			created_by_id integer,
			updated_by_id integer,
			version integer default 1,
			created_at datetime default CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS je_line (
			id integer primary key autoincrement,
			journal_entry_id integer not null,
			line_number integer not null,
			gl_account_id integer not null,
			debit numeric,
			credit numeric,
			description varchar(200)
		)`,

		// ── Customer / Price Group ─────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS price_group (
			id integer primary key autoincrement,
			name varchar(50) unique not null,
			description varchar(200),
			is_active tinyint(1) default 1,
			created_at datetime default CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS pg_line (
			id integer primary key autoincrement,
			price_group_id integer not null,
			egg_size varchar(30) not null,
			unit varchar(10) not null default 'Tray',
			price numeric default 0,
			updated_at datetime
		)`,
		`CREATE TABLE IF NOT EXISTS customer (
			id integer primary key autoincrement,
			name varchar(100) not null unique,
			address varchar(200),
			delivery_address varchar(200),
			contact_number varchar(20),
			email varchar(100),
			customer_type varchar(20) default 'Walk-in',
			notes text,
			is_active tinyint(1) default 1,
			created_at datetime default CURRENT_TIMESTAMP,
			created_by_id integer,
			price_group_id integer
		)`,

		// ── Sales Order ───────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS sales_order (
			id integer primary key autoincrement,
			sales_order_number varchar(30) unique not null,
			doc_status varchar(20) default 'Draft',
			date date not null,
			customer_id integer,
			customer_name_snapshot varchar(100),
			customer_address_snapshot varchar(200),
			customer_contact_snapshot varchar(20),
			payment_method varchar(20),
			payment_status varchar(20) default 'Unpaid',
			amount_paid numeric default 0,
			due_date date,
			terms varchar(30),
			grand_total numeric default 0,
			amount_delivered numeric default 0,
			amount_invoiced numeric default 0,
			amount_collected numeric default 0,
			notes text,
			created_at datetime default CURRENT_TIMESTAMP,
			created_by_id integer,
			updated_by_id integer,
			version integer default 1
		)`,
		`CREATE TABLE IF NOT EXISTS so_line (
			id integer primary key autoincrement,
			order_id integer not null,
			sku varchar(30) not null,
			unit varchar(10) not null,
			uom_entry integer not null default 0,
			quantity numeric not null,
			open_qty numeric default 0,
			price_per_unit numeric not null,
			line_total numeric not null
		)`,

		// ── Delivery Order ────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS delivery_order (
			id integer primary key autoincrement,
			delivery_number varchar(30) unique not null,
			date date not null,
			sales_order_id integer not null,
			status varchar(20) default 'Draft',
			delivered_by varchar(100),
			delivered_by_id integer,
			notes text,
			created_at datetime default CURRENT_TIMESTAMP,
			created_by_id integer,
			updated_by_id integer,
			amount_invoiced numeric default 0
		)`,
		`CREATE TABLE IF NOT EXISTS do_line (
			id integer primary key autoincrement,
			delivery_order_id integer not null,
			sales_order_item_id integer,
			sku varchar(30) not null,
			unit varchar(10) not null,
			uom_entry integer not null default 0,
			quantity_ordered numeric default 0,
			quantity_delivered numeric default 0,
			price_per_unit numeric default 0
		)`,

		// ── AR Invoice ────────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS ar_invoice (
			id integer primary key autoincrement,
			invoice_number varchar(30) unique not null,
			date date not null,
			due_date date,
			terms varchar(30),
			sales_order_id integer,
			customer_id integer,
			customer_name_snapshot varchar(100),
			customer_address_snapshot varchar(200),
			customer_contact_snapshot varchar(20),
			total_amount numeric default 0,
			status varchar(20) default 'Open',
			notes text,
			created_at datetime default CURRENT_TIMESTAMP,
			created_by_id integer,
			updated_by_id integer,
			amount_collected numeric default 0,
			version integer default 1
		)`,
		`CREATE TABLE IF NOT EXISTS ar_invoice_line (
			id integer primary key autoincrement,
			ar_invoice_id integer not null,
			delivery_order_id integer,
			sku varchar(30) not null,
			unit varchar(10) not null,
			uom_entry integer not null default 0,
			quantity numeric not null,
			price_per_unit numeric not null
		)`,
		// Many2Many junction: ARInvoice ↔ DeliveryOrder.
		// GORM column naming: <model_singular_snake>_id for each side.
		`CREATE TABLE IF NOT EXISTS ar_invoice_delivery (
			ar_invoice_id integer not null,
			delivery_order_id integer not null,
			primary key (ar_invoice_id, delivery_order_id)
		)`,

		// ── Collection ────────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS collection (
			id integer primary key autoincrement,
			collection_number varchar(30) unique not null,
			date date not null,
			customer_id integer,
			customer_name_snapshot varchar(100),
			total_amount numeric default 0,
			payment_method varchar(20) default 'Cash',
			reference_number varchar(50),
			notes text,
			status varchar(20) default 'Posted',
			created_at datetime default CURRENT_TIMESTAMP,
			created_by_id integer,
			updated_by_id integer
		)`,
		`CREATE TABLE IF NOT EXISTS collection_line (
			id integer primary key autoincrement,
			collection_id integer not null,
			ar_invoice_id integer not null,
			amount_applied numeric not null
		)`,

		// ── Item Master (OITB + OITM) ────────────────────────────────────────
		// Minimal columns: only those read/written by the sales & purchasing service layer.
		// Raw SQL inserts in SeedOITM bypass GORM's full-model insert, so extra columns
		// (num_in_buy, lst_evl_pric, …) are intentionally omitted to keep the DDL lean.
		`CREATE TABLE IF NOT EXISTS oitb (
			itms_grp_cod integer primary key autoincrement,
			itms_grp_nam varchar(100) not null unique,
			description  varchar(255) default '',
			is_active    tinyint(1)   default 1,
			for_sales    tinyint(1)   default 1,
			for_purchasing tinyint(1) default 1,
			for_inventory  tinyint(1) default 1,
			for_production tinyint(1) default 1
		)`,
		`CREATE TABLE IF NOT EXISTS oitm (
			id           integer primary key autoincrement,
			item_code    varchar(50)  unique not null,
			item_name    varchar(150) not null default '',
			itms_grp_cod integer      not null default 0,
			invntry_uom  varchar(20)  not null default 'unit',
			i_uom_entry  integer      not null default 0,
			ugp_entry    integer      not null default 0,
			on_hand      numeric      default 0,
			is_commited  numeric      default 0,
			avg_price    numeric      default 0,
			dflt_wh      varchar(10)  default '',
			sell_item    char(1)      default 'Y',
			valid_for    char(1)      default 'Y'
		)`,

		// ── Purchasing (for future purchasing tests using testutil) ───────────
		`CREATE TABLE IF NOT EXISTS supplier (
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
			is_active tinyint(1) default 1,
			wht_category varchar(30) default 'NONE',
			is_vat_registered tinyint(1) default 0,
			status varchar(20) default 'pending',
			created_at datetime default CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS purchase_header (
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
		`CREATE TABLE IF NOT EXISTS purchase_line (
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
		`CREATE TABLE IF NOT EXISTS delivery_receipt (
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
			created_at datetime default CURRENT_TIMESTAMP,
			created_by_id integer
		)`,
		`CREATE TABLE IF NOT EXISTS dr_line (
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
		`CREATE TABLE IF NOT EXISTS ap_invoice (
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
			created_at datetime default CURRENT_TIMESTAMP,
			created_by_id integer
		)`,
		`CREATE TABLE IF NOT EXISTS ap_invoice_line (
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
		`CREATE TABLE IF NOT EXISTS ap_payment (
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
			created_at datetime default CURRENT_TIMESTAMP,
			created_by_id integer
		)`,
		`CREATE TABLE IF NOT EXISTS ap_payment_line (
			id integer primary key autoincrement,
			payment_id integer not null,
			apinvoice_id integer not null,
			amount_applied numeric not null
		)`,
	}
}
