package handlers

import (
	"time"

	"egglayererp/app/middleware"
	"egglayererp/app/services/reports"
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
