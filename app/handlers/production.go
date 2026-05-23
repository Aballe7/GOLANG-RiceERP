package handlers

import (
	"ricemill/app/middleware"
	"ricemill/app/models"
	productionsvc "ricemill/app/services/production"
)

// All handler methods in this file require CanAccess("Production").

func checkProduction() (Response, bool) {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized(), false
	}
	if !middleware.Store.CanAccess("Production") {
		return forbidden("Production"), false
	}
	return Response{}, true
}

// ─────────────────────────────────────────────
// Milling Orders
// ─────────────────────────────────────────────

func ListMillingOrders() Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	list, err := productionsvc.ListMillingOrders()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

func GetMillingOrder(id uint) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	mo, err := productionsvc.GetMillingOrder(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", mo)
}

func CreateMillingOrder(req productionsvc.CreateMillingOrderRequest) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	mo, err := productionsvc.CreateMillingOrder(req, userID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("Milling order created", mo)
}

func StartMillingOrder(id uint) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := productionsvc.StartMillingOrder(id, userID); err != nil {
		return errResponse(err)
	}
	mo, err := productionsvc.GetMillingOrder(id)
	if err != nil {
		return okResponse("Milling order started", nil)
	}
	return okResponse("Milling order started", mo)
}

func CompleteMillingOrder(id uint, req productionsvc.CompleteMillingRequest) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	result, err := productionsvc.CompleteMillingOrder(id, req, userID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("Milling order completed", result)
}

func CancelMillingOrder(id uint) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := productionsvc.CancelMillingOrder(id, userID); err != nil {
		return errResponse(err)
	}
	mo, err := productionsvc.GetMillingOrder(id)
	if err != nil {
		return okResponse("Milling order cancelled", nil)
	}
	return okResponse("Milling order cancelled", mo)
}

// ─────────────────────────────────────────────
// BOM (OITT / ITT1)
// ─────────────────────────────────────────────

func ListBOMs() Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	boms, err := productionsvc.ListBOMs()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", boms)
}

func GetBOM(code string) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	bom, err := productionsvc.GetBOM(code)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", bom)
}

func UpsertBOM(bom models.ProductTree) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	if err := productionsvc.UpsertBOM(&bom); err != nil {
		return errResponse(err)
	}
	return okResponse("BOM saved", &bom)
}

func DeleteBOM(code string) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	if err := productionsvc.DeleteBOM(code); err != nil {
		return errResponse(err)
	}
	return okResponse("BOM deleted", nil)
}

// ─────────────────────────────────────────────
// Work Orders (OWOR / WOR1)
// ─────────────────────────────────────────────

func ListWorkOrders() Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	list, err := productionsvc.ListWorkOrders()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", list)
}

func GetWorkOrder(docEntry uint) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	wo, err := productionsvc.GetWorkOrder(docEntry)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", wo)
}

func CreateWorkOrder(req productionsvc.CreateWORequest) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	wo, err := productionsvc.CreateWorkOrder(req, userID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("Work order created", wo)
}

func ReleaseWorkOrder(docEntry uint) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := productionsvc.ReleaseWorkOrder(docEntry, userID); err != nil {
		return errResponse(err)
	}
	return okResponse("Work order released", nil)
}

func CloseWorkOrder(docEntry uint, cmpltQty, rjctQty float64) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := productionsvc.CloseWorkOrder(docEntry, userID, cmpltQty, rjctQty); err != nil {
		return errResponse(err)
	}
	return okResponse("Work order closed", nil)
}

func CancelWorkOrder(docEntry uint) Response {
	if r, ok := checkProduction(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := productionsvc.CancelWorkOrder(docEntry, userID); err != nil {
		return errResponse(err)
	}
	return okResponse("Work order cancelled", nil)
}
