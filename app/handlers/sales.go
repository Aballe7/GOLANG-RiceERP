package handlers

import (
	"ricemill/app/middleware"
	"ricemill/app/models"
	"ricemill/app/services/sales"
	audit "ricemill/app/services/audit"
)

// All handler methods in this file require CanAccess("Sales").

func checkSales() (Response, bool) {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized(), false
	}
	if !middleware.Store.CanAccess("Sales") {
		return forbidden("Sales"), false
	}
	return Response{}, true
}

// ─────────────────────────────────────────────
// Customer
// ─────────────────────────────────────────────

func ListCustomers(activeOnly bool) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	customers, err := sales.ListCustomers(activeOnly)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", customers)
}

func GetCustomer(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	c, err := sales.GetCustomer(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", c)
}

func CreateCustomer(c models.Customer) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	if err := sales.CreateCustomer(&c); err != nil {
		return errResponse(err)
	}
	return okResponse("Customer created", c)
}

func UpdateCustomer(id uint, updates map[string]interface{}) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	if err := sales.UpdateCustomer(id, updates); err != nil {
		return errResponse(err)
	}
	c, err := sales.GetCustomer(id)
	if err != nil {
		return okResponse("Customer updated", true)
	}
	return okResponse("Customer updated", c)
}

func DeleteCustomer(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	if err := sales.DeleteCustomer(id); err != nil {
		return errResponse(err)
	}
	return okResponse("Customer deactivated", nil)
}

// ─────────────────────────────────────────────
// PriceGroup
// ─────────────────────────────────────────────

func ListPriceGroups(activeOnly bool) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	groups, err := sales.ListPriceGroups(activeOnly)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", groups)
}

func GetPriceGroup(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	pg, err := sales.GetPriceGroup(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", pg)
}

func CreatePriceGroup(pg models.PriceGroup) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	if err := sales.CreatePriceGroup(&pg); err != nil {
		return errResponse(err)
	}
	return okResponse("Price group created", pg)
}

func UpdatePriceGroup(id uint, updates map[string]interface{}) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	if err := sales.UpdatePriceGroup(id, updates); err != nil {
		return errResponse(err)
	}
	return okResponse("Price group updated", nil)
}

func DeletePriceGroup(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	if err := sales.DeletePriceGroup(id); err != nil {
		return errResponse(err)
	}
	return okResponse("Price group deactivated", nil)
}

type UpsertPriceGroupItemRequest struct {
	PriceGroupID uint    `json:"price_group_id"`
	EggSize      string  `json:"egg_size"`
	Unit         string  `json:"unit"`
	Price        float64 `json:"price"`
}

func UpsertPriceGroupItem(req UpsertPriceGroupItemRequest) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	if err := sales.UpsertPriceGroupItem(req.PriceGroupID, req.EggSize, req.Unit, req.Price); err != nil {
		return errResponse(err)
	}
	return okResponse("Price group item updated", nil)
}

// ─────────────────────────────────────────────
// Sales Order
// ─────────────────────────────────────────────

type CreateSalesOrderRequest struct {
	Date            string                  `json:"date"`
	CustomerID      *uint                   `json:"customer_id"`
	CustomerName    string                  `json:"customer_name"`
	CustomerAddr    string                  `json:"customer_addr"`
	CustomerContact string                  `json:"customer_contact"`
	PaymentMethod   string                  `json:"payment_method"`
	Terms           string                  `json:"terms"`
	DueDate         *string                 `json:"due_date"`
	Notes           string                  `json:"notes"`
	Items           []models.SalesOrderItem `json:"items"`
}

func CreateSalesOrder(req CreateSalesOrderRequest) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	so, err := sales.CreateSalesOrder(
		req.Date, req.CustomerID,
		req.CustomerName, req.CustomerAddr, req.CustomerContact,
		req.PaymentMethod, req.Terms, req.DueDate, req.Notes,
		req.Items, userID,
	)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Sales", "SalesOrder", so.SalesOrderNumber, "Sales order created for "+so.CustomerNameSnapshot, so.ID)
	return okResponse("Sales order created", so)
}

func GetSalesOrder(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	so, err := sales.GetSalesOrder(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", so)
}

func ListSalesOrders() Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	orders, err := sales.ListSalesOrders()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", orders)
}

func VoidSalesOrder(id uint) Response {
	return CancelSalesOrder(id)
}

type UpdateSalesOrderRequest struct {
	Version         uint                    `json:"version"`
	Date            string                  `json:"date"`
	CustomerID      *uint                   `json:"customer_id"`
	CustomerName    string                  `json:"customer_name"`
	CustomerAddr    string                  `json:"customer_addr"`
	CustomerContact string                  `json:"customer_contact"`
	PaymentMethod   string                  `json:"payment_method"`
	Terms           string                  `json:"terms"`
	DueDate         *string                 `json:"due_date"`
	Notes           string                  `json:"notes"`
	Items           []models.SalesOrderItem `json:"items"`
}

