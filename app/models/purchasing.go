package models

import "time"

type Supplier struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"type:varchar(150);not null" json:"name"`
	SupplierType  string    `gorm:"type:varchar(30);default:'Regular'" json:"supplier_type"`
	ContactPerson string    `gorm:"type:varchar(100)" json:"contact_person"`
	ContactNumber string    `gorm:"type:varchar(20)" json:"contact_number"`
	Email         string    `gorm:"type:varchar(100)" json:"email"`
	Address       string    `gorm:"type:varchar(200)" json:"address"`
	TINNumber     string    `gorm:"type:varchar(30)" json:"tin_number"`
	PaymentTerms  string    `gorm:"type:varchar(30);default:'COD'" json:"payment_terms"`
	BankDetails   string    `gorm:"type:text" json:"bank_details"`
	Categories    string    `gorm:"type:varchar(200)" json:"categories"`
	Notes         string    `gorm:"type:text" json:"notes"`
	IsActive      bool      `gorm:"type:tinyint(1);default:1" json:"is_active"`
	CreatedAt     time.Time `gorm:"type:date" json:"created_at"`
}

func (Supplier) TableName() string { return "supplier" }

type Purchase struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Date           time.Time `gorm:"type:date;not null" json:"date"`
	Category       string    `gorm:"type:varchar(50)" json:"category"`
	ItemName       string    `gorm:"type:varchar(100)" json:"item_name"`
	Quantity       float64   `gorm:"type:decimal(12,3)" json:"quantity"`
	Unit           string    `gorm:"type:varchar(20)" json:"unit"`
	UnitPrice      float64   `gorm:"type:decimal(15,4);default:0" json:"unit_price"`
	TotalCost      float64   `gorm:"type:decimal(15,4)" json:"total_cost"`
	Supplier       string    `gorm:"type:varchar(100)" json:"supplier"`
	SupplierID     *uint     `json:"supplier_id"`
	PONumber       string    `gorm:"type:varchar(50)" json:"po_number"`
	InvoiceNumber  string    `gorm:"type:varchar(50)" json:"invoice_number"`
	ReceivedBy     string    `gorm:"type:varchar(100)" json:"received_by"`
	PaymentStatus  string    `gorm:"type:varchar(20);default:'Paid'" json:"payment_status"`
	AmountPaid     float64   `gorm:"type:decimal(15,4);default:0" json:"amount_paid"`
	PaymentMethod  string    `gorm:"type:varchar(20);default:'Cash'" json:"payment_method"`
	Remarks        string    `gorm:"type:text" json:"remarks"`
	CreatedByID    *uint     `json:"created_by_id"`
	AmountReceived float64   `gorm:"type:decimal(15,4);default:0" json:"amount_received"`
	AmountInvoiced float64   `gorm:"type:decimal(15,4);default:0" json:"amount_invoiced"`
	AmountSettled  float64   `gorm:"type:decimal(15,4);default:0" json:"amount_settled"`
	IsActive       bool      `gorm:"type:tinyint(1);default:1" json:"is_active"`

	SupplierRel      *Supplier         `gorm:"foreignKey:SupplierID" json:"supplier_rel,omitempty"`
	DeliveryReceipts []DeliveryReceipt `gorm:"foreignKey:PurchaseID" json:"delivery_receipts,omitempty"`
	APInvoices       []APInvoice       `gorm:"foreignKey:PurchaseID" json:"ap_invoices,omitempty"`
}

func (Purchase) TableName() string { return "purchase" }

type DeliveryReceipt struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	DRNumber        string    `gorm:"type:varchar(30);uniqueIndex;not null" json:"dr_number"`
	Date            time.Time `gorm:"type:date;not null" json:"date"`
	PurchaseID      uint      `gorm:"not null;index" json:"purchase_id"`
	Status          string    `gorm:"type:varchar(20);default:'Draft'" json:"status"` // Draft | Received | Cancelled
	ReceivedBy      string    `gorm:"type:varchar(100)" json:"received_by"`
	SupplierDRRef   string    `gorm:"type:varchar(50)" json:"supplier_dr_ref"`
	Notes           string    `gorm:"type:text" json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedByID     *uint     `json:"created_by_id"`
	AmountInvoiced  float64   `gorm:"type:decimal(15,4);default:0" json:"amount_invoiced"`

	Purchase   *Purchase             `gorm:"foreignKey:PurchaseID" json:"purchase,omitempty"`
	Items      []DeliveryReceiptItem `gorm:"foreignKey:DeliveryReceiptID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	APInvoices []APInvoice           `gorm:"foreignKey:DeliveryReceiptID" json:"ap_invoices,omitempty"`
}

func (DeliveryReceipt) TableName() string { return "delivery_receipt" }

type DeliveryReceiptItem struct {
	ID                uint    `gorm:"primaryKey" json:"id"`
	DeliveryReceiptID uint    `gorm:"not null;index" json:"delivery_receipt_id"`
	Category          string  `gorm:"type:varchar(50)" json:"category"`
	ItemName          string  `gorm:"type:varchar(100);not null" json:"item_name"`
	Unit              string  `gorm:"type:varchar(20);not null" json:"unit"`
	QuantityOrdered   float64 `gorm:"type:decimal(12,3);default:0" json:"quantity_ordered"`
	QuantityReceived  float64 `gorm:"type:decimal(12,3);default:0" json:"quantity_received"`
	UnitPrice         float64 `gorm:"type:decimal(15,4);default:0" json:"unit_price"`
}

func (DeliveryReceiptItem) TableName() string { return "delivery_receipt_item" }

func (i *DeliveryReceiptItem) LineTotal() float64 {
	return i.QuantityReceived * i.UnitPrice
}

type APInvoice struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	InvoiceNumber       string     `gorm:"type:varchar(30);uniqueIndex;not null" json:"invoice_number"`
	Date                time.Time  `gorm:"type:date;not null" json:"date"`
	DueDate             *time.Time `gorm:"type:date" json:"due_date"`
	Terms               string     `gorm:"type:varchar(30)" json:"terms"`
	PurchaseID          uint       `gorm:"not null;index" json:"purchase_id"`
	DeliveryReceiptID   *uint      `gorm:"index" json:"delivery_receipt_id"`
	SupplierName        string     `gorm:"type:varchar(150)" json:"supplier_name"`
	SupplierInvoiceRef  string     `gorm:"type:varchar(50)" json:"supplier_invoice_ref"`
	TotalAmount         float64    `gorm:"type:decimal(15,4);default:0" json:"total_amount"`
	Status              string     `gorm:"type:varchar(20);default:'Open'" json:"status"` // Open | Partial | Paid | Cancelled
	Notes               string     `gorm:"type:text" json:"notes"`
	CreatedAt           time.Time  `json:"created_at"`
	CreatedByID         *uint      `json:"created_by_id"`
	AmountPaidStored    float64    `gorm:"type:decimal(15,4);default:0" json:"amount_paid_stored"`

	Purchase        *Purchase        `gorm:"foreignKey:PurchaseID" json:"purchase,omitempty"`
	DeliveryReceipt *DeliveryReceipt `gorm:"foreignKey:DeliveryReceiptID" json:"delivery_receipt,omitempty"`
	Items           []APInvoiceItem  `gorm:"foreignKey:APInvoiceID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	PaymentLines    []APPaymentLine  `gorm:"foreignKey:APInvoiceID" json:"payment_lines,omitempty"`
}

