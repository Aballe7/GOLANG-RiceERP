package app

import (
	"context"
	"fmt"
	"io"
	"os"

	appconfig "ricemill/app/config"
	"ricemill/app/constants"
	"ricemill/app/db"
	"ricemill/app/handlers"
	"ricemill/app/middleware"
	"ricemill/app/models"
	auditsvc "ricemill/app/services/audit"
	backupsvc "ricemill/app/services/backup"
	invsvc "ricemill/app/services/inventory"
	productionsvc "ricemill/app/services/production"
	purchasingsvc "ricemill/app/services/purchasing"
	reportssvc "ricemill/app/services/reports"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct. All exported methods are callable from JS.
type App struct {
	ctx           context.Context
	cfg           *appconfig.Config
	defaultConfig []byte
}

func NewApp(defaultConfig []byte) *App { return &App{defaultConfig: defaultConfig} }

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) DomReady(ctx context.Context) {
	a.ctx = ctx

	cfg, err := appconfig.Load(a.defaultConfig)
	if err != nil {
		runtime.EventsEmit(ctx, "startup:error", fmt.Sprintf(
			"Failed to load config: %s", err.Error(),
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

	// RunOITMSchemaMigrations MUST run before AutoMigrateAll so that table and
	// column renames are applied before GORM tries to create new tables.
	if err := db.RunOITMSchemaMigrations(); err != nil {
		runtime.EventsEmit(ctx, "startup:error", "OITM schema migration failed: "+err.Error())
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
func (a *App) DemoLogin() handlers.Response                                { return handlers.DemoLogin() }
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

// ─────────────────────── Inventory ──────────────────────────────────────────

func (a *App) ListItemCategories() handlers.Response { return handlers.ListItemCategories(true) }
func (a *App) GetItemCategory(id int) handlers.Response  { return handlers.GetItemCategory(id) }
func (a *App) CreateItemCategory(cat models.OITB) handlers.Response {
	return handlers.CreateItemCategory(cat)
}
func (a *App) UpdateItemCategory(id int, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateItemCategory(id, updates)
}
func (a *App) DeleteItemCategory(id int) handlers.Response { return handlers.DeleteItemCategory(id) }
func (a *App) ListItemMasters() handlers.Response        { return handlers.ListItemMasters(true) }
func (a *App) ListProductionItems() handlers.Response    { return handlers.ListProductionItems() }
func (a *App) ListSalesItems() handlers.Response         { return handlers.ListSalesItems() }
func (a *App) ListPurchasingItems() handlers.Response    { return handlers.ListPurchasingItems() }
func (a *App) ListInventoryItems() handlers.Response     { return handlers.ListInventoryItems() }
func (a *App) ListItemCategoriesForModule(module string) handlers.Response {
	return handlers.ListItemCategoriesForModule(module)
}
func (a *App) GetItemMaster(id uint) handlers.Response { return handlers.GetItemMaster(id) }
func (a *App) CreateItemMaster(item models.ItemMaster) handlers.Response {
	return handlers.CreateItemMaster(item)
}
func (a *App) UpdateItemMaster(id uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateItemMaster(id, updates)
}
func (a *App) DeleteItemMaster(id uint) handlers.Response { return handlers.DeleteItemMaster(id) }

// ─────────────────────── ItemUoMPrice (ITM9) ────────────────────────────────

func (a *App) GetItemUoMPrices(itemCode string) handlers.Response {
	return handlers.GetItemUoMPrices(itemCode)
}
func (a *App) UpsertItemUoMPrice(req handlers.UpsertItemUoMPriceRequest) handlers.Response {
	return handlers.UpsertItemUoMPrice(req)
}
func (a *App) DeleteItemUoMPrice(itemCode string, uomEntry uint) handlers.Response {
	return handlers.DeleteItemUoMPrice(itemCode, uomEntry)
}
func (a *App) GetItemUoMMapping(itemCode string) handlers.Response {
	return handlers.GetItemUoMMapping(itemCode)
}

// ─────────────────────── UoM Master (OUOM) ──────────────────────────────────

func (a *App) ListUoMMasters() handlers.Response { return handlers.ListUoMMasters() }
func (a *App) GetUoMMaster(entry uint) handlers.Response { return handlers.GetUoMMaster(entry) }
func (a *App) CreateUoMMaster(item models.UoMMaster) handlers.Response {
	return handlers.CreateUoMMaster(item)
}
func (a *App) UpdateUoMMaster(entry uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateUoMMaster(entry, updates)
}
func (a *App) DeleteUoMMaster(entry uint) handlers.Response { return handlers.DeleteUoMMaster(entry) }

// ─────────────────────── UoM Groups (OUGP) ──────────────────────────────────

func (a *App) ListUoMGroups() handlers.Response { return handlers.ListUoMGroups() }
func (a *App) GetUoMGroup(entry uint) handlers.Response { return handlers.GetUoMGroup(entry) }
func (a *App) CreateUoMGroup(req invsvc.CreateUoMGroupRequest) handlers.Response {
	return handlers.CreateUoMGroup(req)
}
func (a *App) UpdateUoMGroup(entry uint, ugpCode, ugpName string, baseUom int, lines []invsvc.UoMGroupLineInput) handlers.Response {
	return handlers.UpdateUoMGroup(entry, ugpCode, ugpName, baseUom, lines)
}
func (a *App) DeleteUoMGroup(entry uint) handlers.Response { return handlers.DeleteUoMGroup(entry) }

// ── Goods Receipt ──────────────────────────────────────────────────────────
func (a *App) ListGoodsReceipts() handlers.Response { return handlers.ListGoodsReceipts() }
func (a *App) GetGoodsReceipt(id uint) handlers.Response { return handlers.GetGoodsReceipt(id) }
func (a *App) CreateGoodsReceipt(req invsvc.CreateGoodsReceiptRequest) handlers.Response {
	return handlers.CreateGoodsReceipt(req)
}
func (a *App) CancelGoodsReceipt(id uint) handlers.Response { return handlers.CancelGoodsReceipt(id) }

// ── Goods Issue ──────────────────────────────────────────────────────────────
func (a *App) ListGoodsIssues() handlers.Response { return handlers.ListGoodsIssues() }
func (a *App) GetGoodsIssue(id uint) handlers.Response { return handlers.GetGoodsIssue(id) }
func (a *App) CreateGoodsIssue(req invsvc.CreateGoodsIssueRequest) handlers.Response {
	return handlers.CreateGoodsIssue(req)
}
func (a *App) CancelGoodsIssue(id uint) handlers.Response { return handlers.CancelGoodsIssue(id) }

// ─────────────────────── OWHS — Warehouse Master ─────────────────────────────

func (a *App) ListWarehouses() handlers.Response { return handlers.ListWarehouses() }
func (a *App) GetWarehouse(whsCode string) handlers.Response {
	return handlers.GetWarehouse(whsCode)
}
func (a *App) CreateWarehouse(whs models.OWHS) handlers.Response {
	return handlers.CreateWarehouse(whs)
}
func (a *App) UpdateWarehouse(whsCode string, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateWarehouse(whsCode, updates)
}

// ─────────────────────── OITW — Per-Warehouse Stock ──────────────────────────

func (a *App) GetItemWarehouseStock(itemCode string) handlers.Response {
	return handlers.GetItemWarehouseStock(itemCode)
}
func (a *App) GetWarehouseStock(whsCode string) handlers.Response {
	return handlers.GetWarehouseStock(whsCode)
}
func (a *App) ListAllWarehouseStock() handlers.Response {
	return handlers.ListAllWarehouseStock()
}
func (a *App) ListOIVL(itemCode string) handlers.Response {
	return handlers.ListOIVL(itemCode)
}
func (a *App) UpdateOITWMinMax(itemCode, whsCode string, minStock, maxStock float64) handlers.Response {
	return handlers.UpdateOITWMinMax(itemCode, whsCode, minStock, maxStock)
}

// ─────────────────────── ITM1 — Item Price List ───────────────────────────────

func (a *App) ListItemPrices(itemCode string) handlers.Response {
	return handlers.ListItemPrices(itemCode)
}
func (a *App) UpsertItemPrice(req handlers.UpsertItemPriceRequest) handlers.Response {
	return handlers.UpsertItemPrice(req)
}

// ─────────────────────── OSPP — Special Prices ───────────────────────────────

func (a *App) ListSpecialPrices(itemCode string) handlers.Response {
	return handlers.ListSpecialPrices(itemCode)
}
func (a *App) UpsertSpecialPrice(sp models.OSPP) handlers.Response {
	return handlers.UpsertSpecialPrice(sp)
}
func (a *App) DeleteSpecialPrice(cardCode, itemCode string) handlers.Response {
	return handlers.DeleteSpecialPrice(cardCode, itemCode)
}

// ─────────────────────── Purchasing ─────────────────────────────────────────

func (a *App) ListPurchasingLookup(category string) handlers.Response {
	return handlers.ListPurchasingLookup(category)
}

func (a *App) ListSuppliers(activeOnly bool) handlers.Response  { return handlers.ListSuppliers(activeOnly) }
func (a *App) GetSupplier(id uint) handlers.Response           { return handlers.GetSupplier(id) }
func (a *App) CreateSupplier(s models.Supplier) handlers.Response { return handlers.CreateSupplier(s) }
func (a *App) UpdateSupplier(id uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateSupplier(id, updates)
}
func (a *App) DeleteSupplier(id uint) handlers.Response  { return handlers.DeleteSupplier(id) }
func (a *App) ApproveSupplier(id uint) handlers.Response { return handlers.ApproveSupplier(id) }
func (a *App) RejectSupplier(id uint) handlers.Response  { return handlers.RejectSupplier(id) }

func (a *App) ListPurchaseHeaders() handlers.Response                        { return handlers.ListPurchaseHeaders() }
func (a *App) GetPurchaseHeader(id uint) handlers.Response                   { return handlers.GetPurchaseHeader(id) }
func (a *App) ListOpenPurchaseHeaders(supplierID uint) handlers.Response     { return handlers.ListOpenPurchaseHeaders(supplierID) }
func (a *App) CreatePurchaseHeader(req purchasingsvc.CreatePurchaseParams) handlers.Response {
	return handlers.CreatePurchaseHeader(req)
}
func (a *App) CancelPurchaseHeader(id uint) handlers.Response { return handlers.CancelPurchaseHeader(id) }
func (a *App) GetNextPONumber() handlers.Response             { return handlers.GetNextPONumber() }

func (a *App) CreateDeliveryReceipt(req handlers.CreateDeliveryReceiptRequest) handlers.Response {
	return handlers.CreateDeliveryReceipt(req)
}
func (a *App) GetDeliveryReceipt(id uint) handlers.Response { return handlers.GetDeliveryReceipt(id) }
func (a *App) ListDeliveryReceipts(purchaseHeaderID uint) handlers.Response {
	return handlers.ListDeliveryReceipts(purchaseHeaderID)
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
func (a *App) ListAPInvoices(purchaseHeaderID *uint, status string) handlers.Response {
	return handlers.ListAPInvoices(purchaseHeaderID, status)
}
func (a *App) CancelAPInvoice(id uint) handlers.Response { return handlers.CancelAPInvoice(id) }

func (a *App) CreateAPPayment(req handlers.CreateAPPaymentRequest) handlers.Response {
	return handlers.CreateAPPayment(req)
}
func (a *App) GetAPPayment(id uint) handlers.Response  { return handlers.GetAPPayment(id) }
func (a *App) ListAPPayments() handlers.Response       { return handlers.ListAPPayments() }
func (a *App) CancelAPPayment(id uint) handlers.Response { return handlers.CancelAPPayment(id) }

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
func (a *App) SubmitSalesOrder(id uint) handlers.Response { return handlers.SubmitSalesOrder(id) }
func (a *App) CancelSalesOrder(id uint) handlers.Response { return handlers.CancelSalesOrder(id) }
func (a *App) UpdateSalesOrder(id uint, req handlers.UpdateSalesOrderRequest) handlers.Response {
	return handlers.UpdateSalesOrder(id, req)
}

func (a *App) CreateDeliveryOrder(req handlers.CreateDeliveryOrderRequest) handlers.Response {
	return handlers.CreateDeliveryOrder(req)
}
func (a *App) GetDeliveryOrder(id uint) handlers.Response { return handlers.GetDeliveryOrder(id) }
func (a *App) ListDeliveryOrders(soID uint) handlers.Response { return handlers.ListDeliveryOrders(soID) }
func (a *App) ConfirmDeliveryOrder(id uint) handlers.Response { return handlers.ConfirmDeliveryOrder(id) }
func (a *App) CancelDeliveryOrder(id uint) handlers.Response  { return handlers.CancelDeliveryOrder(id) }

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
func (a *App) CancelCollection(id uint) handlers.Response { return handlers.CancelCollection(id) }
func (a *App) GetCollection(id uint) handlers.Response    { return handlers.GetCollection(id) }
func (a *App) ListCollections() handlers.Response         { return handlers.ListCollections() }

// ─────────────────────── Accounting ─────────────────────────────────────────

func (a *App) ListGLAccounts() handlers.Response                { return handlers.ListGLAccounts() }
func (a *App) GetGLAccounts() handlers.Response                 { return handlers.ListGLAccounts() }
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

// ─────────────────────── Withholding Tax (BIR EWT) ───────────────────────────

func (a *App) ComputeWHT(req handlers.ComputeWHTRequest) handlers.Response {
	return handlers.ComputeWHT(req)
}
func (a *App) ListWHTEntries(year, quarter int) handlers.Response {
	return handlers.ListWHTEntries(year, quarter)
}
func (a *App) GetForm0619E(year, month int) handlers.Response {
	return handlers.GetForm0619E(year, month)
}
func (a *App) GetForm1601EQ(year, quarter int) handlers.Response {
	return handlers.GetForm1601EQ(year, quarter)
}
func (a *App) GetForm2307(supplierID uint, year, quarter int, company string) handlers.Response {
	return handlers.GetForm2307(supplierID, year, quarter, company)
}
func (a *App) ListSuppliersWithWHT(year int) handlers.Response {
	return handlers.ListSuppliersWithWHT(year)
}

// ─────────────────────── Reports ─────────────────────────────────────────────

func (a *App) GetPnL(startDate, endDate string) handlers.Response { return handlers.GetPnL(startDate, endDate) }
func (a *App) GetBalanceSheet(asOf string) handlers.Response      { return handlers.GetBalanceSheet(asOf) }
func (a *App) GetARAgingReport() handlers.Response                              { return handlers.GetARAgingReport() }
func (a *App) GetAPAgingReport() handlers.Response                              { return handlers.GetAPAgingReport() }
func (a *App) GetTrialBalance(startDate, endDate string) handlers.Response      { return handlers.GetTrialBalance(startDate, endDate) }
func (a *App) GetAnalytics(from, to string) handlers.Response                  { return handlers.GetAnalytics(from, to) }
func (a *App) GetCashFlowStatement(startDate, endDate string) handlers.Response { return handlers.GetCashFlowStatement(startDate, endDate) }

// ── Inventory Valuation Reports ───────────────────────────────────────────────

// GetInventoryValuationReport returns stock quantities and values as of asOf.
// whsCode="" = all warehouses; itmGrpCod=0 = all item groups.
func (a *App) GetInventoryValuationReport(asOf, whsCode string, itmGrpCod int) handlers.Response {
	return handlers.GetInventoryValuationReport(asOf, whsCode, itmGrpCod)
}

// GetInventoryMovementReport returns OIVL ledger entries for the given filters.
// dateFrom/dateTo: YYYY-MM-DD; itemCode/whsCode/transType: "" = no filter.
func (a *App) GetInventoryMovementReport(dateFrom, dateTo, itemCode, whsCode, transType string) handlers.Response {
	return handlers.GetInventoryMovementReport(dateFrom, dateTo, itemCode, whsCode, transType)
}

func (a *App) ExportBalanceSheetExcel(asOf string) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized. Please log in."}
	}
	if !middleware.Store.CanAccess("Reports") {
		return handlers.Response{OK: false, Message: "Access denied: you don't have permission to access Reports."}
	}
	bs, err := reportssvc.GetBalanceSheet(asOf)
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	destPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Balance Sheet",
		DefaultFilename: fmt.Sprintf("BalanceSheet_%s.xlsx", asOf),
		Filters: []runtime.FileFilter{
			{DisplayName: "Excel Files (*.xlsx)", Pattern: "*.xlsx"},
		},
	})
	if err != nil || destPath == "" {
		return handlers.Response{OK: false, Message: "Export cancelled"}
	}
	if err := reportssvc.ExportBalanceSheetExcel(bs, destPath); err != nil {
		return handlers.Response{OK: false, Message: "Export failed: " + err.Error()}
	}
	auditsvc.Log("EXPORT", "Report", "BalanceSheet", asOf, "Balance Sheet exported: "+destPath, nil)
	return handlers.OkResponse("Balance Sheet exported to "+destPath, nil)
}

func (a *App) ExportPnLExcel(startDate, endDate string) handlers.Response {
	if !middleware.Store.IsLoggedIn() {
		return handlers.Response{OK: false, Message: "Unauthorized. Please log in."}
	}
	if !middleware.Store.CanAccess("Reports") {
		return handlers.Response{OK: false, Message: "Access denied: you don't have permission to access Reports."}
	}
	pnl, err := reportssvc.GetPnL(startDate, endDate)
	if err != nil {
		return handlers.Response{OK: false, Message: err.Error()}
	}
	destPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Profit & Loss",
		DefaultFilename: fmt.Sprintf("PnL_%s_to_%s.xlsx", startDate, endDate),
		Filters: []runtime.FileFilter{
			{DisplayName: "Excel Files (*.xlsx)", Pattern: "*.xlsx"},
		},
	})
	if err != nil || destPath == "" {
		return handlers.Response{OK: false, Message: "Export cancelled"}
	}
	if err := reportssvc.ExportPnLExcel(pnl, destPath); err != nil {
		return handlers.Response{OK: false, Message: "Export failed: " + err.Error()}
	}
	auditsvc.Log("EXPORT", "Report", "PnL", startDate+"_"+endDate, "P&L exported: "+destPath, nil)
	return handlers.OkResponse("P&L exported to "+destPath, nil)
}

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
	auditsvc.Log("CREATE", "Backup", "Backup", bf.FileName, "Database backup created: "+bf.FileName, nil)
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
	auditsvc.Log("EXPORT", "Backup", "Backup", name, "Database backup exported: "+name+" → "+destPath, nil)
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

