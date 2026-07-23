package main

import (
	"fmt"
	"os"

	appconfig "ricemill/app/config"
	"ricemill/app/db"
	invsvc "ricemill/app/services/inventory"
)

func must(err error, step string) {
	if err != nil {
		fmt.Printf("FAIL [%s]: %v\n", step, err)
		os.Exit(1)
	}
	fmt.Printf("OK   [%s]\n", step)
}

func main() {
	cfg, err := appconfig.Load(nil)
	must(err, "load config")
	must(db.Connect(cfg), "connect")
	must(db.PreMigrateFixup(), "PreMigrateFixup")
	must(db.AutoMigrateAll(), "AutoMigrateAll")
	must(db.RunColumnMigrations(), "RunColumnMigrations")
	must(db.SeedDefaults(), "SeedDefaults")
	must(invsvc.BackfillOIVL(), "BackfillOIVL")

	// ── Diagnostic: what does the stock ledger actually contain? ──────────────
	type row struct {
		TransType string
		N         int
		MinD      string
		MaxD      string
	}
	var rows []row
	must(db.DB.Raw(`SELECT trans_type, COUNT(*) AS n, MIN(doc_date) AS min_d, MAX(doc_date) AS max_d
		FROM oivl GROUP BY trans_type ORDER BY trans_type`).Scan(&rows).Error, "query oivl")
	fmt.Println("\noivl rows by trans_type:")
	if len(rows) == 0 {
		fmt.Println("  (none)")
	}
	for _, r := range rows {
		fmt.Printf("  %-10s n=%-5d %s .. %s\n", r.TransType, r.N, r.MinD, r.MaxD)
	}

	var giCount int64
	db.DB.Raw(`SELECT COUNT(*) FROM oige`).Scan(&giCount)
	fmt.Printf("\ngoods issue headers (oige): %d\n", giCount)

	type statRow struct {
		Status string
		N      int
	}
	var drStats, doStats []statRow
	db.DB.Raw(`SELECT status, COUNT(*) AS n FROM delivery_receipt GROUP BY status`).Scan(&drStats)
	db.DB.Raw(`SELECT status, COUNT(*) AS n FROM delivery_order GROUP BY status`).Scan(&doStats)
	fmt.Printf("delivery_receipt by status: %v\n", drStats)
	fmt.Printf("delivery_order by status:   %v\n", doStats)

	type gi struct {
		ID          uint
		GINumber    string
		PostingDate string
		Status      string
	}
	var gis []gi
	db.DB.Raw(`SELECT id, gi_number, posting_date, status FROM oige ORDER BY id DESC LIMIT 5`).Scan(&gis)
	for _, g := range gis {
		fmt.Printf("  GI %-20s date=%-12q status=%s\n", g.GINumber, g.PostingDate, g.Status)
	}
}
