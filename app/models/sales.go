package models

import "time"

type Customer struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`
	Address         string    `gorm:"type:varchar(200)" json:"address"`
	DeliveryAddress string    `gorm:"type:varchar(200)" json:"delivery_address"`
	ContactNumber   string    `gorm:"type:varchar(20)" json:"contact_number"`
	Email           string    `gorm:"type:varchar(100)" json:"email"`
	CustomerType    string    `gorm:"type:varchar(20);default:'Walk-in'" json:"customer_type"` // Account | Walk-in
	Notes           string    `gorm:"type:text" json:"notes"`
	IsActive        bool      `gorm:"type:tinyint(1);default:1" json:"is_active"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	CreatedByID     *uint     `json:"created_by_id"`
	PriceGroupID    *uint     `gorm:"index" json:"price_group_id"`

	PriceGroup *PriceGroup   `gorm:"foreignKey:PriceGroupID" json:"price_group,omitempty"`
	Orders     []SalesOrder  `gorm:"foreignKey:CustomerID" json:"-"`
}

func (Customer) TableName() string { return "customer" }

type PriceGroup struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
	Description string    `gorm:"type:varchar(200)" json:"description"`
	IsActive    bool      `gorm:"type:tinyint(1);default:1" json:"is_active"`
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`

	Items     []PriceGroupItem `gorm:"foreignKey:PriceGroupID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	Customers []Customer       `gorm:"foreignKey:PriceGroupID" json:"-"`
}

func (PriceGroup) TableName() string { return "price_group" }

func (pg *PriceGroup) GetPrice(eggSize, unit string) *float64 {
	for _, item := range pg.Items {
		if item.EggSize == eggSize && item.Unit == unit {
			return &item.Price
		}
	}
	return nil
}

type PriceGroupItem struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	PriceGroupID uint      `gorm:"not null;index" json:"price_group_id"`
	EggSize      string    `gorm:"type:varchar(30);not null" json:"egg_size"`
	Unit         string    `gorm:"type:varchar(10);not null;default:'Tray'" json:"unit"`
	Price        float64   `gorm:"type:decimal(15,4);not null;default:0" json:"price"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (PriceGroupItem) TableName() string { return "pg_line" }

type SalesOrder struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	SalesOrderNumber        string    `gorm:"type:varchar(30);uniqueIndex;not null" json:"sales_order_number"`
	DocStatus               string    `gorm:"type:varchar(20);default:'Draft'" json:"doc_status"` // Draft | Open | Closed | Cancelled
	Date                    time.Time `gorm:"type:date;not null" json:"date"`
	CustomerID              *uint     `gorm:"index" json:"customer_id"`
	CustomerNameSnapshot    string    `gorm:"type:varchar(100)" json:"customer_name_snapshot"`
	CustomerAddressSnapshot string    `gorm:"type:varchar(200)" json:"customer_address_snapshot"`
	CustomerContactSnapshot string    `gorm:"type:varchar(20)" json:"customer_contact_snapshot"`
	PaymentMethod           string    `gorm:"type:varchar(20)" json:"payment_method"`
	PaymentStatus           string    `gorm:"type:varchar(20);default:'Unpaid'" json:"payment_status"` // Paid | Unpaid | Partial
	AmountPaid              float64   `gorm:"type:decimal(15,4);default:0" json:"amount_paid"`
	DueDate                 *time.Time `gorm:"type:date" json:"due_date"`
	Terms                   string    `gorm:"type:varchar(30)" json:"terms"`
	GrandTotal              float64   `gorm:"type:decimal(15,4);default:0" json:"grand_total"`
	AmountDelivered         float64   `gorm:"type:decimal(15,4);default:0" json:"amount_delivered"`
	AmountInvoiced          float64   `gorm:"type:decimal(15,4);default:0" json:"amount_invoiced"`
	AmountCollected         float64   `gorm:"type:decimal(15,4);default:0" json:"amount_collected"`
	Notes                   string    `gorm:"type:text" json:"notes"`
	CreatedAt               time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	CreatedByID             *uint     `json:"created_by_id"`
	UpdatedByID             *uint     `json:"updated_by_id"`
	Version                 uint      `gorm:"default:1" json:"version"`

	Customer   *Customer       `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Items      []SalesOrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	Deliveries []DeliveryOrder `gorm:"foreignKey:SalesOrderID" json:"deliveries,omitempty"`
	ARInvoices []ARInvoice     `gorm:"foreignKey:SalesOrderID" json:"ar_invoices,omitempty"`
}

