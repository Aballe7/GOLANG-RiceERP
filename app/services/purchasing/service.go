package purchasing

// SAP B1-aligned P2P purchasing service.
// Document chain: PurchaseHeader (OPOR) → DeliveryReceipt (OPDN) → APInvoice (OPCH) → APPayment (OVPM)
// The old single-row "Purchase" table has been fully removed. All document links are at line level.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ricemill/app/db"
	"ricemill/app/models"
	"ricemill/app/services/accounting"
	"ricemill/app/services/docnumber"
	invsvc "ricemill/app/services/inventory"

	"gorm.io/gorm"
)

// tinRegexp validates Philippine TIN formats.
var tinRegexp = regexp.MustCompile(`^(\d{3}-\d{3}-\d{3}(-\d{3,5})?)$|^\d{9,12}$`)

const minQtyDelta = 0.005

// ─────────────────────────────────────────────
// Supplier CRUD
// ─────────────────────────────────────────────

func ListPurchasingLookup(category string) ([]models.PurchasingLookup, error) {
	var rows []models.PurchasingLookup
	err := db.DB.Where("category = ? AND is_active = 1", category).
		Order("sort_order, value").Find(&rows).Error
	return rows, err
}

func ListSuppliers(activeOnly bool) ([]models.Supplier, error) {
	var suppliers []models.Supplier
	q := db.DB.Order("name")
	if activeOnly {
		q = q.Where("is_active = 1")
	}
	return suppliers, q.Find(&suppliers).Error
}

func GetSupplier(id uint) (*models.Supplier, error) {
	var s models.Supplier
	err := db.DB.First(&s, id).Error
	return &s, err
}

func CreateSupplier(s *models.Supplier) error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("supplier name is required")
	}
	tin := strings.TrimSpace(s.TINNumber)
	if tin != "" {
		if !tinRegexp.MatchString(tin) {
			return fmt.Errorf("invalid TIN format — expected NNN-NNN-NNN or NNN-NNN-NNN-NNN")
		}
		var count int64
		db.DB.Model(&models.Supplier{}).Where("tin_number = ?", tin).Count(&count)
		if count > 0 {
			return fmt.Errorf("a supplier with TIN '%s' already exists", tin)
		}
		s.TINNumber = tin
	}
	s.Status = "pending"
	s.IsActive = false
	s.CreatedAt = time.Now()
	return db.DB.Create(s).Error
}

func UpdateSupplier(id uint, updates map[string]interface{}) error {
	for _, key := range []string{"status", "is_active", "approved_by", "approved_at"} {
		delete(updates, key)
	}
	return db.DB.Model(&models.Supplier{}).Where("id = ?", id).Updates(updates).Error
}

