package app

import (
	"context"
	"fmt"
	"io"
	"os"

	appconfig "egglayererp/app/config"
	"egglayererp/app/constants"
	"egglayererp/app/db"
	"egglayererp/app/handlers"
	"egglayererp/app/middleware"
	"egglayererp/app/models"
	backupsvc "egglayererp/app/services/backup"
	flockssvc "egglayererp/app/services/flocks"
	opssvc "egglayererp/app/services/operations"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct. All exported methods are callable from JS.
type App struct {
	ctx context.Context
	cfg *appconfig.Config
}

func NewApp() *App { return &App{} }

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) DomReady(ctx context.Context) {
	a.ctx = ctx

	cfg, err := appconfig.Load()
	if err != nil {
		runtime.EventsEmit(ctx, "startup:error", fmt.Sprintf(
			"Failed to load config.toml: %s\n\nPlease ensure config.toml is in the same folder as the executable.",
			err.Error(),
		))
		return
	}
	a.cfg = cfg
	handlers.SetConfig(cfg)

	if err := db.Connect(cfg); err != nil {
		runtime.EventsEmit(ctx, "startup:db_error", fmt.Sprintf(
			"Cannot connect to MySQL.\n\nDSN: %s\nError: %s\n\nPlease check config.toml and ensure MySQL is running.",
			cfg.Database.DSN, err.Error(),
		))
		return
	}

	if err := db.PreMigrateFixup(); err != nil {
		runtime.EventsEmit(ctx, "startup:error", "Pre-migration fixup failed: "+err.Error())
		return
	}

	if err := db.AutoMigrateAll(); err != nil {
		runtime.EventsEmit(ctx, "startup:error", "Database migration failed: "+err.Error())
		return
	}

	if err := db.RunColumnMigrations(); err != nil {
		runtime.EventsEmit(ctx, "startup:error", "Column migration failed: "+err.Error())
		return
	}

	if err := db.SeedDefaults(); err != nil {
		runtime.EventsEmit(ctx, "startup:error", "Seed defaults failed: "+err.Error())
		return
	}

	runtime.EventsEmit(ctx, "startup:ready", nil)
}
func (a *App) Shutdown(ctx context.Context)  {}

// ─────────────────────── Auth ───────────────────────────────────────────────

func (a *App) Login(req handlers.LoginRequest) handlers.Response           { return handlers.Login(req) }
func (a *App) Logout() handlers.Response                                   { return handlers.Logout() }
func (a *App) GetCurrentUser() handlers.Response                           { return handlers.GetCurrentUser() }
func (a *App) ListUsers() handlers.Response                                { return handlers.ListUsers() }
func (a *App) CreateUser(req handlers.CreateUserRequest) handlers.Response { return handlers.CreateUser(req) }
func (a *App) UpdateUser(req handlers.UpdateUserRequest) handlers.Response { return handlers.UpdateUser(req) }
func (a *App) ChangePassword(req handlers.ChangePasswordRequest) handlers.Response {
	return handlers.ChangePassword(req)
}
func (a *App) SetPermissions(req handlers.SetPermissionsRequest) handlers.Response {
	return handlers.SetPermissions(req)
}
func (a *App) DeleteUser(userID uint) handlers.Response { return handlers.DeleteUser(userID) }

// ─────────────────────── Dashboard ──────────────────────────────────────────

func (a *App) GetDashboard() handlers.Response { return handlers.GetDashboard() }

// ─────────────────────── Flocks ─────────────────────────────────────────────

func (a *App) GetFlocks() handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	active, retired, err := flockssvc.ListFlocks()
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Data: map[string]interface{}{"active": active, "retired": retired}}
}

func (a *App) GetFlock(id uint) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	f, err := flockssvc.GetFlock(id)
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Data: f}
}

func (a *App) CreateFlock(req flockssvc.CreateFlockRequest) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	if !middleware.Store.CanAccess("Flocks") {
		return handlers.Response{OK: false, Message: "Access denied"}
	}
	req.CreatedByID = middleware.Store.UserID()
	f, err := flockssvc.CreateFlock(req)
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Flock created", Data: f}
}

func (a *App) UpdateFlock(req flockssvc.UpdateFlockRequest) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	if err := flockssvc.UpdateFlock(req); err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Flock updated"}
}

