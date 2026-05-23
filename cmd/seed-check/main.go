package main

import (
	"fmt"

	appconfig "ricemill/app/config"
	"ricemill/app/db"
	"ricemill/app/models"
)

func main() {
	cfg, _ := appconfig.Load(nil)
	db.Connect(cfg)

	counts := func(m interface{}, label string) {
		var n int64
		db.DB.Model(m).Count(&n)
		fmt.Printf("  %-25s %d\n", label+":", n)
	}

	fmt.Println("=== Seed Data Summary ===")
	counts(&models.Supplier{}, "Suppliers")
	counts(&models.Customer{}, "Customers")
	counts(&models.PriceGroup{}, "Price Groups")
	counts(&models.PriceGroupItem{}, "Price Group Items")
	counts(&models.ItemMaster{}, "Item Masters")
	counts(&models.DeliveryReceipt{}, "Delivery Receipts")
	counts(&models.APInvoice{}, "AP Invoices")
	counts(&models.APPayment{}, "AP Payments")
	counts(&models.SalesOrder{}, "Sales Orders")
	counts(&models.DeliveryOrder{}, "Delivery Orders")
	counts(&models.ARInvoice{}, "AR Invoices")
	counts(&models.Collection{}, "Collections")
	counts(&models.AccountDetermination{}, "AD Rules")
	counts(&models.PaymentMethodAccount{}, "Payment Methods")
	counts(&models.JournalEntry{}, "Journal Entries")

	var paidAP, paidAR, partialAR int64
	db.DB.Model(&models.APInvoice{}).Where("status = 'Paid'").Count(&paidAP)
	db.DB.Model(&models.ARInvoice{}).Where("status = 'Paid'").Count(&paidAR)
	db.DB.Model(&models.ARInvoice{}).Where("status = 'Partial'").Count(&partialAR)
	fmt.Printf("\n  AP Invoices Paid:         %d\n", paidAP)
	fmt.Printf("  AR Invoices Paid:         %d\n", paidAR)
	fmt.Printf("  AR Invoices Partial:      %d\n", partialAR)
}
