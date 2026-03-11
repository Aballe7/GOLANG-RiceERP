package handlers

import (
	"egglayererp/app/middleware"
	"egglayererp/app/services/operations"
)

// RecordDailyLog saves a layer house daily log.
func RecordDailyLog(req operations.RecordDailyLogRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Operations") {
		return forbidden("Operations")
	}
	req.CreatedByID = middleware.Store.UserID()
	if err := operations.RecordDailyLog(req); err != nil {
		return errResponse(err)
	}
	return okResponse("Daily log recorded successfully", nil)
}

// RecordGrowerLog saves a grower house daily log.
func RecordGrowerLog(req operations.RecordGrowerLogRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Operations") {
		return forbidden("Operations")
	}
	req.CreatedByID = middleware.Store.UserID()
	if err := operations.RecordGrowerLog(req); err != nil {
		return errResponse(err)
	}
	return okResponse("Grower log recorded successfully", nil)
}

// UpdateDailyLog edits an existing layer log.
func UpdateDailyLog(id uint, req operations.RecordDailyLogRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	if err := operations.UpdateDailyLog(id, req); err != nil {
		return errResponse(err)
	}
	return okResponse("Daily log updated successfully", nil)
}

// UpdateGrowerLog edits an existing grower log.
func UpdateGrowerLog(id uint, req operations.RecordGrowerLogRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	if err := operations.UpdateGrowerLog(id, req); err != nil {
		return errResponse(err)
	}
	return okResponse("Grower log updated successfully", nil)
}

// GetDailyLog returns a single daily log by ID.
func GetDailyLog(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	log, err := operations.GetDailyLog(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", log)
}

// GetGrowerLog returns a single grower log by ID.
func GetGrowerLog(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	log, err := operations.GetGrowerLog(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", log)
}

// ListFeedStocks returns all feed stocks.
func ListFeedStocks() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	stocks, err := operations.ListFeedStocks()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", stocks)
}

// UpsertFeedStock creates or updates a feed stock entry.
func UpsertFeedStock(name string, totalSacks float64, kgPerSack float64) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Operations") {
		return forbidden("Operations")
	}
	if err := operations.UpsertFeedStock(name, totalSacks, kgPerSack); err != nil {
		return errResponse(err)
	}
	return okResponse("Feed stock updated successfully", nil)
}

// ListVaccineStocks returns all vaccine stocks.
func ListVaccineStocks() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	stocks, err := operations.ListVaccineStocks()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", stocks)
}

// ListVaccineSchedules returns vaccine schedules for a flock.
func ListVaccineSchedules(flockID uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	schedules, err := operations.ListVaccineSchedules(flockID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", schedules)
}

// CreateVaccineSchedule creates a new vaccine schedule entry.
func CreateVaccineSchedule(flockID uint, vaccineName, scheduledDate, adminMethod string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Operations") {
		return forbidden("Operations")
	}
	if err := operations.CreateVaccineSchedule(flockID, vaccineName, scheduledDate, adminMethod, middleware.Store.UserID()); err != nil {
		return errResponse(err)
	}
	return okResponse("Vaccine schedule created successfully", nil)
}

// MarkVaccineComplete marks a vaccine schedule as completed.
func MarkVaccineComplete(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Operations") {
		return forbidden("Operations")
	}
	if err := operations.MarkVaccineComplete(id); err != nil {
		return errResponse(err)
	}
	return okResponse("Vaccine marked as completed", nil)
}