func (a *App) TransferFlock(req flockssvc.TransferFlockRequest) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	if err := flockssvc.TransferFlock(req); err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Flock transferred to Layer house"}
}

func (a *App) RetireFlock(req flockssvc.RetireFlockRequest) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	req.CreatedByID = middleware.Store.UserID()
	if err := flockssvc.RetireFlock(req); err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Flock retired"}
}

func (a *App) GetLayerHouses() handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	houses, err := flockssvc.GetLayerHouses()
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Data: houses}
}

func (a *App) GetHouseHistory(flockID uint) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	logs, logType, err := flockssvc.GetHouseHistory(flockID)
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Data: map[string]interface{}{"logs": logs, "log_type": logType}}
}

func (a *App) GetBodyWeightData(flockID uint) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	data, err := flockssvc.GetBodyWeightData(flockID)
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Data: data}
}

func (a *App) AddBodyWeightLog(flockID uint, date string, weekAge int, avgWeightG float64, sampleSize int, notes string) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	err := flockssvc.AddBodyWeightLog(flockID, date, weekAge, avgWeightG, sampleSize, notes, middleware.Store.UserID())
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Weight recorded"}
}

func (a *App) UpdateBodyWeightLog(id uint, date string, avgWeightG float64, sampleSize int, notes string) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	if err := flockssvc.UpdateBodyWeightLog(id, date, avgWeightG, sampleSize, notes); err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Weight updated"}
}

func (a *App) DeleteBodyWeightLog(id uint) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	if err := flockssvc.DeleteBodyWeightLog(id); err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Weight record removed"}
}

// ─────────────────────── Operations ─────────────────────────────────────────

func (a *App) RecordDailyLog(req opssvc.RecordDailyLogRequest) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	if !middleware.Store.CanAccess("Operations") {
		return handlers.Response{OK: false, Message: "Access denied"}
	}
	req.CreatedByID = middleware.Store.UserID()
	if err := opssvc.RecordDailyLog(req); err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Daily log saved"}
}

func (a *App) RecordGrowerLog(req opssvc.RecordGrowerLogRequest) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	if !middleware.Store.CanAccess("Operations") {
		return handlers.Response{OK: false, Message: "Access denied"}
	}
	req.CreatedByID = middleware.Store.UserID()
	if err := opssvc.RecordGrowerLog(req); err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Grower log saved"}
}

func (a *App) GetDailyLog(id uint) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	log, err := opssvc.GetDailyLog(id)
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Data: log}
}

func (a *App) GetGrowerLog(id uint) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	log, err := opssvc.GetGrowerLog(id)
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Data: log}
}

func (a *App) UpdateDailyLog(id uint, req opssvc.RecordDailyLogRequest) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	if !middleware.Store.CanAccess("Operations") {
		return handlers.Response{OK: false, Message: "Access denied"}
	}
	if err := opssvc.UpdateDailyLog(id, req); err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Log updated"}
}

func (a *App) UpdateGrowerLog(id uint, req opssvc.RecordGrowerLogRequest) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	if !middleware.Store.CanAccess("Operations") {
		return handlers.Response{OK: false, Message: "Access denied"}
	}
	if err := opssvc.UpdateGrowerLog(id, req); err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Message: "Log updated"}
}

func (a *App) ListFeedStocks() handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	stocks, err := opssvc.ListFeedStocks()
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.Response{OK: true, Data: stocks}
}

func (a *App) ListVaccineSchedules(flockID uint) handlers.Response {
	return handlers.ListVaccineSchedules(flockID)
}
func (a *App) CreateVaccineSchedule(flockID uint, vaccineName, scheduledDate, adminMethod string) handlers.Response {
	return handlers.CreateVaccineSchedule(flockID, vaccineName, scheduledDate, adminMethod)
}
func (a *App) MarkVaccineComplete(id uint) handlers.Response {
	return handlers.MarkVaccineComplete(id)
}

// ─────────────────────── Inventory ──────────────────────────────────────────

