package handlers

import (
	"ricemill/app/middleware"
	"ricemill/app/models"
	"ricemill/app/services/purchasing"
	audit "ricemill/app/services/audit"
)

// All handler methods in this file require CanAccess("Purchasing").

// ListPurchasingLookup returns active dropdown values for a given category.
// No permission gate — read-only reference data needed even on form load.
func ListPurchasingLookup(category string) Response {
	rows, err := purchasing.ListPurchasingLookup(category)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", rows)
}

func checkPurchasing() (Response, bool) {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized(), false
	}
	if !middleware.Store.CanAccess("Purchasing") {
		return forbidden("Purchasing"), false
	}
	return Response{}, true
}

// ─────────────────────────────────────────────
// Supplier
// ─────────────────────────────────────────────

func ListSuppliers(activeOnly bool) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	suppliers, err := purchasing.ListSuppliers(activeOnly)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", suppliers)
}

func GetSupplier(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	s, err := purchasing.GetSupplier(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", s)
}

func CreateSupplier(s models.Supplier) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	if err := purchasing.CreateSupplier(&s); err != nil {
		return errResponse(err)
	}
	return okResponse("Supplier created", s)
}

func UpdateSupplier(id uint, updates map[string]interface{}) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	if err := purchasing.UpdateSupplier(id, updates); err != nil {
		return errResponse(err)
	}
	return okResponse("Supplier updated", nil)
}

func DeleteSupplier(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	if err := purchasing.DeleteSupplier(id); err != nil {
		return errResponse(err)
	}
	return okResponse("Supplier deactivated", nil)
}

func ApproveSupplier(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	if !middleware.Store.IsAdmin() {
		return errMsg("Only Admin users can approve suppliers.")
	}
	u := middleware.Store.CurrentUser()
	userID := middleware.Store.UserID()
	if err := purchasing.ApproveSupplier(id, u.FullName, userID); err != nil {
		return errResponse(err)
	}
	audit.Create("Purchasing", "Supplier", "", "Supplier approved and activated", id)
	return okResponse("Supplier approved and activated", nil)
}

func RejectSupplier(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	if !middleware.Store.IsAdmin() {
		return errMsg("Only Admin users can reject suppliers.")
	}
	if err := purchasing.RejectSupplier(id); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Purchasing", "Supplier", "", "Supplier rejected", id)
	return okResponse("Supplier rejected", nil)
}

// ─────────────────────────────────────────────
// Purchase Header (OPOR-aligned)
// ─────────────────────────────────────────────

// ListPurchaseHeaders returns all PurchaseHeader records (newest first).
func ListPurchaseHeaders() Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	headers, err := purchasing.ListPurchaseHeaders()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", headers)
}

// GetPurchaseHeader returns a fully-preloaded PurchaseHeader with all child documents.
func GetPurchaseHeader(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	h, err := purchasing.GetPurchaseHeader(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", h)
}

// ListOpenPurchaseHeaders returns PurchaseHeaders with open lines for DR creation.
// Pass supplierID = 0 to get all suppliers.
func ListOpenPurchaseHeaders(supplierID uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	data, err := purchasing.ListOpenPurchaseHeaders(supplierID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", data)
}

// CreatePurchaseHeader handles creation of a new PO with multiple line items.
func CreatePurchaseHeader(req purchasing.CreatePurchaseParams) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	h, err := purchasing.CreatePurchaseHeader(req, userID)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Purchasing", "Purchase", h.PONumber, "Purchase order created", h.ID)
	return okResponse("Purchase created", h)
}

// CancelPurchaseHeader cancels an open PO (guards against confirmed DRs).
func CancelPurchaseHeader(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := purchasing.CancelPurchaseHeader(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Purchasing", "Purchase", "", "Purchase order cancelled", id)
	return okResponse("Purchase order cancelled", nil)
}

// GetNextPONumber returns the next available PO number for UI preview.
func GetNextPONumber() Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	num, err := purchasing.NextPONumberForUI()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", num)
}

// ─────────────────────────────────────────────
// Delivery Receipt
// ─────────────────────────────────────────────

// DRLineInput is the DTO for a single DR line submitted from the frontend.
type DRLineInput struct {
	PurchaseHeaderID uint    `json:"purchase_header_id"` // PurchaseHeader.ID — stored in BaseDocEntry
	PurchaseLineID   uint    `json:"purchase_line_id"`   // PurchaseLine.ID   — stored in BaseLineNum for precise open_qty update
	Description      string  `json:"description"`
	Category         string  `json:"category"`
	Unit             string  `json:"unit"`
	UomEntry         uint    `json:"uom_entry"`
	QuantityOrdered  float64 `json:"quantity_ordered"`
	Quantity         float64 `json:"quantity"`
	Price            float64 `json:"price"`
}

type CreateDeliveryReceiptRequest struct {
	Date       string        `json:"date"`
	ReceivedBy string        `json:"received_by"`
	RefNumber  string        `json:"ref_number"`
	Comments   string        `json:"comments"`
	Lines      []DRLineInput `json:"lines"`
}

func CreateDeliveryReceipt(req CreateDeliveryReceiptRequest) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	if len(req.Lines) == 0 {
		return errMsg("At least one line item is required")
	}
	userID := middleware.Store.UserID()

	// Use first line's PurchaseHeaderID as the primary source PO.
	primaryHeaderID := req.Lines[0].PurchaseHeaderID

	// Convert DRLineInputs to model items; tag each with its source PurchaseHeader.
	items := make([]models.DeliveryReceiptItem, len(req.Lines))
	for i, l := range req.Lines {
		phid := l.PurchaseHeaderID
		var baseLine *int
		if l.PurchaseLineID > 0 {
			bl := int(l.PurchaseLineID)
			baseLine = &bl
		}
		items[i] = models.DeliveryReceiptItem{
			LineNum:         i,
			Description:     l.Description,
			Category:        l.Category,
			Unit:            l.Unit,
			UomEntry:        l.UomEntry,
			QuantityOrdered: l.QuantityOrdered,
			Quantity:        l.Quantity,
			Price:           l.Price,
			LineTotal:       l.Quantity * l.Price,
			LineStatus:      "O",
			BaseDocEntry:    &phid,
			BaseLineNum:     baseLine,
		}
	}

	dr, err := purchasing.CreateDeliveryReceipt(
		primaryHeaderID, req.Date, req.ReceivedBy,
		req.RefNumber, req.Comments, items, userID,
	)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Purchasing", "DeliveryReceipt", dr.DRNumber, "Delivery receipt created", dr.ID)
	return okResponse("Delivery receipt created", dr)
}