func ApproveSupplier(id uint, approverName string, userID *uint) error {
	now := time.Now()
	result := db.DB.Model(&models.Supplier{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":         "active",
		"is_active":      true,
		"approved_by":    approverName,
		"approved_by_id": userID,
		"approved_at":    now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("supplier not found")
	}
	return nil
}

func RejectSupplier(id uint) error {
	result := db.DB.Model(&models.Supplier{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":    "rejected",
		"is_active": false,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("supplier not found")
	}
	return nil
}

func DeleteSupplier(id uint) error {
	return db.DB.Model(&models.Supplier{}).Where("id = ?", id).
		Update("is_active", false).Error
}

// ─────────────────────────────────────────────
// Purchase Order (PurchaseHeader / PurchaseLine)
// ─────────────────────────────────────────────

// POLineInput is the DTO for a single PO line submitted from the UI.
type POLineInput struct {
	ItemCode  string  `json:"item_code"`
	Category  string  `json:"category"`
	ItemName  string  `json:"item_name"`
	Unit      string  `json:"unit"`
	IUoMEntry uint    `json:"i_uom_entry"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// CreatePurchaseParams holds header + lines for a new PO.
type CreatePurchaseParams struct {
	Date          string        `json:"date"`
	SupplierID    *uint         `json:"supplier_id"`
	SupplierName  string        `json:"supplier"`
	PaymentMethod string        `json:"payment_method"`
	Remarks       string        `json:"remarks"`
	PONumber      string        `json:"po_number"`
	Lines         []POLineInput `json:"lines"`
}

func nextPONumber() (string, error) {
	var last string
	if err := db.DB.
		Model(&models.PurchaseHeader{}).
		Where("po_number <> ''").
		Order("id DESC").
		Limit(1).
		Pluck("po_number", &last).Error; err != nil {
		return "", err
	}

	seq := 1
	if last != "" {
		code := strings.TrimSpace(last)
		if strings.HasPrefix(code, "PO-") {
			code = strings.TrimPrefix(code, "PO-")
		}
		code = strings.ReplaceAll(code, "-", "")
		if n, err := strconv.Atoi(code); err == nil && n >= 0 {
			seq = n + 1
		}
	}

	digits := fmt.Sprintf("%08d", seq)
	return fmt.Sprintf("PO-%s-%s", digits[:3], digits[3:]), nil
}

func NextPONumberForUI() (string, error) {
	return nextPONumber()
}

// ListPurchaseHeaders returns all PO headers with their lines, newest first.
func ListPurchaseHeaders() ([]models.PurchaseHeader, error) {
	var headers []models.PurchaseHeader
	err := db.DB.
		Preload("Supplier").
		Preload("Lines").
		Order("id DESC").
		Find(&headers).Error
	return headers, err
}

// GetPurchaseHeader returns a single PO header with full detail:
// lines, delivery receipts (with lines + AP invoices), and AP invoices.
func GetPurchaseHeader(id uint) (*models.PurchaseHeader, error) {
	var h models.PurchaseHeader
	err := db.DB.
		Preload("Supplier").
		Preload("Lines").
		Preload("DeliveryReceipts.Lines").
		Preload("DeliveryReceipts.APInvoices").
		Preload("APInvoices.Lines").
		Preload("APInvoices.PaymentLines.Payment").
		First(&h, id).Error
	return &h, err
}

// CreatePurchaseHeader inserts a new PurchaseHeader with its PurchaseLines.
func CreatePurchaseHeader(params CreatePurchaseParams, createdByID *uint) (*models.PurchaseHeader, error) {
	if len(params.Lines) == 0 {
		return nil, fmt.Errorf("at least one line item is required")
	}

	parsedDate, err := time.Parse(time.RFC3339, params.Date)
	if err != nil {
		// Try plain date format as fallback
		parsedDate, err = time.Parse("2006-01-02", params.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %w", err)
		}
	}

	poNum := strings.TrimSpace(params.PONumber)
	if poNum == "" {
		poNum, err = nextPONumber()
		if err != nil {
			return nil, fmt.Errorf("failed to generate PO number: %w", err)
		}
	}

	var grandTotal float64
	for _, l := range params.Lines {
		grandTotal += l.Quantity * l.UnitPrice
	}

	var header *models.PurchaseHeader
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		h := &models.PurchaseHeader{
			PONumber:      poNum,
			Status:        "Open",
			SupplierID:    params.SupplierID,
			SupplierName:  params.SupplierName,
			PostingDate:   parsedDate,
			Currency:      "PHP",
			DocTotal:      grandTotal,
			Comments:      params.Remarks,
			PaymentMethod: params.PaymentMethod,
			CreatedByID:   createdByID,
		}
		if err := tx.Create(h).Error; err != nil {
			return err
		}

		for idx, l := range params.Lines {
			// Validate category is enabled for purchasing
			if l.Category != "" {
				var cat models.OITB
				if err := tx.Where("itms_grp_nam = ?", l.Category).First(&cat).Error; err == nil && !cat.ForPurchasing {
					return fmt.Errorf("category '%s' is not enabled for Purchasing", l.Category)
				}
			}
			pl := &models.PurchaseLine{
				HeaderID:    h.ID,
				LineNum:     idx,
				ItemCode:    l.ItemCode,
				Description: l.ItemName,
				Unit:        l.Unit,
				IUoMEntry:   l.IUoMEntry,
				Quantity:    l.Quantity,
				OpenQty:     l.Quantity,
				Price:       l.UnitPrice,
				LineTotal:   l.Quantity * l.UnitPrice,
				LineStatus:  "O",
			}
			if err := tx.Create(pl).Error; err != nil {
				return err
			}
		}

		header = h
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Reload with lines so callers get accurate IDs for the created PurchaseLines.
	return GetPurchaseHeader(header.ID)
}

// CancelPurchaseHeader cancels a PO. Guard: no confirmed DRs may exist.
func CancelPurchaseHeader(headerID uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var h models.PurchaseHeader
		if err := tx.Preload("Lines").First(&h, headerID).Error; err != nil {
			return fmt.Errorf("purchase order not found: %w", err)
		}
		if h.Status == "Cancelled" {
			return fmt.Errorf("purchase order %s is already cancelled", h.PONumber)
		}

		// Guard: no confirmed DRs
		var confirmedCount int64
		tx.Model(&models.DeliveryReceipt{}).
			Where("purchase_header_id = ? AND status = 'Received'", headerID).
			Count(&confirmedCount)
		if confirmedCount > 0 {
			return fmt.Errorf("cannot cancel PO %s: %d confirmed delivery receipt(s) exist", h.PONumber, confirmedCount)
		}

		// Close all lines
		for _, line := range h.Lines {
			tx.Model(&models.PurchaseLine{}).Where("id = ?", line.ID).
				Updates(map[string]interface{}{"line_status": "C", "open_qty": 0})
		}

		return tx.Model(&h).Updates(map[string]interface{}{
			"status":        "Cancelled",
			"updated_by_id": userID,
			"version":       gorm.Expr("version + 1"),
		}).Error
	})
}

// ─── Open PO DTOs for DR creation ────────────────────────────────────────────

// OpenPOLineDTO is a flat view of one open PurchaseLine.
type OpenPOLineDTO struct {
	ID          uint    `json:"id"`
	LineNum     int     `json:"line_num"`
	Description string  `json:"description"`
	Unit        string  `json:"unit"`
	IUoMEntry   uint    `json:"i_uom_entry"`
	Quantity    float64 `json:"quantity"`
	OpenQty     float64 `json:"open_qty"`
	Price       float64 `json:"price"`
}

// OpenPOHeaderDTO groups open lines by their PurchaseHeader for the DR creation form.
type OpenPOHeaderDTO struct {
	ID           uint            `json:"id"` // PurchaseHeader.ID
	PONumber     string          `json:"po_number"`
	SupplierID   *uint           `json:"supplier_id"`
	SupplierName string          `json:"supplier_name"`
	PostingDate  time.Time       `json:"posting_date"`
	Status       string          `json:"status"`
	Lines        []OpenPOLineDTO `json:"lines"`
}

// ListOpenPurchaseHeaders returns PurchaseHeaders with open lines (open_qty > 0).
// If supplierID > 0 only that supplier's headers are returned.
func ListOpenPurchaseHeaders(supplierID uint) ([]OpenPOHeaderDTO, error) {
	var headers []models.PurchaseHeader
	q := db.DB.Where("status NOT IN ('Cancelled','Closed')")
	if supplierID > 0 {
		q = q.Where("supplier_id = ?", supplierID)
	}
	if err := q.Preload("Lines", "open_qty > 0.001").Order("id DESC").Find(&headers).Error; err != nil {
		return nil, err
	}

	result := make([]OpenPOHeaderDTO, 0, len(headers))
	for _, h := range headers {
		if len(h.Lines) == 0 {
			continue
		}
		lines := make([]OpenPOLineDTO, 0, len(h.Lines))
		for _, l := range h.Lines {
			if l.OpenQty <= minQtyDelta {
				continue
			}
			lines = append(lines, OpenPOLineDTO{
				ID:          l.ID,
				LineNum:     l.LineNum,
				Description: l.Description,
				Unit:        l.Unit,
				IUoMEntry:   l.IUoMEntry,
				Quantity:    l.Quantity,
				OpenQty:     l.OpenQty,
				Price:       l.Price,
			})
		}
		if len(lines) == 0 {
			continue
		}
		result = append(result, OpenPOHeaderDTO{
			ID:           h.ID,
			PONumber:     h.PONumber,
			SupplierID:   h.SupplierID,
			SupplierName: h.SupplierName,
			PostingDate:  h.PostingDate,
			Status:       h.Status,
			Lines:        lines,
		})
	}
	return result, nil
}

// ─────────────────────────────────────────────
// Delivery Receipt (OPDN / PDN1)
// ─────────────────────────────────────────────

// CreateDeliveryReceipt creates a new DR (status=Draft) linked to a PurchaseHeader.
// purchaseHeaderID is the primary source PO at the header level; line-level source
// is carried on each DeliveryReceiptItem via BaseDocEntry + BaseLineNum.
func CreateDeliveryReceipt(
	purchaseHeaderID uint,
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

	// Denormalise supplier info from the PurchaseHeader
	var ph models.PurchaseHeader
	if err := db.DB.First(&ph, purchaseHeaderID).Error; err != nil {
		return nil, fmt.Errorf("purchase order not found: %w", err)
	}

	// Guard: only one Draft DR allowed per PO at a time
	var draftCount int64
	if err := db.DB.Model(&models.DeliveryReceipt{}).
		Where("purchase_header_id = ? AND status = 'Draft'", purchaseHeaderID).
		Count(&draftCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check existing draft delivery receipts: %w", err)
	}
	if draftCount > 0 {
		return nil, fmt.Errorf("a draft delivery receipt already exists for this purchase order — confirm or cancel it first")
	}

	// Per-item quantity validation: requested qty must not exceed current open_qty
	var poLines []models.PurchaseLine
	if err := db.DB.Where("header_id = ?", purchaseHeaderID).Find(&poLines).Error; err != nil {
		return nil, fmt.Errorf("failed to load purchase order lines: %w", err)
	}
	poLineMap := make(map[uint]models.PurchaseLine, len(poLines))
	for _, pl := range poLines {
		poLineMap[pl.ID] = pl
	}
	for _, drItem := range items {
		if drItem.BaseLineNum == nil {
			continue
		}
		pl, ok := poLineMap[uint(*drItem.BaseLineNum)]
		if !ok {
			continue
		}
		if drItem.Quantity > pl.OpenQty+0.001 {
			return nil, fmt.Errorf("quantity %.3f for item %s exceeds open quantity %.3f",
				drItem.Quantity, drItem.ItemCode, pl.OpenQty)
		}
	}

	phID := purchaseHeaderID
	dr := &models.DeliveryReceipt{
		DRNumber:         drNum,
		PostingDate:      parsedDate,
		PurchaseHeaderID: &phID,
		SupplierID:       ph.SupplierID,
		SupplierName:     ph.SupplierName,
		Currency:         "PHP",
		Status:           "Draft",
		ReceivedBy:       receivedBy,
		ReceivedByID:     createdByID,
		RefNumber:        supplierDRRef,
		Comments:         notes,
		CreatedAt:        time.Now(),
		CreatedByID:      createdByID,
		Lines:            items,
	}

	if err := db.DB.Create(dr).Error; err != nil {
		return nil, err
	}
	return dr, nil
}

// GetDeliveryReceipt fetches a DR with its lines, source PO header, and AP invoices.
func GetDeliveryReceipt(id uint) (*models.DeliveryReceipt, error) {
	var dr models.DeliveryReceipt
	err := db.DB.
		Preload("Lines").
		Preload("PurchaseHeader").
		Preload("APInvoices").
		First(&dr, id).Error
	return &dr, err
}

// ListDeliveryReceipts returns DRs. If purchaseHeaderID > 0, filter by that PO.
func ListDeliveryReceipts(purchaseHeaderID uint) ([]models.DeliveryReceipt, error) {
	var drs []models.DeliveryReceipt
	q := db.DB.Preload("Lines").Preload("PurchaseHeader").Order("id DESC")
	if purchaseHeaderID != 0 {
		q = q.Where("purchase_header_id = ?", purchaseHeaderID)
	}
	return drs, q.Find(&drs).Error
}

// ConfirmDeliveryReceipt marks a DR as Received, decrements PurchaseLine open_qty,
// and posts the GRNI journal entry (Dr Inventory / Cr GRNI).
func ConfirmDeliveryReceipt(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var dr models.DeliveryReceipt
		if err := tx.Preload("Lines").First(&dr, id).Error; err != nil {
			return err
		}
		if dr.Status != "Draft" {
			return fmt.Errorf("delivery receipt %s is already %s", dr.DRNumber, dr.Status)
		}

		totalReceived := 0.0
		for _, item := range dr.Lines {
			totalReceived += item.ReceivedTotal()
		}

		// Mark as Received
		if err := tx.Model(&dr).Updates(map[string]interface{}{
			"status":        "Received",
			"updated_by_id": userID,
		}).Error; err != nil {
			return err
		}

		// Set open_qty on each DR line (full qty is uninvoiced at this point)
		for _, line := range dr.Lines {
			if err := tx.Model(&models.DeliveryReceiptItem{}).Where("id = ?", line.ID).
				Update("open_qty", line.Quantity).Error; err != nil {
				return fmt.Errorf("set dr_line open_qty: %w", err)
			}
		}

		// Decrement open_qty on source PurchaseLines — strict check, no silent clamp
		// Pre-fetch current open_qty values inside the transaction for validation.
		var plIDs []uint
		for _, line := range dr.Lines {
			if line.BaseLineNum != nil && *line.BaseLineNum > 0 {
				plIDs = append(plIDs, uint(*line.BaseLineNum))
			}
		}
		var currentPOLines []models.PurchaseLine
		if len(plIDs) > 0 {
			if err := tx.Where("id IN ?", plIDs).Find(&currentPOLines).Error; err != nil {
				return fmt.Errorf("failed to verify open quantities: %w", err)
			}
		}
		poLineQtyMap := make(map[uint]float64, len(currentPOLines))
		for _, pl := range currentPOLines {
			poLineQtyMap[pl.ID] = pl.OpenQty
		}

		poHeadersSeen := map[uint]bool{}
		for _, line := range dr.Lines {
			if line.BaseLineNum != nil && *line.BaseLineNum > 0 {
				plID := uint(*line.BaseLineNum)
				currentQty := poLineQtyMap[plID]
				if line.Quantity > currentQty+0.001 {
					return fmt.Errorf("cannot confirm: qty %.3f for %s exceeds remaining open qty %.3f — the purchase order may have been partially received by another receipt",
						line.Quantity, line.ItemCode, currentQty)
				}
				if err := tx.Model(&models.PurchaseLine{}).
					Where("id = ?", plID).
					UpdateColumn("open_qty", gorm.Expr("open_qty - ?", line.Quantity)).Error; err != nil {
					return fmt.Errorf("decrement purchase_line open_qty: %w", err)
				}
				var pl models.PurchaseLine
				if err := tx.Select("header_id").First(&pl, plID).Error; err == nil {
					poHeadersSeen[pl.HeaderID] = true
				}
			} else if line.BaseDocEntry != nil {
				// Fallback: BaseDocEntry = PurchaseHeader.ID, match by description
				if err := tx.Model(&models.PurchaseLine{}).
					Where("header_id = ? AND description = ? AND open_qty > 0", *line.BaseDocEntry, line.Description).
					UpdateColumn("open_qty", gorm.Expr("open_qty - ?", line.Quantity)).Error; err != nil {
					return fmt.Errorf("decrement purchase_line open_qty (fallback): %w", err)
				}
				poHeadersSeen[*line.BaseDocEntry] = true
			}
		}

		// Auto-close PurchaseHeader when all lines are fully received
		for phID := range poHeadersSeen {
			var remaining int64
			tx.Model(&models.PurchaseLine{}).
				Where("header_id = ? AND open_qty > ? AND line_status = 'O'", phID, minQtyDelta).
				Count(&remaining)
			if remaining == 0 {
				tx.Model(&models.PurchaseHeader{}).Where("id = ?", phID).Updates(map[string]interface{}{
					"status":        "Closed",
					"updated_by_id": userID,
				})
			}
		}

		// Update on_hand / OITW / OIVL for each received line
		var createdByID uint
		if userID != nil {
			createdByID = *userID
		}
		for _, item := range dr.Lines {
			var oitm models.OITM
			if tx.Where("item_code = ? OR item_name = ?", item.ItemCode, item.Description).First(&oitm).Error != nil {
				continue
			}
			uomEntry := item.UomEntry
			if uomEntry == 0 {
				uomEntry = oitm.IUoMEntry
			}
			invQty := invsvc.ConvertToInventoryQty(tx, &oitm, uomEntry, item.Quantity)

			// Convert line price to per-inventory-unit cost
			// e.g. 2 SACK × ₱2500/SACK → invQty=100 KG → ₱50/KG
			invUnitPrice := item.Price
			if invQty > 0 && item.Quantity > 0 {
				invUnitPrice = (item.Quantity * item.Price) / invQty
			}

			// Recalc moving average BEFORE incrementing on_hand
			newAvg := invsvc.RecalcMovingAvgPrice(tx, &oitm, invQty, invUnitPrice)

			tx.Model(&models.OITM{}).Where("item_code = ?", oitm.ItemCode).
				UpdateColumn("on_hand", gorm.Expr("on_hand + ?", invQty))
			_ = invsvc.UpsertOITWOnHand(tx, oitm.ItemCode, item.WarehouseCode, oitm.DfltWh, invQty)
			_ = invsvc.AppendOIVL(tx, oitm.ItemCode, oitm.ItemName, item.WarehouseCode, oitm.DfltWh,
				"DR", dr.PostingDate.Format("2006-01-02"), int(dr.ID),
				invQty, 0, newAvg, createdByID)
		}

		// Post GRNI journal entry: Dr Inventory / Cr GRNI
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

// CancelDeliveryReceipt cancels a DR. Confirmed DRs are reversed if no AP Invoices exist.
func CancelDeliveryReceipt(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var dr models.DeliveryReceipt
		if err := tx.Preload("Lines").First(&dr, id).Error; err != nil {
			return err
		}
		if dr.Status == "Cancelled" {
			return fmt.Errorf("delivery receipt %s is already cancelled", dr.DRNumber)
		}

		if dr.Status == "Received" {
			// Guard: no AP Invoices
			var invCount int64
			tx.Model(&models.APInvoice{}).Where("delivery_receipt_id = ?", id).Count(&invCount)
			if invCount > 0 {
				return fmt.Errorf("cannot cancel %s: %d AP invoice(s) exist — cancel them first", dr.DRNumber, invCount)
			}

			// Restore PurchaseLine open_qty
			poHeadersSeen := map[uint]bool{}
			for _, line := range dr.Lines {
				tx.Model(&models.DeliveryReceiptItem{}).Where("id = ?", line.ID).Update("open_qty", 0)

				if line.BaseLineNum != nil && *line.BaseLineNum > 0 {
					plID := uint(*line.BaseLineNum)
					tx.Model(&models.PurchaseLine{}).Where("id = ?", plID).
						UpdateColumn("open_qty", gorm.Expr("open_qty + ?", line.Quantity))
					tx.Model(&models.PurchaseLine{}).Where("id = ?", plID).Update("line_status", "O")
					var pl models.PurchaseLine
					if tx.Select("header_id").First(&pl, plID).Error == nil {
						poHeadersSeen[pl.HeaderID] = true
					}
				} else if line.BaseDocEntry != nil {
					tx.Model(&models.PurchaseLine{}).
						Where("header_id = ? AND description = ?", *line.BaseDocEntry, line.Description).
						Updates(map[string]interface{}{
							"open_qty":    gorm.Expr("open_qty + ?", line.Quantity),
							"line_status": "O",
						})
					poHeadersSeen[*line.BaseDocEntry] = true
				}
			}

			// Reopen PurchaseHeader if it was auto-closed
			for phID := range poHeadersSeen {
				tx.Model(&models.PurchaseHeader{}).
					Where("id = ? AND status = 'Closed'", phID).
					Updates(map[string]interface{}{
						"status":        "Open",
						"updated_by_id": userID,
					})
			}

			// Reverse stock: decrement on_hand / OITW / OIVL for each line
			var cancelUID uint
			if userID != nil {
				cancelUID = *userID
			}
			for _, item := range dr.Lines {
				var oitm models.OITM
				if tx.Where("item_code = ? OR item_name = ?", item.ItemCode, item.Description).First(&oitm).Error != nil {
					continue
				}
				uomEntry := item.UomEntry
				if uomEntry == 0 {
					uomEntry = oitm.IUoMEntry
				}
				invQty := invsvc.ConvertToInventoryQty(tx, &oitm, uomEntry, item.Quantity)
				tx.Model(&models.OITM{}).Where("item_code = ?", oitm.ItemCode).
					UpdateColumn("on_hand", gorm.Expr("on_hand - ?", invQty))
				_ = invsvc.UpsertOITWOnHand(tx, oitm.ItemCode, item.WarehouseCode, oitm.DfltWh, -invQty)
				_ = invsvc.AppendOIVL(tx, oitm.ItemCode, oitm.ItemName, item.WarehouseCode, oitm.DfltWh,
					"DR_CANCEL", dr.PostingDate.Format("2006-01-02"), int(dr.ID),
					0, invQty, oitm.AvgPrice, cancelUID)
			}

			// Reverse GRNI journal entry
			var originalJE models.JournalEntry
			if tx.Where("source_type = ? AND source_id = ?", "DR_CONFIRM", dr.ID).
				First(&originalJE).Error == nil {
				if _, err := accounting.ReverseJournalEntry(tx, &originalJE, nil, userID); err != nil {
					return fmt.Errorf("reverse GRNI entry: %w", err)
				}
			}
		}

		return tx.Model(&dr).Updates(map[string]interface{}{
			"status":        "Cancelled",
			"updated_by_id": userID,
		}).Error
	})
}

// ─────────────────────────────────────────────
// AP Invoice (OPCH / PCH1)
// ─────────────────────────────────────────────

// CreateAPInvoice creates an AP Invoice linked to a PurchaseHeader (and optionally a DR).
// Posts: Dr GRNI / Cr AP Payable (+ Cr WHT Payable when whtAmount > 0).
func CreateAPInvoice(
	purchaseHeaderID uint,
	drID *uint,
	date, terms, supplierName, supplierInvRef, notes string,
	dueDate *string,
	items []models.APInvoiceItem,
	createdByID *uint,
	vatExclusiveAmount float64,
	whtRate float64,
	whtAmount float64,
	whtATCCode string,
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

	total := 0.0
	for _, item := range items {
		total += item.Quantity * item.Price
	}

	var invBaseDocNum *int
	if drID != nil {
		var dr models.DeliveryReceipt
		if db.DB.Select("doc_num").First(&dr, *drID).Error == nil && dr.DocNum != 0 {
			invBaseDocNum = &dr.DocNum
		}
	}

	netPayable := total
	if whtAmount > 0 {
		netPayable = total - whtAmount
	}

	phID := purchaseHeaderID
	inv := &models.APInvoice{
		InvoiceNumber:      apNum,
		PostingDate:        parsedDate,
		DueDate:            parsedDue,
		Terms:              terms,
		PurchaseHeaderID:   &phID,
		DeliveryReceiptID:  drID,
		BaseDocEntry:       drID,
		BaseDocNum:         invBaseDocNum,
		SupplierName:       supplierName,
		RefNumber:          supplierInvRef,
		DocTotal:           total,
		Status:             "Open",
		Comments:           notes,
		CreatedAt:          time.Now(),
		CreatedByID:        createdByID,
		Lines:              items,
		VATExclusiveAmount: vatExclusiveAmount,
		WHTRate:            whtRate,
		WHTAmount:          whtAmount,
		WHTATCCode:         whtATCCode,
		NetPayable:         netPayable,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(inv).Error; err != nil {
			return err
		}

		// Update DR.doc_total (amount_invoiced) when linked to a DR
		if drID != nil {
			if err := tx.Model(&models.DeliveryReceipt{}).Where("id = ?", *drID).
				Updates(map[string]interface{}{
					"amount_invoiced": gorm.Expr("amount_invoiced + ?", total),
					"updated_by_id":   createdByID,
				}).Error; err != nil {
				return err
			}
		}

		// Decrement open_qty on DR lines; close lines when fully invoiced
		for _, item := range inv.Lines {
			if item.DRLineID == nil {
				continue
			}
			if err := tx.Model(&models.DeliveryReceiptItem{}).Where("id = ?", *item.DRLineID).
				UpdateColumn("open_qty", gorm.Expr("open_qty - ?", item.Quantity)).
				Error; err != nil {
				return fmt.Errorf("decrement dr_line open_qty: %w", err)
			}
			// Close DR line if fully invoiced
			var drLine models.DeliveryReceiptItem
			if err := tx.First(&drLine, *item.DRLineID).Error; err == nil && drLine.OpenQty <= minQtyDelta {
				tx.Model(&drLine).Update("line_status", "C")
				// Close the source PurchaseLine if fully received AND fully invoiced
				if drLine.BaseLineNum != nil && *drLine.BaseLineNum > 0 {
					tx.Model(&models.PurchaseLine{}).
						Where("id = ? AND open_qty <= ?", *drLine.BaseLineNum, minQtyDelta).
						Update("line_status", "C")
				}
			}
		}

		// Post journal entry
		apAcct, err := accounting.GetAccount(tx, "PURCHASING", "AP_PAYABLE")
		if err != nil {
			return err
		}
		var debitAcct *models.GLAccount
		if drID != nil {
			debitAcct, err = accounting.GetAccount(tx, "PURCHASING", "GRNI")
		} else {
			debitAcct, err = accounting.GetAccount(tx, "PURCHASING", "INVENTORY_RECEIVED")
		}
		if err != nil {
			return err
		}

		jeLines := []accounting.JELine{
			{Account: debitAcct, Debit: total, Description: "GRNI cleared / inventory received"},
		}
		if whtAmount > 0 {
			whtAcct, whtErr := accounting.GetAccount(tx, "PURCHASING", "AP_WHT_PAYABLE")
			if whtErr != nil {
				return whtErr
			}
			jeLines = append(jeLines,
				accounting.JELine{Account: apAcct,  Credit: netPayable, Description: "Accounts Payable (net of WHT)"},
				accounting.JELine{Account: whtAcct, Credit: whtAmount,  Description: fmt.Sprintf("WHT Payable — %s %.0f%%", whtATCCode, whtRate*100)},
			)
		} else {
			jeLines = append(jeLines,
				accounting.JELine{Account: apAcct, Credit: total, Description: "Accounts Payable"},
			)
		}

		_, err = accounting.PostJournalEntry(tx, accounting.PostJEParams{
			Module:      "PURCHASING",
			SourceType:  "AP_INVOICE",
			SourceID:    &inv.ID,
			SourceRef:   inv.InvoiceNumber,
			Narration:   "AP Invoice: " + inv.InvoiceNumber + " — " + supplierName,
			CreatedByID: createdByID,
			Lines:       jeLines,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// GetAPInvoice fetches an AP Invoice with lines, payment lines, DR, and source PO header.
func GetAPInvoice(id uint) (*models.APInvoice, error) {
	var inv models.APInvoice
	err := db.DB.
		Preload("Lines").
		Preload("PaymentLines.Payment").
		Preload("DeliveryReceipt").
		Preload("PurchaseHeader").
		First(&inv, id).Error
	return &inv, err
}

// APInvoiceFilters holds optional filter parameters for ListAPInvoices.
type APInvoiceFilters struct {
	PurchaseHeaderID *uint
	Status           string
}

// ListAPInvoices returns AP invoices, optionally filtered by PO header or status.
func ListAPInvoices(filters APInvoiceFilters) ([]models.APInvoice, error) {
	var invoices []models.APInvoice
	q := db.DB.Preload("Lines").Order("id DESC")
	if filters.PurchaseHeaderID != nil {
		q = q.Where("purchase_header_id = ?", *filters.PurchaseHeaderID)
	}
	if filters.Status != "" {
		q = q.Where("status = ?", filters.Status)
	}
	return invoices, q.Find(&invoices).Error
}

// CancelAPInvoice reverses the AP accounting JE and sets the invoice to Cancelled.
func CancelAPInvoice(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var inv models.APInvoice
		if err := tx.Preload("Lines").First(&inv, id).Error; err != nil {
			return err
		}
		if inv.Status == "Cancelled" {
			return fmt.Errorf("AP invoice %s is already cancelled", inv.InvoiceNumber)
		}
		if inv.AmountPaidStored > 0 {
			return fmt.Errorf("AP invoice %s has payments applied; cannot cancel directly", inv.InvoiceNumber)
		}

		// Restore open_qty on DR lines; reopen PurchaseLines closed by this invoice
		for _, line := range inv.Lines {
			if line.DRLineID == nil {
				continue
			}
			var drLine models.DeliveryReceiptItem
			if err := tx.First(&drLine, *line.DRLineID).Error; err != nil {
				return fmt.Errorf("load dr_line for cancel: %w", err)
			}
			if err := tx.Model(&drLine).Updates(map[string]interface{}{
				"open_qty":    gorm.Expr("open_qty + ?", line.Quantity),
				"line_status": "O",
			}).Error; err != nil {
				return fmt.Errorf("restore dr_line open_qty: %w", err)
			}
			// Reopen source PurchaseLine
			if drLine.BaseLineNum != nil && *drLine.BaseLineNum > 0 {
				tx.Model(&models.PurchaseLine{}).
					Where("id = ? AND line_status = 'C'", *drLine.BaseLineNum).
					Update("line_status", "O")
			}
		}

		// Restore DR.doc_total (amount_invoiced)
		if inv.DeliveryReceiptID != nil {
			tx.Model(&models.DeliveryReceipt{}).Where("id = ?", *inv.DeliveryReceiptID).
				Updates(map[string]interface{}{
					"amount_invoiced": gorm.Expr("amount_invoiced - ?", inv.DocTotal),
					"updated_by_id":   userID,
				})
		}

		if err := tx.Model(&inv).Updates(map[string]interface{}{
			"status":        "Cancelled",
			"updated_by_id": userID,
			"version":       gorm.Expr("version + 1"),
		}).Error; err != nil {
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
// AP Payment (OVPM / VPM2)
// ─────────────────────────────────────────────

// APPaymentLineInput is the DTO for a single payment line.
type APPaymentLineInput struct {
	APInvoiceID   uint    `json:"ap_invoice_id"`
	AmountApplied float64 `json:"amount_applied"`
}

// CreateAPPayment creates an APPayment, settles the nominated invoices,
// and posts Dr AP / Cr Cash journal entry.
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

		// Batch-fetch invoices
		apInvIDs := make([]uint, len(lines))
		for i, l := range lines {
			apInvIDs[i] = l.APInvoiceID
		}
		var apInvoices []models.APInvoice
		if err := tx.Where("id IN ?", apInvIDs).Find(&apInvoices).Error; err != nil {
			return fmt.Errorf("fetch AP invoices: %w", err)
		}
		apInvMap := make(map[uint]*models.APInvoice, len(apInvoices))
		for i := range apInvoices {
			apInvMap[apInvoices[i].ID] = &apInvoices[i]
		}

		for _, l := range lines {
			line := models.APPaymentLine{
				PaymentID:     payment.ID,
				APInvoiceID:   l.APInvoiceID,
				AmountApplied: l.AmountApplied,
			}
			if err := tx.Create(&line).Error; err != nil {
				return fmt.Errorf("create payment line for invoice %d: %w", l.APInvoiceID, err)
			}

			inv := apInvMap[l.APInvoiceID]
			if inv == nil {
				return fmt.Errorf("AP invoice %d not found", l.APInvoiceID)
			}
			newPaid := inv.AmountPaidStored + l.AmountApplied
			newStatus := inv.Status
			if newPaid >= inv.DocTotal-minQtyDelta {
				newStatus = "Paid"
			} else if newPaid > 0 {
				newStatus = "Partial"
			}
			if err := tx.Model(inv).Updates(map[string]interface{}{
				"amount_paid_stored": newPaid,
				"status":             newStatus,
				"updated_by_id":      createdByID,
			}).Error; err != nil {
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

// CancelAPPayment voids a payment, restoring linked AP Invoice statuses and reversing the JE.
func CancelAPPayment(id uint, userID *uint) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var pay models.APPayment
		if err := tx.Preload("Lines").First(&pay, id).Error; err != nil {
			return err
		}
		if pay.Status == "Cancelled" {
			return fmt.Errorf("payment %s is already cancelled", pay.PaymentNumber)
		}

		for _, line := range pay.Lines {
			var inv models.APInvoice
			if err := tx.First(&inv, line.APInvoiceID).Error; err != nil {
				return fmt.Errorf("load invoice %d: %w", line.APInvoiceID, err)
			}
			newPaid := inv.AmountPaidStored - line.AmountApplied
			if newPaid < 0 {
				newPaid = 0
			}
			newStatus := "Open"
			if newPaid > minQtyDelta {
				newStatus = "Partial"
			}
			if err := tx.Model(&inv).Updates(map[string]interface{}{
				"amount_paid_stored": newPaid,
				"status":             newStatus,
				"updated_by_id":      userID,
			}).Error; err != nil {
				return fmt.Errorf("restore invoice %d: %w", line.APInvoiceID, err)
			}
		}

		var originalJE models.JournalEntry
		if tx.Where("source_type = ? AND source_id = ?", "AP_PAYMENT", pay.ID).
			First(&originalJE).Error == nil {
			if _, err := accounting.ReverseJournalEntry(tx, &originalJE, nil, userID); err != nil {
				return fmt.Errorf("reverse payment JE: %w", err)
			}
		}

		return tx.Model(&pay).Updates(map[string]interface{}{
			"status":        "Cancelled",
			"updated_by_id": userID,
		}).Error
	})
}

// ListAPPayments returns all AP payments, newest first.
func ListAPPayments() ([]models.APPayment, error) {
	var payments []models.APPayment
	err := db.DB.Preload("Lines").Order("id DESC").Find(&payments).Error
	return payments, err
}