func (a *App) GetEggInventory() handlers.Response  { return handlers.GetEggInventory() }
func (a *App) ListInventory() handlers.Response    { return handlers.ListInventory() }
func (a *App) ListItemCategories() handlers.Response { return handlers.ListItemCategories(true) }
func (a *App) GetItemCategory(id uint) handlers.Response { return handlers.GetItemCategory(id) }
func (a *App) CreateItemCategory(cat models.ItemCategory) handlers.Response {
	return handlers.CreateItemCategory(cat)
}
func (a *App) UpdateItemCategory(id uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateItemCategory(id, updates)
}
func (a *App) DeleteItemCategory(id uint) handlers.Response { return handlers.DeleteItemCategory(id) }
func (a *App) ListItemMasters() handlers.Response  { return handlers.ListItemMasters(true) }
func (a *App) GetItemMaster(id uint) handlers.Response { return handlers.GetItemMaster(id) }
func (a *App) CreateItemMaster(item models.ItemMaster) handlers.Response {
	return handlers.CreateItemMaster(item)
}
func (a *App) UpdateItemMaster(id uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateItemMaster(id, updates)
}
func (a *App) DeleteItemMaster(id uint) handlers.Response { return handlers.DeleteItemMaster(id) }

// ─────────────────────── Purchasing ─────────────────────────────────────────

func (a *App) ListSuppliers(activeOnly bool) handlers.Response  { return handlers.ListSuppliers(activeOnly) }
func (a *App) GetSupplier(id uint) handlers.Response           { return handlers.GetSupplier(id) }
func (a *App) CreateSupplier(s models.Supplier) handlers.Response { return handlers.CreateSupplier(s) }
func (a *App) UpdateSupplier(id uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateSupplier(id, updates)
}
func (a *App) DeleteSupplier(id uint) handlers.Response { return handlers.DeleteSupplier(id) }

func (a *App) ListPurchases() handlers.Response       { return handlers.ListPurchases() }
func (a *App) GetPurchase(id uint) handlers.Response  { return handlers.GetPurchase(id) }
func (a *App) CreatePurchase(p models.Purchase) handlers.Response { return handlers.CreatePurchase(p) }
func (a *App) UpdatePurchase(id uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdatePurchase(id, updates)
}

func (a *App) CreateDeliveryReceipt(req handlers.CreateDeliveryReceiptRequest) handlers.Response {
	return handlers.CreateDeliveryReceipt(req)
}
func (a *App) GetDeliveryReceipt(id uint) handlers.Response { return handlers.GetDeliveryReceipt(id) }
func (a *App) ListDeliveryReceipts(purchaseID uint) handlers.Response {
	return handlers.ListDeliveryReceipts(purchaseID)
}
func (a *App) ConfirmDeliveryReceipt(id uint) handlers.Response {
	return handlers.ConfirmDeliveryReceipt(id)
}
func (a *App) CancelDeliveryReceipt(id uint) handlers.Response {
	return handlers.CancelDeliveryReceipt(id)
}

func (a *App) CreateAPInvoice(req handlers.CreateAPInvoiceRequest) handlers.Response {
	return handlers.CreateAPInvoice(req)
}
func (a *App) GetAPInvoice(id uint) handlers.Response { return handlers.GetAPInvoice(id) }
func (a *App) ListAPInvoices(purchaseID *uint, status string) handlers.Response {
	return handlers.ListAPInvoices(purchaseID, status)
}
func (a *App) CancelAPInvoice(id uint) handlers.Response { return handlers.CancelAPInvoice(id) }

func (a *App) CreateAPPayment(req handlers.CreateAPPaymentRequest) handlers.Response {
	return handlers.CreateAPPayment(req)
}
func (a *App) GetAPPayment(id uint) handlers.Response { return handlers.GetAPPayment(id) }
func (a *App) ListAPPayments() handlers.Response      { return handlers.ListAPPayments() }

// ─────────────────────── Sales ───────────────────────────────────────────────

func (a *App) ListCustomers(activeOnly bool) handlers.Response  { return handlers.ListCustomers(activeOnly) }
func (a *App) GetCustomer(id uint) handlers.Response           { return handlers.GetCustomer(id) }
func (a *App) CreateCustomer(c models.Customer) handlers.Response { return handlers.CreateCustomer(c) }
func (a *App) UpdateCustomer(id uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateCustomer(id, updates)
}
func (a *App) DeleteCustomer(id uint) handlers.Response { return handlers.DeleteCustomer(id) }

