package handlers

import (
	"egglayererp/app/middleware"
	"egglayererp/app/models"
	"egglayererp/app/services/purchasing"
	audit "egglayererp/app/services/audit"
)

// All handler methods in this file require CanAccess("Purchasing").

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

// ─────────────────────────────────────────────
// Purchase
// ─────────────────────────────────────────────

func ListPurchases() Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	purchases, err := purchasing.ListPurchases()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", purchases)
}

func GetPurchase(id uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	p, err := purchasing.GetPurchase(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", p)
}

func CreatePurchase(p models.Purchase) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	if err := purchasing.CreatePurchase(&p); err != nil {
		return errResponse(err)
	}
	return okResponse("Purchase created", p)
}

func UpdatePurchase(id uint, updates map[string]interface{}) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	if err := purchasing.UpdatePurchase(id, updates); err != nil {
		return errResponse(err)
	}
	return okResponse("Purchase updated", nil)
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

type CreateDeliveryReceiptRequest struct {
	PurchaseIDs   []uint                       `json:"purchase_ids"`
	Date          string                       `json:"date"`
	ReceivedBy    string                       `json:"received_by"`
	SupplierDRRef string                       `json:"supplier_dr_ref"`
	Notes         string                       `json:"notes"`
	Items         []models.DeliveryReceiptItem `json:"items"`
}

func CreateDeliveryReceipt(req CreateDeliveryReceiptRequest) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	if len(req.PurchaseIDs) == 0 {
		return errMsg("At least one purchase order must be selected")
	}
	userID := middleware.Store.UserID()
	var created []*models.DeliveryReceipt
	for _, pid := range req.PurchaseIDs {
		dr, err := purchasing.CreateDeliveryReceipt(
			pid, req.Date, req.ReceivedBy,
			req.SupplierDRRef, req.Notes, req.Items, userID,
		)
		if err != nil {
			return errResponse(err)
		}
		audit.Create("Purchasing", "DeliveryReceipt", dr.DRNumber, "Delivery receipt created", dr.ID)
		created = append(created, dr)
	}
	return okResponse("Delivery receipt(s) created", created)
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

func ListDeliveryReceipts(purchaseID uint) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	drs, err := purchasing.ListDeliveryReceipts(purchaseID)
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
	PurchaseID     uint                    `json:"purchase_id"`
	DRID           *uint                   `json:"dr_id"`
	Date           string                  `json:"date"`
	Terms          string                  `json:"terms"`
	SupplierName   string                  `json:"supplier_name"`
	SupplierInvRef string                  `json:"supplier_inv_ref"`
	Notes          string                  `json:"notes"`
	DueDate        *string                 `json:"due_date"`
	Items          []models.APInvoiceItem  `json:"items"`
}

func CreateAPInvoice(req CreateAPInvoiceRequest) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	inv, err := purchasing.CreateAPInvoice(
		req.PurchaseID, req.DRID, req.Date, req.Terms,
		req.SupplierName, req.SupplierInvRef, req.Notes,
		req.DueDate, req.Items, userID,
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

func ListAPInvoices(purchaseID *uint, status string) Response {
	if r, ok := checkPurchasing(); !ok {
		return r
	}
	invoices, err := purchasing.ListAPInvoices(purchasing.APInvoiceFilters{
		PurchaseID: purchaseID,
		Status:     status,
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
	Date          string                           `json:"date"`
	SupplierID    *uint                            `json:"supplier_id"`
	SupplierName  string                           `json:"supplier_name"`
	PaymentMethod string                           `json:"payment_method"`
	RefNum        string                           `json:"ref_num"`
	Notes         string                           `json:"notes"`
	Lines         []purchasing.APPaymentLineInput  `json:"lines"`
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