func SubmitSalesOrder(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := sales.SubmitSalesOrder(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Confirm("Sales", "SalesOrder", "", "Sales order submitted", id)
	return okResponse("Sales order submitted", nil)
}

func UpdateSalesOrder(id uint, req UpdateSalesOrderRequest) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	so, err := sales.UpdateSalesOrder(
		id, req.Version,
		req.CustomerID,
		req.CustomerName, req.CustomerAddr, req.CustomerContact,
		req.Date, req.PaymentMethod, req.Terms, req.DueDate, req.Notes,
		req.Items, userID,
	)
	if err != nil {
		return errResponse(err)
	}
	audit.Update("Sales", "SalesOrder", so.SalesOrderNumber, "Sales order updated", so.ID)
	return okResponse("Sales order updated", so)
}

func CancelSalesOrder(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := sales.CancelSalesOrder(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Sales", "SalesOrder", "", "Sales order cancelled", id)
	return okResponse("Sales order cancelled", nil)
}

// ─────────────────────────────────────────────
// Delivery Order
// ─────────────────────────────────────────────

type CreateDeliveryOrderRequest struct {
	SalesOrderID uint                       `json:"sales_order_id"`
	Date         string                     `json:"date"`
	DeliveredBy  string                     `json:"delivered_by"`
	Notes        string                     `json:"notes"`
	Items        []models.DeliveryOrderItem `json:"items"`
}

func CreateDeliveryOrder(req CreateDeliveryOrderRequest) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	do, err := sales.CreateDeliveryOrder(
		req.SalesOrderID, req.Date, req.DeliveredBy, req.Notes, req.Items, userID,
	)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Sales", "DeliveryOrder", do.DeliveryNumber, "Delivery order created", do.ID)
	return okResponse("Delivery order created", do)
}

func ConfirmDeliveryOrder(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := sales.ConfirmDeliveryOrder(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Confirm("Sales", "DeliveryOrder", "", "Delivery order confirmed", id)
	return okResponse("Delivery order confirmed", nil)
}

func CancelDeliveryOrder(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := sales.CancelDeliveryOrder(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Sales", "DeliveryOrder", "", "Delivery order cancelled", id)
	return okResponse("Delivery order cancelled", nil)
}

func GetDeliveryOrder(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	do, err := sales.GetDeliveryOrder(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", do)
}

func ListDeliveryOrders(soID uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	orders, err := sales.ListDeliveryOrders(soID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", orders)
}

// ─────────────────────────────────────────────
// AR Invoice
// ─────────────────────────────────────────────

type CreateARInvoiceRequest struct {
	DeliveryOrderIDs []uint                  `json:"delivery_order_ids"`
	CustomerID       *uint                   `json:"customer_id"`
	CustomerName     string                  `json:"customer_name"`
	CustomerAddr     string                  `json:"customer_addr"`
	CustomerContact  string                  `json:"customer_contact"`
	Date             string                  `json:"date"`
	Terms            string                  `json:"terms"`
	Notes            string                  `json:"notes"`
	DueDate          *string                 `json:"due_date"`
	Items            []models.ARInvoiceItem  `json:"items"`
}

func CreateARInvoice(req CreateARInvoiceRequest) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	inv, err := sales.CreateARInvoice(
		req.DeliveryOrderIDs, req.CustomerID,
		req.CustomerName, req.CustomerAddr, req.CustomerContact,
		req.Date, req.Terms, req.Notes, req.DueDate,
		req.Items, userID,
	)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Sales", "ARInvoice", inv.InvoiceNumber, "AR invoice created for "+inv.CustomerNameSnapshot, inv.ID)
	return okResponse("AR invoice created", inv)
}

func GetARInvoice(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	inv, err := sales.GetARInvoice(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", inv)
}

func ListARInvoices(customerID *uint, soID *uint, status string) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	invoices, err := sales.ListARInvoices(sales.ARInvoiceFilters{
		CustomerID:   customerID,
		SalesOrderID: soID,
		Status:       status,
	})
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", invoices)
}

func CancelARInvoice(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := sales.CancelARInvoice(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Sales", "ARInvoice", "", "AR invoice cancelled", id)
	return okResponse("AR invoice cancelled", nil)
}

// ─────────────────────────────────────────────
// Collection
// ─────────────────────────────────────────────

type CreateCollectionRequest struct {
	Date          string                      `json:"date"`
	CustomerID    *uint                       `json:"customer_id"`
	CustomerName  string                      `json:"customer_name"`
	PaymentMethod string                      `json:"payment_method"`
	RefNum        string                      `json:"ref_num"`
	Notes         string                      `json:"notes"`
	Lines         []sales.CollectionLineInput `json:"lines"`
}

func CreateCollection(req CreateCollectionRequest) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	collection, err := sales.CreateCollection(
		req.Date, req.CustomerID, req.CustomerName,
		req.PaymentMethod, req.RefNum, req.Notes,
		req.Lines, userID,
	)
	if err != nil {
		return errResponse(err)
	}
	audit.Create("Sales", "Collection", collection.CollectionNumber, "Collection created for "+collection.CustomerNameSnapshot, collection.ID)
	return okResponse("Collection created", collection)
}

func CancelCollection(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	userID := middleware.Store.UserID()
	if err := sales.CancelCollection(id, userID); err != nil {
		return errResponse(err)
	}
	audit.Cancel("Sales", "Collection", "", "Collection cancelled", id)
	return okResponse("Collection cancelled", nil)
}

func GetCollection(id uint) Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	collection, err := sales.GetCollection(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", collection)
}

func ListCollections() Response {
	if r, ok := checkSales(); !ok {
		return r
	}
	collections, err := sales.ListCollections()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", collections)
}
