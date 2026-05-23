// cmd/seed-demo/main.go — 2-month rice mill demo transactions.
//
// Prerequisites: run seed-master first (items, customers, suppliers must exist).
// Run with:  go run ./cmd/seed-demo/main.go
// Idempotent: guarded by "ricemill_demo_done" FarmSetting key.
// To re-run:  DELETE FROM farm_settings WHERE `key` = 'ricemill_demo_done';
package main

import (
	"fmt"
	"log"
	"time"

	appconfig "ricemill/app/config"
	"ricemill/app/db"
	"ricemill/app/models"
	accountingsvc "ricemill/app/services/accounting"
	invsvc "ricemill/app/services/inventory"
	productionsvc "ricemill/app/services/production"
	purchasingsvc "ricemill/app/services/purchasing"
	salessvc "ricemill/app/services/sales"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func must(err error, ctx string) {
	if err != nil {
		log.Fatalf("[SEED-DEMO] %s: %v", ctx, err)
	}
}

func warn(err error, ctx string) bool {
	if err != nil {
		fmt.Printf("[SEED-DEMO]   WARN %s: %v\n", ctx, err)
		return true
	}
	return false
}

// day returns the date string N days after startDate.
func day(start time.Time, offset int) string {
	return start.AddDate(0, 0, offset).Format("2006-01-02")
}

// itemByCode fetches an ItemMaster or fatals.
func itemByCode(code string) *models.ItemMaster {
	var item models.ItemMaster
	if err := db.DB.Where("item_code = ?", code).First(&item).Error; err != nil {
		log.Fatalf("[SEED-DEMO] item %q not found — run seed-master first: %v", code, err)
	}
	return &item
}

// customerByName fetches a Customer or fatals.
func customerByName(name string) *models.Customer {
	var c models.Customer
	if err := db.DB.Where("name LIKE ?", "%"+name+"%").First(&c).Error; err != nil {
		log.Fatalf("[SEED-DEMO] customer %q not found — run seed-master first: %v", name, err)
	}
	return &c
}

// supplierByName fetches a Supplier or fatals.
func supplierByName(name string) *models.Supplier {
	var s models.Supplier
	if err := db.DB.Where("name LIKE ?", "%"+name+"%").First(&s).Error; err != nil {
		log.Fatalf("[SEED-DEMO] supplier %q not found — run seed-master first: %v", name, err)
	}
	return &s
}

// glID fetches a GL account ID by code or fatals.
func glID(code string) uint {
	var acct models.GLAccount
	if err := db.DB.Where("`code` = ?", code).First(&acct).Error; err != nil {
		log.Fatalf("[SEED-DEMO] GL account %s not found: %v", code, err)
	}
	return acct.ID
}

// ─── main ────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("[SEED-DEMO] Connecting to database...")
	cfg, err := appconfig.Load(nil)
	must(err, "load config")
	must(db.Connect(cfg), "connect db")
	must(db.AutoMigrateAll(), "migrate")
	must(db.RunColumnMigrations(), "column migrations")
	must(db.SeedDefaults(), "seed defaults")

	// Idempotency guard
	var sentinel models.FarmSettings
	if err := db.DB.Where("`key` = ?", "ricemill_demo_done").First(&sentinel).Error; err == nil {
		fmt.Println("[SEED-DEMO] Already seeded. DELETE FROM farm_settings WHERE `key`='ricemill_demo_done' to re-run.")
		return
	}

	// Admin user for created_by references
	var admin models.User
	must(db.DB.First(&admin, "username = ?", "admin").Error, "find admin user")
	adminID := &admin.ID

	// Demo period: 2 months ending yesterday
	now := time.Now()
	start := now.AddDate(0, -2, 0).Truncate(24 * time.Hour)

	fmt.Printf("[SEED-DEMO] Period: %s → %s\n", start.Format("Jan 2, 2006"), now.Format("Jan 2, 2006"))

	// ── Resolve master records ────────────────────────────────────────────────
	fmt.Println("[SEED-DEMO] Resolving master data references...")

	// Items
	palNSIC222  := itemByCode("PAL-001") // Palay - NSIC Rc222
	palNSIC160  := itemByCode("PAL-002") // Palay - NSIC Rc160
	palDinorado := itemByCode("PAL-006") // Palay - Dinorado
	ricePremy   := itemByCode("RICE-001") // White Rice - Premium Grade
	riceWellM   := itemByCode("RICE-002") // White Rice - Well Milled
	riceRegM    := itemByCode("RICE-003") // White Rice - Regularly Milled
	riceDino    := itemByCode("RICE-005") // Dinorado Milled Rice
	bran        := itemByCode("BYP-001") // Rice Bran (Darak)
	hull        := itemByCode("BYP-002") // Rice Hull (Ipa)
	diesel      := itemByCode("FUEL-001") // Diesel Fuel
	sack50      := itemByCode("PACK-001") // PP Sack 50kg
	sack25      := itemByCode("PACK-002") // PP Sack 25kg
	vbelt       := itemByCode("PART-001") // V-Belt
	fumigant    := itemByCode("CHEM-002") // Phostoxin Fumigant Tablet

	// Customers
	custNFA     := customerByName("NFA - Cavite")
	custSM      := customerByName("SM Supermarket")
	custFeeds   := customerByName("Cavite Feeds")   // buys bran/hull
	custWalkin  := customerByName("Walk-in Buyer")

	// Suppliers
	supPalay    := supplierByName("RCPI Farmers Cooperative")
	supPalay2   := supplierByName("Tanza Palay Farmers")
	supFuel     := supplierByName("Petron Bulk Sales")
	supPack     := supplierByName("Camuning Packaging")

	fmt.Println("[SEED-DEMO]   ✓ Master data resolved")

	// ── Account Determination (upsert — always applied so demo re-runs stay correct) ──
	// These upserts run unconditionally to correct any stale or missing rules,
	// especially MILLING_OUTPUT which must point to Milled Rice (1-1330), not Palay.
	fmt.Println("[SEED-DEMO] Verifying account determination rules...")
	adRules := []struct{ module, event, code string }{
		// Purchasing
		{"PURCHASING", "GRNI",               "1-1400"}, // DR Confirm  — Cr GRNI
		{"PURCHASING", "INVENTORY_RECEIVED", "1-1310"}, // DR Confirm  — Dr Palay Inventory (catch-all)
		{"PURCHASING", "AP_PAYABLE",         "2-1100"}, // AP Invoice  — Cr Accounts Payable
		{"PURCHASING", "AP_CLEARING",        "2-1100"}, // AP Payment  — Dr Accounts Payable
		// Sales
		{"SALES", "AR_RECEIVABLE",     "1-1200"}, // AR Invoice  — Dr Accounts Receivable
		{"SALES", "SALES_REVENUE",     "4-1000"}, // AR Invoice  — Cr Rice Sales Revenue
		{"SALES", "COLLECTION_CLEARING", "1-1200"}, // Collection  — Cr Accounts Receivable
		// Production
		{"PRODUCTION", "MILLING_WIP",    "1-1320"}, // Milling Start    — Dr WIP
		{"PRODUCTION", "MILLING_INPUT",  "1-1310"}, // Milling Start    — Cr Palay Inventory
		{"PRODUCTION", "MILLING_OUTPUT", "1-1330"}, // Milling Complete — Dr Milled Rice (catch-all)
	}
	for _, r := range adRules {
		id := glID(r.code)
		warn(accountingsvc.UpsertAccountDetermination(r.module, r.event, nil, id, "Demo seed"), "AD "+r.event)
	}
	pmRules := []struct{ method, bank, dir, code string }{
		{"Cash",          "",    "BOTH", "1-1100"},
		{"Bank Transfer", "BDO", "BOTH", "1-1120"},
		{"GCash",         "",    "BOTH", "1-1110"},
		{"Check",         "",    "BOTH", "1-1120"},
	}
	for _, p := range pmRules {
		id := glID(p.code)
		warn(accountingsvc.UpsertPaymentMethodAccount(p.method, p.bank, id, p.dir), "PM "+p.method)
	}
	fmt.Println("[SEED-DEMO]   ✓ Account determination verified")

	// ═════════════════════════════════════════════════════════════════════════
	// PHASE 1 — STANDALONE GOODS RECEIPTS (initial + replenishment stock)
	// ═════════════════════════════════════════════════════════════════════════
	fmt.Println("[SEED-DEMO] Phase 1: Goods Receipts (initial palay stock)...")

	type grSpec struct {
		date    int
		remarks string
		lines   []invsvc.GRLineInput
	}
	grSpecs := []grSpec{
		{
			date:    0,
			remarks: "Opening stock — initial palay inventory",
			lines: []invsvc.GRLineInput{
				{ItemID: palNSIC222.ID, Quantity: 250, Price: palNSIC222.AvgPrice},
				{ItemID: palNSIC160.ID, Quantity: 180, Price: palNSIC160.AvgPrice},
			},
		},
		{
			date:    3,
			remarks: "Opening stock — diesel and supplies",
			lines: []invsvc.GRLineInput{
				{ItemID: diesel.ID, Quantity: 300, Price: diesel.AvgPrice},
				{ItemID: sack50.ID, Quantity: 1000, Price: sack50.AvgPrice},
				{ItemID: sack25.ID, Quantity: 500, Price: sack25.AvgPrice},
				{ItemID: fumigant.ID, Quantity: 200, Price: fumigant.AvgPrice},
			},
		},
		{
			date:    28,
			remarks: "Mid-period palay replenishment — RCPI delivery",
			lines: []invsvc.GRLineInput{
				{ItemID: palNSIC222.ID, Quantity: 300, Price: palNSIC222.AvgPrice},
				{ItemID: palDinorado.ID, Quantity: 120, Price: palDinorado.AvgPrice},
			},
		},
		{
			date:    32,
			remarks: "Diesel refill — operational needs",
			lines: []invsvc.GRLineInput{
				{ItemID: diesel.ID, Quantity: 400, Price: diesel.AvgPrice},
				{ItemID: vbelt.ID, Quantity: 5, Price: vbelt.AvgPrice},
			},
		},
	}
	grCount := 0
	for _, spec := range grSpecs {
		req := invsvc.CreateGoodsReceiptRequest{
			PostingDate: day(start, spec.date),
			Remarks:     spec.remarks,
			Lines:       spec.lines,
		}
		_, err := invsvc.CreateGoodsReceipt(req, adminID)
		if !warn(err, "create GR day "+fmt.Sprint(spec.date)) {
			grCount++
		}
	}
	fmt.Printf("[SEED-DEMO]   ✓ %d goods receipts\n", grCount)

	// ═════════════════════════════════════════════════════════════════════════
	// PHASE 2 — PURCHASE ORDERS → DR → AP INVOICE → AP PAYMENT
	// ═════════════════════════════════════════════════════════════════════════
	fmt.Println("[SEED-DEMO] Phase 2: Purchase cycles...")

	type poCycle struct {
		daysOffset   int
		supplier     *models.Supplier
		category     string
		itemName     string
		unit         string
		qty          float64
		unitPrice    float64
		payDaysAfter int // days after PO to pay
	}
	poCycles := []poCycle{
		// Month 1 — palay from farmers
		{5, supPalay, "Palay", "Palay - NSIC Rc222", "sack", 100, palNSIC222.AvgPrice, 3},
		{8, supPalay2, "Palay", "Palay - NSIC Rc160", "sack", 80, palNSIC160.AvgPrice, 5},
		// Month 1 — fuel & supplies
		{10, supFuel, "Fuel & Lubricants", "Diesel Fuel", "liter", 500, diesel.AvgPrice, 2},
		{14, supPack, "Packaging", "PP Sack 50kg (Plain)", "pc", 2000, sack50.AvgPrice, 7},
		{20, supPalay, "Palay", "Palay - NSIC Rc222", "sack", 150, palNSIC222.AvgPrice, 3},
		// Month 2 — palay restocking
		{33, supPalay, "Palay", "Palay - NSIC Rc222", "sack", 200, palNSIC222.AvgPrice, 3},
		{36, supPalay2, "Palay", "Palay - NSIC Rc160", "sack", 100, palNSIC160.AvgPrice, 5},
		// Month 2 — fuel & packaging replenishment
		{40, supFuel, "Fuel & Lubricants", "Diesel Fuel", "liter", 600, diesel.AvgPrice, 2},
		{45, supPack, "Packaging", "PP Sack 50kg (Plain)", "pc", 3000, sack50.AvgPrice, 7},
		{50, supPalay, "Palay", "Palay - Dinorado", "sack", 80, palDinorado.AvgPrice, 3},
	}

	poCount := 0
	for _, c := range poCycles {
		d := start.AddDate(0, 0, c.daysOffset)
		dateStr := d.Format("2006-01-02")
		total := c.qty * c.unitPrice

		// Create PurchaseHeader (SAP B1-aligned OPOR)
		ph, err := purchasingsvc.CreatePurchaseHeader(purchasingsvc.CreatePurchaseParams{
			Date:         dateStr,
			SupplierID:   &c.supplier.ID,
			SupplierName: c.supplier.Name,
			Lines: []purchasingsvc.POLineInput{
				{Category: c.category, ItemName: c.itemName, Unit: c.unit, Quantity: c.qty, UnitPrice: c.unitPrice},
			},
		}, adminID)
		if warn(err, "create purchase header day "+fmt.Sprint(c.daysOffset)) {
			continue
		}

		// Delivery Receipt (OPDN/PDN1)
		drItems := []models.DeliveryReceiptItem{
			{Category: c.category, Description: c.itemName, Unit: c.unit,
				QuantityOrdered: c.qty, Quantity: c.qty, Price: c.unitPrice},
		}
		dr, err := purchasingsvc.CreateDeliveryReceipt(ph.ID, dateStr, "Warehouse Staff", "", "", drItems, adminID)
		if warn(err, "create DR") {
			continue
		}
		if warn(purchasingsvc.ConfirmDeliveryReceipt(dr.ID, adminID), "confirm DR") {
			continue
		}

		// AP Invoice (OPCH/PCH1)
		drIDPtr := &dr.ID
		dueStr := d.AddDate(0, 0, 30).Format("2006-01-02")
		apItems := []models.APInvoiceItem{
			{Category: c.category, Description: c.itemName, Unit: c.unit, Quantity: c.qty, Price: c.unitPrice},
		}
		apInv, err := purchasingsvc.CreateAPInvoice(ph.ID, drIDPtr, dateStr, "Net 30", c.supplier.Name, "", "", &dueStr, apItems, adminID, 0, 0, 0, "")
		if warn(err, "create AP invoice") {
			continue
		}

		// AP Payment (OVPM)
		payDate := d.AddDate(0, 0, c.payDaysAfter).Format("2006-01-02")
		_, err = purchasingsvc.CreateAPPayment(payDate, &c.supplier.ID, c.supplier.Name, "Cash", "", "",
			[]purchasingsvc.APPaymentLineInput{{APInvoiceID: apInv.ID, AmountApplied: total}}, adminID)
		warn(err, "create AP payment")

		poCount++
	}
	fmt.Printf("[SEED-DEMO]   ✓ %d purchase cycles (PO → DR → AP Invoice → Payment)\n", poCount)

	// ═════════════════════════════════════════════════════════════════════════
	// PHASE 3 — STANDALONE GOODS ISSUES (operational supplies)
	// ═════════════════════════════════════════════════════════════════════════
	fmt.Println("[SEED-DEMO] Phase 3: Goods Issues (operational use)...")

	type giSpec struct {
		date    int
		remarks string
		lines   []invsvc.GILineInput
	}
	giSpecs := []giSpec{
		{6, "Diesel consumed — generator and dryer operations (Week 1)",
			[]invsvc.GILineInput{{ItemID: diesel.ID, Quantity: 80, Price: diesel.AvgPrice}}},
		{13, "Fumigant application — Warehouse 1 storage treatment",
			[]invsvc.GILineInput{{ItemID: fumigant.ID, Quantity: 50, Price: fumigant.AvgPrice}}},
		{19, "Diesel consumed — Week 3 operations",
			[]invsvc.GILineInput{{ItemID: diesel.ID, Quantity: 100, Price: diesel.AvgPrice}}},
		{27, "V-Belt replacement — Huller no. 2",
			[]invsvc.GILineInput{{ItemID: vbelt.ID, Quantity: 2, Price: vbelt.AvgPrice}}},
		{34, "Diesel consumed — Week 5 operations",
			[]invsvc.GILineInput{{ItemID: diesel.ID, Quantity: 120, Price: diesel.AvgPrice}}},
		{42, "Fumigant application — monthly grain protection",
			[]invsvc.GILineInput{{ItemID: fumigant.ID, Quantity: 80, Price: fumigant.AvgPrice}}},
		{50, "Diesel consumed — Week 8 operations",
			[]invsvc.GILineInput{{ItemID: diesel.ID, Quantity: 100, Price: diesel.AvgPrice}}},
		{55, "Packaging issued to milling floor",
			[]invsvc.GILineInput{
				{ItemID: sack50.ID, Quantity: 500, Price: sack50.AvgPrice},
				{ItemID: sack25.ID, Quantity: 200, Price: sack25.AvgPrice},
			}},
	}
	giCount := 0
	for _, spec := range giSpecs {
		req := invsvc.CreateGoodsIssueRequest{
			PostingDate: day(start, spec.date),
			Remarks:     spec.remarks,
			Lines:       spec.lines,
		}
		_, err := invsvc.CreateGoodsIssue(req, adminID)
		if !warn(err, "create GI day "+fmt.Sprint(spec.date)) {
			giCount++
		}
	}
	fmt.Printf("[SEED-DEMO]   ✓ %d goods issues\n", giCount)

	// ═════════════════════════════════════════════════════════════════════════
	// PHASE 4 — MILLING ORDERS (Create → Start → Complete)
	// ═════════════════════════════════════════════════════════════════════════
	fmt.Println("[SEED-DEMO] Phase 4: Milling orders...")

	type millingSpec struct {
		postDay    int  // posting date offset
		expectDay  int  // expected completion offset
		inputItem  *models.ItemMaster
		inputQty   float64
		completionNote string
		outputs    []productionsvc.CompleteMillingLineInput
	}

	millingSpecs := []millingSpec{
		{
			postDay: 7, expectDay: 9,
			inputItem: palNSIC222, inputQty: 100,
			completionNote: "Batch 1 — NSIC Rc222 standard milling",
			outputs: []productionsvc.CompleteMillingLineInput{
				{OutputItemID: riceWellM.ID, ActualQty: 62},
				{OutputItemID: bran.ID, ActualQty: 12},
				{OutputItemID: hull.ID, ActualQty: 22},
			},
		},
		{
			postDay: 15, expectDay: 17,
			inputItem: palNSIC160, inputQty: 80,
			completionNote: "Batch 2 — NSIC Rc160 premium milling",
			outputs: []productionsvc.CompleteMillingLineInput{
				{OutputItemID: ricePremy.ID, ActualQty: 52},
				{OutputItemID: bran.ID, ActualQty: 10},
				{OutputItemID: hull.ID, ActualQty: 16},
			},
		},
		{
			postDay: 22, expectDay: 24,
			inputItem: palNSIC222, inputQty: 150,
			completionNote: "Batch 3 — large run for NFA delivery",
			outputs: []productionsvc.CompleteMillingLineInput{
				{OutputItemID: riceRegM.ID, ActualQty: 88},
				{OutputItemID: riceWellM.ID, ActualQty: 25},
				{OutputItemID: bran.ID, ActualQty: 18},
				{OutputItemID: hull.ID, ActualQty: 17},
			},
		},
		{
			postDay: 30, expectDay: 32,
			inputItem: palDinorado, inputQty: 60,
			completionNote: "Batch 4 — Dinorado specialty milling",
			outputs: []productionsvc.CompleteMillingLineInput{
				{OutputItemID: riceDino.ID, ActualQty: 38},
				{OutputItemID: bran.ID, ActualQty: 8},
				{OutputItemID: hull.ID, ActualQty: 12},
			},
		},
		{
			postDay: 38, expectDay: 40,
			inputItem: palNSIC222, inputQty: 200,
			completionNote: "Batch 5 — Month 2 main production run",
			outputs: []productionsvc.CompleteMillingLineInput{
				{OutputItemID: riceRegM.ID, ActualQty: 118},
				{OutputItemID: riceWellM.ID, ActualQty: 30},
				{OutputItemID: bran.ID, ActualQty: 24},
				{OutputItemID: hull.ID, ActualQty: 25},
			},
		},
		{
			postDay: 46, expectDay: 48,
			inputItem: palNSIC160, inputQty: 100,
			completionNote: "Batch 6 — premium grade production",
			outputs: []productionsvc.CompleteMillingLineInput{
				{OutputItemID: ricePremy.ID, ActualQty: 64},
				{OutputItemID: riceWellM.ID, ActualQty: 10},
				{OutputItemID: bran.ID, ActualQty: 14},
				{OutputItemID: hull.ID, ActualQty: 10},
			},
		},
		{
			postDay: 53, expectDay: 55,
			inputItem: palNSIC222, inputQty: 120,
			completionNote: "Batch 7 — end-of-month production",
			outputs: []productionsvc.CompleteMillingLineInput{
				{OutputItemID: riceWellM.ID, ActualQty: 72},
				{OutputItemID: bran.ID, ActualQty: 15},
				{OutputItemID: hull.ID, ActualQty: 30},
			},
		},
	}

	moCount := 0
	for _, spec := range millingSpecs {
		// Create (Draft)
		mo, err := productionsvc.CreateMillingOrder(productionsvc.CreateMillingOrderRequest{
			PostingDate:  day(start, spec.postDay),
			ExpectedDate: day(start, spec.expectDay),
			InputItemID:  spec.inputItem.ID,
			InputQty:     spec.inputQty,
			Remarks:      spec.completionNote,
		}, adminID)
		if warn(err, fmt.Sprintf("create MO day %d", spec.postDay)) {
			continue
		}

		// Start (creates GI internally)
		if err := productionsvc.StartMillingOrder(mo.ID, adminID); err != nil {
			warn(err, "start MO "+mo.MoNumber)
			continue
		}

		// Complete (creates GR internally)
		completeReq := productionsvc.CompleteMillingRequest{
			Lines:          spec.outputs,
			CompletionNote: spec.completionNote,
		}
		if _, err := productionsvc.CompleteMillingOrder(mo.ID, completeReq, adminID); err != nil {
			warn(err, "complete MO "+mo.MoNumber)
			continue
		}

		moCount++
	}
	fmt.Printf("[SEED-DEMO]   ✓ %d milling orders (Draft → In Progress → Completed)\n", moCount)

	// ═════════════════════════════════════════════════════════════════════════
	// PHASE 5 — SALES ORDERS → DO → AR INVOICE → COLLECTION
	// ═════════════════════════════════════════════════════════════════════════
	fmt.Println("[SEED-DEMO] Phase 5: Sales cycles...")

	type saleLine struct {
		sku   string
		unit  string
		qty   float64
		price float64
	}
	type saleSpec struct {
		daysOffset  int
		customer    *models.Customer
		payMethod   string
		terms       string
		collectFull bool
		lines       []saleLine
	}

	saleSpecs := []saleSpec{
		// NFA — large bulk orders, account payment, Net 30
		{11, custNFA, "Bank Transfer", "Net 30", true, []saleLine{
			{"White Rice - Well Milled", "sack", 40, riceWellM.AvgPrice},
			{"White Rice - Regularly Milled", "sack", 30, riceRegM.AvgPrice},
		}},
		{25, custNFA, "Bank Transfer", "Net 30", true, []saleLine{
			{"White Rice - Well Milled", "sack", 50, riceWellM.AvgPrice},
			{"White Rice - Regularly Milled", "sack", 50, riceRegM.AvgPrice},
		}},
		{42, custNFA, "Bank Transfer", "Net 30", false, []saleLine{ // partial — still outstanding
			{"White Rice - Well Milled", "sack", 60, riceWellM.AvgPrice},
			{"White Rice - Regularly Milled", "sack", 40, riceRegM.AvgPrice},
		}},

		// SM Supermarket — premium grade, Net 15
		{13, custSM, "Bank Transfer", "Net 15", true, []saleLine{
			{"White Rice - Premium Grade", "sack", 20, ricePremy.AvgPrice},
			{"Dinorado Milled Rice", "sack", 10, riceDino.AvgPrice},
		}},
		{31, custSM, "Bank Transfer", "Net 15", true, []saleLine{
			{"White Rice - Premium Grade", "sack", 25, ricePremy.AvgPrice},
			{"Dinorado Milled Rice", "sack", 15, riceDino.AvgPrice},
		}},
		{48, custSM, "Bank Transfer", "Net 15", true, []saleLine{
			{"White Rice - Premium Grade", "sack", 30, ricePremy.AvgPrice},
		}},

		// Cavite Feeds — by-products (bran, hull)
		{16, custFeeds, "Cash", "COD", true, []saleLine{
			{"Rice Bran (Darak)", "sack", 20, bran.AvgPrice},
			{"Rice Hull (Ipa)", "sack", 30, hull.AvgPrice},
		}},
		{33, custFeeds, "Cash", "COD", true, []saleLine{
			{"Rice Bran (Darak)", "sack", 25, bran.AvgPrice},
			{"Rice Hull (Ipa)", "sack", 40, hull.AvgPrice},
		}},
		{50, custFeeds, "Cash", "COD", true, []saleLine{
			{"Rice Bran (Darak)", "sack", 30, bran.AvgPrice},
			{"Rice Hull (Ipa)", "sack", 50, hull.AvgPrice},
		}},

		// Walk-in buyers — small cash sales throughout
		{9,  custWalkin, "Cash", "COD", true, []saleLine{{"White Rice - Regularly Milled", "sack", 5, riceRegM.AvgPrice}}},
		{17, custWalkin, "Cash", "COD", true, []saleLine{{"White Rice - Well Milled", "sack", 8, riceWellM.AvgPrice}}},
		{23, custWalkin, "GCash", "COD", true, []saleLine{{"White Rice - Premium Grade", "sack", 3, ricePremy.AvgPrice}}},
		{29, custWalkin, "Cash", "COD", true, []saleLine{{"White Rice - Regularly Milled", "sack", 10, riceRegM.AvgPrice}}},
		{37, custWalkin, "Cash", "COD", true, []saleLine{{"Rice Bran (Darak)", "sack", 5, bran.AvgPrice}}},
		{44, custWalkin, "GCash", "COD", true, []saleLine{{"White Rice - Well Milled", "sack", 6, riceWellM.AvgPrice}}},
		{52, custWalkin, "Cash", "COD", true, []saleLine{{"White Rice - Regularly Milled", "sack", 8, riceRegM.AvgPrice}}},
		{57, custWalkin, "Cash", "COD", false, []saleLine{{"White Rice - Premium Grade", "sack", 4, ricePremy.AvgPrice}}},
	}

	soCount := 0
	for _, sc := range saleSpecs {
		d := start.AddDate(0, 0, sc.daysOffset)
		dateStr := d.Format("2006-01-02")
		dueStr := d.AddDate(0, 0, 30).Format("2006-01-02")

		// Build SO items
		soItems := make([]models.SalesOrderItem, 0, len(sc.lines))
		grandTotal := 0.0
		for _, l := range sc.lines {
			lt := l.qty * l.price
			grandTotal += lt
			soItems = append(soItems, models.SalesOrderItem{
				SKU: l.sku, Unit: l.unit,
				Quantity: l.qty, PricePerUnit: l.price, LineTotal: lt,
			})
		}

		custIDPtr := &sc.customer.ID
		so, err := salessvc.CreateSalesOrder(
			dateStr, custIDPtr,
			sc.customer.Name, sc.customer.Address, sc.customer.ContactNumber,
			sc.payMethod, sc.terms, &dueStr, "",
			soItems, adminID,
		)
		if warn(err, "create SO day "+fmt.Sprint(sc.daysOffset)) {
			continue
		}

		// Delivery Order
		doItems := make([]models.DeliveryOrderItem, len(sc.lines))
		for i, l := range sc.lines {
			doItems[i] = models.DeliveryOrderItem{
				SKU: l.sku, Unit: l.unit,
				QuantityOrdered: l.qty, QuantityDelivered: l.qty, PricePerUnit: l.price,
			}
		}
		dOrder, err := salessvc.CreateDeliveryOrder(so.ID, dateStr, "Driver", "", doItems, adminID)
		if warn(err, "create DO") {
			continue
		}
		if warn(salessvc.ConfirmDeliveryOrder(dOrder.ID, adminID), "confirm DO") {
			continue
		}

		// AR Invoice
		arItems := make([]models.ARInvoiceItem, len(sc.lines))
		for i, l := range sc.lines {
			arItems[i] = models.ARInvoiceItem{SKU: l.sku, Unit: l.unit, Quantity: l.qty, PricePerUnit: l.price}
		}
		arInv, err := salessvc.CreateARInvoice(
			[]uint{dOrder.ID}, custIDPtr,
			sc.customer.Name, sc.customer.Address, sc.customer.ContactNumber,
			dateStr, sc.terms, "", &dueStr,
			arItems, adminID,
		)
		if warn(err, "create AR invoice") {
			continue
		}

		// Collection
		collectAmt := grandTotal
		if !sc.collectFull {
			collectAmt = grandTotal * 0.5
		}
		collectDate := d.AddDate(0, 0, 1).Format("2006-01-02")
		_, err = salessvc.CreateCollection(collectDate, custIDPtr, sc.customer.Name,
			sc.payMethod, "", "",
			[]salessvc.CollectionLineInput{{ARInvoiceID: arInv.ID, AmountApplied: collectAmt}},
			adminID,
		)
		warn(err, "create collection")
		soCount++
	}
	fmt.Printf("[SEED-DEMO]   ✓ %d sales cycles (SO → DO → AR Invoice → Collection)\n", soCount)

	// ─── Done ────────────────────────────────────────────────────────────────
	db.DB.Save(&models.FarmSettings{Key: "ricemill_demo_done", Value: now.Format(time.RFC3339)})
	fmt.Println("[SEED-DEMO] ✅ Done. 2-month demo data loaded.")
	fmt.Println("[SEED-DEMO]    Modules covered: GR, GI, Purchase/PO, Milling, Sales (SO/DO/AR/Collections)")
}
