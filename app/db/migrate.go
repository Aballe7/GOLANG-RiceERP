package db

import (
	"egglayererp/app/models"
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

		// Flocks
		&models.Flock{},
		&models.DailyLog{},
		&models.GrowerLog{},
		&models.BodyWeightLog{},
		&models.EggProduction{},

		// Operations
		&models.VaccineSchedule{},
		&models.FeedStock{},
		&models.VaccineStock{},

		// Inventory
		&models.Inventory{},
		&models.ItemCategory{},
		&models.ItemMaster{},
		&models.UoMMaster{},
		&models.UoMGroup{},
		&models.UoMGroupLine{},

		// Purchasing
		&models.Supplier{},
		&models.Purchase{},
		&models.DeliveryReceipt{},
		&models.DeliveryReceiptItem{},
		&models.PurchaseHeader{},
		&models.PurchaseLine{},
		&models.APInvoice{},
		&models.APInvoiceItem{},
		&models.APPayment{},
		&models.APPaymentLine{},

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
		if err := seedDefaultCOA(); err != nil {
			return fmt.Errorf("failed to seed chart of accounts: %w", err)
		}
	}

	return nil
}

func seedDefaultCOA() error {
	accounts := []models.GLAccount{
		// ASSETS
		{Code: "1-0000", Name: "ASSETS", Section: "ASSET", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1000", Name: "Current Assets", Section: "ASSET", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1100", Name: "Cash on Hand", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1110", Name: "GCash", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1120", Name: "Bank Account", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1200", Name: "Accounts Receivable", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1300", Name: "Inventory", Section: "ASSET", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1310", Name: "Inventory — Feeds", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1320", Name: "Inventory — Eggs", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1330", Name: "Inventory — Live Birds", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1400", Name: "Goods Received Not Invoiced (GRNI)", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "1-1500", Name: "Input VAT Creditable", Section: "ASSET", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},

		// LIABILITIES
		{Code: "2-0000", Name: "LIABILITIES", Section: "LIABILITY", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "2-1000", Name: "Current Liabilities", Section: "LIABILITY", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "2-1100", Name: "Accounts Payable", Section: "LIABILITY", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "2-1200", Name: "Output VAT Payable", Section: "LIABILITY", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},

		// EQUITY
		{Code: "3-0000", Name: "EQUITY", Section: "EQUITY", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "3-1000", Name: "Owner's Capital", Section: "EQUITY", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "3-2000", Name: "Retained Earnings", Section: "EQUITY", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},

		// REVENUE
		{Code: "4-0000", Name: "REVENUE", Section: "REVENUE", AccountType: "HEADER", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "4-1000", Name: "Egg Sales Revenue", Section: "REVENUE", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "4-2000", Name: "Bird Sales Revenue", Section: "REVENUE", AccountType: "POSTING", NormalBalance: "CREDIT", IsSystem: true},
		{Code: "4-9000", Name: "Sales Discount", Section: "REVENUE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},

		// EXPENSES
		{Code: "5-0000", Name: "EXPENSES", Section: "EXPENSE", AccountType: "HEADER", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-1000", Name: "Cost of Goods Sold", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-2000", Name: "Feed Expense", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-3000", Name: "Mortality Loss", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-4000", Name: "Veterinary & Medicine Expense", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-5000", Name: "Supplies Expense", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
		{Code: "5-6000", Name: "Purchase Price Variance", Section: "EXPENSE", AccountType: "POSTING", NormalBalance: "DEBIT", IsSystem: true},
	}

	now := time.Now()
	for i := range accounts {
		accounts[i].CreatedAt = now
	}

	// Insert in order (headers first so foreign keys work if added later)
	return DB.CreateInBatches(accounts, 10).Error
}
