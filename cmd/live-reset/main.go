// cmd/live-reset/main.go — Pre-go-live reset for rice mill client.
//
// Clears ALL transaction data and demo records.
// PRESERVES: users, chart of accounts, item master, item categories,
//            customers, suppliers, UoM masters/groups, account determination,
//            payment method accounts.
//
// Run with:  go run ./cmd/live-reset/main.go
// The operator must type "YES" to confirm before any data is deleted.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	appconfig "ricemill/app/config"
	"ricemill/app/db"
)

func main() {
	cfg, err := appconfig.Load(nil)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := db.Connect(cfg); err != nil {
		log.Fatalf("connect db: %v", err)
	}

	// ── Read current record counts so the operator can verify ────────────────
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║        RICE MILL — PRE-GO-LIVE RESET UTILITY        ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This will PERMANENTLY delete all transaction records:")
	fmt.Println()

	type tableInfo struct {
		table   string
		label   string
		keep    bool
	}
	tables := []tableInfo{
		// Transaction tables — will be cleared
		{"collection_line",     "Collection lines",          false},
		{"collection",          "Collections",               false},
		{"ar_invoice_line",     "AR Invoice lines",          false},
		{"ar_invoice",          "AR Invoices",               false},
		{"do_line",             "Delivery Order lines",      false},
		{"delivery_order",      "Delivery Orders",           false},
		{"so_line",             "Sales Order lines",         false},
		{"sales_order",         "Sales Orders",              false},
		{"ap_payment_line",     "AP Payment lines",          false},
		{"ap_payment",          "AP Payments",               false},
		{"ap_invoice_line",     "AP Invoice lines",          false},
		{"ap_invoice",          "AP Invoices",               false},
		{"dr_line",             "Delivery Receipt lines",    false},
		{"delivery_receipt",    "Delivery Receipts",         false},
		{"purchase",            "Purchase records",          false},
		{"purchase_line",       "Purchase Order lines",      false},
		{"purchase_header",     "Purchase Order headers",    false},
		{"mor1",                "Milling Order output lines",false},
		{"omor",                "Milling Orders",            false},
		{"ige1",                "Goods Issue lines",         false},
		{"oige",                "Goods Issues",              false},
		{"ign1",                "Goods Receipt lines",       false},
		{"oign",                "Goods Receipts",            false},
		{"je_line",             "Journal Entry lines",       false},
		{"journal_entry",       "Journal Entries",           false},
		{"audit_log",           "Audit Log entries",         false},
		// Master tables — will be KEPT
		{"item_master",         "Item Masters",              true},
		{"item_categories",     "Item Categories",           true},
		{"customer",            "Customers",                 true},
		{"supplier",            "Suppliers",                 true},
		{"ouom",                "UoM Masters",               true},
		{"ougp",                "UoM Groups",                true},
		{"ugp1",                "UoM Group Lines",           true},
		{"account_determination","Account Determination",   true},
		{"payment_method_account","Payment Method Accounts",true},
		{"gl_account",          "Chart of Accounts",         true},
		{"user",                "Users",                     true},
	}

	// Print current row counts
	fmt.Printf("  %-35s  %8s  %s\n", "Table", "Rows", "Action")
	fmt.Printf("  %-35s  %8s  %s\n", strings.Repeat("─", 35), strings.Repeat("─", 8), "──────")
	for _, t := range tables {
		var count int64
		db.DB.Raw("SELECT COUNT(*) FROM `" + t.table + "`").Scan(&count)
		action := "DELETE"
		if t.keep {
			action = "keep"
		}
		fmt.Printf("  %-35s  %8d  %s\n", t.label, count, action)
	}

	// Reset on_hand_qty will be set to 0 on item_master
	fmt.Println()
	fmt.Println("  Additionally:")
	fmt.Println("  • on_hand_qty on all item_master rows will be set to 0")
	fmt.Println("  • Document number counters (docnum_*) will be reset")
	fmt.Println("  • Demo sentinel keys will be removed (ricemill_demo_done, seed_done)")
	fmt.Println()
	fmt.Println("  PRESERVED: users, COA, items, customers, suppliers, UoMs, account rules")
	fmt.Println()

	// ── Confirmation prompt ───────────────────────────────────────────────────
	fmt.Print("Type  YES  to proceed, or anything else to abort: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "YES" {
		fmt.Println("Aborted. No changes were made.")
		os.Exit(0)
	}

	fmt.Println()
	fmt.Println("Proceeding...")

	// ── Delete transactions in dependency order ───────────────────────────────
	toDelete := []string{
		// Sales chain (deepest first)
		"collection_line",
		"collection",
		"ar_invoice_line",
		"ar_invoice",
		"do_line",
		"delivery_order",
		"so_line",
		"sales_order",
		// Purchasing chain
		"ap_payment_line",
		"ap_payment",
		"ap_invoice_line",
		"ap_invoice",
		"dr_line",
		"delivery_receipt",
		"purchase",
		"purchase_line",
		"purchase_header",
		// Production
		"mor1",
		"omor",
		// Inventory movements
		"ige1",
		"oige",
		"ign1",
		"oign",
		// Accounting
		"je_line",
		"journal_entry",
		// Audit
		"audit_log",
	}

	errors := 0
	for _, t := range toDelete {
		// Use TRUNCATE where possible for speed, fall back to DELETE
		result := db.DB.Exec("DELETE FROM `" + t + "`")
		if result.Error != nil {
			fmt.Printf("  ✗ %-30s  ERROR: %v\n", t, result.Error)
			errors++
		} else {
			fmt.Printf("  ✓ %-30s  deleted %d rows\n", t, result.RowsAffected)
		}
	}

	// ── Reset on_hand_qty on all item_master rows ────────────────────────────
	r := db.DB.Exec("UPDATE `item_master` SET on_hand_qty = 0")
	if r.Error != nil {
		fmt.Printf("  ✗ item_master on_hand_qty reset  ERROR: %v\n", r.Error)
		errors++
	} else {
		fmt.Printf("  ✓ %-30s  reset %d items to 0\n", "item_master.on_hand_qty", r.RowsAffected)
	}

	// ── Reset document number counters ───────────────────────────────────────
	r2 := db.DB.Exec("DELETE FROM farm_settings WHERE `key` LIKE 'docnum_%'")
	if r2.Error != nil {
		fmt.Printf("  ✗ docnum counters  ERROR: %v\n", r2.Error)
	} else {
		fmt.Printf("  ✓ %-30s  removed %d counters\n", "docnum counters", r2.RowsAffected)
	}

	// ── Remove demo / seed sentinel keys ─────────────────────────────────────
	sentinels := []string{"seed_done", "ricemill_demo_done"}
	for _, k := range sentinels {
		db.DB.Exec("DELETE FROM farm_settings WHERE `key` = ?", k)
	}
	fmt.Printf("  ✓ %-30s  removed demo sentinels\n", "farm_settings")

	// ── Summary ───────────────────────────────────────────────────────────────
	fmt.Println()
	if errors > 0 {
		fmt.Printf("⚠️  Reset completed with %d error(s). Check warnings above.\n", errors)
	} else {
		fmt.Println("✅ Live reset complete. System is ready for client go-live.")
		fmt.Println()
		fmt.Println("   Next steps:")
		fmt.Println("   1. Verify master data (items, customers, suppliers, UoMs, COA)")
		fmt.Println("   2. Configure account determination rules if needed")
		fmt.Println("   3. Enter opening stock via Goods Receipts")
		fmt.Println("   4. Create the first live transactions")
	}
	fmt.Println()
}