func (SalesOrder) TableName() string { return "sales_order" }

type SalesOrderItem struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	OrderID      uint    `gorm:"not null;index" json:"order_id"`
	SKU          string  `gorm:"type:varchar(30);not null" json:"sku"`
	Unit         string  `gorm:"type:varchar(10);not null" json:"unit"`
	UomEntry     uint    `gorm:"column:uom_entry;not null;default:0" json:"uom_entry"`
	Quantity     float64 `gorm:"type:decimal(12,3);not null" json:"quantity"`
	OpenQty      float64 `gorm:"type:decimal(12,3);default:0" json:"open_qty"`
	PricePerUnit float64 `gorm:"type:decimal(15,4);not null" json:"price_per_unit"`
	LineTotal    float64 `gorm:"type:decimal(15,4);not null" json:"line_total"`
}

func (SalesOrderItem) TableName() string { return "so_line" }

type DeliveryOrder struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	DeliveryNumber string    `gorm:"type:varchar(30);uniqueIndex;not null" json:"delivery_number"`
	Date           time.Time `gorm:"type:date;not null" json:"date"`
	SalesOrderID   uint      `gorm:"not null;index" json:"sales_order_id"`
	Status         string    `gorm:"type:varchar(20);default:'Draft'" json:"status"` // Draft | Delivered | Cancelled
	DeliveredBy    string    `gorm:"type:varchar(100)" json:"delivered_by"` // snapshot of deliverer name
	DeliveredByID  *uint     `gorm:"index" json:"delivered_by_id"`
	Notes          string    `gorm:"type:text" json:"notes"`
	CreatedAt      time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	CreatedByID    *uint     `json:"created_by_id"`
	UpdatedByID    *uint     `json:"updated_by_id"`
	AmountInvoiced float64   `gorm:"type:decimal(15,4);default:0" json:"amount_invoiced"`

	SalesOrder *SalesOrder         `gorm:"foreignKey:SalesOrderID" json:"sales_order,omitempty"`
	Items      []DeliveryOrderItem `gorm:"foreignKey:DeliveryOrderID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	ARInvoices []ARInvoice         `gorm:"many2many:ar_invoice_delivery;" json:"ar_invoices,omitempty"`
}

func (DeliveryOrder) TableName() string { return "delivery_order" }

func (d *DeliveryOrder) TotalAmount() float64 {
	total := 0.0
	for _, item := range d.Items {
		total += item.LineTotal()
	}
	return total
}

type DeliveryOrderItem struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	DeliveryOrderID  uint    `gorm:"not null;index" json:"delivery_order_id"`
	SalesOrderItemID *uint   `gorm:"index" json:"sales_order_item_id"`
	SKU              string  `gorm:"type:varchar(30);not null" json:"sku"`
	Unit             string  `gorm:"type:varchar(10);not null" json:"unit"`
	UomEntry         uint    `gorm:"column:uom_entry;not null;default:0" json:"uom_entry"`
	QuantityOrdered  float64 `gorm:"type:decimal(12,3);default:0" json:"quantity_ordered"`
	QuantityDelivered float64 `gorm:"type:decimal(12,3);default:0" json:"quantity_delivered"`
	PricePerUnit     float64 `gorm:"type:decimal(15,4);default:0" json:"price_per_unit"`
}

func (DeliveryOrderItem) TableName() string { return "do_line" }

func (i *DeliveryOrderItem) LineTotal() float64 {
	return i.QuantityDelivered * i.PricePerUnit
}

type ARInvoice struct {
	ID                      uint       `gorm:"primaryKey" json:"id"`
	InvoiceNumber           string     `gorm:"type:varchar(30);uniqueIndex;not null" json:"invoice_number"`
	Date                    time.Time  `gorm:"type:date;not null" json:"date"`
	DueDate                 *time.Time `gorm:"type:date" json:"due_date"`
	Terms                   string     `gorm:"type:varchar(30)" json:"terms"`
	SalesOrderID            *uint      `gorm:"index" json:"sales_order_id"`
	CustomerID              *uint      `gorm:"index" json:"customer_id"`
	CustomerNameSnapshot    string     `gorm:"type:varchar(100)" json:"customer_name_snapshot"`
	CustomerAddressSnapshot string     `gorm:"type:varchar(200)" json:"customer_address_snapshot"`
	CustomerContactSnapshot string     `gorm:"type:varchar(20)" json:"customer_contact_snapshot"`
	TotalAmount             float64    `gorm:"type:decimal(15,4);default:0" json:"total_amount"`
	Status                  string     `gorm:"type:varchar(20);default:'Open'" json:"status"` // Open | Partial | Paid | Cancelled
	Notes                   string     `gorm:"type:text" json:"notes"`
	CreatedAt               time.Time  `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	CreatedByID             *uint      `json:"created_by_id"`
	UpdatedByID             *uint      `json:"updated_by_id"`
	AmountCollected         float64    `gorm:"type:decimal(15,4);default:0" json:"amount_collected"`
	Version                 uint       `gorm:"default:1" json:"version"`

	SalesOrder      *SalesOrder      `gorm:"foreignKey:SalesOrderID" json:"sales_order,omitempty"`
	DeliveryOrders  []DeliveryOrder  `gorm:"many2many:ar_invoice_delivery;" json:"delivery_orders,omitempty"`
	Customer        *Customer        `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Items           []ARInvoiceItem  `gorm:"foreignKey:ARInvoiceID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	CollectionLines []CollectionLine `gorm:"foreignKey:ARInvoiceID" json:"collection_lines,omitempty"`
}

