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

	"ricemill/app/db"
	"ricemill/app/models"
	"ricemill/app/services/accounting"
	"ricemill/app/services/docnumber"
	invsvc "ricemill/app/services/inventory"

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
	// GORM Updates(map) skips nil values, so handle price_group_id=null explicitly.
	if v, exists := updates["price_group_id"]; exists && v == nil {
		if err := db.DB.Model(&models.Customer{}).Where("id = ?", id).
			UpdateColumn("price_group_id", nil).Error; err != nil {
			return err
		}
		delete(updates, "price_group_id")
	}
	if len(updates) == 0 {
		return nil
	}
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

	// Validate category visibility for sales
	for _, item := range items {
		var oitm models.OITM
		if err := db.DB.Where("item_code = ? OR item_name = ?", item.SKU, item.SKU).First(&oitm).Error; err == nil {
			var cat models.OITB
			if err := db.DB.First(&cat, oitm.ItmsGrpCod).Error; err == nil && !cat.ForSales {
				return nil, fmt.Errorf("item %s belongs to category '%s' which is not enabled for Sales", item.SKU, cat.ItmsGrpNam)
			}
		}
	}

	grandTotal := 0.0
	for _, item := range items {
		grandTotal += item.LineTotal
	}

	// Set OpenQty = Quantity for each item
	for i := range items {
		items[i].OpenQty = items[i].Quantity
	}

	so := &models.SalesOrder{
		SalesOrderNumber:        invNum,
		DocStatus:               "Draft",
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
		Preload("ARInvoices.CollectionLines.Collection").
		First(&so, id).Error
	return &so, err
}

// ListSalesOrders returns all sales orders, newest first.
func ListSalesOrders() ([]models.SalesOrder, error) {
	var orders []models.SalesOrder
	err := db.DB.Preload("Customer").Order("id DESC").Find(&orders).Error
	return orders, err
}

// VoidSalesOrder is kept for backward compatibility. Use CancelSalesOrder instead.
func VoidSalesOrder(id uint, userID *uint) error {
	return CancelSalesOrder(id, userID)
}

// SubmitSalesOrder transitions a Draft SO to Open, making it available for delivery.
func SubmitSalesOrder(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var so models.SalesOrder
		if err := tx.First(&so, id).Error; err != nil {
			return err
		}
		if so.DocStatus != "Draft" {
			return fmt.Errorf("sales order %s is not in Draft status (current: %s)", so.SalesOrderNumber, so.DocStatus)
		}
		if err := tx.Model(&so).Updates(map[string]interface{}{
			"doc_status":    "Open",
			"updated_by_id": userID,
			"version":       gorm.Expr("version + 1"),
		}).Error; err != nil {
			return err
		}

		// Increment is_commited for each line (SO is now Open and locked for editing)
		var soItems []models.SalesOrderItem
		if err := tx.Where("order_id = ?", id).Find(&soItems).Error; err == nil {
			for _, item := range soItems {
				var oitm models.OITM
				if tx.Where("item_code = ? OR item_name = ?", item.SKU, item.SKU).First(&oitm).Error != nil {
					continue
				}
				uomEntry := item.UomEntry
				if uomEntry == 0 {
					uomEntry = oitm.IUoMEntry
				}
				invQty := invsvc.ConvertToInventoryQty(tx, &oitm, uomEntry, item.Quantity)
				tx.Model(&models.OITM{}).Where("item_code = ?", oitm.ItemCode).
					UpdateColumn("is_commited", gorm.Expr("is_commited + ?", invQty))
			}
		}
		return nil
	})
}

