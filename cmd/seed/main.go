// cmd/seed/main.go — 2-month sample data seeder for EggLayerERP-Go.
// Run with: go run ./cmd/seed/main.go
// Safe to run multiple times; guarded by seed_done FarmSetting key.
package main

import (
	"fmt"
	"log"
	"time"

	appconfig "ricemill/app/config"
	"ricemill/app/db"
	"ricemill/app/models"
	accountingsvc "ricemill/app/services/accounting"
	purchasingsvc "ricemill/app/services/purchasing"
	salessvc "ricemill/app/services/sales"
)

func main() {
	// Load config and connect
	fmt.Println("[SEED] Connecting to database...")
	cfg, err := appconfig.Load(nil)
	must(err, "load config")
	must(db.Connect(cfg), "connect db")
	must(db.AutoMigrateAll(), "migrate")
	must(db.RunColumnMigrations(), "column migrations")
	must(db.SeedDefaults(), "seed defaults")

	// Idempotency guard
	var sentinel models.FarmSettings
	if err := db.DB.Where("`key` = ?", "seed_done").First(&sentinel).Error; err == nil {
		fmt.Println("[SEED] Already seeded. Delete the 'seed_done' farm_settings row to re-run.")
		return
	}

	// Admin user for created_by references
	var admin models.User
	if err := db.DB.First(&admin, "username = ?", "admin").Error; err != nil {
		log.Fatalf("[SEED] admin user not found: %v", err)
	}
	adminID := &admin.ID

	now := time.Now()
	startDate := now.AddDate(0, -2, 0).Truncate(24 * time.Hour)

	// ─────────────────────────────────────────────
	// Phase 1 — Master Data
	// ─────────────────────────────────────────────
	fmt.Println("[SEED] Creating master data...")

	// Suppliers
	sup1 := &models.Supplier{
		Name:          "AgriFeeds Co.",
		ContactPerson: "Ramon Santos",
		ContactNumber: "09171234567",
		Email:         "ramon@agrifeeds.com",
		Address:       "Silang, Cavite",
		Categories:    "Feeds",
		PaymentTerms:  "Net 30",
	}
	sup2 := &models.Supplier{
		Name:          "Vet Supplies PH",
		ContactPerson: "Ana Reyes",
		ContactNumber: "09281234567",
		Email:         "ana@vetsuppliesph.com",
		Address:       "General Trias, Cavite",
		Categories:    "Medicines",
		PaymentTerms:  "COD",
	}
	must(purchasingsvc.CreateSupplier(sup1), "create supplier 1")
	must(purchasingsvc.CreateSupplier(sup2), "create supplier 2")
	fmt.Println("[SEED]   ✓ 2 suppliers")

	// Price Groups
	pgWholesale := &models.PriceGroup{Name: "Wholesale", Description: "Supermarket & bulk buyers"}
	pgRetail := &models.PriceGroup{Name: "Retail", Description: "Local market & walk-in"}
	must(salessvc.CreatePriceGroup(pgWholesale), "create price group wholesale")
	must(salessvc.CreatePriceGroup(pgRetail), "create price group retail")

	// Price group items (Wholesale)
	wholesalePrices := map[string]float64{
		"Small": 180, "Medium": 210, "Large": 240, "ExtraLarge": 265, "Jumbo": 290,
	}
	for size, price := range wholesalePrices {
		must(salessvc.UpsertPriceGroupItem(pgWholesale.ID, size, "Tray", price), "upsert wholesale price "+size)
	}
	// Price group items (Retail)
	retailPrices := map[string]float64{
		"Small": 190, "Medium": 220, "Large": 250, "ExtraLarge": 275, "Jumbo": 300,
	}
	for size, price := range retailPrices {
		must(salessvc.UpsertPriceGroupItem(pgRetail.ID, size, "Tray", price), "upsert retail price "+size)
	}

	// Customers
	cust1 := &models.Customer{
		Name:         "Robinsons Supermarket",
		Address:      "Tanza, Cavite",
		ContactNumber: "09501234567",
		CustomerType: "Account",
		PriceGroupID: &pgWholesale.ID,
	}
	cust2 := &models.Customer{
		Name:         "Local Market — Tanza",
		Address:      "Tanza Public Market, Cavite",
		ContactNumber: "09661234567",
		CustomerType: "Account",
		PriceGroupID: &pgRetail.ID,
	}
	cust3 := &models.Customer{
		Name:         "Walk-in Buyer",
		CustomerType: "Walk-in",
		PriceGroupID: &pgRetail.ID,
	}
	must(salessvc.CreateCustomer(cust1), "create customer 1")
	must(salessvc.CreateCustomer(cust2), "create customer 2")
	must(salessvc.CreateCustomer(cust3), "create customer 3")
	fmt.Printf("[SEED]   ✓ 3 customers (2 price groups)\n")

	// Item categories and masters (for purchasing reference)
	catFeeds := &models.OITB{ItmsGrpNam: "Feeds", Description: "Layer and grower feeds"}
	catMeds := &models.OITB{ItmsGrpNam: "Medicines", Description: "Vaccines and medications"}
	must(db.DB.Create(catFeeds).Error, "create cat feeds")
	must(db.DB.Create(catMeds).Error, "create cat meds")

	item1 := &models.OITM{ItemCode: "FEED-LAY-001", ItemName: "Layer Complete Feed", ItmsGrpCod: catFeeds.ItmsGrpCod, InvntryUom: "kg", AvgPrice: 28, ValidFor: "Y"}
	item2 := &models.OITM{ItemCode: "FEED-GRW-001", ItemName: "Grower Feed", ItmsGrpCod: catFeeds.ItmsGrpCod, InvntryUom: "kg", AvgPrice: 24, ValidFor: "Y"}
	item3 := &models.OITM{ItemCode: "MED-NDV-001", ItemName: "Newcastle Vaccine", ItmsGrpCod: catMeds.ItmsGrpCod, InvntryUom: "vial", AvgPrice: 350, ValidFor: "Y"}
	must(db.DB.Create(item1).Error, "create item 1")
	must(db.DB.Create(item2).Error, "create item 2")
	must(db.DB.Create(item3).Error, "create item 3")
	fmt.Println("[SEED]   ✓ 3 item masters")

	// ─────────────────────────────────────────────
	// Phase 2 — Account Determination + Payment Method Accounts
	// (required so service functions can post GL entries)
	// ─────────────────────────────────────────────
	fmt.Println("[SEED] Setting up account determination...")

	// Resolve GL account IDs by code
	glID := func(code string) uint {
		var acct models.GLAccount
		if err := db.DB.Where("`code` = ?", code).First(&acct).Error; err != nil {
			log.Fatalf("[SEED] GL account %s not found: %v", code, err)
		}
		return acct.ID
	}

	// Account Determination mappings
	adRules := []struct {
		module, event string
		glCode        string
	}{
		// Purchasing
		{"PURCHASING", "GRNI", "1-1400"},
		{"PURCHASING", "INVENTORY_RECEIVED", "1-1310"},
		{"PURCHASING", "AP_PAYABLE", "2-1100"},
		{"PURCHASING", "AP_CLEARING", "2-1100"},
		// Sales
		{"SALES", "AR_RECEIVABLE", "1-1200"},
		{"SALES", "SALES_REVENUE", "4-1000"},
		{"SALES", "COLLECTION_CLEARING", "1-1200"},
	}
	for _, r := range adRules {
		id := glID(r.glCode)
		if err := accountingsvc.UpsertAccountDetermination(r.module, r.event, nil, id, "Seeded"); err != nil {
			log.Fatalf("[SEED] upsert AD %s/%s: %v", r.module, r.event, err)
		}
	}

	// Payment Method Accounts
	pmRules := []struct {
		method, bank, dir string
		glCode            string
	}{
		{"Cash", "", "BOTH", "1-1100"},
		{"Bank Transfer", "BDO", "BOTH", "1-1120"},
		{"GCash", "", "BOTH", "1-1110"},
	}
	for _, p := range pmRules {
		id := glID(p.glCode)
		if err := accountingsvc.UpsertPaymentMethodAccount(p.method, p.bank, id, p.dir); err != nil {
			log.Fatalf("[SEED] upsert payment method %s: %v", p.method, err)
		}
	}
	fmt.Println("[SEED]   ✓ Account determination and payment method accounts configured")

	// ─────────────────────────────────────────────
	// Phase 3 — Purchasing Transactions
	// ─────────────────────────────────────────────
	fmt.Println("[SEED] Creating purchasing transactions...")

	type purchaseCycle struct {
		supplierID   *uint
		supplierName string
		category     string
		itemName     string
		unit         string
		qty          float64
		unitPrice    float64
		dayOffset    int
	}

	cycles := []purchaseCycle{
		// Month 1
		{&sup1.ID, sup1.Name, "Feeds", "Layer Complete Feed", "sack", 50, 1400, 3},
		{&sup1.ID, sup1.Name, "Feeds", "Grower Feed", "sack", 30, 1200, 7},
		{&sup1.ID, sup1.Name, "Feeds", "Layer Complete Feed", "sack", 50, 1400, 18},
		{&sup2.ID, sup2.Name, "Medicines", "Newcastle Vaccine", "vial", 10, 350, 22},
		// Month 2
		{&sup1.ID, sup1.Name, "Feeds", "Layer Complete Feed", "sack", 50, 1400, 33},
		{&sup1.ID, sup1.Name, "Feeds", "Grower Feed", "sack", 30, 1200, 38},
		{&sup1.ID, sup1.Name, "Feeds", "Layer Complete Feed", "sack", 50, 1400, 48},
		{&sup2.ID, sup2.Name, "Medicines", "Newcastle Vaccine", "vial", 10, 350, 52},
	}

	purchaseCount := 0
	for _, c := range cycles {
		d := startDate.AddDate(0, 0, c.dayOffset)
		dateStr := d.Format("2006-01-02")
		total := c.qty * c.unitPrice

		// Create PurchaseHeader (SAP B1-aligned OPOR)
		ph, err := purchasingsvc.CreatePurchaseHeader(purchasingsvc.CreatePurchaseParams{
			Date:         dateStr,
			SupplierID:   c.supplierID,
			SupplierName: c.supplierName,
			Lines: []purchasingsvc.POLineInput{
				{Category: c.category, ItemName: c.itemName, Unit: c.unit, Quantity: c.qty, UnitPrice: c.unitPrice},
			},
		}, adminID)
		if err != nil {
			fmt.Printf("[SEED]   WARN: create purchase header: %v\n", err)
			continue
		}

		// Delivery Receipt (OPDN/PDN1)
		drItems := []models.DeliveryReceiptItem{
			{Category: c.category, Description: c.itemName, Unit: c.unit,
				QuantityOrdered: c.qty, Quantity: c.qty, Price: c.unitPrice},
		}
		dr, err := purchasingsvc.CreateDeliveryReceipt(ph.ID, dateStr, "Warehouse Staff", "", "", drItems, adminID)
		if err != nil {
			fmt.Printf("[SEED]   WARN: create DR: %v\n", err)
			continue
		}

		if err := purchasingsvc.ConfirmDeliveryReceipt(dr.ID, adminID); err != nil {
			// GL posting may fail if account determination not configured — that's OK
			fmt.Printf("[SEED]   INFO: confirm DR %s: %v\n", dr.DRNumber, err)
			// Force-set status to Received anyway
			db.DB.Model(&models.DeliveryReceipt{}).Where("id = ?", dr.ID).Update("status", "Received")
		}

		// AP Invoice (OPCH/PCH1)
		drIDPtr := &dr.ID
		apDueStr := d.AddDate(0, 0, 30).Format("2006-01-02")
		apItems := []models.APInvoiceItem{
			{Category: c.category, Description: c.itemName, Unit: c.unit, Quantity: c.qty, Price: c.unitPrice},
		}
		apInv, err := purchasingsvc.CreateAPInvoice(ph.ID, drIDPtr, dateStr, "Net 30", c.supplierName, "", "", &apDueStr, apItems, adminID, 0, 0, 0, "")
		if err != nil {
			fmt.Printf("[SEED]   WARN: create AP invoice: %v\n", err)
			continue
		}

		// AP Payment (OVPM)
		payDate := d.AddDate(0, 0, 5).Format("2006-01-02")
		payLines := []purchasingsvc.APPaymentLineInput{
			{APInvoiceID: apInv.ID, AmountApplied: total},
		}
		_, err = purchasingsvc.CreateAPPayment(payDate, c.supplierID, c.supplierName, "Cash", "", "", payLines, adminID)
		if err != nil {
			fmt.Printf("[SEED]   WARN: create AP payment: %v\n", err)
		}

		purchaseCount++
	}
	fmt.Printf("[SEED]   ✓ %d purchase cycles (PO → DR → APInvoice → APPayment)\n", purchaseCount)

	// ─────────────────────────────────────────────
	// Phase 4 — Sales Transactions
	// ─────────────────────────────────────────────
	fmt.Println("[SEED] Creating sales transactions...")

	type saleCycle struct {
		customer     *models.Customer
		items        []models.SalesOrderItem
		dayOffset    int
		collectFull  bool // false = partial (50%)
	}

	// Helper to make a tray line
	trayLine := func(size string, qty, price float64) models.SalesOrderItem {
		return models.SalesOrderItem{
			SKU:          size + "-TRAY",
			Unit:         "Tray",
			Quantity:     qty,
			PricePerUnit: price,
			LineTotal:    qty * price,
		}
	}

	salesCycles := []saleCycle{
		// Robinsons (wholesale) — 4 times across 2 months, fully collected
		{cust1, []models.SalesOrderItem{trayLine("Large", 200, 240), trayLine("ExtraLarge", 100, 265)}, 5, true},
		{cust1, []models.SalesOrderItem{trayLine("Large", 200, 240), trayLine("ExtraLarge", 100, 265)}, 12, true},
		{cust1, []models.SalesOrderItem{trayLine("Large", 180, 240), trayLine("ExtraLarge", 120, 265)}, 35, true},
		{cust1, []models.SalesOrderItem{trayLine("Large", 200, 240), trayLine("ExtraLarge", 100, 265)}, 50, true},

		// Local Market — 6 times, 2 with partial collection
		{cust2, []models.SalesOrderItem{trayLine("Medium", 100, 220), trayLine("Large", 50, 250)}, 4, true},
		{cust2, []models.SalesOrderItem{trayLine("Medium", 100, 220), trayLine("Large", 50, 250)}, 11, true},
		{cust2, []models.SalesOrderItem{trayLine("Medium", 80, 220), trayLine("Large", 40, 250)}, 20, false}, // partial
		{cust2, []models.SalesOrderItem{trayLine("Medium", 100, 220), trayLine("Large", 50, 250)}, 33, true},
		{cust2, []models.SalesOrderItem{trayLine("Medium", 100, 220), trayLine("Large", 50, 250)}, 44, true},
		{cust2, []models.SalesOrderItem{trayLine("Medium", 100, 220), trayLine("Large", 60, 250)}, 55, false}, // partial

		// Walk-in — 10 small orders, all fully collected
		{cust3, []models.SalesOrderItem{trayLine("Medium", 10, 220)}, 2, true},
		{cust3, []models.SalesOrderItem{trayLine("Large", 15, 250)}, 8, true},
		{cust3, []models.SalesOrderItem{trayLine("Medium", 20, 220), trayLine("Large", 10, 250)}, 14, true},
		{cust3, []models.SalesOrderItem{trayLine("ExtraLarge", 10, 275)}, 19, true},
		{cust3, []models.SalesOrderItem{trayLine("Large", 30, 250)}, 25, true},
		{cust3, []models.SalesOrderItem{trayLine("Medium", 15, 220)}, 31, true},
		{cust3, []models.SalesOrderItem{trayLine("Large", 20, 250), trayLine("ExtraLarge", 5, 275)}, 37, true},
		{cust3, []models.SalesOrderItem{trayLine("Medium", 25, 220)}, 43, true},
		{cust3, []models.SalesOrderItem{trayLine("Large", 10, 250)}, 49, true},
		{cust3, []models.SalesOrderItem{trayLine("Medium", 30, 220), trayLine("Large", 10, 250)}, 56, true},
	}

	salesCount := 0
	for _, sc := range salesCycles {
		d := startDate.AddDate(0, 0, sc.dayOffset)
		dateStr := d.Format("2006-01-02")

		// Grand total
		grandTotal := 0.0
		for _, item := range sc.items {
			grandTotal += item.LineTotal
		}

		custIDPtr := &sc.customer.ID
		dueStr := d.AddDate(0, 0, 30).Format("2006-01-02")
		payMethod := "Cash"
		terms := "COD"
		if sc.customer.CustomerType == "Account" {
			payMethod = "Bank Transfer"
			terms = "Net 30"
		}

		so, err := salessvc.CreateSalesOrder(
			dateStr,
			custIDPtr,
			sc.customer.Name, sc.customer.Address, sc.customer.ContactNumber,
			payMethod, terms, &dueStr,
			"",
			sc.items,
			adminID,
		)
		if err != nil {
			fmt.Printf("[SEED]   WARN: create SO: %v\n", err)
			continue
		}

		// Delivery Order
		doItems := make([]models.DeliveryOrderItem, len(sc.items))
		for i, item := range sc.items {
			doItems[i] = models.DeliveryOrderItem{
				SKU:               item.SKU,
				Unit:              item.Unit,
				QuantityOrdered:   item.Quantity,
				QuantityDelivered: item.Quantity,
				PricePerUnit:      item.PricePerUnit,
			}
		}
		dOrder, err := salessvc.CreateDeliveryOrder(so.ID, dateStr, "Driver", "", doItems, adminID)
		if err != nil {
			fmt.Printf("[SEED]   WARN: create DO: %v\n", err)
			continue
		}
		if err := salessvc.ConfirmDeliveryOrder(dOrder.ID, adminID); err != nil {
			fmt.Printf("[SEED]   INFO: confirm DO %s: %v\n", dOrder.DeliveryNumber, err)
			db.DB.Model(&models.DeliveryOrder{}).Where("id = ?", dOrder.ID).Update("status", "Delivered")
			db.DB.Model(&models.SalesOrder{}).Where("id = ?", so.ID).UpdateColumn("amount_delivered", grandTotal)
		}

		// AR Invoice
		arItems := make([]models.ARInvoiceItem, len(sc.items))
		for i, item := range sc.items {
			arItems[i] = models.ARInvoiceItem{
				SKU:          item.SKU,
				Unit:         item.Unit,
				Quantity:     item.Quantity,
				PricePerUnit: item.PricePerUnit,
			}
		}
		arInv, err := salessvc.CreateARInvoice(
			[]uint{dOrder.ID}, custIDPtr,
			sc.customer.Name, sc.customer.Address, sc.customer.ContactNumber,
			dateStr, terms, "",
			&dueStr,
			arItems,
			adminID,
		)
		if err != nil {
			fmt.Printf("[SEED]   WARN: create AR invoice: %v\n", err)
			continue
		}

		// Collection
		collectAmt := grandTotal
		if !sc.collectFull {
			collectAmt = grandTotal * 0.5 // 50% partial
		}
		collectDate := d.AddDate(0, 0, 1).Format("2006-01-02")
		collectLines := []salessvc.CollectionLineInput{
			{ARInvoiceID: arInv.ID, AmountApplied: collectAmt},
		}
		_, err = salessvc.CreateCollection(collectDate, custIDPtr, sc.customer.Name, payMethod, "", "", collectLines, adminID)
		if err != nil {
			fmt.Printf("[SEED]   WARN: create collection: %v\n", err)
		}

		salesCount++
	}
	fmt.Printf("[SEED]   ✓ %d sales cycles (SO → DO → ARInvoice → Collection)\n", salesCount)

	// ─────────────────────────────────────────────
	// Mark seed done
	// ─────────────────────────────────────────────
	db.DB.Save(&models.FarmSettings{Key: "seed_done", Value: time.Now().Format(time.RFC3339)})
	fmt.Println("[SEED] Done. Seed key written.")
}

func must(err error, ctx string) {
	if err != nil {
		log.Fatalf("[SEED] %s: %v", ctx, err)
	}
}