func (ARInvoice) TableName() string { return "ar_invoice" }

func (a *ARInvoice) Balance() float64 {
	balance := a.TotalAmount - a.AmountCollected
	if balance < 0 {
		return 0
	}
	return balance
}

type ARInvoiceItem struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	ARInvoiceID     uint    `gorm:"not null;index" json:"ar_invoice_id"`
	DeliveryOrderID *uint   `gorm:"index" json:"delivery_order_id"` // which DO this line came from
	SKU             string  `gorm:"type:varchar(30);not null" json:"sku"`
	Unit            string  `gorm:"type:varchar(10);not null" json:"unit"`
	UomEntry        uint    `gorm:"column:uom_entry;not null;default:0" json:"uom_entry"`
	Quantity        float64 `gorm:"type:decimal(12,3);not null" json:"quantity"`
	PricePerUnit    float64 `gorm:"type:decimal(15,4);not null" json:"price_per_unit"`
}

func (ARInvoiceItem) TableName() string { return "ar_invoice_line" }

func (i *ARInvoiceItem) LineTotal() float64 {
	return i.Quantity * i.PricePerUnit
}

type Collection struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	CollectionNumber     string    `gorm:"type:varchar(30);uniqueIndex;not null" json:"collection_number"`
	Date                 time.Time `gorm:"type:date;not null" json:"date"`
	CustomerID           *uint     `gorm:"index" json:"customer_id"`
	CustomerNameSnapshot string    `gorm:"type:varchar(100)" json:"customer_name_snapshot"`
	TotalAmount          float64   `gorm:"type:decimal(15,4);default:0" json:"total_amount"`
	PaymentMethod        string    `gorm:"type:varchar(20);default:'Cash'" json:"payment_method"`
	ReferenceNumber      string    `gorm:"type:varchar(50)" json:"reference_number"`
	Notes                string    `gorm:"type:text" json:"notes"`
	Status               string    `gorm:"type:varchar(20);default:'Posted'" json:"status"` // Posted | Cancelled
	CreatedAt            time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	CreatedByID          *uint     `json:"created_by_id"`
	UpdatedByID          *uint     `json:"updated_by_id"`

	Customer *Customer        `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Lines    []CollectionLine `gorm:"foreignKey:CollectionID;constraint:OnDelete:CASCADE" json:"lines,omitempty"`
}

func (Collection) TableName() string { return "collection" }

type CollectionLine struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	CollectionID  uint    `gorm:"not null;index" json:"collection_id"`
	ARInvoiceID   uint    `gorm:"not null;index" json:"ar_invoice_id"`
	AmountApplied float64 `gorm:"type:decimal(15,4);not null" json:"amount_applied"`

	Collection *Collection `gorm:"foreignKey:CollectionID" json:"collection,omitempty"`
	ARInvoice  *ARInvoice  `gorm:"foreignKey:ARInvoiceID"  json:"ar_invoice,omitempty"`
}

func (CollectionLine) TableName() string { return "collection_line" }
