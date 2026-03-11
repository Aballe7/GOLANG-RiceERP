package purchasing

// Complete P2P purchasing service.
// Covers: Supplier CRUD, Purchase CRUD, Delivery Receipt P2P,
// AP Invoice, and AP Payment with settlement logic.
//
// Accounting engine calls are marked TODO; they will be wired
// to egglayererp/app/services/accounting once GL accounts are configured.

import (
	"fmt"
	"time"

	"egglayererp/app/db"
	"egglayererp/app/models"
	"egglayererp/app/services/docnumber"

	"egglayererp/app/services/accounting"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────
// Supplier CRUD
// ─────────────────────────────────────────────

// ListSuppliers returns all suppliers ordered by name.
func ListSuppliers(activeOnly bool) ([]models.Supplier, error) {
	var suppliers []models.Supplier
	q := db.DB.Order("name")
	if activeOnly {
		q = q.Where("is_active = 1")
	}
	return suppliers, q.Find(&suppliers).Error
}

// GetSupplier fetches a single Supplier by ID.
func GetSupplier(id uint) (*models.Supplier, error) {
	var s models.Supplier
	err := db.DB.First(&s, id).Error
	return &s, err
}

// CreateSupplier inserts a new Supplier.
func CreateSupplier(s *models.Supplier) error {
	s.IsActive = true
	s.CreatedAt = time.Now()
	return db.DB.Create(s).Error
}

// UpdateSupplier applies partial updates to a Supplier.
func UpdateSupplier(id uint, updates map[string]interface{}) error {
	return db.DB.Model(&models.Supplier{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteSupplier soft-deletes a Supplier (is_active=false).
func DeleteSupplier(id uint) error {
	return db.DB.Model(&models.Supplier{}).Where("id = ?", id).
		Update("is_active", false).Error
}

// ─────────────────────────────────────────────
// Purchase CRUD
// ─────────────────────────────────────────────

// ListPurchases returns all purchases, newest first.
func ListPurchases() ([]models.Purchase, error) {
	var purchases []models.Purchase
	err := db.DB.Preload("SupplierRel").Order("id DESC").Find(&purchases).Error
	return purchases, err
}

// GetPurchase fetches a Purchase with DeliveryReceipts and APInvoices.
func GetPurchase(id uint) (*models.Purchase, error) {
	var p models.Purchase
	err := db.DB.
		Preload("SupplierRel").
		Preload("DeliveryReceipts.Items").
		Preload("APInvoices.Items").
		First(&p, id).Error
	return &p, err
}

// CreatePurchase inserts a new Purchase record.
func CreatePurchase(p *models.Purchase) error {
	p.IsActive = true
	return db.DB.Create(p).Error
}

// UpdatePurchase applies partial updates to a Purchase.
func UpdatePurchase(id uint, updates map[string]interface{}) error {
	return db.DB.Model(&models.Purchase{}).Where("id = ?", id).Updates(updates).Error
}

// ─────────────────────────────────────────────
// Delivery Receipt
// ─────────────────────────────────────────────

// CreateDeliveryReceipt creates a new DeliveryReceipt (status=Draft) linked to a Purchase.
func CreateDeliveryReceipt(
	purchaseID uint,
	date, receivedBy, supplierDRRef, notes string,
	items []models.DeliveryReceiptItem,
	createdByID *uint,
) (*models.DeliveryReceipt, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
	}

	drNum, err := docnumber.Svc.NextDRNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate DR number: %w", err)
	}

	dr := &models.DeliveryReceipt{
		DRNumber:      drNum,
		Date:          parsedDate,
		PurchaseID:    purchaseID,
		Status:        "Draft",
		ReceivedBy:    receivedBy,
		SupplierDRRef: supplierDRRef,
		Notes:         notes,
		CreatedAt:     time.Now(),
		CreatedByID:   createdByID,
		Items:         items,
	}

	if err := db.DB.Create(dr).Error; err != nil {
		return nil, err
	}
	return dr, nil
}

// GetDeliveryReceipt fetches a DeliveryReceipt with its items.
func GetDeliveryReceipt(id uint) (*models.DeliveryReceipt, error) {
	var dr models.DeliveryReceipt
	err := db.DB.Preload("Items").Preload("Purchase").Preload("APInvoices").First(&dr, id).Error
	return &dr, err
}

// ListDeliveryReceipts returns DeliveryReceipts. If purchaseID is 0, all records are returned.
func ListDeliveryReceipts(purchaseID uint) ([]models.DeliveryReceipt, error) {
	var drs []models.DeliveryReceipt
	q := db.DB.Preload("Items").Order("id DESC")
	if purchaseID != 0 {
		q = q.Where("purchase_id = ?", purchaseID)
	}
	err := q.Find(&drs).Error
	return drs, err
}

// ConfirmDeliveryReceipt sets the DR status to 'Received', updates the parent
// Purchase.amount_received, and posts the GRNI / INVENTORY_RECEIVED journal entry.
func ConfirmDeliveryReceipt(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var dr models.DeliveryReceipt
		if err := tx.Preload("Items").First(&dr, id).Error; err != nil {
			return err
		}
		if dr.Status != "Draft" {
			return fmt.Errorf("delivery receipt %s is already %s", dr.DRNumber, dr.Status)
		}

		// Calculate total received value
		totalReceived := 0.0
		for _, item := range dr.Items {
			totalReceived += item.LineTotal()
		}

		// Mark as Received
		if err := tx.Model(&dr).Updates(map[string]interface{}{
			"status": "Received",
		}).Error; err != nil {
			return err
		}

		// Update Purchase.amount_received (increment)
		if err := tx.Model(&models.Purchase{}).Where("id = ?", dr.PurchaseID).
			UpdateColumn("amount_received", gorm.Expr("amount_received + ?", totalReceived)).
			Error; err != nil {
			return err
		}

		grniAcct, err := accounting.GetAccount(tx, "PURCHASING", "GRNI")
		if err != nil {
			return err
		}
		invAcct, err := accounting.GetAccount(tx, "PURCHASING", "INVENTORY_RECEIVED")
		if err != nil {
			return err
		}
		_, err = accounting.PostJournalEntry(tx, accounting.PostJEParams{
			Module:      "PURCHASING",
			SourceType:  "DR_CONFIRM",
			SourceID:    &dr.ID,
			SourceRef:   dr.DRNumber,
			Narration:   "Goods received: " + dr.DRNumber,
			CreatedByID: userID,
			Lines: []accounting.JELine{
				{Account: invAcct,  Debit:  totalReceived, Description: "Inventory received"},
				{Account: grniAcct, Credit: totalReceived, Description: "GRNI — goods received not invoiced"},
			},
		})
		return err
	})
}