func (a *App) ListPriceGroups(activeOnly bool) handlers.Response { return handlers.ListPriceGroups(activeOnly) }
func (a *App) GetPriceGroup(id uint) handlers.Response           { return handlers.GetPriceGroup(id) }
func (a *App) CreatePriceGroup(pg models.PriceGroup) handlers.Response {
	return handlers.CreatePriceGroup(pg)
}
func (a *App) UpdatePriceGroup(id uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdatePriceGroup(id, updates)
}
func (a *App) DeletePriceGroup(id uint) handlers.Response { return handlers.DeletePriceGroup(id) }
func (a *App) UpsertPriceGroupItem(req handlers.UpsertPriceGroupItemRequest) handlers.Response {
	return handlers.UpsertPriceGroupItem(req)
}

func (a *App) CreateSalesOrder(req handlers.CreateSalesOrderRequest) handlers.Response {
	return handlers.CreateSalesOrder(req)
}
func (a *App) GetSalesOrder(id uint) handlers.Response  { return handlers.GetSalesOrder(id) }
func (a *App) ListSalesOrders() handlers.Response       { return handlers.ListSalesOrders() }
func (a *App) VoidSalesOrder(id uint) handlers.Response { return handlers.VoidSalesOrder(id) }

func (a *App) CreateDeliveryOrder(req handlers.CreateDeliveryOrderRequest) handlers.Response {
	return handlers.CreateDeliveryOrder(req)
}
func (a *App) GetDeliveryOrder(id uint) handlers.Response { return handlers.GetDeliveryOrder(id) }
func (a *App) ListDeliveryOrders(soID uint) handlers.Response { return handlers.ListDeliveryOrders(soID) }
func (a *App) ConfirmDeliveryOrder(id uint) handlers.Response { return handlers.ConfirmDeliveryOrder(id) }

func (a *App) CreateARInvoice(req handlers.CreateARInvoiceRequest) handlers.Response {
	return handlers.CreateARInvoice(req)
}
func (a *App) GetARInvoice(id uint) handlers.Response { return handlers.GetARInvoice(id) }
func (a *App) ListARInvoices(customerID *uint, status string) handlers.Response {
	return handlers.ListARInvoices(customerID, nil, status)
}
func (a *App) CancelARInvoice(id uint) handlers.Response { return handlers.CancelARInvoice(id) }

func (a *App) CreateCollection(req handlers.CreateCollectionRequest) handlers.Response {
	return handlers.CreateCollection(req)
}
func (a *App) GetCollection(id uint) handlers.Response { return handlers.GetCollection(id) }
func (a *App) ListCollections() handlers.Response      { return handlers.ListCollections() }

// ─────────────────────── Accounting ─────────────────────────────────────────

func (a *App) ListGLAccounts() handlers.Response                { return handlers.ListGLAccounts() }
func (a *App) GetChartOfAccounts() handlers.Response            { return handlers.ListGLAccounts() }
func (a *App) GetGLAccount(id uint) handlers.Response           { return handlers.GetGLAccount(id) }
func (a *App) CreateGLAccount(req handlers.CreateGLAccountRequest) handlers.Response {
	return handlers.CreateGLAccount(req)
}
func (a *App) UpdateGLAccount(id uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateGLAccount(id, updates)
}
func (a *App) DeleteGLAccount(id uint) handlers.Response { return handlers.DeleteGLAccount(id) }

func (a *App) ListAccountDeterminations() handlers.Response { return handlers.ListAccountDeterminations() }
func (a *App) UpsertAccountDetermination(req handlers.UpsertAccountDeterminationRequest) handlers.Response {
	return handlers.UpsertAccountDetermination(req)
}
func (a *App) DeleteAccountDetermination(id uint) handlers.Response {
	return handlers.DeleteAccountDetermination(id)
}
func (a *App) GetPostingEvents() handlers.Response {
	return handlers.OkResponse("", constants.PostingEvents)
}

func (a *App) ListPaymentMethodAccounts() handlers.Response { return handlers.ListPaymentMethodAccounts() }
func (a *App) UpsertPaymentMethodAccount(req handlers.UpsertPaymentMethodAccountRequest) handlers.Response {
	return handlers.UpsertPaymentMethodAccount(req)
}

func (a *App) ListJournalEntries(module string, limit int) handlers.Response {
	return handlers.ListJournalEntries(module, limit)
}
func (a *App) GetJournalEntry(id uint) handlers.Response { return handlers.GetJournalEntry(id) }
func (a *App) PostManualJournalEntry(req handlers.ManualJERequest) handlers.Response {
	return handlers.PostManualJournalEntry(req)
}
func (a *App) ReverseJournalEntry(id uint) handlers.Response {
	return handlers.ReverseJournalEntry(id)
}

