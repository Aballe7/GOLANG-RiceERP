package sales

// Complete O2D sales service.
// Covers: Customer CRUD, PriceGroup CRUD, Sales Order, Delivery Order,
// AR Invoice, and Collection with settlement logic.
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
// Customer CRUD
// ─────────────────────────────────────────────

// ListCustomers returns all customers ordered by name.
func ListCustomers(activeOnly bool) ([]models.Customer, error) {
	var customers []models.Customer
	q := db.DB.Preload("PriceGroup").Order("name")
	if activeOnly {
		q = q.Where("is_active = 1")
	}
	return customers, q.Find(&customers).Error
}

// GetCustomer fetches a single Customer by ID.
func GetCustomer(id uint) (*models.Customer, error) {
	var c models.Customer
	err := db.DB.Preload("PriceGroup.Items").First(&c, id).Error
	return &c, err
}

// CreateCustomer inserts a new Customer.
func CreateCustomer(c *models.Customer) error {
	c.IsActive = true
	c.CreatedAt = time.Now()
	return db.DB.Create(c).Error
}

// UpdateCustomer applies partial updates to a Customer.
func UpdateCustomer(id uint, updates map[string]interface{}) error {
	return db.DB.Model(&models.Customer{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteCustomer soft-deletes a Customer (is_active=false).
func DeleteCustomer(id uint) error {
	return db.DB.Model(&models.Customer{}).Where("id = ?", id).
		Update("is_active", false).Error
}

// ─────────────────────────────────────────────
// PriceGroup CRUD
// ─────────────────────────────────────────────

// ListPriceGroups returns all price groups with their items and customer counts.
func ListPriceGroups(activeOnly bool) ([]models.PriceGroup, error) {
	var groups []models.PriceGroup
	q := db.DB.Preload("Items").Preload("Customers").Order("name")
	if activeOnly {
		q = q.Where("is_active = 1")
	}
	return groups, q.Find(&groups).Error
}

// GetPriceGroup fetches a single PriceGroup with all its items and assigned customers.
func GetPriceGroup(id uint) (*models.PriceGroup, error) {
	var pg models.PriceGroup
	err := db.DB.Preload("Items").Preload("Customers").First(&pg, id).Error
	return &pg, err
}

// CreatePriceGroup inserts a new PriceGroup.
func CreatePriceGroup(pg *models.PriceGroup) error {
	pg.IsActive = true
	pg.CreatedAt = time.Now()
	return db.DB.Create(pg).Error
}

// UpdatePriceGroup applies partial updates to a PriceGroup.
func UpdatePriceGroup(id uint, updates map[string]interface{}) error {
	return db.DB.Model(&models.PriceGroup{}).Where("id = ?", id).Updates(updates).Error
}

// DeletePriceGroup soft-deletes a PriceGroup (is_active=false).
func DeletePriceGroup(id uint) error {
	return db.DB.Model(&models.PriceGroup{}).Where("id = ?", id).
		Update("is_active", false).Error
}

// UpsertPriceGroupItem creates or updates a single price line within a PriceGroup.
// Matches on price_group_id + egg_size + unit.
func UpsertPriceGroupItem(pgID uint, eggSize, unit string, price float64) error {
	var item models.PriceGroupItem
	err := db.DB.
		Where("price_group_id = ? AND egg_size = ? AND unit = ?", pgID, eggSize, unit).
		First(&item).Error

	now := time.Now()
	if err == gorm.ErrRecordNotFound {
		item = models.PriceGroupItem{
			PriceGroupID: pgID,
			EggSize:      eggSize,
			Unit:         unit,
			Price:        price,
			UpdatedAt:    now,
		}
		return db.DB.Create(&item).Error
	}
	if err != nil {
		return err
	}
	return db.DB.Model(&item).Updates(map[string]interface{}{
		"price":      price,
		"updated_at": now,
	}).Error
}

// ─────────────────────────────────────────────
// Sales Order
// ─────────────────────────────────────────────

// CreateSalesOrder creates a new Sales Order with line items.
func CreateSalesOrder(
	date string,
	customerID *uint,
	customerName, customerAddr, customerContact string,
	paymentMethod, terms string,
	dueDate *string,
	notes string,
	items []models.SalesOrderItem,
	createdByID *uint,
) (*models.SalesOrder, error) {
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

	invNum, err := docnumber.Svc.NextInvoiceNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}

	grandTotal := 0.0
	for _, item := range items {
		grandTotal += item.LineTotal
	}

	so := &models.SalesOrder{
		SalesOrderNumber:        invNum,
		Date:                    parsedDate,
		CustomerID:              customerID,
		CustomerNameSnapshot:    customerName,
		CustomerAddressSnapshot: customerAddr,
		CustomerContactSnapshot: customerContact,
		PaymentMethod:           paymentMethod,
		PaymentStatus:           "Unpaid",
		Terms:                   terms,
		DueDate:                 parsedDue,
		GrandTotal:              grandTotal,
		Notes:                   notes,
		CreatedAt:               time.Now(),
		CreatedByID:             createdByID,
		Items:                   items,
	}

	if err := db.DB.Create(so).Error; err != nil {
		return nil, err
	}
	return so, nil
}

// GetSalesOrder fetches a SalesOrder with items, deliveries, and AR invoices.
func GetSalesOrder(id uint) (*models.SalesOrder, error) {
	var so models.SalesOrder
	err := db.DB.
		Preload("Customer").
		Preload("Items").
		Preload("Deliveries.Items").
		Preload("ARInvoices.Items").
		First(&so, id).Error
	return &so, err
}

// ListSalesOrders returns all sales orders, newest first.
func ListSalesOrders() ([]models.SalesOrder, error) {
	var orders []models.SalesOrder
	err := db.DB.Preload("Customer").Order("id DESC").Find(&orders).Error
	return orders, err
}

// VoidSalesOrder sets the SalesOrder.payment_status to 'Void'.
// Only orders with no deliveries and no AR invoices can be voided.
func VoidSalesOrder(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var so models.SalesOrder
		if err := tx.Preload("Deliveries").Preload("ARInvoices").First(&so, id).Error; err != nil {
			return err
		}
		if so.PaymentStatus == "Void" {
			return fmt.Errorf("sales order %s is already voided", so.SalesOrderNumber)
		}
		if len(so.Deliveries) > 0 {
			return fmt.Errorf("sales order %s has delivery orders; cannot void", so.SalesOrderNumber)
		}
		if len(so.ARInvoices) > 0 {
			return fmt.Errorf("sales order %s has AR invoices; cannot void", so.SalesOrderNumber)
		}
		_ = userID
		return tx.Model(&so).Update("payment_status", "Void").Error
	})
}

