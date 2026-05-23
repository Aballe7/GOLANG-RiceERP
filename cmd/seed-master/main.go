// cmd/seed-master/main.go — Rice mill master data seeder.
// Inserts UoM masters/groups, ~50 item masters, 20 customers, and 20 suppliers.
// Run with: go run ./cmd/seed-master/main.go
// Safe to run multiple times; guarded by sentinel FarmSetting keys.
package main

import (
	"fmt"
	"log"

	appconfig "ricemill/app/config"
	"ricemill/app/db"
	"ricemill/app/models"
	invsvc "ricemill/app/services/inventory"
	purchasingsvc "ricemill/app/services/purchasing"
	salessvc "ricemill/app/services/sales"
)

func main() {
	fmt.Println("[SEED-MASTER] Connecting to database...")
	cfg, err := appconfig.Load(nil)
	must(err, "load config")
	must(db.Connect(cfg), "connect db")
	must(db.AutoMigrateAll(), "migrate")
	must(db.RunColumnMigrations(), "column migrations")
	must(db.SeedDefaults(), "seed defaults")

	// ─────────────────────────────────────────────
	// Item Categories (guarded separately)
	// ─────────────────────────────────────────────
	var catSentinel models.FarmSettings
	if err := db.DB.Where("`key` = ?", "ricemill_categories_done").First(&catSentinel).Error; err != nil {
		fmt.Println("[SEED-MASTER] Creating item categories...")
		seedCategories()
		db.DB.Save(&models.FarmSettings{Key: "ricemill_categories_done", Value: "1"})
		fmt.Println("[SEED-MASTER]   ✓ Category seed done")
	} else {
		fmt.Println("[SEED-MASTER] Categories already seeded — skipping.")
	}

	// ─────────────────────────────────────────────
	// UoM Masters & Groups (guarded separately so they can be re-run)
	// ─────────────────────────────────────────────
	var uomSentinel models.FarmSettings
	if err := db.DB.Where("`key` = ?", "ricemill_uom_done").First(&uomSentinel).Error; err != nil {
		fmt.Println("[SEED-MASTER] Creating UoM masters and groups...")
		seedUoMs()
		db.DB.Save(&models.FarmSettings{Key: "ricemill_uom_done", Value: "1"})
		fmt.Println("[SEED-MASTER]   ✓ UoM seed done")
	} else {
		fmt.Println("[SEED-MASTER] UoM already seeded — skipping.")
	}

	// Idempotency guard for items/customers/suppliers
	var sentinel models.FarmSettings
	if err := db.DB.Where("`key` = ?", "ricemill_master_done").First(&sentinel).Error; err == nil {
		fmt.Println("[SEED-MASTER] Already seeded. Delete the 'ricemill_master_done' farm_settings row to re-run.")
		return
	}

	// ─────────────────────────────────────────────
	// Item Masters — 50 items
	// ─────────────────────────────────────────────
	fmt.Println("[SEED-MASTER] Creating item masters...")

	// Helper: look up ItmsGrpCod by name (categories must already be seeded)
	catID := func(name string) int {
		var cat models.OITB
		db.DB.Where("itms_grp_nam = ?", name).First(&cat)
		return cat.ItmsGrpCod
	}

	items := []models.OITM{
		// ── Palay (Raw Paddy) — production input ──────────────────────────
		{ItemCode: "PAL-001", ItemName: "Palay - NSIC Rc222", ItmsGrpCod: catID("Palay"), InvntryUom: "sack", AvgPrice: 1200, MinLevel: 50, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "PAL-002", ItemName: "Palay - NSIC Rc160", ItmsGrpCod: catID("Palay"), InvntryUom: "sack", AvgPrice: 1100, MinLevel: 50, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "PAL-003", ItemName: "Palay - NSIC Rc238", ItmsGrpCod: catID("Palay"), InvntryUom: "sack", AvgPrice: 1150, MinLevel: 50, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "PAL-004", ItemName: "Palay - IR64", ItmsGrpCod: catID("Palay"), InvntryUom: "sack", AvgPrice: 1050, MinLevel: 50, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "PAL-005", ItemName: "Palay - Mestizo Hybrid", ItmsGrpCod: catID("Palay"), InvntryUom: "sack", AvgPrice: 1300, MinLevel: 30, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "PAL-006", ItemName: "Palay - Dinorado", ItmsGrpCod: catID("Palay"), InvntryUom: "sack", AvgPrice: 1400, MinLevel: 20, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "PAL-007", ItemName: "Palay - Sinandomeng", ItmsGrpCod: catID("Palay"), InvntryUom: "sack", AvgPrice: 1350, MinLevel: 20, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "PAL-008", ItemName: "Palay - Red Rice", ItmsGrpCod: catID("Palay"), InvntryUom: "sack", AvgPrice: 1100, MinLevel: 20, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "PAL-009", ItemName: "Palay - Malagkit (Glutinous)", ItmsGrpCod: catID("Palay"), InvntryUom: "sack", AvgPrice: 1250, MinLevel: 20, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "PAL-010", ItemName: "Palay - NSIC Rc480", ItmsGrpCod: catID("Palay"), InvntryUom: "sack", AvgPrice: 1180, MinLevel: 50, ValidFor: "Y", MakItem: "Y"},

		// ── Milled Rice — production output ───────────────────────────────
		{ItemCode: "RICE-001", ItemName: "White Rice - Premium Grade", ItmsGrpCod: catID("Milled Rice"), InvntryUom: "sack", AvgPrice: 2600, MinLevel: 30, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "RICE-002", ItemName: "White Rice - Well Milled", ItmsGrpCod: catID("Milled Rice"), InvntryUom: "sack", AvgPrice: 2400, MinLevel: 30, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "RICE-003", ItemName: "White Rice - Regularly Milled", ItmsGrpCod: catID("Milled Rice"), InvntryUom: "sack", AvgPrice: 2200, MinLevel: 50, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "RICE-004", ItemName: "Sinandomeng Milled Rice", ItmsGrpCod: catID("Milled Rice"), InvntryUom: "sack", AvgPrice: 3200, MinLevel: 20, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "RICE-005", ItemName: "Dinorado Milled Rice", ItmsGrpCod: catID("Milled Rice"), InvntryUom: "sack", AvgPrice: 3000, MinLevel: 20, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "RICE-006", ItemName: "Red Rice Milled", ItmsGrpCod: catID("Milled Rice"), InvntryUom: "sack", AvgPrice: 2800, MinLevel: 10, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "RICE-007", ItemName: "Malagkit (Glutinous Rice)", ItmsGrpCod: catID("Milled Rice"), InvntryUom: "sack", AvgPrice: 2900, MinLevel: 10, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "RICE-008", ItemName: "Broken Rice (Butil)", ItmsGrpCod: catID("Milled Rice"), InvntryUom: "sack", AvgPrice: 1400, MinLevel: 20, ValidFor: "Y", MakItem: "Y"},

		// ── By-Products — production output ───────────────────────────────
		{ItemCode: "BYP-001", ItemName: "Rice Bran (Darak)", ItmsGrpCod: catID("By-Products"), InvntryUom: "sack", AvgPrice: 450, MinLevel: 20, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "BYP-002", ItemName: "Rice Hull (Ipa)", ItmsGrpCod: catID("By-Products"), InvntryUom: "sack", AvgPrice: 180, MinLevel: 20, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "BYP-003", ItemName: "Rice Polish", ItmsGrpCod: catID("By-Products"), InvntryUom: "sack", AvgPrice: 550, MinLevel: 10, ValidFor: "Y", MakItem: "Y"},
		{ItemCode: "BYP-004", ItemName: "Broken Rice - Pinawa", ItmsGrpCod: catID("By-Products"), InvntryUom: "sack", AvgPrice: 900, MinLevel: 10, ValidFor: "Y", MakItem: "Y"},

		// ── Fuel & Lubricants — not production items ───────────────────────
		{ItemCode: "FUEL-001", ItemName: "Diesel Fuel", ItmsGrpCod: catID("Fuel & Lubricants"), InvntryUom: "liter", AvgPrice: 65, MinLevel: 200, ValidFor: "Y"},
		{ItemCode: "FUEL-002", ItemName: "Gasoline (RON 95)", ItmsGrpCod: catID("Fuel & Lubricants"), InvntryUom: "liter", AvgPrice: 72, MinLevel: 100, ValidFor: "Y"},
		{ItemCode: "FUEL-003", ItemName: "Hydraulic Oil", ItmsGrpCod: catID("Fuel & Lubricants"), InvntryUom: "liter", AvgPrice: 280, MinLevel: 10, ValidFor: "Y"},
		{ItemCode: "FUEL-004", ItemName: "Engine Oil SAE 40", ItmsGrpCod: catID("Fuel & Lubricants"), InvntryUom: "liter", AvgPrice: 320, MinLevel: 10, ValidFor: "Y"},
		{ItemCode: "FUEL-005", ItemName: "Gear Oil", ItmsGrpCod: catID("Fuel & Lubricants"), InvntryUom: "liter", AvgPrice: 250, MinLevel: 5, ValidFor: "Y"},
		{ItemCode: "FUEL-006", ItemName: "Lithium Grease", ItmsGrpCod: catID("Fuel & Lubricants"), InvntryUom: "kg", AvgPrice: 380, MinLevel: 5, ValidFor: "Y"},
		{ItemCode: "FUEL-007", ItemName: "Penetrating Oil (WD-40)", ItmsGrpCod: catID("Fuel & Lubricants"), InvntryUom: "can", AvgPrice: 250, MinLevel: 3, ValidFor: "Y"},

		// ── Packaging — not production items ──────────────────────────────
		{ItemCode: "PACK-001", ItemName: "PP Sack 50kg (Plain)", ItmsGrpCod: catID("Packaging Materials"), InvntryUom: "pc", AvgPrice: 18, MinLevel: 500, ValidFor: "Y"},
		{ItemCode: "PACK-002", ItemName: "PP Sack 25kg (Plain)", ItmsGrpCod: catID("Packaging Materials"), InvntryUom: "pc", AvgPrice: 12, MinLevel: 300, ValidFor: "Y"},
		{ItemCode: "PACK-003", ItemName: "Woven Bag 50kg (Branded)", ItmsGrpCod: catID("Packaging Materials"), InvntryUom: "pc", AvgPrice: 22, MinLevel: 200, ValidFor: "Y"},
		{ItemCode: "PACK-004", ItemName: "Plastic Liner (Inner Bag)", ItmsGrpCod: catID("Packaging Materials"), InvntryUom: "pc", AvgPrice: 5, MinLevel: 500, ValidFor: "Y"},
		{ItemCode: "PACK-005", ItemName: "Nylon Twine", ItmsGrpCod: catID("Packaging Materials"), InvntryUom: "roll", AvgPrice: 95, MinLevel: 10, ValidFor: "Y"},
		{ItemCode: "PACK-006", ItemName: "Packing Tape", ItmsGrpCod: catID("Packaging Materials"), InvntryUom: "roll", AvgPrice: 35, MinLevel: 20, ValidFor: "Y"},
		{ItemCode: "PACK-007", ItemName: "Printed Label Sticker", ItmsGrpCod: catID("Packaging Materials"), InvntryUom: "pc", AvgPrice: 2, MinLevel: 1000, ValidFor: "Y"},

		// ── Spare Parts — not production items ────────────────────────────
		{ItemCode: "PART-001", ItemName: "V-Belt (B-Section)", ItmsGrpCod: catID("Spare Parts"), InvntryUom: "pc", AvgPrice: 350, MinLevel: 5, ValidFor: "Y"},
		{ItemCode: "PART-002", ItemName: "Ball Bearing 6205", ItmsGrpCod: catID("Spare Parts"), InvntryUom: "pc", AvgPrice: 280, MinLevel: 4, ValidFor: "Y"},
		{ItemCode: "PART-003", ItemName: "Ball Bearing 6206", ItmsGrpCod: catID("Spare Parts"), InvntryUom: "pc", AvgPrice: 310, MinLevel: 4, ValidFor: "Y"},
		{ItemCode: "PART-004", ItemName: "Rubber Roll (Huller)", ItmsGrpCod: catID("Spare Parts"), InvntryUom: "pc", AvgPrice: 1800, MinLevel: 2, ValidFor: "Y"},
		{ItemCode: "PART-005", ItemName: "Steel Screen / Sieve", ItmsGrpCod: catID("Spare Parts"), InvntryUom: "pc", AvgPrice: 950, MinLevel: 2, ValidFor: "Y"},
		{ItemCode: "PART-006", ItemName: "Drive Chain (80H)", ItmsGrpCod: catID("Spare Parts"), InvntryUom: "meter", AvgPrice: 280, MinLevel: 5, ValidFor: "Y"},

		// ── Chemicals & Pest Control — not production items ────────────────
		{ItemCode: "CHEM-001", ItemName: "Rodenticide (Rat Bait)", ItmsGrpCod: catID("Chemicals"), InvntryUom: "kg", AvgPrice: 850, MinLevel: 2, ValidFor: "Y"},
		{ItemCode: "CHEM-002", ItemName: "Phostoxin Fumigant Tablet", ItmsGrpCod: catID("Chemicals"), InvntryUom: "tablet", AvgPrice: 12, MinLevel: 100, ValidFor: "Y"},
		{ItemCode: "CHEM-003", ItemName: "Insecticide Spray", ItmsGrpCod: catID("Chemicals"), InvntryUom: "can", AvgPrice: 380, MinLevel: 5, ValidFor: "Y"},
		{ItemCode: "CHEM-004", ItemName: "Herbicide (Roundup 1L)", ItmsGrpCod: catID("Chemicals"), InvntryUom: "bottle", AvgPrice: 450, MinLevel: 3, ValidFor: "Y"},

		// ── Miscellaneous — not production items ───────────────────────────
		{ItemCode: "MISC-001", ItemName: "Tarpaulin (10x20ft)", ItmsGrpCod: catID("Miscellaneous"), InvntryUom: "pc", AvgPrice: 850, MinLevel: 2, ValidFor: "Y"},
		{ItemCode: "MISC-002", ItemName: "Wooden Pallet", ItmsGrpCod: catID("Miscellaneous"), InvntryUom: "pc", AvgPrice: 650, MinLevel: 10, ValidFor: "Y"},
		{ItemCode: "MISC-003", ItemName: "Broom (Walis Tambo)", ItmsGrpCod: catID("Miscellaneous"), InvntryUom: "pc", AvgPrice: 85, MinLevel: 5, ValidFor: "Y"},
		{ItemCode: "MISC-004", ItemName: "Safety Gloves (Pair)", ItmsGrpCod: catID("Miscellaneous"), InvntryUom: "pair", AvgPrice: 45, MinLevel: 10, ValidFor: "Y"},
	}

	for i, item := range items {
		item := item // capture loop var
		if err := db.DB.Create(&item).Error; err != nil {
			log.Printf("[SEED-MASTER]   WARN: item %d (%s): %v", i+1, item.ItemCode, err)
		}
	}
	fmt.Printf("[SEED-MASTER]   ✓ %d item masters\n", len(items))

	// ─────────────────────────────────────────────
	// Customers — 20
	// ─────────────────────────────────────────────
	fmt.Println("[SEED-MASTER] Creating customers...")

	customers := []*models.Customer{
		// Account customers
		{Name: "NFA - Cavite Province", Address: "NFA Compound, Trece Martires, Cavite", ContactNumber: "046-419-1234", Email: "cavite@nfa.gov.ph", CustomerType: "Account"},
		{Name: "SM Supermarket - Tanza", Address: "SM Center Tanza, Cavite", ContactNumber: "09171000001", Email: "procurement@sm.com.ph", CustomerType: "Account"},
		{Name: "Robinsons Supermarket - Imus", Address: "Robinsons Place Imus, Cavite", ContactNumber: "09181000002", Email: "procurement@robinsons.com.ph", CustomerType: "Account"},
		{Name: "Puregold - General Trias", Address: "Puregold Gen. Trias Branch, Cavite", ContactNumber: "09191000003", Email: "buying@puregold.com.ph", CustomerType: "Account"},
		{Name: "Metro Gaisano - Tanza", Address: "Metro Gaisano Tanza, Cavite", ContactNumber: "09201000004", Email: "rice@metrogaisano.com", CustomerType: "Account"},
		{Name: "Tanza Public Market Rice Section", Address: "Tanza Public Market, Tanza, Cavite", ContactNumber: "09211000005", CustomerType: "Account"},
		{Name: "Dasmariñas Rice Retailers Coop", Address: "Dasmariñas, Cavite", ContactNumber: "09221000006", Email: "drrc.coop@gmail.com", CustomerType: "Account"},
		{Name: "Imus Rice Trading Co.", Address: "Imus, Cavite", ContactNumber: "09231000007", Email: "imus.rice@gmail.com", CustomerType: "Account"},
		{Name: "Trece Martires Rice Supply", Address: "Trece Martires City, Cavite", ContactNumber: "09241000008", CustomerType: "Account"},
		{Name: "Noveleta Rice & Grain Store", Address: "Noveleta, Cavite", ContactNumber: "09251000009", CustomerType: "Account"},
		{Name: "BaySide Restaurant Group", Address: "Rosario, Cavite", ContactNumber: "09261000010", Email: "supply@baysidegroup.ph", CustomerType: "Account"},
		{Name: "Catering by Maria Events", Address: "Kawit, Cavite", ContactNumber: "09271000011", CustomerType: "Account"},
		{Name: "San Jose Parish Canteen", Address: "San Jose, Tanza, Cavite", ContactNumber: "09281000012", CustomerType: "Account"},
		{Name: "Cavite Feeds & Livestock Supply", Address: "Silang, Cavite", ContactNumber: "09291000013", Email: "procurement@cavitefeeds.ph", CustomerType: "Account", Notes: "Buys rice bran and by-products for feed"},
		{Name: "Green Agri Nutrition Corp", Address: "General Trias, Cavite", ContactNumber: "09301000014", Email: "orders@greenagri.ph", CustomerType: "Account", Notes: "By-products buyer — darak and ipa"},
		{Name: "Magallanes Rice Dealership", Address: "Magallanes, Cavite", ContactNumber: "09311000015", CustomerType: "Account"},
		{Name: "Bacoor Grain & Rice Trading", Address: "Bacoor, Cavite", ContactNumber: "09321000016", Email: "bgrt@gmail.com", CustomerType: "Account"},
		{Name: "Alfonso Farmers Market", Address: "Alfonso, Cavite", ContactNumber: "09331000017", CustomerType: "Account"},
		// Walk-in customers
		{Name: "Walk-in Buyer", CustomerType: "Walk-in"},
		{Name: "Farmer Buyback (Palay)", CustomerType: "Walk-in", Notes: "Farmers selling palay directly at the mill"},
	}

	for i, c := range customers {
		if err := salessvc.CreateCustomer(c); err != nil {
			log.Printf("[SEED-MASTER]   WARN: customer %d (%s): %v", i+1, c.Name, err)
		}
	}
	fmt.Printf("[SEED-MASTER]   ✓ %d customers\n", len(customers))

	// ─────────────────────────────────────────────
	// Suppliers — 20
	// ─────────────────────────────────────────────
	fmt.Println("[SEED-MASTER] Creating suppliers...")

	suppliers := []*models.Supplier{
		// Palay / Grain suppliers
		{Name: "RCPI Farmers Cooperative", SupplierType: "Regular", ContactPerson: "Eduardo Reyes", ContactNumber: "09171100001", Email: "rcpi.coop@gmail.com", Address: "Tanza, Cavite", Categories: "Palay", PaymentTerms: "COD", Notes: "Primary palay supplier — NSIC Rc222 and IR64"},
		{Name: "Tanza Palay Farmers Assoc.", SupplierType: "Regular", ContactPerson: "Rodrigo Magtibay", ContactNumber: "09181100002", Address: "Tanza, Cavite", Categories: "Palay", PaymentTerms: "COD"},
		{Name: "Silang Agri Growers Coop", SupplierType: "Regular", ContactPerson: "Nilo Santos", ContactNumber: "09191100003", Email: "sagc@gmail.com", Address: "Silang, Cavite", Categories: "Palay", PaymentTerms: "COD", Notes: "Dinorado and Sinandomeng varieties"},
		{Name: "General Trias Rice Farmers", SupplierType: "Seasonal", ContactPerson: "Crisanto Delos Reyes", ContactNumber: "09201100004", Address: "General Trias, Cavite", Categories: "Palay", PaymentTerms: "COD"},
		{Name: "PhilRice Seed Center - Cavite", SupplierType: "Regular", ContactPerson: "Dr. Ana Villanueva", ContactNumber: "046-419-5678", Email: "seedcenter.cavite@philrice.gov.ph", Address: "Trece Martires, Cavite", Categories: "Palay", PaymentTerms: "Net 30", Notes: "Certified seeds and specialty varieties"},
		// Fuel suppliers
		{Name: "Petron Bulk Sales - Tanza", SupplierType: "Regular", ContactPerson: "Mark Rivera", ContactNumber: "09211100005", Email: "bulk.tanza@petron.com", Address: "Tanza, Cavite", Categories: "Fuel & Lubricants", PaymentTerms: "Net 15", TINNumber: "012-345-678-000"},
		{Name: "Shell Fleet Services Cavite", SupplierType: "Regular", ContactPerson: "Jerico Lim", ContactNumber: "09221100006", Email: "fleet.cavite@shell.ph", Address: "Imus, Cavite", Categories: "Fuel & Lubricants", PaymentTerms: "Net 15"},
		{Name: "UniOil Petroleum Corp", SupplierType: "Regular", ContactPerson: "Carlo Magno", ContactNumber: "09231100007", Email: "orders@unioil.com.ph", Address: "Bacoor, Cavite", Categories: "Fuel & Lubricants", PaymentTerms: "COD"},
		// Packaging suppliers
		{Name: "Camuning Packaging Solutions", SupplierType: "Regular", ContactPerson: "Lourdes Cruz", ContactNumber: "09241100008", Email: "lourdes@camunungpak.com", Address: "Quezon City, Metro Manila", Categories: "Packaging", PaymentTerms: "Net 30", TINNumber: "098-765-432-000"},
		{Name: "Plastisak Industrial Corp", SupplierType: "Regular", ContactPerson: "Roberto Sy", ContactNumber: "09251100009", Email: "sales@plastisak.com.ph", Address: "Caloocan, Metro Manila", Categories: "Packaging", PaymentTerms: "Net 15", TINNumber: "111-222-333-000"},
		{Name: "Regalado Agri Trading", SupplierType: "Regular", ContactPerson: "Perla Regalado", ContactNumber: "09261100010", Address: "Tanza, Cavite", Categories: "Packaging|Miscellaneous", PaymentTerms: "COD"},
		// Spare parts / mechanical suppliers
		{Name: "ABC Mechanical Parts Center", SupplierType: "Regular", ContactPerson: "Alfredo Castillo", ContactNumber: "09271100011", Email: "abc.parts@gmail.com", Address: "Dasmariñas, Cavite", Categories: "Spare Parts", PaymentTerms: "COD"},
		{Name: "Bearings & Belts Plus Inc.", SupplierType: "Regular", ContactPerson: "Danilo Tan", ContactNumber: "09281100012", Email: "sales@bbplus.ph", Address: "Quezon City, Metro Manila", Categories: "Spare Parts", PaymentTerms: "Net 15"},
		{Name: "Agri Mechanical Supply Co.", SupplierType: "Regular", ContactPerson: "Ramon Flores", ContactNumber: "09291100013", Address: "General Trias, Cavite", Categories: "Spare Parts|Equipment", PaymentTerms: "COD", Notes: "Huller rubber rolls and screens specialist"},
		// Chemical / pest control suppliers
		{Name: "Delta Agri Products Corp", SupplierType: "Regular", ContactPerson: "Susan Dimaculangan", ContactNumber: "09301100014", Email: "orders@deltaagri.ph", Address: "Calamba, Laguna", Categories: "Chemicals", PaymentTerms: "Net 30", TINNumber: "444-555-666-000"},
		{Name: "Agrisave Chemical Trading", SupplierType: "Regular", ContactPerson: "Eduardo Ponce", ContactNumber: "09311100015", Email: "edsave@agrisave.com", Address: "Manila", Categories: "Chemicals", PaymentTerms: "COD"},
		{Name: "Pest Guard Solutions Inc.", SupplierType: "Regular", ContactPerson: "Maricel Torres", ContactNumber: "09321100016", Email: "maricel@pestguard.ph", Address: "Las Piñas, Metro Manila", Categories: "Chemicals", PaymentTerms: "COD"},
		// General / miscellaneous suppliers
		{Name: "Liwayway Milling Supply", SupplierType: "Regular", ContactPerson: "Teresita Liwayway", ContactNumber: "09331100017", Address: "Cavite City, Cavite", Categories: "Packaging|Miscellaneous", PaymentTerms: "COD"},
		{Name: "Universal Hardware - Tanza", SupplierType: "Regular", ContactPerson: "Jaime dela Cruz", ContactNumber: "09341100018", Address: "Tanza, Cavite", Categories: "Spare Parts|Miscellaneous", PaymentTerms: "COD"},
		{Name: "Cavite Industrial Equipment", SupplierType: "One-time", ContactPerson: "Benjamin Ocampo", ContactNumber: "09351100019", Email: "cie@gmail.com", Address: "Imus, Cavite", Categories: "Equipment|Spare Parts", PaymentTerms: "Net 30", Notes: "Milling equipment overhaul and major spare parts"},
	}

	for i, s := range suppliers {
		if err := purchasingsvc.CreateSupplier(s); err != nil {
			log.Printf("[SEED-MASTER]   WARN: supplier %d (%s): %v", i+1, s.Name, err)
		}
	}
	fmt.Printf("[SEED-MASTER]   ✓ %d suppliers\n", len(suppliers))

	// Mark as done
	db.DB.Create(&models.FarmSettings{Key: "ricemill_master_done", Value: "1"})
	fmt.Println("[SEED-MASTER] Done.")
}