// ─────────────────────── Reports ─────────────────────────────────────────────

func (a *App) GetPnL(startDate, endDate string) handlers.Response { return handlers.GetPnL(startDate, endDate) }
func (a *App) GetBalanceSheet(asOf string) handlers.Response      { return handlers.GetBalanceSheet(asOf) }
func (a *App) GetARAgingReport() handlers.Response                { return handlers.GetARAgingReport() }
func (a *App) GetAnalytics(from, to string) handlers.Response     { return handlers.GetAnalytics(from, to) }

// ─────────────────────── Backup ──────────────────────────────────────────────

// CreateBackup runs mysqldump to the managed backups directory (~\EggLayerERP\backups).
// The file is auto-named with a timestamp so it always appears in Backup History.
// Use DownloadBackup to export a copy to a custom location.
func (a *App) CreateBackup() handlers.Response {
	if !middleware.Store.IsAdmin() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	if a.cfg == nil {
		return handlers.Response{OK: false, Message: "App configuration not loaded"}
	}
	bf, err := backupsvc.CreateBackup(a.cfg)
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	return handlers.OkResponse("Backup created successfully", bf)
}

func (a *App) GetBackupHistory() handlers.Response        { return handlers.ListBackups() }
func (a *App) DeleteBackup(name string) handlers.Response { return handlers.DeleteBackup(name) }

// BackupDiagnostics returns a plain-text diagnostic report to help debug backup issues.
func (a *App) BackupDiagnostics() handlers.Response {
	lines := []string{}
	add := func(s string) { lines = append(lines, s) }

	add("=== Backup Diagnostics ===")
	add(fmt.Sprintf("IsAdmin: %v", middleware.Store.IsAdmin()))
	add(fmt.Sprintf("cfg nil: %v", a.cfg == nil))
	add(fmt.Sprintf("BackupDir: %s", backupsvc.GetBackupDir()))

	// check dir accessible
	if _, err := os.Stat(backupsvc.GetBackupDir()); err != nil {
		add("BackupDir stat error: " + err.Error())
	} else {
		add("BackupDir: accessible")
	}

	if a.cfg != nil {
		add(fmt.Sprintf("DSN prefix: %.30s...", a.cfg.Database.DSN))
		// try a real backup
		add("--- Running mysqldump ---")
		bf, err := backupsvc.CreateBackup(a.cfg)
		if err != nil {
			add("BACKUP ERROR: " + err.Error())
		} else {
			add(fmt.Sprintf("SUCCESS: %s (%d bytes)", bf.FileName, bf.FileSizeBytes))
			// clean up the test file
			_ = os.Remove(bf.Path)
			add("(test file removed)")
		}
	}

	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return handlers.OkResponse("", result)
}

// DownloadBackup opens a save-file dialog and copies the backup to the chosen location.
func (a *App) DownloadBackup(name string) handlers.Response {
	if !middleware.Store.IsAdmin() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	srcPath, err := backupsvc.GetBackupPath(name)
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	destPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Backup File",
		DefaultFilename: name,
		Filters: []runtime.FileFilter{
			{DisplayName: "SQL Files (*.sql)", Pattern: "*.sql"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || destPath == "" {
		return handlers.Response{OK: false, Message: "Save cancelled"}
	}
	if err := copyFile(srcPath, destPath); err != nil {
		return handlers.Response{OK: false, Message: "Copy failed: " + err.Error()}
	}
	return handlers.OkResponse("Backup saved to "+destPath, nil)
}

// GetBackupDir returns the current directory where backups are stored.
func (a *App) GetBackupDir() handlers.Response {
	if !middleware.Store.IsAdmin() {
		return handlers.Response{OK: false, Message: "Unauthorized"}
	}
	return handlers.OkResponse("", backupsvc.GetBackupDir())
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ─────────────────────── Settings ────────────────────────────────────────────

func (a *App) GetFarmSettings() handlers.Response { return handlers.GetFarmSettings() }
func (a *App) UpdateFarmSetting(key, value string) handlers.Response {
	return handlers.UpdateFarmSetting(key, value)
}
func (a *App) UpdateFarmSettings(settings map[string]string) handlers.Response {
	return handlers.UpdateFarmSettings(settings)
}
func (a *App) GetAuditLogs(limit int) handlers.Response { return handlers.GetAuditLogs(limit) }
