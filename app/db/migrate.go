package db

import (
	"ricemill/app/models"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AutoMigrateAll creates/updates all tables.
func AutoMigrateAll() error {
	return DB.AutoMigrate(
		// Users & Audit
		&models.User{},
		&models.AuditLog{},

		// Inventory — Item Master (SAP B1 OITM / OITB and sub-tables)
		&models.OITB{},        // Item Groups (was ItemCategory / item_categories)
		&models.OITM{},        // Item Master Data (was ItemMaster / item_master)
		&models.UoMMaster{},
		&models.UoMGroup{},
		&models.UoMGroupLine{},
		&models.ItemUoMPrice{}, // ITM9 — now uses item_code PK
		&models.OWHS{},         // Warehouse Master
		&models.OITW{},         // Per-Warehouse Stock
		&models.OIVL{},         // Inventory Ledger / Audit Trail
		&models.ITM1{},         // Item Prices per Price List
		&models.ITM2{},         // Multiple Preferred Vendors per Item
		&models.ITM12{},        // UoM per Item Mapping
		&models.OSPP{},         // Special Prices (BP-Item overrides)
		&models.GoodsReceipt{},
		&models.GoodsReceiptItem{},
		&models.GoodsIssue{},
		&models.GoodsIssueItem{},

		// Purchasing
		&models.Supplier{},
		&models.DeliveryReceipt{},
		&models.DeliveryReceiptItem{},
		&models.PurchaseHeader{},
		&models.PurchaseLine{},
		&models.APInvoice{},
		&models.APInvoiceItem{},
		&models.APPayment{},
		&models.APPaymentLine{},
		&models.PurchasingLookup{},

		// Sales
		&models.Customer{},
		&models.PriceGroup{},
		&models.PriceGroupItem{},
		&models.SalesOrder{},
		&models.SalesOrderItem{},
		&models.DeliveryOrder{},
		&models.DeliveryOrderItem{},
		&models.ARInvoice{},
		&models.ARInvoiceItem{},
		&models.Collection{},
		&models.CollectionLine{},

		// Accounting
		&models.GLAccount{},
		&models.AccountDetermination{},
		&models.PaymentMethodAccount{},
		&models.JournalEntry{},
		&models.JournalEntryLine{},

		// Production
		&models.MillingOrder{},
		&models.MillingOrderLine{},
		&models.ProductTree{},
		&models.ProductTreeLine{},
		&models.WorkOrder{},
		&models.WorkOrderLine{},

		// Fixed Assets
		&models.FixedAsset{},
		&models.DepreciationEntry{},

		// Settings
		&models.FarmSettings{},
	)
}

// PreMigrateFixup repairs created_at columns before AutoMigrateAll runs.
//
// MySQL 9.x rejects  datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP  (without
// the matching fractional-seconds precision on CURRENT_TIMESTAMP), which is
// exactly what GORM's AutoMigrate generates.  We pre-add / pre-fix every
// existing table so GORM sees the column already present and skips the ADD.
//
// Must be called BEFORE AutoMigrateAll.
func PreMigrateFixup() error {
	// 1. Fix any table whose created_at is the wrong base type (e.g. 'date').
	//    MODIFY to datetime(3) NULL so existing date values are preserved.
	type wrongCol struct{ Table string }
	var wrongCols []wrongCol
	DB.Raw(`SELECT table_name AS ` + "`table`" + `
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND column_name  = 'created_at'
		  AND data_type   != 'datetime'`).Scan(&wrongCols)
	for _, c := range wrongCols {
		sql := fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN created_at datetime(3) NULL", c.Table)
		if err := DB.Exec(sql).Error; err != nil {
			return fmt.Errorf("fix %s.created_at type: %w", c.Table, err)
		}
	}

	// 2. For every existing table that has NO created_at column at all, add
	//    one using CURRENT_TIMESTAMP(3) — the precision-qualified form that
	//    MySQL 9.x accepts.  GORM will then skip the ADD when it runs.
	type tblRow struct{ Table string }
	var existing []tblRow
	DB.Raw(`SELECT table_name AS ` + "`table`" + `
		FROM information_schema.tables
		WHERE table_schema = DATABASE()`).Scan(&existing)

	for _, t := range existing {
		var cnt int64
		DB.Raw(`SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = DATABASE()
			  AND table_name   = ?
			  AND column_name  = 'created_at'`, t.Table).Scan(&cnt)
		if cnt == 0 {
			sql := fmt.Sprintf(
				"ALTER TABLE `%s` ADD COLUMN created_at datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)",
				t.Table,
			)
			if err := DB.Exec(sql).Error; err != nil {
				return fmt.Errorf("add %s.created_at: %w", t.Table, err)
			}
		}
	}

	return nil
}

// RunColumnMigrations handles column renames that GORM AutoMigrate cannot do automatically.
func RunColumnMigrations() error {
	// sales_order.invoice_number → sales_order_number
	var oldExists, newExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'sales_order' AND column_name = 'invoice_number'").Scan(&oldExists)
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'sales_order' AND column_name = 'sales_order_number'").Scan(&newExists)

	if oldExists > 0 && newExists > 0 {
		if err := DB.Exec("ALTER TABLE sales_order DROP COLUMN invoice_number").Error; err != nil {
			return fmt.Errorf("drop sales_order.invoice_number: %w", err)
		}
	} else if oldExists > 0 && newExists == 0 {
		if err := DB.Exec("ALTER TABLE sales_order RENAME COLUMN invoice_number TO sales_order_number").Error; err != nil {
			return fmt.Errorf("rename sales_order.invoice_number: %w", err)
		}
	}

	// Add on_hand_qty to oitm if it doesn't exist yet
	var onHandExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'oitm' AND column_name = 'on_hand_qty'").Scan(&onHandExists)
	if onHandExists == 0 {
		if err := DB.Exec("ALTER TABLE oitm ADD COLUMN on_hand_qty DECIMAL(12,3) NOT NULL DEFAULT 0").Error; err != nil {
			return fmt.Errorf("add oitm.on_hand_qty: %w", err)
		}
	}

	// Add is_production_item to oitm if it doesn't exist yet
	var isProdExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'oitm' AND column_name = 'is_production_item'").Scan(&isProdExists)
	if isProdExists == 0 {
		if err := DB.Exec("ALTER TABLE oitm ADD COLUMN is_production_item TINYINT(1) NOT NULL DEFAULT 0").Error; err != nil {
			return fmt.Errorf("add oitm.is_production_item: %w", err)
		}
	}

	// Add i_uom_entry to oitm if it doesn't exist yet
	var iuomExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'oitm' AND column_name = 'i_uom_entry'").Scan(&iuomExists)
	if iuomExists == 0 {
		if err := DB.Exec("ALTER TABLE oitm ADD COLUMN i_uom_entry INT UNSIGNED NOT NULL DEFAULT 0 AFTER `invntry_uom`").Error; err != nil {
			return fmt.Errorf("add oitm.i_uom_entry: %w", err)
		}
	}

	// Add s_uom_entry to oitm if it doesn't exist yet
	var sUomExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'oitm' AND column_name = 's_uom_entry'").Scan(&sUomExists)
	if sUomExists == 0 {
		if err := DB.Exec("ALTER TABLE oitm ADD COLUMN s_uom_entry INT UNSIGNED NOT NULL DEFAULT 0 AFTER `i_uom_entry`").Error; err != nil {
			return fmt.Errorf("add oitm.s_uom_entry: %w", err)
		}
	}

	// Add p_uom_entry to oitm if it doesn't exist yet
	var pUomExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'oitm' AND column_name = 'p_uom_entry'").Scan(&pUomExists)
	if pUomExists == 0 {
		if err := DB.Exec("ALTER TABLE oitm ADD COLUMN p_uom_entry INT UNSIGNED NOT NULL DEFAULT 0 AFTER `s_uom_entry`").Error; err != nil {
			return fmt.Errorf("add oitm.p_uom_entry: %w", err)
		}
	}

	// Add ugp_entry to oitm if it doesn't exist yet
	var ugpExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'oitm' AND column_name = 'ugp_entry'").Scan(&ugpExists)
	if ugpExists == 0 {
		if err := DB.Exec("ALTER TABLE oitm ADD COLUMN ugp_entry INT UNSIGNED NOT NULL DEFAULT 0 AFTER `p_uom_entry`").Error; err != nil {
			return fmt.Errorf("add oitm.ugp_entry: %w", err)
		}
	}

	// Add qty_received to purchase if it doesn't exist yet
	var qtyRecvExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'purchase' AND column_name = 'qty_received'").Scan(&qtyRecvExists)
	if qtyRecvExists == 0 {
		if err := DB.Exec("ALTER TABLE purchase ADD COLUMN qty_received DECIMAL(12,3) NOT NULL DEFAULT 0").Error; err != nil {
			return fmt.Errorf("add purchase.qty_received: %w", err)
		}
	}

	// Add dr_line_id to ap_invoice_line if it doesn't exist yet
	var drLineIdExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'ap_invoice_line' AND column_name = 'dr_line_id'").Scan(&drLineIdExists)
	if drLineIdExists == 0 {
		if err := DB.Exec("ALTER TABLE ap_invoice_line ADD COLUMN dr_line_id INT UNSIGNED NULL DEFAULT NULL").Error; err != nil {
			return fmt.Errorf("add ap_invoice_line.dr_line_id: %w", err)
		}
	}

	// Widen purchase_header.status from varchar(1) to varchar(20) and remap 'O'/'C' to full words
	var phStatusType string
	DB.Raw("SELECT DATA_TYPE FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'purchase_header' AND column_name = 'status'").Scan(&phStatusType)
	if phStatusType == "varchar" {
		var phStatusLen int64
		DB.Raw("SELECT CHARACTER_MAXIMUM_LENGTH FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'purchase_header' AND column_name = 'status'").Scan(&phStatusLen)
		if phStatusLen == 1 {
			DB.Exec(`ALTER TABLE purchase_header MODIFY COLUMN status VARCHAR(20) NOT NULL DEFAULT 'Open'`)
			DB.Exec(`UPDATE purchase_header SET status='Open'   WHERE status='O'`)
			DB.Exec(`UPDATE purchase_header SET status='Closed' WHERE status='C'`)
		}
	}

	// Add status column to ap_payment if it doesn't exist yet
	var apPayStatusExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'ap_payment' AND column_name = 'status'").Scan(&apPayStatusExists)
	if apPayStatusExists == 0 {
		if err := DB.Exec("ALTER TABLE ap_payment ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'Posted'").Error; err != nil {
			return fmt.Errorf("add ap_payment.status: %w", err)
		}
	}

	// Initialize open_qty for confirmed DR lines where it's still 0 (one-time data fix)
	DB.Exec(`UPDATE dr_line dl JOIN delivery_receipt dr ON dr.id = dl.delivery_receipt_id SET dl.open_qty = dl.quantity_received WHERE dr.status = 'Received' AND dl.open_qty = 0 AND dl.quantity_received > 0`)

	// sales_order.doc_status: backfill existing rows after AutoMigrate adds the column
	// Existing active SOs become 'Open'; old Void SOs become 'Cancelled'
	DB.Exec(`UPDATE sales_order SET doc_status = 'Open' WHERE doc_status = 'Draft' AND payment_status != 'Void'`)
	DB.Exec(`UPDATE sales_order SET doc_status = 'Cancelled' WHERE payment_status = 'Void'`)

	// so_line.open_qty: initialize from quantity for any rows not yet set
	DB.Exec(`UPDATE so_line SET open_qty = quantity WHERE open_qty = 0 AND quantity > 0`)

	// Add uom_entry to document line tables if not present (7 tables)
	for _, tbl := range []string{"ign1", "ige1", "dr_line", "ap_invoice_line", "so_line", "do_line", "ar_invoice_line"} {
		var uomEntryExists int64
		DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = 'uom_entry'", tbl).Scan(&uomEntryExists)
		if uomEntryExists == 0 {
			if err := DB.Exec("ALTER TABLE `" + tbl + "` ADD COLUMN uom_entry INT UNSIGNED NOT NULL DEFAULT 0").Error; err != nil {
				return fmt.Errorf("add %s.uom_entry: %w", tbl, err)
			}
		}
	}

	// Deduplicate customer rows — keep lowest ID per name, reroute FKs, add unique index.
	// Runs only if the unique index doesn't exist yet (i.e. first time after dedup is needed).
	var custUniqueExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'customer' AND index_name = 'idx_customer_name_unique'").Scan(&custUniqueExists)
	if custUniqueExists == 0 {
		// Reroute FK references from duplicate customer IDs to the canonical (lowest-ID) row
		for _, tbl := range []string{"sales_order", "ar_invoice", "collection"} {
			DB.Exec(`UPDATE ` + "`" + tbl + "`" + ` t
				JOIN customer dup ON dup.id = t.customer_id
				JOIN (SELECT name, MIN(id) AS keep_id FROM customer GROUP BY name) kept ON kept.name = dup.name
				SET t.customer_id = kept.keep_id
				WHERE t.customer_id != kept.keep_id`)
		}
		// Delete duplicate customer rows (keep lowest ID per name)
		DB.Exec(`DELETE c FROM customer c
			INNER JOIN (SELECT name, MIN(id) AS keep_id FROM customer GROUP BY name) kept
				ON c.name = kept.name AND c.id != kept.keep_id`)
		// Add unique index to prevent future duplicates
		if err := DB.Exec("ALTER TABLE customer ADD UNIQUE INDEX idx_customer_name_unique (name(100))").Error; err != nil {
			return fmt.Errorf("add customer unique name index: %w", err)
		}
	}

	// Deduplicate supplier rows — keep lowest ID per name, reroute FKs, add unique index.
	// Runs only if the unique index doesn't exist yet.
	var supUniqueExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'supplier' AND index_name = 'idx_supplier_name_unique'").Scan(&supUniqueExists)
	if supUniqueExists == 0 {
		// Reroute supplier_id FK references from duplicate rows to the canonical (lowest-ID) row
		for _, tbl := range []string{"purchase", "delivery_receipt", "ap_invoice", "ap_payment", "purchase_header"} {
			DB.Exec(`UPDATE ` + "`" + tbl + "`" + ` t
				JOIN supplier dup ON dup.id = t.supplier_id
				JOIN (SELECT name, MIN(id) AS keep_id FROM supplier GROUP BY name) kept ON kept.name = dup.name
				SET t.supplier_id = kept.keep_id
				WHERE t.supplier_id != kept.keep_id`)
		}
		// Delete duplicate supplier rows (keep lowest ID per name)
		DB.Exec(`DELETE s FROM supplier s
			INNER JOIN (SELECT name, MIN(id) AS keep_id FROM supplier GROUP BY name) kept
				ON s.name = kept.name AND s.id != kept.keep_id`)
		// Add unique index to prevent future duplicates
		if err := DB.Exec("ALTER TABLE supplier ADD UNIQUE INDEX idx_supplier_name_unique (name(100))").Error; err != nil {
			return fmt.Errorf("add supplier unique name index: %w", err)
		}
	}

	// Backfill supplier.status for rows created before the approval workflow was added.
	// Rows with is_active=1 and no status → "active"; is_active=0 and no status → "rejected".
	DB.Exec(`UPDATE supplier SET status = 'active'   WHERE is_active = 1 AND (status = '' OR status IS NULL)`)
	DB.Exec(`UPDATE supplier SET status = 'rejected' WHERE is_active = 0 AND (status = '' OR status IS NULL)`)

	// ── BIR WHT columns (added with WHT automation feature) ──────────────────
	// supplier: wht_category, is_vat_registered
	type whtColSpec struct {
		Table  string
		Column string
		DDL    string
	}
	for _, s := range []whtColSpec{
		{"supplier",    "wht_category",       "ALTER TABLE supplier    ADD COLUMN wht_category    VARCHAR(30)    NOT NULL DEFAULT 'NONE'"},
		{"supplier",    "is_vat_registered",   "ALTER TABLE supplier    ADD COLUMN is_vat_registered TINYINT(1) NOT NULL DEFAULT 0"},
		{"ap_invoice",  "vat_exclusive_amount","ALTER TABLE ap_invoice  ADD COLUMN vat_exclusive_amount DECIMAL(15,4) NOT NULL DEFAULT 0"},
		{"ap_invoice",  "wht_rate",            "ALTER TABLE ap_invoice  ADD COLUMN wht_rate        DECIMAL(7,4)   NOT NULL DEFAULT 0"},
		{"ap_invoice",  "wht_amount",          "ALTER TABLE ap_invoice  ADD COLUMN wht_amount      DECIMAL(15,4)  NOT NULL DEFAULT 0"},
		{"ap_invoice",  "wht_atc_code",        "ALTER TABLE ap_invoice  ADD COLUMN wht_atc_code    VARCHAR(10)    NOT NULL DEFAULT ''"},
		{"ap_invoice",  "net_payable",         "ALTER TABLE ap_invoice  ADD COLUMN net_payable     DECIMAL(15,4)  NOT NULL DEFAULT 0"},
	} {
		var cnt int64
		DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?",
			s.Table, s.Column).Scan(&cnt)
		if cnt == 0 {
			if err := DB.Exec(s.DDL).Error; err != nil {
				return fmt.Errorf("add %s.%s: %w", s.Table, s.Column, err)
			}
		}
	}
	// Backfill net_payable for existing AP invoices that had no WHT (net_payable = doc_total)
	DB.Exec(`UPDATE ap_invoice SET net_payable = total_amount WHERE net_payable = 0 AND total_amount > 0 AND wht_amount = 0`)

	// Table renames for line tables
	type tblState struct {
		Old string
		New string
	}
	for _, pair := range []tblState{
		{"sales_order_item", "so_line"},
		{"journal_entry_line", "je_line"},
		{"delivery_order_item", "do_line"},
		{"delivery_receipt_item", "dr_line"},
		{"price_group_item", "pg_line"},
		{"ar_invoice_item", "ar_invoice_line"},
		{"ap_invoice_item", "ap_invoice_line"},
	} {
		var oldCnt, newCnt int64
		DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", pair.Old).Scan(&oldCnt)
		DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", pair.New).Scan(&newCnt)
		if oldCnt > 0 && newCnt == 0 {
			sql := fmt.Sprintf("RENAME TABLE %s TO %s", pair.Old, pair.New)
			if err := DB.Exec(sql).Error; err != nil {
				return fmt.Errorf("rename table %s → %s: %w", pair.Old, pair.New, err)
			}
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// RunOITMSchemaMigrations performs idempotent table/column renames and data
// migrations to align the Item Master module with the SAP B1 OITM schema.
//
// MUST be called BEFORE AutoMigrateAll — so that AutoMigrate targets the
// already-renamed tables (oitm / oitb) instead of creating new empty ones.
// ─────────────────────────────────────────────────────────────────────────────
func RunOITMSchemaMigrations() error {

	// ── STEP A: Table renames ──────────────────────────────────────────────
	// A1. item_categories → oitb
	// A2. item_master     → oitm
	for _, pair := range []struct{ Old, New string }{
		{"item_categories", "oitb"},
		{"item_master", "oitm"},
	} {
		var oldCnt, newCnt int64
		DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", pair.Old).Scan(&oldCnt)
		DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", pair.New).Scan(&newCnt)
		if oldCnt > 0 && newCnt == 0 {
			sql := fmt.Sprintf("RENAME TABLE `%s` TO `%s`", pair.Old, pair.New)
			if err := DB.Exec(sql).Error; err != nil {
				return fmt.Errorf("rename table %s → %s: %w", pair.Old, pair.New, err)
			}
		}
	}

	// ── STEP B: oitb column renames ────────────────────────────────────────
	// B1. id → itms_grp_cod
	var oitbIDExists, oitbCodExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='oitb' AND column_name='id'").Scan(&oitbIDExists)
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='oitb' AND column_name='itms_grp_cod'").Scan(&oitbCodExists)
	if oitbIDExists > 0 && oitbCodExists == 0 {
		if err := DB.Exec("ALTER TABLE oitb CHANGE COLUMN id itms_grp_cod INT NOT NULL AUTO_INCREMENT").Error; err != nil {
			return fmt.Errorf("oitb: rename id → itms_grp_cod: %w", err)
		}
	}
	// B2. name → itms_grp_nam
	oitmColRenameIfExists("oitb", "name", "itms_grp_nam", "VARCHAR(100) NOT NULL DEFAULT ''")

	// ── STEP C: oitm column renames ────────────────────────────────────────
	for _, r := range []struct{ Old, New, Def string }{
		{"name", "item_name", "VARCHAR(150) NOT NULL DEFAULT ''"},
		{"unit", "invntry_uom", "VARCHAR(20) NOT NULL DEFAULT ''"},
		{"unit_price", "avg_price", "DECIMAL(15,4) NOT NULL DEFAULT 0"},
		{"reorder_level", "min_level", "DECIMAL(12,3) NOT NULL DEFAULT 0"},
		{"on_hand_qty", "on_hand", "DECIMAL(12,3) NOT NULL DEFAULT 0"},
	} {
		oitmColRenameIfExists("oitm", r.Old, r.New, r.Def)
	}

	// ── STEP D: Bool → CHAR(1) Y/N conversions ────────────────────────────
	// D1. is_active (TINYINT) → valid_for (CHAR)
	oitmAddCharFromBool("oitm", "is_active", "valid_for")
	// D2. is_production_item (TINYINT) → mak_item (CHAR)
	oitmAddCharFromBool("oitm", "is_production_item", "mak_item")

	// ── STEP E: Category FK migration (category VARCHAR → itms_grp_cod INT) ─
	var catColExists, grpCodInOitm int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='oitm' AND column_name='category'").Scan(&catColExists)
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='oitm' AND column_name='itms_grp_cod'").Scan(&grpCodInOitm)
	if catColExists > 0 && grpCodInOitm == 0 {
		if err := DB.Exec("ALTER TABLE oitm ADD COLUMN itms_grp_cod INT NOT NULL DEFAULT 0").Error; err != nil {
			return fmt.Errorf("oitm: add itms_grp_cod: %w", err)
		}
		// Match existing string category names to oitb group codes
		DB.Exec("UPDATE oitm o JOIN oitb b ON b.itms_grp_nam = o.category SET o.itms_grp_cod = b.itms_grp_cod")
	}

	// ── STEP F: itm9 FK migration (item_id uint → item_code string) ────────
	var itm9ItemIDExists, itm9ItemCodeExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='itm9' AND column_name='item_id'").Scan(&itm9ItemIDExists)
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='itm9' AND column_name='item_code'").Scan(&itm9ItemCodeExists)
	if itm9ItemIDExists > 0 && itm9ItemCodeExists == 0 {
		if err := DB.Exec("ALTER TABLE itm9 ADD COLUMN item_code VARCHAR(50) NOT NULL DEFAULT ''").Error; err != nil {
			return fmt.Errorf("itm9: add item_code column: %w", err)
		}
		// Populate item_code from oitm using old item_id FK
		DB.Exec("UPDATE itm9 t JOIN oitm o ON o.id = t.item_id SET t.item_code = o.item_code")
		// Rebuild primary key: drop old (item_id, uom_entry), add new (item_code, uom_entry)
		if err := DB.Exec("ALTER TABLE itm9 DROP PRIMARY KEY").Error; err != nil {
			return fmt.Errorf("itm9: drop old PK: %w", err)
		}
		if err := DB.Exec("ALTER TABLE itm9 ADD PRIMARY KEY (item_code, uom_entry)").Error; err != nil {
			return fmt.Errorf("itm9: add new PK: %w", err)
		}
	}

	// ── STEP G: Seed default warehouse if owhs table is empty ──────────────
	var whsTblExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='owhs'").Scan(&whsTblExists)
	if whsTblExists > 0 {
		var whsRowCnt int64
		DB.Raw("SELECT COUNT(*) FROM owhs").Scan(&whsRowCnt)
		if whsRowCnt == 0 {
			DB.Exec("INSERT INTO owhs (whs_code, whs_name, inactive) VALUES ('WH01','Main Warehouse','N')")
		}
	}

	// ── STEP H: Add cash_flow_class to gl_account (PAS 7) ─────────────────
	var cfClassExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='gl_account' AND column_name='cash_flow_class'").Scan(&cfClassExists)
	if cfClassExists == 0 {
		if err := DB.Exec("ALTER TABLE gl_account ADD COLUMN cash_flow_class VARCHAR(30) NOT NULL DEFAULT ''").Error; err != nil {
			return fmt.Errorf("add gl_account.cash_flow_class: %w", err)
		}
		// Back-fill classifications for existing installations based on known account codes.
		cfBackfill := []struct {
			Class string
			Codes []string
		}{
			{"CASH",                   []string{"1-1100", "1-1110", "1-1120"}},
			{"OPERATING_WC_ASSET",     []string{"1-1200", "1-1310", "1-1320", "1-1330", "1-1340", "1-1350", "1-1360", "1-1400", "1-1500"}},
			{"OPERATING_WC_LIABILITY", []string{"2-1100", "2-1200", "2-1300"}},
			{"OPERATING_NON_CASH",     []string{"5-7000", "5-8000"}},
			{"INVESTING_PPE",          []string{"1-2110", "1-2120", "1-2130", "1-2140"}},
			{"FINANCING_DEBT",         []string{"2-2100"}},
			{"FINANCING_EQUITY",       []string{"3-1000"}},
		}
		for _, b := range cfBackfill {
			for _, code := range b.Codes {
				DB.Exec("UPDATE gl_account SET cash_flow_class = ? WHERE code = ?", b.Class, code)
			}
		}
	}

	// ── STEP I: Add purchase_header_id to delivery_receipt ────────────────
	var drPhIdExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='delivery_receipt' AND column_name='purchase_header_id'").Scan(&drPhIdExists)
	if drPhIdExists == 0 {
		if err := DB.Exec("ALTER TABLE delivery_receipt ADD COLUMN purchase_header_id BIGINT UNSIGNED NULL, ADD INDEX idx_delivery_receipt_purchase_header_id (purchase_header_id)").Error; err != nil {
			return fmt.Errorf("add delivery_receipt.purchase_header_id: %w", err)
		}
		// Back-fill: join delivery_receipt → purchase (by purchase_id) → purchase_header (by po_number)
		DB.Exec(`
			UPDATE delivery_receipt dr
			JOIN purchase p ON p.id = dr.purchase_id
			JOIN purchase_header ph ON ph.po_number = p.po_number
			SET dr.purchase_header_id = ph.id
			WHERE dr.purchase_header_id IS NULL AND dr.purchase_id IS NOT NULL
		`)
	}

	// ── STEP J: Add purchase_header_id to ap_invoice ───────────────────────
	var apPhIdExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='ap_invoice' AND column_name='purchase_header_id'").Scan(&apPhIdExists)
	if apPhIdExists == 0 {
		if err := DB.Exec("ALTER TABLE ap_invoice ADD COLUMN purchase_header_id BIGINT UNSIGNED NULL, ADD INDEX idx_ap_invoice_purchase_header_id (purchase_header_id)").Error; err != nil {
			return fmt.Errorf("add ap_invoice.purchase_header_id: %w", err)
		}
		// Back-fill: join ap_invoice → purchase → purchase_header
		DB.Exec(`
			UPDATE ap_invoice ai
			JOIN purchase p ON p.id = ai.purchase_id
			JOIN purchase_header ph ON ph.po_number = p.po_number
			SET ai.purchase_header_id = ph.id
			WHERE ai.purchase_header_id IS NULL AND ai.purchase_id IS NOT NULL
		`)
	}

	// ── STEP K: Back-fill dr_line.base_doc_entry (purchase.id → purchase_header.id) ──
	// BaseDocEntry on dr_line used to store legacy Purchase.ID; now it must store
	// PurchaseHeader.ID so that the SAP B1-aligned BaseEntry+BaseLine chain works.
	DB.Exec(`
		UPDATE dr_line dl
		JOIN purchase p ON p.id = dl.base_doc_entry
		JOIN purchase_header ph ON ph.po_number = p.po_number
		SET dl.base_doc_entry = ph.id
		WHERE dl.base_doc_entry IS NOT NULL
		  AND dl.base_doc_entry NOT IN (SELECT id FROM purchase_header)
	`)

	// ── STEP L: Add payment_method to purchase_header if missing ───────────
	var phPayMethodExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='purchase_header' AND column_name='payment_method'").Scan(&phPayMethodExists)
	if phPayMethodExists == 0 {
		DB.Exec("ALTER TABLE purchase_header ADD COLUMN payment_method VARCHAR(30) NOT NULL DEFAULT ''")
	}

	// ── STEP M: Relax delivery_receipt.purchase_id NOT NULL → NULL ──────────
	// The legacy purchase_id FK is no longer populated on new inserts.
	// MySQL requires a default value or NULL-ability; we drop the NOT NULL constraint.
	var drPurchaseIdNullable int64
	DB.Raw(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name   = 'delivery_receipt'
		  AND column_name  = 'purchase_id'
		  AND is_nullable  = 'NO'
	`).Scan(&drPurchaseIdNullable)
	if drPurchaseIdNullable > 0 {
		if err := DB.Exec("ALTER TABLE delivery_receipt MODIFY COLUMN purchase_id BIGINT UNSIGNED NULL DEFAULT NULL").Error; err != nil {
			return fmt.Errorf("relax delivery_receipt.purchase_id NOT NULL: %w", err)
		}
	}

	// ── STEP N: Relax ap_invoice.purchase_id NOT NULL → NULL ─────────────────
	var apPurchaseIdNullable int64
	DB.Raw(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name   = 'ap_invoice'
		  AND column_name  = 'purchase_id'
		  AND is_nullable  = 'NO'
	`).Scan(&apPurchaseIdNullable)
	if apPurchaseIdNullable > 0 {
		if err := DB.Exec("ALTER TABLE ap_invoice MODIFY COLUMN purchase_id BIGINT UNSIGNED NULL DEFAULT NULL").Error; err != nil {
			return fmt.Errorf("relax ap_invoice.purchase_id NOT NULL: %w", err)
		}
	}

	// ── STEP O: OIVL performance indexes for inventory valuation reports ──────
	// These composite indexes dramatically speed up:
	//   • Historical inventory valuation (aggregate by item + date)
	//   • Stock card (per-item running balance)
	//   • Movement report (filtered by date / warehouse / trans_type)
	for _, idxDDL := range []string{
		`CREATE INDEX IF NOT EXISTS idx_oivl_item_date      ON oivl (item_code, doc_date)`,
		`CREATE INDEX IF NOT EXISTS idx_oivl_whs_date       ON oivl (warehouse, doc_date)`,
		`CREATE INDEX IF NOT EXISTS idx_oivl_trans_date     ON oivl (trans_type, doc_date)`,
	} {
		if err := DB.Exec(idxDDL).Error; err != nil {
			// Non-fatal: index creation failure should not block startup
			fmt.Printf("[DB] WARN: could not create OIVL index: %v\n", err)
		}
	}

	// ── STEP P: Normalise charset / collation to utf8mb4_unicode_ci ────────
	// MySQL 8.0 defaults to utf8mb4_0900_ai_ci. Tables created at different
	// versions or via GORM AutoMigrate may carry mismatched column collations,
	// causing Error 1267 on any cross-table JOIN (e.g. oitm JOIN oitw on
	// item_code). Two sub-steps:
	//   1. Set the DATABASE default so every future AutoMigrate CREATE TABLE
	//      inherits utf8mb4_unicode_ci without needing DSN changes.
	//   2. CONVERT each existing table that still has a mismatched column.
	// This block is idempotent: on subsequent runs the information_schema
	// query returns no rows and the loop body never executes.
	{
		var dbName string
		if scanErr := DB.Raw("SELECT DATABASE()").Scan(&dbName).Error; scanErr == nil && dbName != "" {
			if alterErr := DB.Exec(fmt.Sprintf(
				"ALTER DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName,
			)).Error; alterErr != nil {
				fmt.Printf("[DB] WARN: could not set database collation: %v\n", alterErr)
			}
		}

		type mismatchRow struct{ TableName string }
		var mismatches []mismatchRow
		DB.Raw(`
			SELECT DISTINCT TABLE_NAME AS table_name
			FROM   information_schema.COLUMNS
			WHERE  TABLE_SCHEMA  = DATABASE()
			  AND  COLLATION_NAME IS NOT NULL
			  AND  COLLATION_NAME != 'utf8mb4_unicode_ci'
			ORDER  BY TABLE_NAME
		`).Scan(&mismatches)

		for _, row := range mismatches {
			if convertErr := DB.Exec(fmt.Sprintf(
				"ALTER TABLE `%s` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
				row.TableName,
			)).Error; convertErr != nil {
				fmt.Printf("[DB] WARN: collation conversion failed for %s: %v\n", row.TableName, convertErr)
			}
		}
	}

	return nil
}

// oitmColRenameIfExists renames a column in the given table only if the old
// column exists and the new column does not. Safe to call repeatedly.
func oitmColRenameIfExists(table, oldCol, newCol, _ string) {
	var oldExists, newExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", table, oldCol).Scan(&oldExists)
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", table, newCol).Scan(&newExists)
	if oldExists > 0 && newExists == 0 {
		DB.Exec(fmt.Sprintf("ALTER TABLE `%s` RENAME COLUMN `%s` TO `%s`", table, oldCol, newCol))
	}
}

// oitmAddCharFromBool adds a CHAR(1) 'Y'/'N' column derived from a TINYINT bool column.
// Sets 'Y' where the bool is 1, 'N' otherwise. Safe to call repeatedly.
func oitmAddCharFromBool(table, boolCol, charCol string) {
	var boolExists, charExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", table, boolCol).Scan(&boolExists)
	DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", table, charCol).Scan(&charExists)
	if boolExists > 0 && charExists == 0 {
		DB.Exec(fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` CHAR(1) NOT NULL DEFAULT 'N'", table, charCol))
		DB.Exec(fmt.Sprintf("UPDATE `%s` SET `%s` = CASE WHEN `%s` = 1 THEN 'Y' ELSE 'N' END", table, charCol, boolCol))
	}
}

// SeedDefaults seeds default FarmSettings and the first Admin user if none exist.
func SeedDefaults() error {
	// Seed default farm settings
	defaults := map[string]string{
		"farm_name":    "EggLayer Farm",
		"farm_address": "",
		"farm_contact": "",
		"tray_size":    "30",
		"currency":     "PHP",
		"currency_sym": "₱",
	}
	for k, v := range defaults {
		var s models.FarmSettings
		result := DB.Where("`key` = ?", k).First(&s)
		if result.Error == gorm.ErrRecordNotFound {
			DB.Create(&models.FarmSettings{Key: k, Value: v})
		}
	}

	// Seed default admin user if no users exist
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash default admin password: %w", err)
		}
		now := time.Now()
		admin := models.User{
			Username:     "admin",
			FullName:     "System Administrator",
			Role:         "Admin",
			PasswordHash: string(hash),
			IsActive:     true,
			CreatedAt:    now,
		}
		if err := DB.Create(&admin).Error; err != nil {
			return fmt.Errorf("failed to create default admin: %w", err)
		}
	}

	// Seed default Chart of Accounts if none exist
	var acctCount int64
	DB.Model(&models.GLAccount{}).Count(&acctCount)
	if acctCount == 0 {
		if err := seedDefaultAccounts(); err != nil {
			return fmt.Errorf("failed to seed chart of accounts: %w", err)
		}
	}

	// Seed default purchasing lookup values (idempotent — skips existing entries)
	seedPurchasingLookups()

	// Migrate GL accounts to rice mill defaults (handles existing installs)
	if err := migrateRiceMillAccounts(); err != nil {
		return fmt.Errorf("migrateRiceMillAccounts: %w", err)
	}

	// Add Other Income / Other Expenses accounts (required for asset disposal JEs)
	if err := migrateAssetsAccounts(); err != nil {
		return fmt.Errorf("migrateAssetsAccounts: %w", err)
	}

	// Seed account determination rules for all modules
	if err := seedRiceMillAccountDetermination(); err != nil {
		return fmt.Errorf("seedRiceMillAccountDetermination: %w", err)
	}

	// Seed payment method → GL account mappings
	if err := seedRiceMillPaymentMethodAccounts(); err != nil {
		return fmt.Errorf("seedRiceMillPaymentMethodAccounts: %w", err)
	}

	// Seed BANKING module posting events (CASH_INFLOW / CASH_OUTFLOW)
	if err := seedBankingAccountDetermination(); err != nil {
		return fmt.Errorf("seedBankingAccountDetermination: %w", err)
	}

	// Seed WHT posting event (AP_WHT_PAYABLE → 2-1300)
	if err := seedWHTAccountDetermination(); err != nil {
		return fmt.Errorf("seedWHTAccountDetermination: %w", err)
	}

	return nil
}

// seedPurchasingLookups inserts default supplier_type, payment_terms, and payment_method
// values into purchasing_lookup. Existing rows are skipped (FirstOrCreate).
func seedPurchasingLookups() {
	type entry struct {
		category  string
		value     string
		sortOrder int
	}
	defaults := []entry{
		// Supplier types
		{"supplier_type", "Regular", 0},
		{"supplier_type", "Seasonal", 1},
		{"supplier_type", "One-time", 2},
		// Payment terms
		{"payment_terms", "COD", 0},
		{"payment_terms", "Net 7", 1},
		{"payment_terms", "Net 15", 2},
		{"payment_terms", "Net 30", 3},
		{"payment_terms", "Net 60", 4},
		// Payment methods
		{"payment_method", "Cash", 0},
		{"payment_method", "Bank Transfer", 1},
		{"payment_method", "GCash", 2},
		{"payment_method", "Check", 3},
		{"payment_method", "Credit", 4},
	}
	for _, d := range defaults {
		var row models.PurchasingLookup
		DB.Where("category = ? AND value = ?", d.category, d.value).FirstOrCreate(&row, models.PurchasingLookup{
			Category:  d.category,
			Value:     d.value,
			SortOrder: d.sortOrder,
			IsActive:  true,
		})
	}
}

// riceMillAccountEntries returns the default GL accounts for a rice mill.
// CashFlowClass follows PAS 7 (Statement of Cash Flows) classification.
// Used by both seedDefaultAccounts (fresh install) and migrateRiceMillAccounts (existing install).
func riceMillAccountEntries() []models.GLAccount {
	return []models.GLAccount{
		// ── ASSETS ────────────────────────────────────────────────────────────
		{Code: "1-0000", Name: "ASSETS", Section: "ASSET", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1000", Name: "Current Assets", Section: "ASSET", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		// Cash & Cash Equivalents — PAS 7 par. 7
		{Code: "1-1100", Name: "Cash on Hand", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "CASH"},
		{Code: "1-1110", Name: "GCash", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "CASH"},
		{Code: "1-1120", Name: "Bank Account", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "CASH"},
		// Receivables — working capital asset
		{Code: "1-1200", Name: "Accounts Receivable", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_WC_ASSET"},
		// Inventories — PAS 2 / working capital asset
		{Code: "1-1300", Name: "Inventory", Section: "ASSET", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1310", Name: "Inventory — Palay (Raw Paddy)", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_WC_ASSET"},
		{Code: "1-1320", Name: "Inventory — Work In Progress", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_WC_ASSET"},
		{Code: "1-1330", Name: "Inventory — Milled Rice", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_WC_ASSET"},
		{Code: "1-1340", Name: "Inventory — By-Products", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_WC_ASSET"},
		{Code: "1-1350", Name: "Inventory — Packaging Materials", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_WC_ASSET"},
		{Code: "1-1360", Name: "Inventory — Spare Parts", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_WC_ASSET"},
		// Accruals & Tax receivables — working capital asset
		{Code: "1-1400", Name: "Goods Received Not Invoiced (GRNI)", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_WC_ASSET"},
		{Code: "1-1500", Name: "Input VAT Creditable", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_WC_ASSET"},
		// Non-Current Assets — PAS 16 PPE — investing activities
		{Code: "1-2000", Name: "Non-Current Assets", Section: "ASSET", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-2100", Name: "Property, Plant & Equipment", Section: "ASSET", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-2110", Name: "Land", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "INVESTING_PPE"},
		{Code: "1-2120", Name: "Buildings & Structures", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "INVESTING_PPE"},
		{Code: "1-2130", Name: "Milling Equipment", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "INVESTING_PPE"},
		{Code: "1-2140", Name: "Vehicles & Transport", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "INVESTING_PPE"},
		// Accumulated Depreciation — contra-asset, not a cash flow line itself
		{Code: "1-2190", Name: "Accumulated Depreciation", Section: "ASSET", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},

		// ── LIABILITIES ───────────────────────────────────────────────────────
		{Code: "2-0000", Name: "LIABILITIES", Section: "LIABILITY", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "2-1000", Name: "Current Liabilities", Section: "LIABILITY", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "2-1100", Name: "Accounts Payable", Section: "LIABILITY", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true, CashFlowClass: "OPERATING_WC_LIABILITY"},
		{Code: "2-1200", Name: "Output VAT Payable", Section: "LIABILITY", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true, CashFlowClass: "OPERATING_WC_LIABILITY"},
		{Code: "2-1300", Name: "Withholding Tax Payable", Section: "LIABILITY", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true, CashFlowClass: "OPERATING_WC_LIABILITY"},
		{Code: "2-2000", Name: "Non-Current Liabilities", Section: "LIABILITY", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		// Loans Payable — financing activities
		{Code: "2-2100", Name: "Loans Payable — Long Term", Section: "LIABILITY", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true, CashFlowClass: "FINANCING_DEBT"},

		// ── EQUITY ────────────────────────────────────────────────────────────
		{Code: "3-0000", Name: "EQUITY", Section: "EQUITY", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "3-1000", Name: "Owner's Capital", Section: "EQUITY", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true, CashFlowClass: "FINANCING_EQUITY"},
		// Retained Earnings — equity but not a direct cash flow; excluded
		{Code: "3-2000", Name: "Retained Earnings", Section: "EQUITY", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},

		// ── REVENUE — PFRS 15 revenue recognition ─────────────────────────────
		// Revenue accounts are captured via P&L (profit before tax) — no direct CF tag needed
		{Code: "4-0000", Name: "REVENUE", Section: "REVENUE", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "4-1000", Name: "Rice Sales Revenue", Section: "REVENUE", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "4-2000", Name: "By-Product Sales Revenue", Section: "REVENUE", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "4-3000", Name: "Milling Service Fee Income", Section: "REVENUE", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "4-9000", Name: "Sales Discount", Section: "REVENUE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},

		// ── EXPENSES ──────────────────────────────────────────────────────────
		// Cash operating expenses — included in profit; no separate CF tag (indirect method)
		{Code: "5-0000", Name: "EXPENSES", Section: "EXPENSE", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-1000", Name: "Cost of Goods Sold", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-2000", Name: "Milling Labour Expense", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-3000", Name: "Power & Utilities Expense", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-4000", Name: "Repairs & Maintenance Expense", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-5000", Name: "Supplies Expense", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-6000", Name: "Purchase Price Variance", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		// Non-cash expenses — PAS 7 par. 28: add back in indirect method
		{Code: "5-7000", Name: "Depreciation Expense", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_NON_CASH"},
		{Code: "5-8000", Name: "Stock Adjustment Loss", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true, CashFlowClass: "OPERATING_NON_CASH"},

		// ── OTHER INCOME / OTHER EXPENSES (asset disposal, misc) ─────────────
		{Code: "7-0000", Name: "OTHER INCOME", Section: "REVENUE", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "7-1000", Name: "Other Income", Section: "REVENUE", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "8-0000", Name: "OTHER EXPENSES", Section: "EXPENSE", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "8-1000", Name: "Other Expenses", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
	}
}

func seedDefaultAccounts() error {
	accounts := riceMillAccountEntries()
	now := time.Now()
	for i := range accounts {
		accounts[i].CreatedAt = now
	}
	// Insert in order (headers first so foreign keys work if added later)
	return DB.CreateInBatches(accounts, 10).Error
}

// migrateRiceMillAccounts upserts the rice mill default GL accounts for existing
// installations that were previously seeded with other accounts.
// Existing accounts are renamed in-place (preserving their IDs and any journal
// entry FK references); new accounts are inserted.
// Guarded by sentinel key "coa_ricemill_v2_done".
func migrateRiceMillAccounts() error {
	var sentinel models.FarmSettings
	if DB.Where("`key` = ?", "coa_ricemill_v2_done").First(&sentinel).Error == nil {
		return nil // already migrated
	}

	now := time.Now()
	for _, a := range riceMillAccountEntries() {
		var existing models.GLAccount
		err := DB.Where("code = ?", a.Code).First(&existing).Error
		if err != nil {
			// Not found — insert fresh
			a.CreatedAt = now
			if createErr := DB.Create(&a).Error; createErr != nil {
				return fmt.Errorf("migrateRiceMillAccounts create %s: %w", a.Code, createErr)
			}
		} else {
			// Found — update name and metadata (preserves ID and JE FK references)
			if updateErr := DB.Model(&existing).Updates(map[string]interface{}{
				"name":           a.Name,
				"section":        a.Section,
				"account_type":   a.AccountType,
				"normal_balance": a.NormalBalance,
				"is_system":      true,
				"is_active":      true,
			}).Error; updateErr != nil {
				return fmt.Errorf("migrateRiceMillAccounts update %s: %w", a.Code, updateErr)
			}
		}
	}

	DB.Save(&models.FarmSettings{Key: "coa_ricemill_v2_done", Value: now.Format(time.RFC3339)})
	return nil
}

// migrateAssetsAccounts adds the Other Income / Other Expenses headers and posting
// accounts needed for fixed-asset disposal journal entries.
// Guarded by sentinel "coa_assets_v1_done".
func migrateAssetsAccounts() error {
	var sentinel models.FarmSettings
	if DB.Where("`key` = ?", "coa_assets_v1_done").First(&sentinel).Error == nil {
		return nil
	}
	now := time.Now()
	extras := []models.GLAccount{
		{Code: "7-0000", Name: "OTHER INCOME", Section: "REVENUE", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "7-1000", Name: "Other Income", Section: "REVENUE", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "8-0000", Name: "OTHER EXPENSES", Section: "EXPENSE", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "8-1000", Name: "Other Expenses", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
	}
	for _, a := range extras {
		var existing models.GLAccount
		if DB.Where("code = ?", a.Code).First(&existing).Error != nil {
			a.CreatedAt = now
			if err := DB.Create(&a).Error; err != nil {
				return fmt.Errorf("migrateAssetsAccounts create %s: %w", a.Code, err)
			}
		}
	}
	DB.Save(&models.FarmSettings{Key: "coa_assets_v1_done", Value: now.Format(time.RFC3339)})
	return nil
}

// seedRiceMillAccountDetermination seeds account determination rules mapping
// each posting event to the correct GL account per PFRS for a rice mill.
// Guarded by sentinel key "acct_det_ricemill_v1_done".
func seedRiceMillAccountDetermination() error {
	var sentinel models.FarmSettings
	if DB.Where("`key` = ?", "acct_det_ricemill_v1_done").First(&sentinel).Error == nil {
		return nil // already seeded
	}

	// accountIDByCode looks up a GL account ID by its code.
	// Returns 0 (and logs a warning) if not found so remaining rules still seed.
	accountIDByCode := func(code string) uint {
		var acct models.GLAccount
		if err := DB.Select("id").Where("code = ?", code).First(&acct).Error; err != nil {
			fmt.Printf("[ACCT-DET] WARN: GL account %s not found — skipping rule\n", code)
			return 0
		}
		return acct.ID
	}

	// cat returns a *string for item_category (non-nil = category-specific rule).
	cat := func(s string) *string { return &s }

	type ruleSpec struct {
		Module       string
		PostingEvent string
		ItemCategory *string // nil = catch-all rule (item_category IS NULL)
		GLCode       string
		Notes        string
	}

	rules := []ruleSpec{
		// ── SALES ────────────────────────────────────────────────────────────
		{"SALES", "AR_RECEIVABLE",    nil,              "1-1200", "AR Invoice — Accounts Receivable (Dr)"},
		{"SALES", "SALES_REVENUE",    nil,              "4-1000", "AR Invoice — Rice Sales Revenue (Cr)"},
		{"SALES", "COGS",             nil,              "5-1000", "AR Invoice — Cost of Goods Sold (Dr)"},
		{"SALES", "INVENTORY_SOLD",   cat("Milled Rice"),  "1-1330", "AR Invoice — Milled Rice inventory reduction (Cr)"},
		{"SALES", "INVENTORY_SOLD",   cat("By-Products"),  "1-1340", "AR Invoice — By-Product inventory reduction (Cr)"},
		{"SALES", "INVENTORY_SOLD",   nil,              "1-1330", "AR Invoice — Inventory sold fallback (Cr)"},
		{"SALES", "OUTPUT_VAT",       nil,              "2-1200", "AR Invoice — Output VAT 12% (Cr)"},
		{"SALES", "COLLECTION_CLEARING", nil,           "1-1200", "Collection — AR clearing (Cr)"},
		{"SALES", "SALES_DISCOUNT",   nil,              "4-9000", "Collection — Sales discount (Dr)"},

		// ── PURCHASING ───────────────────────────────────────────────────────
		{"PURCHASING", "AP_PAYABLE",              nil,                        "2-1100", "AP Invoice — Accounts Payable (Cr)"},
		{"PURCHASING", "GRNI",                    nil,                        "1-1400", "DR Confirm — Goods Received Not Invoiced (Cr)"},
		{"PURCHASING", "INVENTORY_RECEIVED",      cat("Palay"),               "1-1310", "DR Confirm — Palay received (Dr)"},
		{"PURCHASING", "INVENTORY_RECEIVED",      cat("Milled Rice"),         "1-1330", "DR Confirm — Milled Rice received (Dr)"},
		{"PURCHASING", "INVENTORY_RECEIVED",      cat("By-Products"),         "1-1340", "DR Confirm — By-Products received (Dr)"},
		{"PURCHASING", "INVENTORY_RECEIVED",      cat("Packaging Materials"), "1-1350", "DR Confirm — Packaging Materials (Dr)"},
		{"PURCHASING", "INVENTORY_RECEIVED",      cat("Spare Parts"),         "1-1360", "DR Confirm — Spare Parts (Dr)"},
		{"PURCHASING", "INVENTORY_RECEIVED",      cat("Fuel & Lubricants"),   "5-3000", "DR Confirm — Fuel expensed on receipt (Dr)"},
		{"PURCHASING", "INVENTORY_RECEIVED",      nil,                        "1-1310", "DR Confirm — Inventory received fallback (Dr)"},
		{"PURCHASING", "INPUT_VAT",               nil,                        "1-1500", "AP Invoice — Input VAT 12% (Dr)"},
		{"PURCHASING", "AP_CLEARING",             nil,                        "2-1100", "AP Payment — Payable cleared (Dr)"},
		{"PURCHASING", "PURCHASE_PRICE_VARIANCE", nil,                        "5-6000", "AP Invoice — Purchase price variance (Dr/Cr)"},

		// ── INVENTORY (legacy poultry event names — stub mappings) ────────────
		{"INVENTORY", "FEED_CONSUMED",    nil, "5-5000", "Stub — generic consumption expense"},
		{"INVENTORY", "INVENTORY_FEED",   nil, "1-1350", "Stub — generic inventory reduction"},
		{"INVENTORY", "MORTALITY_LOSS",   nil, "5-8000", "Stub — stock adjustment loss"},
		{"INVENTORY", "INVENTORY_BIRDS",  nil, "1-1310", "Stub — generic raw material"},
		{"INVENTORY", "STOCK_ADJUSTMENT", nil, "5-8000", "Stock write-down / shrinkage"},

		// ── PRODUCTION ───────────────────────────────────────────────────────
		{"PRODUCTION", "MILLING_WIP",    nil,               "1-1320", "Milling Start — WIP account (Dr)"},
		{"PRODUCTION", "MILLING_INPUT",  cat("Palay"),      "1-1310", "Milling Start — Palay (paddy) input (Cr)"},
		{"PRODUCTION", "MILLING_INPUT",  nil,               "1-1310", "Milling Start — Raw material input fallback (Cr)"},
		{"PRODUCTION", "MILLING_OUTPUT", cat("Milled Rice"),"1-1330", "Milling Complete — Milled Rice output (Dr)"},
		{"PRODUCTION", "MILLING_OUTPUT", cat("By-Products"),"1-1340", "Milling Complete — By-Product output (Dr)"},
		{"PRODUCTION", "MILLING_OUTPUT", nil,               "1-1330", "Milling Complete — Output inventory fallback (Dr)"},
	}

	for _, r := range rules {
		acctID := accountIDByCode(r.GLCode)
		if acctID == 0 {
			continue
		}

		rule := models.AccountDetermination{
			Module:       r.Module,
			PostingEvent: r.PostingEvent,
			ItemCategory: r.ItemCategory,
			GLAccountID:  acctID,
			IsActive:     true,
			Notes:        r.Notes,
		}

		// Use explicit SQL for IS NULL / = ? to avoid GORM silently dropping nil pointer fields.
		if r.ItemCategory == nil {
			DB.Where("module = ? AND posting_event = ? AND item_category IS NULL", r.Module, r.PostingEvent).
				Assign(models.AccountDetermination{GLAccountID: acctID, Notes: r.Notes, IsActive: true}).
				FirstOrCreate(&rule)
		} else {
			DB.Where("module = ? AND posting_event = ? AND item_category = ?", r.Module, r.PostingEvent, *r.ItemCategory).
				Assign(models.AccountDetermination{GLAccountID: acctID, Notes: r.Notes, IsActive: true}).
				FirstOrCreate(&rule)
		}
	}

	DB.Save(&models.FarmSettings{Key: "acct_det_ricemill_v1_done", Value: time.Now().Format(time.RFC3339)})
	return nil
}

// seedRiceMillPaymentMethodAccounts seeds payment method → GL account mappings
// so that cash/bank/GCash collections and payments post to the correct accounts.
// Guarded by sentinel key "payment_method_accts_v1_done".
func seedRiceMillPaymentMethodAccounts() error {
	var sentinel models.FarmSettings
	if DB.Where("`key` = ?", "payment_method_accts_v1_done").First(&sentinel).Error == nil {
		return nil // already seeded
	}

	accountIDByCode := func(code string) uint {
		var acct models.GLAccount
		if err := DB.Select("id").Where("code = ?", code).First(&acct).Error; err != nil {
			fmt.Printf("[PAY-ACCT] WARN: GL account %s not found — skipping\n", code)
			return 0
		}
		return acct.ID
	}

	type pmaSpec struct {
		PaymentMethod string
		BankName      string
		GLCode        string
		Direction     string
	}

	specs := []pmaSpec{
		{"Cash",          "", "1-1100", "BOTH"},
		{"GCash",         "", "1-1110", "BOTH"},
		{"Bank Transfer", "", "1-1120", "BOTH"},
		{"Check",         "", "1-1120", "BOTH"},
		{"Credit",        "", "1-1200", "INFLOW"},
	}

	for _, s := range specs {
		acctID := accountIDByCode(s.GLCode)
		if acctID == 0 {
			continue
		}
		row := models.PaymentMethodAccount{
			PaymentMethod: s.PaymentMethod,
			BankName:      s.BankName,
			Direction:     s.Direction,
			GLAccountID:   acctID,
			IsActive:      true,
		}
		DB.Where("payment_method = ? AND direction = ?", s.PaymentMethod, s.Direction).
			FirstOrCreate(&row)
	}

	DB.Save(&models.FarmSettings{Key: "payment_method_accts_v1_done", Value: time.Now().Format(time.RFC3339)})
	return nil
}

// seedBankingAccountDetermination seeds BANKING module posting events
// (CASH_INFLOW and CASH_OUTFLOW) which were omitted from the original
// acct_det_ricemill_v1_done seed.
// Guarded by sentinel "acct_det_banking_v1_done".
func seedBankingAccountDetermination() error {
	var sentinel models.FarmSettings
	if DB.Where("`key` = ?", "acct_det_banking_v1_done").First(&sentinel).Error == nil {
		return nil // already seeded
	}

	accountIDByCode := func(code string) uint {
		var acct models.GLAccount
		if err := DB.Select("id").Where("code = ?", code).First(&acct).Error; err != nil {
			fmt.Printf("[ACCT-DET] WARN: GL account %s not found — skipping rule\n", code)
			return 0
		}
		return acct.ID
	}

	type ruleSpec struct {
		Module       string
		PostingEvent string
		GLCode       string
		Notes        string
	}

	rules := []ruleSpec{
		// Cash on Hand (1-1100) is the default cash account for both directions.
		// The banking service uses PaymentMethodAccount to resolve the specific
		// account per payment method; this catch-all satisfies the AD check.
		{"BANKING", "CASH_INFLOW",  "1-1100", "Collection — Cash / Bank Account (Dr)"},
		{"BANKING", "CASH_OUTFLOW", "1-1100", "AP Payment — Cash / Bank Account (Cr)"},
	}

	for _, r := range rules {
		acctID := accountIDByCode(r.GLCode)
		if acctID == 0 {
			continue
		}
		rule := models.AccountDetermination{
			Module:       r.Module,
			PostingEvent: r.PostingEvent,
			GLAccountID:  acctID,
			IsActive:     true,
			Notes:        r.Notes,
		}
		DB.Where("module = ? AND posting_event = ? AND item_category IS NULL", r.Module, r.PostingEvent).
			Assign(models.AccountDetermination{GLAccountID: acctID, Notes: r.Notes, IsActive: true}).
			FirstOrCreate(&rule)
	}

	DB.Save(&models.FarmSettings{Key: "acct_det_banking_v1_done", Value: time.Now().Format(time.RFC3339)})
	return nil
}

// seedWHTAccountDetermination seeds the AP_WHT_PAYABLE posting event so that
// CreateAPInvoice can look up the Withholding Tax Payable account (2-1300).
// Guarded by sentinel "acct_det_wht_v1_done".
func seedWHTAccountDetermination() error {
	var sentinel models.FarmSettings
	if DB.Where("`key` = ?", "acct_det_wht_v1_done").First(&sentinel).Error == nil {
		return nil // already seeded
	}

	var whtAcct models.GLAccount
	if err := DB.Select("id").Where("code = ?", "2-1300").First(&whtAcct).Error; err != nil {
		fmt.Printf("[ACCT-DET] WARN: GL account 2-1300 (WHT Payable) not found — WHT rule skipped\n")
		return nil
	}

	rule := models.AccountDetermination{
		Module:       "PURCHASING",
		PostingEvent: "AP_WHT_PAYABLE",
		GLAccountID:  whtAcct.ID,
		IsActive:     true,
		Notes:        "AP Invoice — BIR Withholding Tax Payable (Cr) — 2-1300",
	}
	DB.Where("module = 'PURCHASING' AND posting_event = 'AP_WHT_PAYABLE' AND item_category IS NULL").
		Assign(models.AccountDetermination{GLAccountID: whtAcct.ID, Notes: rule.Notes, IsActive: true}).
		FirstOrCreate(&rule)

	DB.Save(&models.FarmSettings{Key: "acct_det_wht_v1_done", Value: time.Now().Format(time.RFC3339)})
	return nil
}