func (APInvoice) TableName() string { return "ap_invoice" }

func (a *APInvoice) Balance() float64 {
	balance := a.TotalAmount - a.AmountPaidStored
	if balance < 0 {
		return 0
	}
	return balance
}

type APInvoiceItem struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	APInvoiceID uint    `gorm:"not null;index" json:"ap_invoice_id"`
	Category    string  `gorm:"type:varchar(50)" json:"category"`
	ItemName    string  `gorm:"type:varchar(100);not null" json:"item_name"`
	Unit        string  `gorm:"type:varchar(20);not null" json:"unit"`
	Quantity    float64 `gorm:"type:decimal(12,3);not null" json:"quantity"`
	UnitPrice   float64 `gorm:"type:decimal(15,4);not null" json:"unit_price"`
}

func (APInvoiceItem) TableName() string { return "ap_invoice_item" }

func (i *APInvoiceItem) LineTotal() float64 {
	return i.Quantity * i.UnitPrice
}

type APPayment struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	PaymentNumber         string    `gorm:"type:varchar(30);uniqueIndex;not null" json:"payment_number"`
	Date                  time.Time `gorm:"type:date;not null" json:"date"`
	SupplierID            *uint     `gorm:"index" json:"supplier_id"`
	SupplierNameSnapshot  string    `gorm:"type:varchar(150)" json:"supplier_name_snapshot"`
	TotalAmount           float64   `gorm:"type:decimal(15,4);default:0" json:"total_amount"`
	PaymentMethod         string    `gorm:"type:varchar(20);default:'Cash'" json:"payment_method"`
	ReferenceNumber       string    `gorm:"type:varchar(50)" json:"reference_number"`
	Notes                 string    `gorm:"type:text" json:"notes"`
	CreatedAt             time.Time `json:"created_at"`
	CreatedByID           *uint     `json:"created_by_id"`

	Supplier *Supplier      `gorm:"foreignKey:SupplierID" json:"supplier,omitempty"`
	Lines    []APPaymentLine `gorm:"foreignKey:PaymentID;constraint:OnDelete:CASCADE" json:"lines,omitempty"`
}

func (APPayment) TableName() string { return "ap_payment" }

type APPaymentLine struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	PaymentID     uint    `gorm:"not null;index" json:"payment_id"`
	APInvoiceID   uint    `gorm:"not null;index" json:"ap_invoice_id"`
	AmountApplied float64 `gorm:"type:decimal(15,4);not null" json:"amount_applied"`

	Payment   *APPayment `gorm:"foreignKey:PaymentID"  json:"payment,omitempty"`
	APInvoice *APInvoice `gorm:"foreignKey:APInvoiceID" json:"ap_invoice,omitempty"`
}

func (APPaymentLine) TableName() string { return "ap_payment_line" }