// ─────────────────────────────────────────────
// Delivery Order
// ─────────────────────────────────────────────

// CreateDeliveryOrder creates a new DeliveryOrder (status=Draft) for a SalesOrder.
func CreateDeliveryOrder(
	soID uint,
	date, deliveredBy, notes string,
	items []models.DeliveryOrderItem,
	createdByID *uint,
) (*models.DeliveryOrder, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	doNum, err := docnumber.Svc.NextDeliveryNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate delivery number: %w", err)
	}

	do := &models.DeliveryOrder{
		DeliveryNumber: doNum,
		Date:           parsedDate,
		SalesOrderID:   soID,
		Status:         "Draft",
		DeliveredBy:    deliveredBy,
		Notes:          notes,
		CreatedAt:      time.Now(),
		CreatedByID:    createdByID,
		Items:          items,
	}

	if err := db.DB.Create(do).Error; err != nil {
		return nil, err
	}
	return do, nil
}

// ConfirmDeliveryOrder sets the status to 'Delivered' and updates SO.amount_delivered.
func ConfirmDeliveryOrder(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var do models.DeliveryOrder
		if err := tx.Preload("Items").First(&do, id).Error; err != nil {
			return err
		}
		if do.Status != "Draft" {
			return fmt.Errorf("delivery order %s is already %s", do.DeliveryNumber, do.Status)
		}

		totalDelivered := do.TotalAmount()

		if err := tx.Model(&do).Update("status", "Delivered").Error; err != nil {
			return err
		}

		if err := tx.Model(&models.SalesOrder{}).Where("id = ?", do.SalesOrderID).
			UpdateColumn("amount_delivered", gorm.Expr("amount_delivered + ?", totalDelivered)).
			Error; err != nil {
			return err
		}

		_ = userID
		return nil
	})
}