// UpdateSalesOrder replaces the items and recalculates the total. Only allowed for Draft SOs.
// version must match the current DB version; if not, a conflict error is returned.
func UpdateSalesOrder(
	id uint,
	version uint,
	customerID *uint,
	customerName, customerAddr, customerContact string,
	date, paymentMethod, terms string,
	dueDate *string,
	notes string,
	items []models.SalesOrderItem,
	userID *uint,
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

	grandTotal := 0.0
	for i := range items {
		grandTotal += items[i].LineTotal
		items[i].OpenQty = items[i].Quantity
		items[i].OrderID = id
	}

	var so models.SalesOrder
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&so, id).Error; err != nil {
			return err
		}
		if so.DocStatus != "Draft" {
			return fmt.Errorf("sales order %s cannot be edited (status: %s)", so.SalesOrderNumber, so.DocStatus)
		}

		// Delete existing items and replace
		if err := tx.Where("order_id = ?", id).Delete(&models.SalesOrderItem{}).Error; err != nil {
			return err
		}

		result := tx.Model(&so).
			Where("version = ?", version).
			Updates(map[string]interface{}{
				"customer_id":               customerID,
				"customer_name_snapshot":    customerName,
				"customer_address_snapshot": customerAddr,
				"customer_contact_snapshot": customerContact,
				"date":                      parsedDate,
				"payment_method":            paymentMethod,
				"terms":                     terms,
				"due_date":                  parsedDue,
				"notes":                     notes,
				"grand_total":               grandTotal,
				"updated_by_id":             userID,
				"version":                   gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("conflict: this sales order was modified by another user — please refresh and try again")
		}

		for i := range items {
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Reload with items
	return GetSalesOrder(id)
}

// CancelSalesOrder cancels a Draft or Open SO. Blocked if child DOs or AR Invoices exist.
func CancelSalesOrder(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var so models.SalesOrder
		if err := tx.Preload("Deliveries").Preload("ARInvoices").Preload("Items").First(&so, id).Error; err != nil {
			return err
		}
		if so.DocStatus == "Cancelled" {
			return fmt.Errorf("sales order %s is already cancelled", so.SalesOrderNumber)
		}
		if so.DocStatus == "Closed" {
			return fmt.Errorf("sales order %s is closed and cannot be cancelled", so.SalesOrderNumber)
		}
		if len(so.Deliveries) > 0 {
			return fmt.Errorf("sales order %s has delivery orders; cannot cancel", so.SalesOrderNumber)
		}
		if len(so.ARInvoices) > 0 {
			return fmt.Errorf("sales order %s has AR invoices; cannot cancel", so.SalesOrderNumber)
		}

		// Release is_commited only if SO was Open (incremented at SubmitSalesOrder)
		if so.DocStatus == "Open" {
			for _, item := range so.Items {
				var oitm models.OITM
				if tx.Where("item_code = ? OR item_name = ?", item.SKU, item.SKU).First(&oitm).Error != nil {
					continue
				}
				uomEntry := item.UomEntry
				if uomEntry == 0 {
					uomEntry = oitm.IUoMEntry
				}
				// Use OpenQty: remaining uncommitted portion (full qty if no DOs confirmed yet)
				invQty := invsvc.ConvertToInventoryQty(tx, &oitm, uomEntry, item.OpenQty)
				tx.Model(&models.OITM{}).Where("item_code = ?", oitm.ItemCode).
					UpdateColumn("is_commited", gorm.Expr("GREATEST(0, is_commited - ?)", invQty))
			}
		}

		return tx.Model(&so).Updates(map[string]interface{}{
			"doc_status":    "Cancelled",
			"updated_by_id": userID,
			"version":       gorm.Expr("version + 1"),
		}).Error
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
	// Verify SO exists and is Open
	var so models.SalesOrder
	if err := db.DB.First(&so, soID).Error; err != nil {
		return nil, fmt.Errorf("sales order not found: %w", err)
	}
	if so.DocStatus != "Open" {
		return nil, fmt.Errorf("sales order %s is not Open (status: %s); submit it first", so.SalesOrderNumber, so.DocStatus)
	}

	// Guard: only one Draft DO allowed per SO at a time
	var draftCount int64
	if err := db.DB.Model(&models.DeliveryOrder{}).
		Where("sales_order_id = ? AND status = 'Draft'", soID).
		Count(&draftCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check existing draft delivery orders: %w", err)
	}
	if draftCount > 0 {
		return nil, fmt.Errorf("a draft delivery order already exists for this sales order — confirm or cancel it first")
	}

	// Per-item quantity validation: requested qty must not exceed current open_qty
	var soItems []models.SalesOrderItem
	if err := db.DB.Where("order_id = ?", soID).Find(&soItems).Error; err != nil {
		return nil, fmt.Errorf("failed to load sales order items: %w", err)
	}
	soItemMap := make(map[uint]models.SalesOrderItem, len(soItems))
	for _, si := range soItems {
		soItemMap[si.ID] = si
	}
	for _, doItem := range items {
		if doItem.SalesOrderItemID == nil {
			continue
		}
		si, ok := soItemMap[*doItem.SalesOrderItemID]
		if !ok {
			return nil, fmt.Errorf("sales order item ID %d not found", *doItem.SalesOrderItemID)
		}
		if doItem.QuantityDelivered > si.OpenQty+0.001 {
			return nil, fmt.Errorf("quantity %.3f for item %s exceeds open quantity %.3f",
				doItem.QuantityDelivered, doItem.SKU, si.OpenQty)
		}
	}

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
		DeliveredByID:  createdByID,
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

// ConfirmDeliveryOrder sets the status to 'Delivered', updates SO.amount_delivered,
// decrements SO item open_qty, and auto-closes the SO if all lines are fulfilled.
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

		if err := tx.Model(&do).Updates(map[string]interface{}{
			"status":        "Delivered",
			"updated_by_id": userID,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.SalesOrder{}).Where("id = ?", do.SalesOrderID).
			UpdateColumn("amount_delivered", gorm.Expr("amount_delivered + ?", totalDelivered)).
			Error; err != nil {
			return err
		}

		// Decrement open_qty on each linked SO item — strict check, no silent clamp
		var soItemIDs []uint
		for _, doItem := range do.Items {
			if doItem.SalesOrderItemID != nil {
				soItemIDs = append(soItemIDs, *doItem.SalesOrderItemID)
			}
		}
		var currentSOItems []models.SalesOrderItem
		if len(soItemIDs) > 0 {
			if err := tx.Where("id IN ?", soItemIDs).Find(&currentSOItems).Error; err != nil {
				return fmt.Errorf("failed to verify open quantities: %w", err)
			}
		}
		soItemQtyMap := make(map[uint]float64, len(currentSOItems))
		for _, si := range currentSOItems {
			soItemQtyMap[si.ID] = si.OpenQty
		}
		for _, doItem := range do.Items {
			if doItem.SalesOrderItemID == nil {
				continue
			}
			currentQty := soItemQtyMap[*doItem.SalesOrderItemID]
			if doItem.QuantityDelivered > currentQty+0.001 {
				return fmt.Errorf("cannot confirm: qty %.3f for %s exceeds remaining open qty %.3f — the sales order may have been partially fulfilled by another delivery",
					doItem.QuantityDelivered, doItem.SKU, currentQty)
			}
			if err := tx.Model(&models.SalesOrderItem{}).
				Where("id = ?", *doItem.SalesOrderItemID).
				UpdateColumn("open_qty", gorm.Expr("open_qty - ?", doItem.QuantityDelivered)).
				Error; err != nil {
				return err
			}
		}

		// Auto-close SO if all items are fulfilled (open_qty = 0)
		var openCount int64
		tx.Model(&models.SalesOrderItem{}).
			Where("order_id = ? AND open_qty > 0.001", do.SalesOrderID).
			Count(&openCount)
		if openCount == 0 {
			tx.Model(&models.SalesOrder{}).Where("id = ? AND doc_status = 'Open'", do.SalesOrderID).
				Updates(map[string]interface{}{"doc_status": "Closed", "updated_by_id": userID})
		}

		// Decrement on_hand / OITW / OIVL for each delivered line
		var createdByID uint
		if userID != nil {
			createdByID = *userID
		}
		doDate := do.Date.Format("2006-01-02")
		for _, item := range do.Items {
			var oitm models.OITM
			if tx.Where("item_code = ? OR item_name = ?", item.SKU, item.SKU).First(&oitm).Error != nil {
				continue
			}
			uomEntry := item.UomEntry
			if uomEntry == 0 {
				uomEntry = oitm.IUoMEntry
			}
			invQty := invsvc.ConvertToInventoryQty(tx, &oitm, uomEntry, item.QuantityDelivered)
			tx.Model(&models.OITM{}).Where("item_code = ?", oitm.ItemCode).
				UpdateColumn("on_hand", gorm.Expr("on_hand - ?", invQty))
			tx.Model(&models.OITM{}).Where("item_code = ?", oitm.ItemCode).
				UpdateColumn("is_commited", gorm.Expr("GREATEST(0, is_commited - ?)", invQty))
			_ = invsvc.UpsertOITWOnHand(tx, oitm.ItemCode, "", oitm.DfltWh, -invQty)
			_ = invsvc.AppendOIVL(tx, oitm.ItemCode, oitm.ItemName, "", oitm.DfltWh,
				"DO", doDate, int(do.ID),
				0, invQty, oitm.AvgPrice, createdByID)
		}

		return nil
	})
}

