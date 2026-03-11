package handlers

import (
	"egglayererp/app/middleware"
	"egglayererp/app/services/flocks"
	audit "egglayererp/app/services/audit"
)

// GetFlocks returns all flocks split into active and retired lists.
func GetFlocks() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	active, retired, err := flocks.ListFlocks()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", map[string]interface{}{
		"active":  active,
		"retired": retired,
	})
}

// GetFlock returns a single flock by ID.
func GetFlock(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	f, err := flocks.GetFlock(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", f)
}

// CreateFlock creates a new flock record.
func CreateFlock(req flocks.CreateFlockRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	req.CreatedByID = middleware.Store.UserID()
	f, err := flocks.CreateFlock(req)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Flocks", "Flock", f.Name, "Flock created in house "+f.HouseNumber, f.ID)
	return okResponse("Flock created successfully", f)
}

// UpdateFlock updates flock details.
func UpdateFlock(req flocks.UpdateFlockRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	if err := flocks.UpdateFlock(req); err != nil {
		return errResponse(err)
	}
	audit.Update("Flocks", "Flock", "", "Flock details updated", req.ID)
	return okResponse("Flock updated successfully", nil)
}

// TransferFlock moves a Grower flock to a Layer house.
func TransferFlock(req flocks.TransferFlockRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	if err := flocks.TransferFlock(req); err != nil {
		return errResponse(err)
	}
	audit.Update("Flocks", "Flock", "", "Flock transferred to house "+req.NewHouseNumber, req.ID)
	return okResponse("Flock transferred to Layer house successfully", nil)
}

// RetireFlock retires a flock.
func RetireFlock(req flocks.RetireFlockRequest) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	req.CreatedByID = middleware.Store.UserID()
	if err := flocks.RetireFlock(req); err != nil {
		return errResponse(err)
	}
	audit.Delete("Flocks", "Flock", "", "Flock retired", req.ID)
	return okResponse("Flock retired successfully", nil)
}

// GetLayerHouses returns all Layer flock house numbers.
func GetLayerHouses() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	houses, err := flocks.GetLayerHouses()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", houses)
}

// GetHouseHistory returns logs for a flock.
func GetHouseHistory(flockID uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	logs, houseType, err := flocks.GetHouseHistory(flockID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", map[string]interface{}{
		"logs":       logs,
		"house_type": houseType,
	})
}

// GetBodyWeightData returns body weight data for a flock.
func GetBodyWeightData(flockID uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	data, err := flocks.GetBodyWeightData(flockID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", data)
}

// AddBodyWeightLog records a body weight entry for a flock.
func AddBodyWeightLog(flockID uint, date string, weekAge int, avgWeightG float64, sampleSize int, notes string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	if err := flocks.AddBodyWeightLog(flockID, date, weekAge, avgWeightG, sampleSize, notes, middleware.Store.UserID()); err != nil {
		return errResponse(err)
	}
	return okResponse("Body weight recorded successfully", nil)
}

// UpdateBodyWeightLog edits an existing body weight log.
func UpdateBodyWeightLog(id uint, date string, avgWeightG float64, sampleSize int, notes string) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	if err := flocks.UpdateBodyWeightLog(id, date, avgWeightG, sampleSize, notes); err != nil {
		return errResponse(err)
	}
	return okResponse("Body weight updated successfully", nil)
}

// DeleteBodyWeightLog soft-deletes a body weight log.
func DeleteBodyWeightLog(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	if !middleware.Store.CanAccess("Flocks") {
		return forbidden("Flocks")
	}
	if err := flocks.DeleteBodyWeightLog(id); err != nil {
		return errResponse(err)
	}
	return okResponse("Body weight log deleted", nil)
}
