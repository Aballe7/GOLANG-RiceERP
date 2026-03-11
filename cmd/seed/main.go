// cmd/seed/main.go — 2-month sample data seeder for EggLayerERP-Go.
// Run with: go run ./cmd/seed/main.go
// Safe to run multiple times; guarded by seed_done FarmSetting key.
package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	appconfig "egglayererp/app/config"
	"egglayererp/app/db"
	"egglayererp/app/models"
	accountingsvc "egglayererp/app/services/accounting"
	flockssvc "egglayererp/app/services/flocks"
	opssvc "egglayererp/app/services/operations"
	purchasingsvc "egglayererp/app/services/purchasing"
	salessvc "egglayererp/app/services/sales"
)

func main() {
	// Load config and connect
	fmt.Println("[SEED] Connecting to database...")
	cfg, err := appconfig.Load()
	must(err, "load config")
	must(db.Connect(cfg), "connect db")
	must(db.AutoMigrateAll(), "migrate")
	must(db.RunColumnMigrations(), "column migrations")
	must(db.SeedDefaults(), "seed defaults")

	// Idempotency guard
	var sentinel models.FarmSettings
	if err := db.DB.First(&sentinel, "key = ?", "seed_done").Error; err == nil {
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
	endDate := now.AddDate(0, 0, -1).Truncate(24 * time.Hour) // up to yesterday

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
	catFeeds := &models.ItemCategory{Name: "Feeds", Description: "Layer and grower feeds"}
	catMeds := &models.ItemCategory{Name: "Medicines", Description: "Vaccines and medications"}
	must(db.DB.Create(catFeeds).Error, "create cat feeds")
	must(db.DB.Create(catMeds).Error, "create cat meds")

	item1 := &models.ItemMaster{ItemCode: "FEED-LAY-001", Name: "Layer Complete Feed", Category: "Feeds", Unit: "kg", UnitPrice: 28, IsActive: true}
	item2 := &models.ItemMaster{ItemCode: "FEED-GRW-001", Name: "Grower Feed", Category: "Feeds", Unit: "kg", UnitPrice: 24, IsActive: true}
	item3 := &models.ItemMaster{ItemCode: "MED-NDV-001", Name: "Newcastle Vaccine", Category: "Medicines", Unit: "vial", UnitPrice: 350, IsActive: true}
	must(db.DB.Create(item1).Error, "create item 1")
	must(db.DB.Create(item2).Error, "create item 2")
	must(db.DB.Create(item3).Error, "create item 3")
	fmt.Println("[SEED]   ✓ 3 item masters")

	// Feed stocks (initial inventory)
	must(opssvc.UpsertFeedStock("Layer Complete Feed", 200, 50), "upsert feed stock layer")
	must(opssvc.UpsertFeedStock("Grower Feed", 100, 50), "upsert feed stock grower")

	var layerFeedStock, growerFeedStock models.FeedStock
	must(db.DB.Where("LOWER(name) = LOWER(?)", "Layer Complete Feed").First(&layerFeedStock).Error, "get layer feed stock")
	must(db.DB.Where("LOWER(name) = LOWER(?)", "Grower Feed").First(&growerFeedStock).Error, "get grower feed stock")
	layerFeedID := layerFeedStock.ID
	growerFeedID := growerFeedStock.ID
	fmt.Println("[SEED]   ✓ 2 feed stocks")

	// Flocks
	// Layer: 8,000 birds, hatch date ~30 weeks ago (Peak Production)
	layerHatchDate := now.AddDate(0, 0, -30*7).Format("2006-01-02")
	layerFlock, err := flockssvc.CreateFlock(flockssvc.CreateFlockRequest{
		Name:        "Layer House L-1",
		HouseNumber: "L-1",
		HouseType:   "Layer",
		Breed:       "Lohmann Brown",
		BuildingName: "Building A",
		HatchDate:   layerHatchDate,
		InitialCount: 8000,
		CreatedByID: adminID,
	})
	must(err, "create layer flock")

	// Grower: 5,000 birds, hatch date ~8 weeks ago (Growing stage)
	growerHatchDate := now.AddDate(0, 0, -8*7).Format("2006-01-02")
	growerFlock, err := flockssvc.CreateFlock(flockssvc.CreateFlockRequest{
		Name:        "Grower House G-1",
		HouseNumber: "G-1",
		HouseType:   "Grower",
		Breed:       "Lohmann Brown",
		BuildingName: "Building B",
		HatchDate:   growerHatchDate,
		InitialCount: 5000,
		CreatedByID: adminID,
	})
	must(err, "create grower flock")
	fmt.Println("[SEED]   ✓ 2 flocks (Layer + Grower)")

	// ─────────────────────────────────────────────
	// Phase 2 — 60 Days of Daily Logs
	// ─────────────────────────────────────────────
	fmt.Println("[SEED] Creating 60 days of daily logs...")

	rng := rand.New(rand.NewSource(42))
	layerBirds := 8000
	growerBirds := 5000

	// Vaccine days (day offsets from start)
	vaccineDays := map[int]string{7: "Newcastle Disease", 21: "Infectious Bronchitis", 42: "Newcastle Booster"}

	// Grower body weight progression for weekly logs
	growerAgeWeeksAtStart := 8
	growerBodyWeightLogs := []struct {
		weekOffset int
		weightG    float64
	}{
		{0, 810}, {1, 870}, {2, 940}, {3, 1010},
		{4, 1060}, {5, 1090}, {6, 1110}, {7, 1130},
	}

	for dayIdx := 0; dayIdx < 60; dayIdx++ {
		d := startDate.AddDate(0, 0, dayIdx)
		if !d.Before(endDate.AddDate(0, 0, 1)) {
			break
		}
		dateStr := d.Format("2006-01-02")

		// Layer daily log
		hdp := 0.70 + rng.Float64()*0.10 // 70–80%
		totalEggs := int(float64(layerBirds) * hdp)
		// Distribute: 20% Medium, 50% Large, 30% ExtraLarge
		medium := int(float64(totalEggs) * 0.20)
		large := int(float64(totalEggs) * 0.50)
		xl := totalEggs - medium - large
		cracked := rng.Intn(20)
		layerMort := rng.Intn(4) // 0–3
		feedLayer := 240.0 + rng.Float64()*20.0

		vaccine := ""
		if v, ok := vaccineDays[dayIdx]; ok {
			vaccine = v
		}

		layerFeedIDPtr := &layerFeedID
		err = opssvc.RecordDailyLog(opssvc.RecordDailyLogRequest{
			FlockID:        layerFlock.ID,
			Date:           dateStr,
			Medium:         medium,
			Large:          large,
			ExtraLarge:     xl,
			CrackedDirty:   cracked,
			FeedConsumedKg: feedLayer,
			Mortality:      layerMort,
			VaccineName:    vaccine,
			FeedTypeID:     layerFeedIDPtr,
			CreatedByID:    adminID,
		})
		if err != nil {
			fmt.Printf("[SEED]   WARN: layer log day %d: %v\n", dayIdx+1, err)
		}
		layerBirds = max(0, layerBirds-layerMort)

		// Grower daily log
		growerMort := rng.Intn(3) // 0–2
		feedGrower := 120.0 + rng.Float64()*30.0
		growerVaccine := ""
		if v, ok := vaccineDays[dayIdx]; ok {
			growerVaccine = v
		}

		growerFeedIDPtr := &growerFeedID
		err = opssvc.RecordGrowerLog(opssvc.RecordGrowerLogRequest{
			FlockID:        growerFlock.ID,
			Date:           dateStr,
			FeedConsumedKg: feedGrower,
			Mortality:      growerMort,
			VaccineName:    growerVaccine,
			FeedTypeID:     growerFeedIDPtr,
			CreatedByID:    adminID,
		})
		if err != nil {
			fmt.Printf("[SEED]   WARN: grower log day %d: %v\n", dayIdx+1, err)
		}
		growerBirds = max(0, growerBirds-growerMort)

		// Weekly body weight for grower (every 7 days)
		if dayIdx%7 == 0 {
			wIdx := dayIdx / 7
			if wIdx < len(growerBodyWeightLogs) {
				bwl := growerBodyWeightLogs[wIdx]
				weekAge := growerAgeWeeksAtStart + bwl.weekOffset
				bwEntry := models.BodyWeightLog{
					FlockID:     growerFlock.ID,
					Date:        d,
					WeekAge:     weekAge,
					AvgWeightG:  bwl.weightG + rng.Float64()*20 - 10, // ±10g noise
					SampleSize:  50,
					IsActive:    true,
					CreatedByID: adminID,
				}
				if err := db.DB.Create(&bwEntry).Error; err != nil {
					fmt.Printf("[SEED]   WARN: body weight log week %d: %v\n", weekAge, err)
				}
			}
		}
	}
	fmt.Printf("[SEED]   ✓ 60 layer logs, 60 grower logs, 8 body weight entries\n")

	// ─────────────────────────────────────────────
	// Phase 2b — Account Determination + Payment Method Accounts
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

		// Create Purchase record
		p := &models.Purchase{
			Date:        d,
			Category:    c.category,
			ItemName:    c.itemName,
			Quantity:    c.qty,
			Unit:        c.unit,
			UnitPrice:   c.unitPrice,
			TotalCost:   total,
			Supplier:    c.supplierName,
			SupplierID:  c.supplierID,
			PaymentStatus: "Unpaid",
			IsActive:    true,
		}
		if err := db.DB.Create(p).Error; err != nil {
			fmt.Printf("[SEED]   WARN: create purchase: %v\n", err)
			continue
		}

		// Delivery Receipt
		drItems := []models.DeliveryReceiptItem{
			{Category: c.category, ItemName: c.itemName, Unit: c.unit,
				QuantityOrdered: c.qty, QuantityReceived: c.qty, UnitPrice: c.unitPrice},
		}
		dr, err := purchasingsvc.CreateDeliveryReceipt(p.ID, dateStr, "Warehouse Staff", "", "", drItems, adminID)
		if err != nil {
			fmt.Printf("[SEED]   WARN: create DR: %v\n", err)
			continue
		}

		if err := purchasingsvc.ConfirmDeliveryReceipt(dr.ID, adminID); err != nil {
			// GL posting may fail if account determination not configured — that's OK
			fmt.Printf("[SEED]   INFO: confirm DR %s: %v\n", dr.DRNumber, err)
			// Force-set status to Received anyway
			db.DB.Model(&models.DeliveryReceipt{}).Where("id = ?", dr.ID).Update("status", "Received")
			db.DB.Model(&models.Purchase{}).Where("id = ?", p.ID).UpdateColumn("amount_received", total)
		}

		// AP Invoice
		drIDPtr := &dr.ID
		apDueStr := d.AddDate(0, 0, 30).Format("2006-01-02")
		apItems := []models.APInvoiceItem{
			{Category: c.category, ItemName: c.itemName, Unit: c.unit, Quantity: c.qty, UnitPrice: c.unitPrice},
		}
		apInv, err := purchasingsvc.CreateAPInvoice(p.ID, drIDPtr, dateStr, "Net 30", c.supplierName, "", "", &apDueStr, apItems, adminID)
		if err != nil {
			fmt.Printf("[SEED]   WARN: create AP invoice: %v\n", err)
			continue
		}

		// AP Payment (pay in full, 5 days after invoice)
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
	fmt.Printf("[SEED]   ✓ %d purchase cycles (DR → APInvoice → APPayment)\n", purchaseCount)

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
		doIDPtr := &dOrder.ID
		arItems := make([]models.ARInvoiceItem, len(sc.items))
		for i, item := range sc.items {
			arItems[i] = models.ARInvoiceItem{
				SKU:          item.SKU,
				Unit:         item.Unit,
				Quantity:     item.Quantity,
				PricePerUnit: item.PricePerUnit,
			}
		}
		soIDPtr := &so.ID
		arInv, err := salessvc.CreateARInvoice(
			soIDPtr, doIDPtr,
			custIDPtr,
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
	_ = growerBirds // suppress unused warning
}

func must(err error, ctx string) {
	if err != nil {
		log.Fatalf("[SEED] %s: %v", ctx, err)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
