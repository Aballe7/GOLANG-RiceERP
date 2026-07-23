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
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"ricemill/app/db"
)

// dbSeq disambiguates DSNs opened within the same clock tick — on Windows the
// nanosecond timestamp alone can repeat across fast successive tests, silently
// sharing one shared-cache DB between them (unique-constraint failures on seeds).
var dbSeq atomic.Int64

// SetupDB opens a fresh in-memory SQLite instance, wires it to db.DB, and
// creates all tables required by the sales O2C and purchasing P2P cycles.
//
// Call at the top of every integration test function.
func SetupDB(t *testing.T) {
	t.Helper()
	dsn := fmt.Sprintf("file:test_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), dbSeq.Add(1))
	var err error
	db.DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "open in-memory SQLite")
	createAllTables(t)
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
			costing_meth char(1)      default 'A',
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
			frgn_name    varchar(150) default '',
			itms_grp_cod integer      not null default 0,
			code_bars    varchar(50)  default '',
			invnt_item   char(1)      default 'Y',
			sell_item    char(1)      default 'Y',
			prchse_item  char(1)      default 'Y',
			mak_item     char(1)      default 'N',
			invntry_uom  varchar(20)  not null default 'unit',
			purchase_unit varchar(20) default '',
			sales_unit   varchar(20)  default '',
			i_uom_entry  integer      not null default 0,
			s_uom_entry  integer      not null default 0,
			p_uom_entry  integer      not null default 0,
			ugp_entry    integer      not null default 0,
			num_in_buy   numeric      default 1,
			num_in_sale  numeric      default 1,
			on_hand      numeric      default 0,
			is_commited  numeric      default 0,
			on_order     numeric      default 0,
			min_level    numeric      default 0,
			max_level    numeric      default 0,
			lead_time    integer      default 0,
			avg_price    numeric      default 0,
			lst_evl_pric numeric      default 0,
			last_pur_prc numeric      default 0,
			eval_system  char(1)      default 'A',
			pricing_cod  varchar(20)  default '',
			dflt_wh      varchar(10)  default '',
			card_code    varchar(15)  default '',
			supp_cat_num varchar(20)  default '',
			vat_gourp_sa varchar(10)  default '',
			wt_liable    char(1)      default 'N',
			man_btch_num char(1)      default 'N',
			man_ser_num  char(1)      default 'N',
			valid_for    char(1)      default 'Y',
			description  text,
			create_date  date,
			update_date  date
		)`,

		// ── Inventory movement + ledger + per-warehouse stock ─────────────────
		`CREATE TABLE IF NOT EXISTS oign (
			id integer primary key autoincrement,
			gr_number varchar(30) unique not null,
			posting_date date not null,
			doc_due_date date,
			status varchar(20) default 'Open',
			remarks text,
			doc_total numeric default 0,
			created_by_id integer,
			updated_by_id integer,
			created_at datetime default CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ign1 (
			id integer primary key autoincrement,
			goods_receipt_id integer not null,
			line_num integer default 0,
			item_id integer not null,
			item_code varchar(50),
			description varchar(200),
			quantity numeric not null,
			unit varchar(20),
			uom_entry integer not null default 0,
			price numeric default 0,
			line_total numeric default 0,
			warehouse_code varchar(20),
			account_code varchar(20),
			project varchar(50),
			batch_no varchar(30) default ''
		)`,
		`CREATE TABLE IF NOT EXISTS oige (
			id integer primary key autoincrement,
			gi_number varchar(30) unique not null,
			posting_date date not null,
			doc_due_date date,
			status varchar(20) default 'Open',
			remarks text,
			doc_total numeric default 0,
			created_by_id integer,
			updated_by_id integer,
			created_at datetime default CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ige1 (
			id integer primary key autoincrement,
			goods_issue_id integer not null,
			line_num integer default 0,
			item_id integer not null,
			item_code varchar(50),
			description varchar(200),
			quantity numeric not null,
			unit varchar(20),
			uom_entry integer not null default 0,
			price numeric default 0,
			line_total numeric default 0,
			warehouse_code varchar(20),
			account_code varchar(20),
			project varchar(50)
		)`,
		`CREATE TABLE IF NOT EXISTS oitw (
			item_code varchar(50) not null,
			whs_code varchar(10) not null,
			on_hand numeric default 0,
			is_commited numeric default 0,
			on_order numeric default 0,
			min_stock numeric default 0,
			max_stock numeric default 0,
			avg_price numeric default 0,
			dflt_bin varchar(20),
			locked char(1) default 'N',
			primary key (item_code, whs_code)
		)`,
		`CREATE TABLE IF NOT EXISTS oivl (
			doc_entry integer primary key autoincrement,
			item_code varchar(50) not null,
			item_name varchar(150),
			warehouse varchar(10),
			doc_date date,
			trans_type varchar(20),
			doc_num integer default 0,
			in_qty numeric default 0,
			out_qty numeric default 0,
			price numeric default 0,
			value numeric default 0,
			batch_no varchar(30) default '',
			created_by_id integer default 0,
			created_at datetime default CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS itm1 (
			item_code varchar(50) not null,
			price_list integer not null,
			price numeric not null default 0,
			currency varchar(3) default 'PHP',
			price_dec integer default 2,
			primary key (item_code, price_list)
		)`,

		// ── UoM master / group / per-item mapping + warehouses + lookups ──────
		`CREATE TABLE IF NOT EXISTS ouom (
			uom_entry integer primary key autoincrement,
			uom_code varchar(20) not null unique,
			uom_name varchar(100) not null default '',
			length numeric default 0, l_type integer default 0,
			width numeric default 0, w_type integer default 0,
			height numeric default 0, h_type integer default 0,
			volume numeric default 0, v_type integer default 0,
			weight numeric default 0, wgt_type integer default 0,
			user_sign integer default 0
		)`,
		`CREATE TABLE IF NOT EXISTS ougp (
			ugp_entry integer primary key autoincrement,
			ugp_code varchar(20) not null unique,
			ugp_name varchar(100) not null default '',
			base_uom integer default 0,
			user_sign integer default 0,
			user_sign2 integer default 0,
			update_date datetime,
			create_date datetime default CURRENT_TIMESTAMP,
			data_source char(1) default 'M'
		)`,
		`CREATE TABLE IF NOT EXISTS itm12 (
			item_code varchar(50) not null,
			uom_entry integer not null,
			uom_type char(1) default '',
			bar_code varchar(50) default '',
			primary key (item_code, uom_entry)
		)`,
		`CREATE TABLE IF NOT EXISTS owhs (
			whs_code varchar(10) primary key,
			whs_name varchar(100) not null default '',
			location varchar(200) default '',
			street varchar(200) default '',
			zip_code varchar(20) default '',
			city varchar(100) default '',
			state varchar(100) default '',
			phone varchar(50) default '',
			inactive char(1) default 'N'
		)`,
		`CREATE TABLE IF NOT EXISTS purchasing_lookup (
			id integer primary key autoincrement,
			category varchar(50) not null,
			value varchar(100) not null,
			sort_order integer default 0,
			is_active tinyint(1) default 1
		)`,

		// ── Production: BOM (oitt/itt1) + Work/Milling orders (owor/wor1) ──────
		`CREATE TABLE IF NOT EXISTS oitt (
			code varchar(50) primary key,
			tree_type char(1) default 'P',
			warehouse varchar(10),
			quantity numeric default 1,
			notes text,
			updated_at datetime
		)`,
		`CREATE TABLE IF NOT EXISTS itt1 (
			id integer primary key autoincrement,
			code varchar(50) not null,
			item_code varchar(50) not null,
			item_name varchar(200),
			quantity numeric not null default 1,
			warehouse varchar(10),
			issue_method char(1) default 'M',
			uom_code varchar(20),
			uom_entry integer default 0,
			price numeric default 0,
			output_type char(1) default ''
		)`,
		`CREATE TABLE IF NOT EXISTS owor (
			doc_entry integer primary key autoincrement,
			doc_num varchar(30) unique not null,
			order_type char(1) default 'P',
			item_code varchar(50) not null,
			item_name varchar(200),
			planned_qty numeric not null,
			cmplt_qty numeric default 0,
			rjct_qty numeric default 0,
			warehouse varchar(10),
			start_date date,
			due_date date,
			status char(1) default 'P',
			notes text,
			batch_no varchar(30) default '',
			moisture_pct numeric default 0,
			out_moisture_pct numeric default 0,
			conv_cost numeric default 0,
			created_by_id integer,
			updated_by_id integer,
			created_at datetime default CURRENT_TIMESTAMP,
			version integer default 1,
			goods_issue_id integer,
			goods_issue_number varchar(30),
			start_je_id integer,
			goods_receipt_id integer,
			goods_receipt_number varchar(30),
			complete_je_id integer
		)`,
		`CREATE TABLE IF NOT EXISTS wor1 (
			id integer primary key autoincrement,
			doc_entry integer not null,
			line_num integer default 0,
			line_dir char(1) default 'I',
			item_id integer default 0,
			item_code varchar(50) not null,
			item_name varchar(200),
			item_category varchar(50),
			planned_qty numeric not null,
			issued_qty numeric default 0,
			warehouse varchar(10),
			issue_method char(1) default 'M',
			uom_entry integer default 0,
			uom_code varchar(20),
			price numeric default 0,
			output_type char(1) default ''
		)`,

		// ── Settings (recovery band, conversion-cost standard) ─────────────────
		`CREATE TABLE IF NOT EXISTS farm_settings (
			key varchar(50) primary key,
			value text
		)`,

		// ── Purchasing ────────────────────────────────────────────────────────
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
			approved_by varchar(100) default '',
			approved_by_id integer,
			approved_at datetime,
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