// ─────────────────────── Production ──────────────────────────────────────────

func (a *App) ListMillingOrders() handlers.Response { return handlers.ListMillingOrders() }
func (a *App) GetMillingOrder(id uint) handlers.Response { return handlers.GetMillingOrder(id) }
func (a *App) CreateMillingOrder(req productionsvc.CreateMillingOrderRequest) handlers.Response {
	return handlers.CreateMillingOrder(req)
}
func (a *App) StartMillingOrder(id uint) handlers.Response { return handlers.StartMillingOrder(id) }
func (a *App) CompleteMillingOrder(id uint, req productionsvc.CompleteMillingRequest) handlers.Response {
	return handlers.CompleteMillingOrder(id, req)
}
func (a *App) CancelMillingOrder(id uint) handlers.Response { return handlers.CancelMillingOrder(id) }

// BOM
func (a *App) ListBOMs() handlers.Response                         { return handlers.ListBOMs() }
func (a *App) GetBOM(code string) handlers.Response                { return handlers.GetBOM(code) }
func (a *App) UpsertBOM(bom models.ProductTree) handlers.Response  { return handlers.UpsertBOM(bom) }
func (a *App) DeleteBOM(code string) handlers.Response             { return handlers.DeleteBOM(code) }

// Work Orders
func (a *App) ListWorkOrders() handlers.Response                   { return handlers.ListWorkOrders() }
func (a *App) GetWorkOrder(docEntry uint) handlers.Response        { return handlers.GetWorkOrder(docEntry) }
func (a *App) CreateWorkOrder(req productionsvc.CreateWORequest) handlers.Response {
	return handlers.CreateWorkOrder(req)
}
func (a *App) ReleaseWorkOrder(docEntry uint) handlers.Response { return handlers.ReleaseWorkOrder(docEntry) }
func (a *App) CloseWorkOrder(docEntry uint, cmpltQty, rjctQty float64) handlers.Response {
	return handlers.CloseWorkOrder(docEntry, cmpltQty, rjctQty)
}
func (a *App) CancelWorkOrder(docEntry uint) handlers.Response { return handlers.CancelWorkOrder(docEntry) }

// ─────────────────────── Fixed Assets ────────────────────────────────────────

func (a *App) ListFixedAssets() handlers.Response                          { return handlers.ListFixedAssets() }
func (a *App) GetFixedAsset(id uint) handlers.Response                     { return handlers.GetFixedAsset(id) }
func (a *App) CreateFixedAsset(req handlers.CreateFixedAssetRequest) handlers.Response {
	return handlers.CreateFixedAsset(req)
}
func (a *App) UpdateFixedAsset(id uint, updates map[string]interface{}) handlers.Response {
	return handlers.UpdateFixedAsset(id, updates)
}
func (a *App) GetDepreciationSchedule(assetID uint) handlers.Response      { return handlers.GetDepreciationSchedule(assetID) }
func (a *App) RunDepreciation(req handlers.RunDepreciationRequest) handlers.Response {
	return handlers.RunDepreciation(req)
}
func (a *App) DisposeAsset(req handlers.DisposeAssetRequest) handlers.Response {
	return handlers.DisposeAsset(req)
}