// CancelDeliveryReceipt sets the DR status to 'Cancelled'.
func CancelDeliveryReceipt(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var dr models.DeliveryReceipt
		if err := tx.First(&dr, id).Error; err != nil {
			return err
		}
		if dr.Status == "Cancelled" {
			return fmt.Errorf("delivery receipt %s is already cancelled", dr.DRNumber)
		}
		if dr.Status == "Received" {
			return fmt.Errorf("delivery receipt %s is already received and cannot be cancelled directly; reverse via an AP Invoice", dr.DRNumber)
		}
		_ = userID
		return tx.Model(&dr).Update("status", "Cancelled").Error
	})
}

// ─────────────────────────────────────────────
// AP Invoice
// ─────────────────────────────────────────────

// CreateAPInvoice creates an AP Invoice (status=Open), links it to a Purchase
// (and optionally a DeliveryReceipt), and posts the AP accounting entry.
func CreateAPInvoice(
	purchaseID uint,
	drID *uint,
	date, terms, supplierName, supplierInvRef, notes string,
	dueDate *string,
	items []models.APInvoiceItem,
	createdByID *uint,
) (*models.APInvoice, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	var parsedDue *time.Time
	if dueDate != nil && *dueDate != "" {
		d, err := time.Parse("2006-01-02", *dueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date format: %w", err)
		}
		parsedDue = &d
	}

	apNum, err := docnumber.Svc.NextAPNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate AP number: %w", err)
	}

	// Calculate total from line items
	total := 0.0
	for _, item := range items {
		total += item.LineTotal()
	}

	inv := &models.APInvoice{
		InvoiceNumber:      apNum,
		Date:               parsedDate,
		DueDate:            parsedDue,
		Terms:              terms,
		PurchaseID:         purchaseID,
		DeliveryReceiptID:  drID,
		SupplierName:       supplierName,
		SupplierInvoiceRef: supplierInvRef,
		TotalAmount:        total,
		Status:             "Open",
		Notes:              notes,
		CreatedAt:          time.Now(),
		CreatedByID:        createdByID,
		Items:              items,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(inv).Error; err != nil {
			return err
		}

		// Update Purchase.amount_invoiced
		if err := tx.Model(&models.Purchase{}).Where("id = ?", purchaseID).
			UpdateColumn("amount_invoiced", gorm.Expr("amount_invoiced + ?", total)).
			Error; err != nil {
			return err
		}

		// If linked to a DR, update DR.amount_invoiced
		if drID != nil {
			if err := tx.Model(&models.DeliveryReceipt{}).Where("id = ?", *drID).
				UpdateColumn("amount_invoiced", gorm.Expr("amount_invoiced + ?", total)).
				Error; err != nil {
				return err
			}
		}

		apAcct, err := accounting.GetAccount(tx, "PURCHASING", "AP_PAYABLE")
		if err != nil {
			return err
		}
		// If linked to a DR, clear GRNI; otherwise debit inventory/expense directly
		var debitAcct *models.GLAccount
		if drID != nil {
			debitAcct, err = accounting.GetAccount(tx, "PURCHASING", "GRNI")
		} else {
			debitAcct, err = accounting.GetAccount(tx, "PURCHASING", "INVENTORY_RECEIVED")
		}
		if err != nil {
			return err
		}
		_, err = accounting.PostJournalEntry(tx, accounting.PostJEParams{
			Module:      "PURCHASING",
			SourceType:  "AP_INVOICE",
			SourceID:    &inv.ID,
			SourceRef:   inv.InvoiceNumber,
			Narration:   "AP Invoice: " + inv.InvoiceNumber + " — " + supplierName,
			CreatedByID: createdByID,
			Lines: []accounting.JELine{
				{Account: debitAcct, Debit:  total, Description: "GRNI cleared / inventory received"},
				{Account: apAcct,    Credit: total, Description: "Accounts Payable"},
			},
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// GetAPInvoice fetches an AP Invoice with its items and payment lines.
func GetAPInvoice(id uint) (*models.APInvoice, error) {
	var inv models.APInvoice
	err := db.DB.
		Preload("Items").
		Preload("PaymentLines.Payment").
		Preload("Purchase").
		Preload("DeliveryReceipt").
		First(&inv, id).Error
	return &inv, err
}

// APInvoiceFilters holds optional filter parameters for ListAPInvoices.
type APInvoiceFilters struct {
	PurchaseID *uint
	Status     string
}

// ListAPInvoices returns AP invoices, optionally filtered.
func ListAPInvoices(filters APInvoiceFilters) ([]models.APInvoice, error) {
	var invoices []models.APInvoice
	q := db.DB.Preload("Items").Order("id DESC")
	if filters.PurchaseID != nil {
		q = q.Where("purchase_id = ?", *filters.PurchaseID)
	}
	if filters.Status != "" {
		q = q.Where("status = ?", filters.Status)
	}
	return invoices, q.Find(&invoices).Error
}

// CancelAPInvoice reverses the AP accounting JE and sets the invoice to 'Cancelled'.
func CancelAPInvoice(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var inv models.APInvoice
		if err := tx.Preload("Items").First(&inv, id).Error; err != nil {
			return err
		}
		if inv.Status == "Cancelled" {
			return fmt.Errorf("AP invoice %s is already cancelled", inv.InvoiceNumber)
		}
		if inv.AmountPaidStored > 0 {
			return fmt.Errorf("AP invoice %s has payments applied; cannot cancel directly", inv.InvoiceNumber)
		}

		// Reverse amount_invoiced on Purchase
		if err := tx.Model(&models.Purchase{}).Where("id = ?", inv.PurchaseID).
			UpdateColumn("amount_invoiced", gorm.Expr("amount_invoiced - ?", inv.TotalAmount)).
			Error; err != nil {
			return err
		}

		// Set status
		if err := tx.Model(&inv).Update("status", "Cancelled").Error; err != nil {
			return err
		}

		var originalJE models.JournalEntry
		if err := tx.Where("source_type = ? AND source_id = ?", "AP_INVOICE", inv.ID).
			First(&originalJE).Error; err == nil {
			if _, err = accounting.ReverseJournalEntry(tx, &originalJE, nil, userID); err != nil {
				return err
			}
		}
		return nil
	})
}

// ─────────────────────────────────────────────
// AP Payment
// ─────────────────────────────────────────────

// APPaymentLineInput is the DTO for a single payment line (invoice + amount applied).
type APPaymentLineInput struct {
	APInvoiceID   uint    `json:"ap_invoice_id"`
	AmountApplied float64 `json:"amount_applied"`
}

// CreateAPPayment creates an APPayment, settles the nominated AP invoices,
// and posts the AP_CLEARING + CASH_OUTFLOW journal entries.
func CreateAPPayment(
	date string,
	supplierID *uint,
	supplierName, paymentMethod, refNum, notes string,
	lines []APPaymentLineInput,
	createdByID *uint,
) (*models.APPayment, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	payNum, err := docnumber.Svc.NextPaymentNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate payment number: %w", err)
	}

	total := 0.0
	for _, l := range lines {
		total += l.AmountApplied
	}

	payment := &models.APPayment{
		PaymentNumber:        payNum,
		Date:                 parsedDate,
		SupplierID:           supplierID,
		SupplierNameSnapshot: supplierName,
		TotalAmount:          total,
		PaymentMethod:        paymentMethod,
		ReferenceNumber:      refNum,
		Notes:                notes,
		CreatedAt:            time.Now(),
		CreatedByID:          createdByID,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(payment).Error; err != nil {
			return err
		}

		for _, l := range lines {
			line := models.APPaymentLine{
				PaymentID:     payment.ID,
				APInvoiceID:   l.APInvoiceID,
				AmountApplied: l.AmountApplied,
			}
			if err := tx.Create(&line).Error; err != nil {
				return fmt.Errorf("failed to create payment line for invoice %d: %w", l.APInvoiceID, err)
			}

			// Settle invoice: increment amount_paid_stored and update status
			var inv models.APInvoice
			if err := tx.First(&inv, l.APInvoiceID).Error; err != nil {
				return fmt.Errorf("AP invoice %d not found: %w", l.APInvoiceID, err)
			}
			newPaid := inv.AmountPaidStored + l.AmountApplied
			newStatus := inv.Status
			if newPaid >= inv.TotalAmount {
				newStatus = "Paid"
			} else if newPaid > 0 {
				newStatus = "Partial"
			}
			if err := tx.Model(&inv).Updates(map[string]interface{}{
				"amount_paid_stored": newPaid,
				"status":             newStatus,
			}).Error; err != nil {
				return err
			}

			// Update parent Purchase.amount_settled
			if err := tx.Model(&models.Purchase{}).Where("id = ?", inv.PurchaseID).
				UpdateColumn("amount_settled", gorm.Expr("amount_settled + ?", l.AmountApplied)).
				Error; err != nil {
				return err
			}
		}

		apAcct, err := accounting.GetAccount(tx, "PURCHASING", "AP_CLEARING")
		if err != nil {
			return err
		}
		cashAcct, err := accounting.GetPaymentAccount(tx, paymentMethod, "OUTFLOW")
		if err != nil {
			return err
		}
		_, err = accounting.PostJournalEntry(tx, accounting.PostJEParams{
			Module:      "PURCHASING",
			SourceType:  "AP_PAYMENT",
			SourceID:    &payment.ID,
			SourceRef:   payment.PaymentNumber,
			Narration:   "AP Payment: " + payment.PaymentNumber + " — " + supplierName,
			CreatedByID: createdByID,
			Lines: []accounting.JELine{
				{Account: apAcct,   Debit:  total, Description: "AP settled / cleared"},
				{Account: cashAcct, Credit: total, Description: "Cash/Bank outflow"},
			},
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return payment, nil
}

// GetAPPayment fetches an APPayment with its settlement lines.
func GetAPPayment(id uint) (*models.APPayment, error) {
	var payment models.APPayment
	err := db.DB.
		Preload("Lines.APInvoice").
		Preload("Supplier").
		First(&payment, id).Error
	return &payment, err
}

// ListAPPayments returns all AP payments, newest first.
func ListAPPayments() ([]models.APPayment, error) {
	var payments []models.APPayment
	err := db.DB.Preload("Lines").Order("id DESC").Find(&payments).Error
	return payments, err
}
