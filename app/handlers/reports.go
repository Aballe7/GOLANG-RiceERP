package handlers

import (
	"time"

	"ricemill/app/middleware"
	"ricemill/app/services/reports"
)

// GetPnL returns the Profit & Loss report for the given date range.
func GetPnL(startDate, endDate string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Reports") {
		return forbidden("Reports")
	}
	report, err := reports.GetPnL(startDate, endDate)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", report)
}

// GetBalanceSheet returns the Balance Sheet as of the given date.
func GetBalanceSheet(asOf string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Reports") {
		return forbidden("Reports")
	}
	if asOf == "" {
		asOf = time.Now().Format("2006-01-02")
	}
	report, err := reports.GetBalanceSheet(asOf)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", report)
}

// GetARAgingReport returns the AR Aging report.
func GetARAgingReport() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Reports") {
		return forbidden("Reports")
	}
	rows, err := reports.GetARAgingReport()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", rows)
}

// GetTrialBalance returns an adjusted Trial Balance for the given date range.
func GetTrialBalance(startDate, endDate string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Reports") {
		return forbidden("Reports")
	}
	report, err := reports.GetTrialBalance(startDate, endDate)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", report)
}

// GetAPAgingReport returns the AP Aging report.
func GetAPAgingReport() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Reports") {
		return forbidden("Reports")
	}
	rows, err := reports.GetAPAgingReport()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", rows)
}

// GetCashFlowStatement returns the PAS 7 Statement of Cash Flows for the given period.
func GetCashFlowStatement(startDate, endDate string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Reports") {
		return forbidden("Reports")
	}
	data, err := reports.GetCashFlowStatement(startDate, endDate)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", data)
}

// GetAnalytics returns analytics data for the given date range.
func GetAnalytics(from, to string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Reports") {
		return forbidden("Reports")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	data, err := reports.GetAnalytics(from, to)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", data)
}

// ─────────────────────── Inventory Valuation Reports ─────────────────────────

// GetInventoryValuationReport returns a snapshot of stock quantities and values.
//   - asOf       : "YYYY-MM-DD" — defaults to today if empty; historical query if past
//   - whsCode    : warehouse filter; "" = all warehouses
//   - itmGrpCod  : item group filter; 0 = all groups
func GetInventoryValuationReport(asOf, whsCode string, itmGrpCod int) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Reports") {
		return forbidden("Reports")
	}
	report, err := reports.GetInventoryValuationSnapshot(asOf, whsCode, itmGrpCod)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", report)
}

// GetInventoryMovementReport returns paginated OIVL ledger entries matching the filters.
//   - dateFrom / dateTo : date range (YYYY-MM-DD)
//   - itemCode          : "" = all items
//   - whsCode           : "" = all warehouses
//   - transType         : "" = all types; e.g. "DR", "GR", "DO"
func GetInventoryMovementReport(dateFrom, dateTo, itemCode, whsCode, transType string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Reports") {
		return forbidden("Reports")
	}
	report, err := reports.GetInventoryMovementReport(dateFrom, dateTo, itemCode, whsCode, transType)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", report)
}