func GetDeliveryReceipt(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	dr, err := purchasing.GetDeliveryReceipt(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", dr)
}

func ListDeliveryReceipts(purchaseHeaderID uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	drs, err := purchasing.ListDeliveryReceipts(purchaseHeaderID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", drs)
}

func ConfirmDeliveryReceipt(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := purchasing.ConfirmDeliveryReceipt(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Confirm("Purchasing", "DeliveryReceipt", "", "Delivery receipt confirmed", id)
	return okResponse("Delivery receipt confirmed", nil)
}

func CancelDeliveryReceipt(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := purchasing.CancelDeliveryReceipt(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Purchasing", "DeliveryReceipt", "", "Delivery receipt cancelled", id)
	return okResponse("Delivery receipt cancelled", nil)
}

// ─────────────────────────────────────────────
// AP Invoice
// ─────────────────────────────────────────────

type CreateAPInvoiceRequest struct {
	PurchaseHeaderID uint                   `json:"purchase_header_id"`
	DRID             *uint                  `json:"dr_id"`
	Date             string                 `json:"date"`
	Terms            string                 `json:"terms"`
	SupplierName     string                 `json:"supplier_name"`
	RefNumber        string                 `json:"ref_number"`
	Comments         string                 `json:"comments"`
	DueDate          *string                `json:"due_date"`
	Lines            []models.APInvoiceItem `json:"lines"`
	// BIR EWT fields — optional; pass zero values when WHT does not apply
	VATExclusiveAmount float64 `json:"vat_exclusive_amount"`
	WHTRate            float64 `json:"wht_rate"`
	WHTAmount          float64 `json:"wht_amount"`
	WHTATCCode         string  `json:"wht_atc_code"`
}

func CreateAPInvoice(req CreateAPInvoiceRequest) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	inv, err := purchasing.CreateAPInvoice(
		req.PurchaseHeaderID, req.DRID, req.Date, req.Terms,
		req.SupplierName, req.RefNumber, req.Comments,
		req.DueDate, req.Lines, userID,
		req.VATExclusiveAmount, req.WHTRate, req.WHTAmount, req.WHTATCCode,
	)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Purchasing", "APInvoice", inv.InvoiceNumber, "AP invoice created for "+inv.SupplierName, inv.ID)
	return okResponse("AP invoice created", inv)
}

func GetAPInvoice(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	inv, err := purchasing.GetAPInvoice(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", inv)
}

func ListAPInvoices(purchaseHeaderID *uint, status string) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	invoices, err := purchasing.ListAPInvoices(purchasing.APInvoiceFilters{
		PurchaseHeaderID: purchaseHeaderID,
		Status:           status,
	})
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", invoices)
}

func CancelAPInvoice(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := purchasing.CancelAPInvoice(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Purchasing", "APInvoice", "", "AP invoice cancelled", id)
	return okResponse("AP invoice cancelled", nil)
}

// ─────────────────────────────────────────────
// AP Payment
// ─────────────────────────────────────────────

type CreateAPPaymentRequest struct {
	Date          string                          `json:"date"`
	SupplierID    *uint                           `json:"supplier_id"`
	SupplierName  string                          `json:"supplier_name"`
	PaymentMethod string                          `json:"payment_method"`
	RefNum        string                          `json:"ref_num"`
	Notes         string                          `json:"notes"`
	Lines         []purchasing.APPaymentLineInput `json:"lines"`
}

func CreateAPPayment(req CreateAPPaymentRequest) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	payment, err := purchasing.CreateAPPayment(
		req.Date, req.SupplierID, req.SupplierName,
		req.PaymentMethod, req.RefNum, req.Notes,
		req.Lines, userID,
	)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Purchasing", "APPayment", payment.PaymentNumber, "AP payment created for "+payment.SupplierNameSnapshot, payment.ID)
	return okResponse("AP payment created", payment)
}

func GetAPPayment(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	payment, err := purchasing.GetAPPayment(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", payment)
}

func ListAPPayments() Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	payments, err := purchasing.ListAPPayments()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", payments)
}

func CancelAPPayment(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := purchasing.CancelAPPayment(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Purchasing", "APPayment", "", "AP payment cancelled", id)
	return okResponse("AP payment cancelled", nil)
}