// GetDeliveryOrder fetches a DeliveryOrder with its items.
func GetDeliveryOrder(id uint) (*models.DeliveryOrder, error) {
	var do models.DeliveryOrder
	err := db.DB.Preload("Items").Preload("SalesOrder").Preload("ARInvoices").First(&do, id).Error
	return &do, err
}

// ListDeliveryOrders returns DeliveryOrders. If soID is 0, all records are returned.
func ListDeliveryOrders(soID uint) ([]models.DeliveryOrder, error) {
	var orders []models.DeliveryOrder
	q := db.DB.Preload("Items").Preload("SalesOrder").Order("id DESC")
	if soID != 0 {
		q = q.Where("sales_order_id = ?", soID)
	}
	err := q.Find(&orders).Error
	return orders, err
}

// ─────────────────────────────────────────────
// AR Invoice
// ─────────────────────────────────────────────

// CreateARInvoice creates an AR Invoice (status=Open) and posts the AR accounting entry.
// soID and doID are optional; at least one linkage is recommended.
func CreateARInvoice(
	soID *uint,
	doID *uint,
	customerID *uint,
	customerName, customerAddr, customerContact string,
	date, terms, notes string,
	dueDate *string,
	items []models.ARInvoiceItem,
	createdByID *uint,
) (*models.ARInvoice, error) {
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

	arNum, err := docnumber.Svc.NextARNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate AR number: %w", err)
	}

	total := 0.0
	for _, item := range items {
		total += item.LineTotal()
	}

	inv := &models.ARInvoice{
		InvoiceNumber:           arNum,
		Date:                    parsedDate,
		DueDate:                 parsedDue,
		Terms:                   terms,
		SalesOrderID:            soID,
		DeliveryOrderID:         doID,
		CustomerID:              customerID,
		CustomerNameSnapshot:    customerName,
		CustomerAddressSnapshot: customerAddr,
		CustomerContactSnapshot: customerContact,
		TotalAmount:             total,
		Status:                  "Open",
		Notes:                   notes,
		CreatedAt:               time.Now(),
		CreatedByID:             createdByID,
		Items:                   items,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(inv).Error; err != nil {
			return err
		}

		// Update SalesOrder.amount_invoiced if linked
		if soID != nil {
			if err := tx.Model(&models.SalesOrder{}).Where("id = ?", *soID).
				UpdateColumn("amount_invoiced", gorm.Expr("amount_invoiced + ?", total)).
				Error; err != nil {
				return err
			}
		}

		// Update DeliveryOrder.amount_invoiced if linked
		if doID != nil {
			if err := tx.Model(&models.DeliveryOrder{}).Where("id = ?", *doID).
				UpdateColumn("amount_invoiced", gorm.Expr("amount_invoiced + ?", total)).
				Error; err != nil {
				return err
			}
		}

		arAcct, err := accounting.GetAccount(tx, "SALES", "AR_RECEIVABLE")
		if err != nil {
			return err
		}
		revAcct, err := accounting.GetAccount(tx, "SALES", "SALES_REVENUE")
		if err != nil {
			return err
		}
		_, err = accounting.PostJournalEntry(tx, accounting.PostJEParams{
			Module:      "SALES",
			SourceType:  "AR_INVOICE",
			SourceID:    &inv.ID,
			SourceRef:   inv.InvoiceNumber,
			Narration:   "AR Invoice: " + inv.InvoiceNumber + " — " + customerName,
			CreatedByID: createdByID,
			Lines: []accounting.JELine{
				{Account: arAcct,  Debit:  total, Description: "Accounts Receivable"},
				{Account: revAcct, Credit: total, Description: "Sales Revenue"},
			},
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// GetARInvoice fetches an AR Invoice with its items and collection lines.
func GetARInvoice(id uint) (*models.ARInvoice, error) {
	var inv models.ARInvoice
	err := db.DB.
		Preload("Items").
		Preload("CollectionLines.Collection").
		Preload("Customer").
		Preload("SalesOrder").
		Preload("DeliveryOrder").
		First(&inv, id).Error
	return &inv, err
}

// ARInvoiceFilters holds optional filter parameters for ListARInvoices.
type ARInvoiceFilters struct {
	CustomerID *uint
	SalesOrderID *uint
	Status     string
}

// ListARInvoices returns AR invoices with optional filters, newest first.
func ListARInvoices(filters ARInvoiceFilters) ([]models.ARInvoice, error) {
	var invoices []models.ARInvoice
	q := db.DB.Preload("Customer").Order("id DESC")
	if filters.CustomerID != nil {
		q = q.Where("customer_id = ?", *filters.CustomerID)
	}
	if filters.SalesOrderID != nil {
		q = q.Where("sales_order_id = ?", *filters.SalesOrderID)
	}
	if filters.Status != "" {
		q = q.Where("status = ?", filters.Status)
	}
	return invoices, q.Find(&invoices).Error
}

// CancelARInvoice reverses the AR accounting JE and sets the invoice to 'Cancelled'.
func CancelARInvoice(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var inv models.ARInvoice
		if err := tx.First(&inv, id).Error; err != nil {
			return err
		}
		if inv.Status == "Cancelled" {
			return fmt.Errorf("AR invoice %s is already cancelled", inv.InvoiceNumber)
		}
		if inv.AmountCollected > 0 {
			return fmt.Errorf("AR invoice %s has collections applied; cannot cancel directly", inv.InvoiceNumber)
		}

		// Reverse amount_invoiced on SalesOrder
		if inv.SalesOrderID != nil {
			if err := tx.Model(&models.SalesOrder{}).Where("id = ?", *inv.SalesOrderID).
				UpdateColumn("amount_invoiced", gorm.Expr("amount_invoiced - ?", inv.TotalAmount)).
				Error; err != nil {
				return err
			}
		}

		// Reverse amount_invoiced on DeliveryOrder
		if inv.DeliveryOrderID != nil {
			if err := tx.Model(&models.DeliveryOrder{}).Where("id = ?", *inv.DeliveryOrderID).
				UpdateColumn("amount_invoiced", gorm.Expr("amount_invoiced - ?", inv.TotalAmount)).
				Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&inv).Update("status", "Cancelled").Error; err != nil {
			return err
		}

		var originalJE models.JournalEntry
		if err := tx.Where("source_type = ? AND source_id = ?", "AR_INVOICE", inv.ID).
			First(&originalJE).Error; err == nil {
			if _, err = accounting.ReverseJournalEntry(tx, &originalJE, nil, userID); err != nil {
				return err
			}
		}
		return nil
	})
}

// ─────────────────────────────────────────────
// Collection
// ─────────────────────────────────────────────

// CollectionLineInput is the DTO for a single collection line (AR invoice + amount applied).
type CollectionLineInput struct {
	ARInvoiceID   uint    `json:"ar_invoice_id"`
	AmountApplied float64 `json:"amount_applied"`
}

// CreateCollection creates a Collection, settles the nominated AR invoices,
// and posts the COLLECTION_CLEARING + CASH_INFLOW journal entries.
func CreateCollection(
	date string,
	customerID *uint,
	customerName, paymentMethod, refNum, notes string,
	lines []CollectionLineInput,
	createdByID *uint,
) (*models.Collection, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	crNum, err := docnumber.Svc.NextCollectionNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate collection number: %w", err)
	}

	total := 0.0
	for _, l := range lines {
		total += l.AmountApplied
	}

	collection := &models.Collection{
		CollectionNumber:     crNum,
		Date:                 parsedDate,
		CustomerID:           customerID,
		CustomerNameSnapshot: customerName,
		TotalAmount:          total,
		PaymentMethod:        paymentMethod,
		ReferenceNumber:      refNum,
		Notes:                notes,
		CreatedAt:            time.Now(),
		CreatedByID:          createdByID,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(collection).Error; err != nil {
			return err
		}

		for _, l := range lines {
			line := models.CollectionLine{
				CollectionID:  collection.ID,
				ARInvoiceID:   l.ARInvoiceID,
				AmountApplied: l.AmountApplied,
			}
			if err := tx.Create(&line).Error; err != nil {
				return fmt.Errorf("failed to create collection line for invoice %d: %w", l.ARInvoiceID, err)
			}

			// Settle AR invoice: increment amount_collected and update status
			var inv models.ARInvoice
			if err := tx.First(&inv, l.ARInvoiceID).Error; err != nil {
				return fmt.Errorf("AR invoice %d not found: %w", l.ARInvoiceID, err)
			}
			newCollected := inv.AmountCollected + l.AmountApplied
			newStatus := inv.Status
			if newCollected >= inv.TotalAmount {
				newStatus = "Paid"
			} else if newCollected > 0 {
				newStatus = "Partial"
			}
			if err := tx.Model(&inv).Updates(map[string]interface{}{
				"amount_collected": newCollected,
				"status":           newStatus,
			}).Error; err != nil {
				return err
			}

			// Update parent SalesOrder.amount_collected if linked
			if inv.SalesOrderID != nil {
				if err := tx.Model(&models.SalesOrder{}).Where("id = ?", *inv.SalesOrderID).
					UpdateColumn("amount_collected", gorm.Expr("amount_collected + ?", l.AmountApplied)).
					Error; err != nil {
					return err
				}
				// Update SalesOrder.amount_paid and payment_status
				var so models.SalesOrder
				if err := tx.First(&so, *inv.SalesOrderID).Error; err == nil {
					newPaid := so.AmountPaid + l.AmountApplied
					soStatus := so.PaymentStatus
					if soStatus != "Void" {
						if newPaid >= so.GrandTotal {
							soStatus = "Paid"
						} else if newPaid > 0 {
							soStatus = "Partial"
						}
					}
					_ = tx.Model(&so).Updates(map[string]interface{}{
						"amount_paid":    newPaid,
						"payment_status": soStatus,
					})
				}
			}
		}

		cashAcct, err := accounting.GetPaymentAccount(tx, paymentMethod, "INFLOW")
		if err != nil {
			return err
		}
		arAcct, err := accounting.GetAccount(tx, "SALES", "COLLECTION_CLEARING")
		if err != nil {
			return err
		}
		_, err = accounting.PostJournalEntry(tx, accounting.PostJEParams{
			Module:      "SALES",
			SourceType:  "COLLECTION",
			SourceID:    &collection.ID,
			SourceRef:   collection.CollectionNumber,
			Narration:   "Collection: " + collection.CollectionNumber + " — " + customerName,
			CreatedByID: createdByID,
			Lines: []accounting.JELine{
				{Account: cashAcct, Debit:  total, Description: "Cash/Bank inflow"},
				{Account: arAcct,   Credit: total, Description: "AR cleared"},
			},
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return collection, nil
}

// GetCollection fetches a Collection with its settlement lines.
func GetCollection(id uint) (*models.Collection, error) {
	var collection models.Collection
	err := db.DB.
		Preload("Lines.ARInvoice").
		Preload("Customer").
		First(&collection, id).Error
	return &collection, err
}

// ListCollections returns all collections, newest first.
func ListCollections() ([]models.Collection, error) {
	var collections []models.Collection
	err := db.DB.Preload("Customer").Order("id DESC").Find(&collections).Error
	return collections, err
}
