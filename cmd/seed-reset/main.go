// cmd/seed-reset/main.go — Clears all seeded sample data so the seed can be re-run.
// Preserves: admin user, chart of accounts, farm settings (except seed_done).
// Run with: go run ./cmd/seed-reset/main.go
package main

import (
	"fmt"
	"log"

	appconfig "egglayererp/app/config"
	"egglayererp/app/db"
)

func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := db.Connect(cfg); err != nil {
		log.Fatalf("connect db: %v", err)
	}

	tables := []string{
		"collection_line", "collection",
		"ar_invoice_item", "ar_invoice",
		"delivery_order_item", "delivery_order",
		"sales_order_item", "sales_order",
		"ap_payment_line", "ap_payment",
		"ap_invoice_item", "ap_invoice",
		"delivery_receipt_item", "delivery_receipt",
		"purchase",
		"body_weight_log", "grower_log", "daily_log", "flock",
		"feed_stock",
		"item_master", "item_categories",
		"price_group_item", "customer", "price_group",
		"supplier",
		"account_determination", "payment_method_account",
		"journal_entry_line", "journal_entry",
	}

	for _, t := range tables {
		if err := db.DB.Exec("DELETE FROM `" + t + "`").Error; err != nil {
			fmt.Printf("WARN: delete %s: %v\n", t, err)
		} else {
			fmt.Printf("  cleared %s\n", t)
		}
	}

	// Remove seed sentinel
	db.DB.Exec("DELETE FROM farm_settings WHERE `key` = 'seed_done'")
	// Reset docnum counters
	db.DB.Exec("DELETE FROM farm_settings WHERE `key` LIKE 'docnum_%'")
	fmt.Println("Reset complete. Run 'go run ./cmd/seed/main.go' to re-seed.")
}