// seedCategories inserts standard rice mill item categories.
func seedCategories() {
	categories := []models.OITB{
		// ── Production Inputs ────────────────────────────────────────────────
		{ItmsGrpNam: "Palay", Description: "Raw paddy rice — all varieties purchased or received for milling", IsActive: true},
		{ItmsGrpNam: "Milled Rice", Description: "Finished white/brown rice ready for sale or distribution", IsActive: true},
		{ItmsGrpNam: "By-Products", Description: "Secondary outputs of milling: rice bran (darak), rice hull (ipa), rice polish, broken rice (butil/pinawa)", IsActive: true},

		// ── Utilities & Energy ───────────────────────────────────────────────
		{ItmsGrpNam: "Fuel & Lubricants", Description: "Diesel, gasoline, engine oil, hydraulic oil, grease, and other lubricants for milling equipment and vehicles", IsActive: true},
		{ItmsGrpNam: "Electricity & Utilities", Description: "Electricity consumption, water supply, and other utility-related costs tracked as stock items or expense items", IsActive: true},

		// ── Machinery & Maintenance ──────────────────────────────────────────
		{ItmsGrpNam: "Spare Parts", Description: "Mechanical replacement parts: rubber rolls, bearings, belts, screens, chains, and other machine components", IsActive: true},
		{ItmsGrpNam: "Tools & Equipment", Description: "Hand tools, measuring instruments, small equipment, and workshop supplies used in maintenance", IsActive: true},
		{ItmsGrpNam: "Cleaning Supplies", Description: "Brooms, mops, dust pans, detergents, and sanitation materials for the mill floor and storage areas", IsActive: true},

		// ── Chemicals & Pest Control ─────────────────────────────────────────
		{ItmsGrpNam: "Chemicals", Description: "Fumigants (Phostoxin), insecticides, rodenticides, herbicides, and other pest control or treatment products", IsActive: true},

		// ── Packaging & Storage ──────────────────────────────────────────────
		{ItmsGrpNam: "Packaging Materials", Description: "PP sacks, woven bags, plastic liners, nylon twine, packing tape, printed labels, and other packing supplies", IsActive: true},
		{ItmsGrpNam: "Storage & Handling", Description: "Wooden pallets, tarpaulins, cargo nets, moisture absorbers, and other items used in warehouse and grain storage", IsActive: true},

		// ── Safety & Compliance ──────────────────────────────────────────────
		{ItmsGrpNam: "Safety & PPE", Description: "Personal protective equipment: gloves, dust masks, safety boots, hard hats, earplugs, and first-aid supplies", IsActive: true},

		// ── Administrative ───────────────────────────────────────────────────
		{ItmsGrpNam: "Office Supplies", Description: "Stationery, printing supplies, record books, weighing tags, and administrative consumables", IsActive: true},

		// ── Capital & Infrastructure ─────────────────────────────────────────
		{ItmsGrpNam: "Equipment & Machinery", Description: "Major capital assets: hullers, polishers, dryers, conveyors, trucks, and other large equipment (for asset tracking)", IsActive: true},

		// ── Catch-all ────────────────────────────────────────────────────────
		{ItmsGrpNam: "Miscellaneous", Description: "Items that do not fit any specific category — review periodically and reclassify as needed", IsActive: true},
	}

	for i, cat := range categories {
		cat := cat
		if err := db.DB.Where(models.OITB{ItmsGrpNam: cat.ItmsGrpNam}).FirstOrCreate(&cat).Error; err != nil {
			log.Printf("[SEED-MASTER]   WARN: category %d (%s): %v", i+1, cat.ItmsGrpNam, err)
		}
	}
	fmt.Printf("[SEED-MASTER]   ✓ %d item categories\n", len(categories))
}