// CancelDeliveryOrder cancels a Delivered DO and reverses the stock decrements.
func CancelDeliveryOrder(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var do models.DeliveryOrder
		if err := tx.Preload("Items").First(&do, id).Error; err != nil {
			return err
		}
		if do.Status == "Cancelled" {
			return fmt.Errorf("delivery order %s is already cancelled", do.DeliveryNumber)
		}

		if do.Status == "Delivered" {
			// Guard: no AR Invoices exist
			var invCount int64
			tx.Model(&models.ARInvoice{}).Where("delivery_order_id = ?", id).Count(&invCount)
			if invCount > 0 {
				return fmt.Errorf("cannot cancel %s: %d AR invoice(s) exist — cancel them first", do.DeliveryNumber, invCount)
			}

			// Restore SO open_qty for each item
			for _, doItem := range do.Items {
				if doItem.SalesOrderItemID != nil {
					tx.Model(&models.SalesOrderItem{}).
						Where("id = ?", *doItem.SalesOrderItemID).
						UpdateColumn("open_qty", gorm.Expr("open_qty + ?", doItem.QuantityDelivered))
				}
			}

			// Reopen SO if it was auto-closed
			tx.Model(&models.SalesOrder{}).
				Where("id = ? AND doc_status = 'Closed'", do.SalesOrderID).
				Updates(map[string]interface{}{"doc_status": "Open", "updated_by_id": userID})

			// Restore on_hand / OITW / OIVL
			var cancelUID uint
			if userID != nil {
				cancelUID = *userID
			}
			doDate := do.Date.Format("2006-01-02")
			for _, item := range do.Items {
				var oitm models.OITM
				if tx.Where("item_code = ? OR item_name = ?", item.SKU, item.SKU).First(&oitm).Error != nil {
					continue
				}
				uomEntry := item.UomEntry
				if uomEntry == 0 {
					uomEntry = oitm.IUoMEntry
				}
				invQty := invsvc.ConvertToInventoryQty(tx, &oitm, uomEntry, item.QuantityDelivered)
				tx.Model(&models.OITM{}).Where("item_code = ?", oitm.ItemCode).
					UpdateColumn("on_hand", gorm.Expr("on_hand + ?", invQty))
				tx.Model(&models.OITM{}).Where("item_code = ?", oitm.ItemCode).
					UpdateColumn("is_commited", gorm.Expr("is_commited + ?", invQty))
				_ = invsvc.UpsertOITWOnHand(tx, oitm.ItemCode, "", oitm.DfltWh, invQty)
				_ = invsvc.AppendOIVL(tx, oitm.ItemCode, oitm.ItemName, "", oitm.DfltWh,
					"DO_CANCEL", doDate, int(do.ID),
					invQty, 0, oitm.AvgPrice, cancelUID)
			}
		}

		return tx.Model(&do).Updates(map[string]interface{}{
			"status":        "Cancelled",
			"updated_by_id": userID,
		}).Error
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
// Base document is the Delivery Order (SAP B1 flow: SO → Delivery → AR Invoice).
// doID is required for DO-based invoices; omit for standalone invoices.
// soID is derived from the DO automatically — callers must not pass it.
func CreateARInvoice(
	doIDs []uint,
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

	// Validate Delivery Orders and derive SO reference
	var dos []models.DeliveryOrder
	var soID *uint
	if len(doIDs) > 0 {
		if err := db.DB.Preload("SalesOrder").Where("id IN ?", doIDs).Find(&dos).Error; err != nil {
			return nil, fmt.Errorf("fetch delivery orders: %w", err)
		}
		if len(dos) != len(doIDs) {
			return nil, fmt.Errorf("one or more delivery orders not found")
		}
		for _, do := range dos {
			if do.Status != "Delivered" {
				return nil, fmt.Errorf("delivery order %s is not Delivered (status: %s)", do.DeliveryNumber, do.Status)
			}
		}
		// Validate same customer across all DOs
		if len(dos) > 1 {
			firstCustID := dos[0].SalesOrder.CustomerID
			for _, do := range dos[1:] {
				if do.SalesOrder.CustomerID != firstCustID {
					return nil, fmt.Errorf("all delivery orders must belong to the same customer")
				}
			}
		}
		// Set soID only when all DOs share the same Sales Order
		firstSOID := dos[0].SalesOrderID
		allSameSO := true
		for _, do := range dos {
			if do.SalesOrderID != firstSOID {
				allSameSO = false
				break
			}
		}
		if allSameSO {
			soID = &firstSOID
		}
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
		DeliveryOrders:          dos,
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

		// Update SalesOrder.amount_invoiced (single SO case)
		if soID != nil {
			if err := tx.Model(&models.SalesOrder{}).Where("id = ?", *soID).
				UpdateColumn("amount_invoiced", gorm.Expr("amount_invoiced + ?", total)).
				Error; err != nil {
				return err
			}
		}

		// Update each DeliveryOrder.amount_invoiced based on its lines
		doAmounts := map[uint]float64{}
		for i := range inv.Items {
			if inv.Items[i].DeliveryOrderID != nil {
				doAmounts[*inv.Items[i].DeliveryOrderID] += inv.Items[i].LineTotal()
			}
		}
		for doID, amt := range doAmounts {
			if err := tx.Model(&models.DeliveryOrder{}).Where("id = ?", doID).
				Updates(map[string]interface{}{
					"amount_invoiced": gorm.Expr("amount_invoiced + ?", amt),
					"updated_by_id":   createdByID,
				}).Error; err != nil {
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
		Preload("DeliveryOrders.Items").
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
		if err := tx.Preload("Items").Preload("DeliveryOrders").First(&inv, id).Error; err != nil {
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

		// Reverse per-DO amount_invoiced based on line items
		doAmounts := map[uint]float64{}
		for i := range inv.Items {
			if inv.Items[i].DeliveryOrderID != nil {
				doAmounts[*inv.Items[i].DeliveryOrderID] += inv.Items[i].LineTotal()
			}
		}
		for doID, amt := range doAmounts {
			if err := tx.Model(&models.DeliveryOrder{}).Where("id = ?", doID).
				Updates(map[string]interface{}{
					"amount_invoiced": gorm.Expr("amount_invoiced - ?", amt),
					"updated_by_id":   userID,
				}).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&inv).Updates(map[string]interface{}{
			"status":        "Cancelled",
			"updated_by_id": userID,
			"version":       gorm.Expr("version + 1"),
		}).Error; err != nil {
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
		Status:               "Posted",
		CreatedAt:            time.Now(),
		CreatedByID:          createdByID,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(collection).Error; err != nil {
			return err
		}

		// Batch-fetch all AR invoices to avoid N+1 queries
		arInvIDs := make([]uint, len(lines))
		for i, l := range lines {
			arInvIDs[i] = l.ARInvoiceID
		}
		var arInvoices []models.ARInvoice
		if err := tx.Where("id IN ?", arInvIDs).Find(&arInvoices).Error; err != nil {
			return fmt.Errorf("fetch AR invoices: %w", err)
		}
		arInvMap := make(map[uint]*models.ARInvoice, len(arInvoices))
		for i := range arInvoices {
			arInvMap[arInvoices[i].ID] = &arInvoices[i]
		}

		// Batch-fetch all linked SalesOrders
		soIDSet := make(map[uint]bool)
		for i := range arInvoices {
			if arInvoices[i].SalesOrderID != nil {
				soIDSet[*arInvoices[i].SalesOrderID] = true
			}
		}
		soMap := make(map[uint]*models.SalesOrder)
		if len(soIDSet) > 0 {
			soIDs := make([]uint, 0, len(soIDSet))
			for id := range soIDSet {
				soIDs = append(soIDs, id)
			}
			var salesOrders []models.SalesOrder
			if err := tx.Where("id IN ?", soIDs).Find(&salesOrders).Error; err != nil {
				return fmt.Errorf("fetch sales orders: %w", err)
			}
			for i := range salesOrders {
				soMap[salesOrders[i].ID] = &salesOrders[i]
			}
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
			inv := arInvMap[l.ARInvoiceID]
			if inv == nil {
				return fmt.Errorf("AR invoice %d not found", l.ARInvoiceID)
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
				"updated_by_id":    createdByID,
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
				// Update SalesOrder.amount_paid and payment_status (use pre-fetched map)
				if so := soMap[*inv.SalesOrderID]; so != nil {
					newPaid := so.AmountPaid + l.AmountApplied
					soStatus := so.PaymentStatus
					if newPaid >= so.GrandTotal {
						soStatus = "Paid"
					} else if newPaid > 0 {
						soStatus = "Partial"
					}
					_ = tx.Model(so).Updates(map[string]interface{}{
						"amount_paid":    newPaid,
						"payment_status": soStatus,
						"updated_by_id":  createdByID,
					})
					so.AmountPaid = newPaid // keep local state consistent for multi-line same SO
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

// CancelCollection reverses a Posted collection:
//  - Restores amount_collected and status on each settled AR Invoice.
//  - Reverses amount_collected and payment_status on linked SalesOrders.
//  - Reverses the collection JE.
//  - Sets collection Status → Cancelled.
func CancelCollection(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var col models.Collection
		if err := tx.Preload("Lines").First(&col, id).Error; err != nil {
			return err
		}
		if col.Status == "Cancelled" {
			return fmt.Errorf("collection %s is already cancelled", col.CollectionNumber)
		}

		// Batch-fetch AR invoices touched by this collection
		arInvIDs := make([]uint, len(col.Lines))
		for i, l := range col.Lines {
			arInvIDs[i] = l.ARInvoiceID
		}
		var arInvoices []models.ARInvoice
		if err := tx.Where("id IN ?", arInvIDs).Find(&arInvoices).Error; err != nil {
			return fmt.Errorf("fetch AR invoices: %w", err)
		}
		arInvMap := make(map[uint]*models.ARInvoice, len(arInvoices))
		for i := range arInvoices {
			arInvMap[arInvoices[i].ID] = &arInvoices[i]
		}

		// Collect unique SalesOrder IDs for amount_collected reversal
		soIDSet := make(map[uint]bool)
		for i := range arInvoices {
			if arInvoices[i].SalesOrderID != nil {
				soIDSet[*arInvoices[i].SalesOrderID] = true
			}
		}

		// Reverse each AR invoice settlement
		for _, line := range col.Lines {
			inv := arInvMap[line.ARInvoiceID]
			if inv == nil {
				continue
			}
			newCollected := inv.AmountCollected - line.AmountApplied
			if newCollected < 0 {
				newCollected = 0
			}
			newStatus := "Open"
			if newCollected >= inv.TotalAmount {
				newStatus = "Paid"
			} else if newCollected > 0 {
				newStatus = "Partial"
			}
			if err := tx.Model(inv).Updates(map[string]interface{}{
				"amount_collected": newCollected,
				"status":           newStatus,
				"updated_by_id":    userID,
			}).Error; err != nil {
				return fmt.Errorf("restore AR invoice %d: %w", inv.ID, err)
			}
		}

		// Reverse amount_collected and payment_status on linked SalesOrders
		for soID := range soIDSet {
			// Sum up how much this collection applied against invoices on this SO
			var soApplied float64
			for _, line := range col.Lines {
				inv := arInvMap[line.ARInvoiceID]
				if inv != nil && inv.SalesOrderID != nil && *inv.SalesOrderID == soID {
					soApplied += line.AmountApplied
				}
			}
			var so models.SalesOrder
			if err := tx.First(&so, soID).Error; err != nil {
				continue
			}
			newPaid := so.AmountPaid - soApplied
			if newPaid < 0 {
				newPaid = 0
			}
			newStatus := "Unpaid"
			if newPaid >= so.GrandTotal {
				newStatus = "Paid"
			} else if newPaid > 0 {
				newStatus = "Partial"
			}
			if err := tx.Model(&so).Updates(map[string]interface{}{
				"amount_paid":       newPaid,
				"amount_collected":  gorm.Expr("GREATEST(0, amount_collected - ?)", soApplied),
				"payment_status":    newStatus,
				"updated_by_id":     userID,
			}).Error; err != nil {
				return fmt.Errorf("restore sales order %d: %w", soID, err)
			}
		}

		// Reverse the collection JE
		var originalJE models.JournalEntry
		if err := tx.Where("source_type = ? AND source_id = ?", "COLLECTION", col.ID).
			First(&originalJE).Error; err == nil {
			if _, err = accounting.ReverseJournalEntry(tx, &originalJE, nil, userID); err != nil {
				return fmt.Errorf("reverse collection JE: %w", err)
			}
		}

		// Mark collection as Cancelled
		return tx.Model(&col).Updates(map[string]interface{}{
			"status":        "Cancelled",
			"updated_by_id": userID,
		}).Error
	})
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
