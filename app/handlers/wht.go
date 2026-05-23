// handlers/wht.go — BIR Expanded Withholding Tax (EWT) handlers.
// All endpoints require Admin role because WHT data is tax-compliance information.
package handlers

import (
	"time"

	"ricemill/app/middleware"
	"ricemill/app/services/wht"
)

func checkWHT() (Response, bool) {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized(), false
	}
	if !middleware.Store.IsAdmin() {
		return forbidden("WHT — Admin only"), false
	}
	return Response{}, true
}

// ComputeWHTRequest is used by the frontend to preview WHT before creating an AP Invoice.
type ComputeWHTRequest struct {
	SupplierID         uint    `json:"supplier_id"`
	WHTCategory        string  `json:"wht_category"`
	VATExclusiveAmount float64 `json:"vat_exclusive_amount"`
	GrossAmount        float64 `json:"gross_amount"`
	InvoiceDateStr     string  `json:"invoice_date"` // YYYY-MM-DD
}

// ComputeWHT returns the WHT calculation result for an AP invoice (preview only — no posting).
func ComputeWHT(req ComputeWHTRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	date, err := time.Parse("2006-01-02", req.InvoiceDateStr)
	if err != nil {
		date = time.Now()
	}
	result, err := wht.ComputeWHT(req.SupplierID, req.WHTCategory, req.VATExclusiveAmount, req.GrossAmount, date)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", result)
}

// ListWHTEntries returns all AP invoices with WHT applied.
// year=0 means all years; quarter=0 means all quarters.
func ListWHTEntries(year, quarter int) Response {
	if r, ok := checkWHT(); !ok {
		return r
	}
	entries, err := wht.ListWHTEntries(year, quarter)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", entries)
}

// GetForm0619E returns monthly remittance data for BIR Form 0619-E.
func GetForm0619E(year, month int) Response {
	if r, ok := checkWHT(); !ok {
		return r
	}
	data, err := wht.GenerateForm0619E(year, month)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", data)
}

// GetForm1601EQ returns quarterly return data for BIR Form 1601-EQ.
func GetForm1601EQ(year, quarter int) Response {
	if r, ok := checkWHT(); !ok {
		return r
	}
	data, err := wht.GenerateForm1601EQ(year, quarter)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", data)
}

// GetForm2307 returns BIR Form 2307 data for a specific supplier and quarter.
// company is the withholding agent's (rice mill) registered name.
func GetForm2307(supplierID uint, year, quarter int, company string) Response {
	if r, ok := checkWHT(); !ok {
		return r
	}
	data, err := wht.GenerateForm2307(supplierID, year, quarter, company)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", data)
}

// ListSuppliersWithWHT returns suppliers that have WHT entries in the given year.
func ListSuppliersWithWHT(year int) Response {
	if r, ok := checkWHT(); !ok {
		return r
	}
	suppliers, err := wht.ListSuppliersWithWHT(year)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", suppliers)
}