// seedUoMs creates common UoM Masters and two conversion groups (Weight, Volume).
func seedUoMs() {
	// ── UoM Masters ──────────────────────────────────────────────────────────
	// Each entry: code, name, weight (kg equivalent), volume (L equivalent)
	masters := []models.UoMMaster{
		// Weight
		{UomCode: "KG", UomName: "Kilogram", Weight: 1},
		{UomCode: "SACK", UomName: "Sack (50 kg)", Weight: 50},
		{UomCode: "BAG", UomName: "Bag (25 kg)", Weight: 25},
		{UomCode: "TON", UomName: "Metric Ton (1,000 kg)", Weight: 1000},
		// Volume
		{UomCode: "L", UomName: "Liter", Volume: 1},
		{UomCode: "ML", UomName: "Milliliter", Volume: 0.001},
		{UomCode: "GAL", UomName: "Gallon", Volume: 3.785},
		{UomCode: "DRUM", UomName: "Drum (200 L)", Volume: 200},
		// Count / length / misc
		{UomCode: "PC", UomName: "Piece"},
		{UomCode: "PAIR", UomName: "Pair"},
		{UomCode: "SET", UomName: "Set"},
		{UomCode: "BOX", UomName: "Box"},
		{UomCode: "ROLL", UomName: "Roll"},
		{UomCode: "CAN", UomName: "Can"},
		{UomCode: "BOTTLE", UomName: "Bottle"},
		{UomCode: "TABLET", UomName: "Tablet"},
		{UomCode: "METER", UomName: "Meter"},
		{UomCode: "BATCH", UomName: "Batch"},
	}

	for i, m := range masters {
		m := m
		if err := invsvc.CreateUoMMaster(&m); err != nil {
			log.Printf("[SEED-MASTER]   WARN: uom %d (%s): %v", i+1, m.UomCode, err)
		}
	}
	fmt.Printf("[SEED-MASTER]   ✓ %d UoM masters\n", len(masters))

	// ── Look up created entries by code ──────────────────────────────────────
	codeToEntry := func(code string) uint {
		var u models.UoMMaster
		db.DB.Where("uom_code = ?", code).First(&u)
		return u.UomEntry
	}

	kgEntry   := codeToEntry("KG")
	sackEntry := codeToEntry("SACK")
	bagEntry  := codeToEntry("BAG")
	tonEntry  := codeToEntry("TON")
	lEntry    := codeToEntry("L")
	mlEntry   := codeToEntry("ML")
	galEntry  := codeToEntry("GAL")
	drumEntry := codeToEntry("DRUM")

	if kgEntry == 0 || lEntry == 0 {
		log.Println("[SEED-MASTER]   WARN: base UoMs not found — skipping groups")
		return
	}

	// ── UoM Groups ───────────────────────────────────────────────────────────
	groups := []invsvc.CreateUoMGroupRequest{
		{
			UgpCode: "WEIGHT-GRP",
			UgpName: "Weight (kg-based)",
			BaseUom: int(kgEntry),
			Lines: []invsvc.UoMGroupLineInput{
				{UomEntry: sackEntry, AltQty: 1, BaseQty: 50},   // 1 SACK = 50 KG
				{UomEntry: bagEntry,  AltQty: 1, BaseQty: 25},   // 1 BAG  = 25 KG
				{UomEntry: tonEntry,  AltQty: 1, BaseQty: 1000}, // 1 TON  = 1000 KG
			},
		},
		{
			UgpCode: "VOLUME-GRP",
			UgpName: "Volume (liter-based)",
			BaseUom: int(lEntry),
			Lines: []invsvc.UoMGroupLineInput{
				{UomEntry: mlEntry,   AltQty: 1, BaseQty: 0.001},  // 1 ML   = 0.001 L
				{UomEntry: galEntry,  AltQty: 1, BaseQty: 3.785},  // 1 GAL  = 3.785 L
				{UomEntry: drumEntry, AltQty: 1, BaseQty: 200},    // 1 DRUM = 200 L
			},
		},
	}

	for _, req := range groups {
		if _, err := invsvc.CreateUoMGroup(req); err != nil {
			log.Printf("[SEED-MASTER]   WARN: group %s: %v", req.UgpCode, err)
		}
	}
	fmt.Printf("[SEED-MASTER]   ✓ %d UoM groups\n", len(groups))
}

func must(err error, ctx string) {
	if err != nil {
		log.Fatalf("[SEED-MASTER] %s: %v", ctx, err)
	}
}
