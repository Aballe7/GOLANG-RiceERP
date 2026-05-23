package models

import "time"

type Supplier struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"type:varchar(150);not null" json:"name"`
	SupplierType  string     `gorm:"type:varchar(30);default:'Regular'" json:"supplier_type"`
	ContactPerson string     `gorm:"type:varchar(100)" json:"contact_person"`
	ContactNumber string     `gorm:"type:varchar(20)" json:"contact_number"`
	Email         string     `gorm:"type:varchar(100)" json:"email"`
	Address       string     `gorm:"type:varchar(200)" json:"address"`
	TINNumber     string     `gorm:"type:varchar(30)" json:"tin_number"`
	PaymentTerms  string     `gorm:"type:varchar(30);default:'COD'" json:"payment_terms"`
	BankDetails   string     `gorm:"type:text" json:"bank_details"`
	Categories    string     `gorm:"type:varchar(200)" json:"categories"`
	Notes         string     `gorm:"type:text" json:"notes"`
	IsActive      bool       `gorm:"type:tinyint(1);default:1" json:"is_active"`
	Status        string     `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending | active | rejected
	ApprovedBy    string     `gorm:"type:varchar(100)" json:"approved_by"` // snapshot of approver name
	ApprovedByID  *uint      `gorm:"index" json:"approved_by_id"`
	ApprovedAt    *time.Time `json:"approved_at"`
	CreatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`

	// BIR Expanded Withholding Tax (EWT) — RR 2-98 as amended
	WHTCategory     string `gorm:"type:varchar(30);default:'NONE'" json:"wht_category"`
	IsVATRegistered bool   `gorm:"type:tinyint(1);default:0" json:"is_vat_registered"`
}

func (Supplier) TableName() string { return "supplier" }

// PurchasingLookup stores user-configurable dropdown options for the purchasing module.
// category: supplier_type | payment_terms | payment_method
type PurchasingLookup struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Category  string `gorm:"type:varchar(50);not null;index" json:"category"`
	Value     string `gorm:"type:varchar(100);not null" json:"value"`
	SortOrder int    `gorm:"default:0" json:"sort_order"`
	IsActive  bool   `gorm:"type:tinyint(1);default:1" json:"is_active"`
}

func (PurchasingLookup) TableName() string { return "purchasing_lookup" }

// PurchaseHeader represents the PO document header — equivalent to SAP B1 OPOR.
// All purchasing flows start here; the old single-row "purchase" table is removed.
type PurchaseHeader struct {
	ID uint `gorm:"primaryKey" json:"id"` // DocEntry

	PONumber string `gorm:"type:varchar(50);uniqueIndex" json:"po_number"` // PO-000-00000
	DocNum   int    `gorm:"type:int" json:"doc_num"`
	Status   string `gorm:"type:varchar(20);default:'Open'" json:"status"` // Open | Closed | Cancelled

	SupplierID   *uint  `gorm:"index" json:"supplier_id"`
	SupplierCode string `gorm:"type:varchar(15)" json:"supplier_code"`
	SupplierName string `gorm:"type:varchar(100)" json:"supplier_name"`

	PostingDate  time.Time  `gorm:"type:date;not null" json:"posting_date"`
	DeliveryDate *time.Time `gorm:"type:date" json:"delivery_date"`
	TaxDate      *time.Time `gorm:"type:date" json:"tax_date"`

	RefNumber     string `gorm:"type:varchar(100)" json:"ref_number"`
	Currency      string `gorm:"type:varchar(3)" json:"currency"`
	PaymentMethod string `gorm:"type:varchar(30)" json:"payment_method"`

	DocTotal float64 `gorm:"type:decimal(15,4);default:0" json:"doc_total"`
	VatSum   float64 `gorm:"type:decimal(15,4);default:0" json:"vat_sum"`

	Comments    string `gorm:"type:varchar(254)" json:"comments"`
	CreatedByID *uint  `json:"created_by_id"`
	UpdatedByID *uint  `json:"updated_by_id"`
	Version     uint   `gorm:"default:1" json:"version"`

	// Relationships
	Supplier        *Supplier         `gorm:"foreignKey:SupplierID" json:"supplier,omitempty"`
	Lines           []PurchaseLine    `gorm:"foreignKey:HeaderID" json:"lines,omitempty"`
	DeliveryReceipts []DeliveryReceipt `gorm:"foreignKey:PurchaseHeaderID" json:"delivery_receipts,omitempty"`
	APInvoices      []APInvoice       `gorm:"foreignKey:PurchaseHeaderID" json:"ap_invoices,omitempty"`
}

func (PurchaseHeader) TableName() string { return "purchase_header" }

// PurchaseLine is one line item on a PurchaseHeader — equivalent to SAP B1 POR1.
// OpenQty tracks remaining quantity not yet received (decremented at DR confirmation).
type PurchaseLine struct {
	ID       uint `gorm:"primaryKey" json:"id"`
	HeaderID uint `gorm:"not null;index" json:"header_id"`

	LineNum int `gorm:"type:int" json:"line_num"`

	ItemCode    string  `gorm:"type:varchar(30)" json:"item_code"`
	Description string  `gorm:"type:varchar(100)" json:"description"`
	Unit        string  `gorm:"type:varchar(20)" json:"unit"`
	Quantity    float64 `gorm:"type:decimal(12,3);default:0" json:"quantity"`
	OpenQty     float64 `gorm:"type:decimal(12,3);default:0" json:"open_qty"` // remaining to receive
	Price       float64 `gorm:"type:decimal(15,4);default:0" json:"price"`
	LineTotal   float64 `gorm:"type:decimal(15,4);default:0" json:"line_total"`

	WarehouseCode string `gorm:"type:varchar(8)" json:"warehouse_code"`
	AccountCode   string `gorm:"type:varchar(15)" json:"account_code"`
	TaxCode       string `gorm:"type:varchar(8)" json:"tax_code"`
	ProjectCode   string `gorm:"type:varchar(20)" json:"project_code"`
	CostCenter    string `gorm:"type:varchar(8)" json:"cost_center"`
	LineStatus    string `gorm:"type:varchar(1);default:'O'" json:"line_status"` // O=Open, C=Closed

	IUoMEntry uint `gorm:"column:i_uom_entry;not null;default:0" json:"i_uom_entry"`

	// Base document chain (PR→PO copy-from traceability, future use)
	BaseDocEntry *uint `json:"base_doc_entry"`
	BaseDocNum   *int  `json:"base_doc_num"`
	BaseLineNum  *int  `json:"base_line_num"`
}

func (PurchaseLine) TableName() string { return "purchase_line" }

// DeliveryReceipt is the goods receipt document — equivalent to SAP B1 OPDN.
// PurchaseHeaderID links to the source PO at the header level.
// Each DeliveryReceiptItem links to the exact PurchaseLine via BaseDocEntry + BaseLineNum.
type DeliveryReceipt struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	DRNumber string `gorm:"type:varchar(30);uniqueIndex;not null" json:"dr_number"`
	DocNum   int    `gorm:"type:int;default:0" json:"doc_num"`
	Status   string `gorm:"type:varchar(20);default:'Draft'" json:"status"` // Draft | Received | Cancelled

	// Supplier (denormalised from PurchaseHeader)
	SupplierID   *uint  `gorm:"index" json:"supplier_id"`
	SupplierCode string `gorm:"type:varchar(15)" json:"supplier_code"`
	SupplierName string `gorm:"type:varchar(100)" json:"supplier_name"`

	PostingDate  time.Time  `gorm:"column:date;type:date;not null" json:"posting_date"` // DB col: date
	DeliveryDate *time.Time `gorm:"type:date" json:"delivery_date"`
	TaxDate      *time.Time `gorm:"type:date" json:"tax_date"`

	RefNumber string `gorm:"column:supplier_dr_ref;type:varchar(50)" json:"ref_number"` // DB col: supplier_dr_ref
	Currency  string `gorm:"type:varchar(3);default:'PHP'" json:"currency"`

	// DocTotal tracks how much of this DR has been invoiced (updated by AP Invoice create/cancel)
	DocTotal float64 `gorm:"column:amount_invoiced;type:decimal(15,4);default:0" json:"doc_total"` // DB col: amount_invoiced
	VatSum   float64 `gorm:"type:decimal(15,4);default:0" json:"vat_sum"`

	Comments string `gorm:"column:notes;type:text" json:"comments"` // DB col: notes

	// SAP B1-aligned: source PO header reference at the DR header level.
	// Line-level source is tracked via DeliveryReceiptItem.BaseDocEntry + BaseLineNum.
	PurchaseHeaderID *uint `gorm:"index" json:"purchase_header_id"`

	ReceivedBy   string    `gorm:"type:varchar(100)" json:"received_by"` // snapshot of receiver name
	ReceivedByID *uint     `gorm:"index" json:"received_by_id"`
	CreatedAt    time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	CreatedByID  *uint     `json:"created_by_id"`
	UpdatedByID  *uint     `json:"updated_by_id"`

	PurchaseHeader *PurchaseHeader      `gorm:"foreignKey:PurchaseHeaderID" json:"purchase_header,omitempty"`
	Lines          []DeliveryReceiptItem `gorm:"foreignKey:DeliveryReceiptID;constraint:OnDelete:CASCADE" json:"lines,omitempty"`
	APInvoices     []APInvoice          `gorm:"foreignKey:DeliveryReceiptID" json:"ap_invoices,omitempty"`
}

func (DeliveryReceipt) TableName() string { return "delivery_receipt" }

// DeliveryReceiptItem is one line in a DeliveryReceipt — equivalent to SAP B1 PDN1.
// BaseDocEntry = PurchaseHeader.ID (source PO header)
// BaseLineNum  = PurchaseLine.ID  (direct FK, not LineNum index — enables O(1) open_qty update)
type DeliveryReceiptItem struct {
	ID                uint `gorm:"primaryKey" json:"id"`
	DeliveryReceiptID uint `gorm:"not null;index" json:"delivery_receipt_id"`
	LineNum           int  `gorm:"type:int;default:0" json:"line_num"`

	ItemCode    string `gorm:"type:varchar(30)" json:"item_code"`
	Description string `gorm:"column:item_name;type:varchar(100);not null" json:"description"` // DB col: item_name
	Unit        string `gorm:"type:varchar(20);not null" json:"unit"`
	Category    string `gorm:"type:varchar(50)" json:"category"`

	Quantity        float64 `gorm:"column:quantity_received;type:decimal(12,3);default:0" json:"quantity"` // DB col: quantity_received
	QuantityOrdered float64 `gorm:"type:decimal(12,3);default:0" json:"quantity_ordered"`
	OpenQty         float64 `gorm:"type:decimal(12,3);default:0" json:"open_qty"` // remaining uninvoiced
	UomEntry        uint    `gorm:"column:uom_entry;not null;default:0" json:"uom_entry"`

	Price     float64 `gorm:"column:unit_price;type:decimal(15,4);default:0" json:"price"` // DB col: unit_price
	LineTotal float64 `gorm:"type:decimal(15,4);default:0" json:"line_total"`

	WarehouseCode string `gorm:"type:varchar(8)" json:"warehouse_code"`
	AccountCode   string `gorm:"type:varchar(15)" json:"account_code"`
	TaxCode       string `gorm:"type:varchar(8)" json:"tax_code"`
	ProjectCode   string `gorm:"type:varchar(20)" json:"project_code"`
	CostCenter    string `gorm:"type:varchar(8)" json:"cost_center"`
	LineStatus    string `gorm:"type:varchar(1);default:'O'" json:"line_status"`

	// SAP B1-aligned base document chain:
	// BaseDocEntry = PurchaseHeader.ID  (BaseType = 22, OPOR equivalent)
	// BaseLineNum  = PurchaseLine.ID    (direct PK — fast open_qty update without description matching)
	BaseDocEntry *uint `json:"base_doc_entry"`
	BaseLineNum  *int  `json:"base_line_num"`
}

func (DeliveryReceiptItem) TableName() string { return "dr_line" }

func (i *DeliveryReceiptItem) ReceivedTotal() float64 {
	return i.Quantity * i.Price
}

// APInvoice is the supplier invoice document — equivalent to SAP B1 OPCH.
// PurchaseHeaderID links to the source PO.
// DeliveryReceiptID links to the source GRPO (when created via DR).
type APInvoice struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	InvoiceNumber string `gorm:"type:varchar(30);uniqueIndex;not null" json:"invoice_number"`
	DocNum        int    `gorm:"type:int;default:0" json:"doc_num"`
	Status        string `gorm:"type:varchar(20);default:'Open'" json:"status"` // Open | Partial | Paid | Cancelled

	SupplierID   *uint  `gorm:"index" json:"supplier_id"`
	SupplierCode string `gorm:"type:varchar(15)" json:"supplier_code"`
	SupplierName string `gorm:"type:varchar(150)" json:"supplier_name"`

	PostingDate time.Time  `gorm:"column:date;type:date;not null" json:"posting_date"` // DB col: date
	DueDate     *time.Time `gorm:"type:date" json:"due_date"`
	TaxDate     *time.Time `gorm:"type:date" json:"tax_date"`
	Terms       string     `gorm:"type:varchar(30)" json:"terms"`

	RefNumber string `gorm:"column:supplier_invoice_ref;type:varchar(50)" json:"ref_number"` // DB col: supplier_invoice_ref
	Currency  string `gorm:"type:varchar(3);default:'PHP'" json:"currency"`

	DocTotal float64 `gorm:"column:total_amount;type:decimal(15,4);default:0" json:"doc_total"` // DB col: total_amount
	VatSum   float64 `gorm:"type:decimal(15,4);default:0" json:"vat_sum"`

	Comments string `gorm:"column:notes;type:text" json:"comments"` // DB col: notes

	// SAP B1-aligned source document links
	PurchaseHeaderID  *uint `gorm:"index" json:"purchase_header_id"`   // FK to purchase_header.id (OPOR)
	DeliveryReceiptID *uint `gorm:"index" json:"delivery_receipt_id"` // FK to delivery_receipt.id (OPDN), nil for direct PO→AP

	// Copy-from chain (SAP B1 BaseEntry/BaseDocNum pattern)
	BaseDocEntry *uint `json:"base_doc_entry"` // DR.id or PurchaseHeader.id
	BaseDocNum   *int  `json:"base_doc_num"`

	AmountPaidStored float64   `gorm:"type:decimal(15,4);default:0" json:"amount_paid_stored"`
	CreatedAt        time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	CreatedByID      *uint     `json:"created_by_id"`
	UpdatedByID      *uint     `json:"updated_by_id"`
	Version          uint      `gorm:"default:1" json:"version"`

	// BIR Expanded Withholding Tax (EWT) — RR 2-98 as amended
	VATExclusiveAmount float64 `gorm:"type:decimal(15,4);default:0" json:"vat_exclusive_amount"`
	WHTRate            float64 `gorm:"type:decimal(7,4);default:0" json:"wht_rate"`
	WHTAmount          float64 `gorm:"type:decimal(15,4);default:0" json:"wht_amount"`
	WHTATCCode         string  `gorm:"type:varchar(10)" json:"wht_atc_code"`
	NetPayable         float64 `gorm:"type:decimal(15,4);default:0" json:"net_payable"`

	PurchaseHeader  *PurchaseHeader  `gorm:"foreignKey:PurchaseHeaderID" json:"purchase_header,omitempty"`
	DeliveryReceipt *DeliveryReceipt `gorm:"foreignKey:DeliveryReceiptID" json:"delivery_receipt,omitempty"`
	Lines           []APInvoiceItem  `gorm:"foreignKey:APInvoiceID;constraint:OnDelete:CASCADE" json:"lines,omitempty"`
	PaymentLines    []APPaymentLine  `gorm:"foreignKey:APInvoiceID" json:"payment_lines,omitempty"`
}

func (APInvoice) TableName() string { return "ap_invoice" }

func (a *APInvoice) Balance() float64 {
	b := a.DocTotal - a.AmountPaidStored
	if b < 0 {
		return 0
	}
	return b
}

// APInvoiceItem is one line in an APInvoice — equivalent to SAP B1 PCH1.
// BaseDocEntry = DeliveryReceipt.ID or PurchaseHeader.ID
// BaseLineNum  = DeliveryReceiptItem.LineNum or PurchaseLine.LineNum
// DRLineID     = DeliveryReceiptItem.ID (direct FK for open_qty decrement)
type APInvoiceItem struct {
	ID          uint `gorm:"primaryKey" json:"id"`
	APInvoiceID uint `gorm:"not null;index" json:"ap_invoice_id"`
	LineNum     int  `gorm:"type:int;default:0" json:"line_num"`

	ItemCode    string  `gorm:"type:varchar(30)" json:"item_code"`
	Description string  `gorm:"column:item_name;type:varchar(100);not null" json:"description"` // DB col: item_name
	Unit        string  `gorm:"type:varchar(20);not null" json:"unit"`
	Category    string  `gorm:"type:varchar(50)" json:"category"`

	Quantity float64 `gorm:"type:decimal(12,3);not null" json:"quantity"`
	OpenQty  float64 `gorm:"type:decimal(12,3);default:0" json:"open_qty"`
	UomEntry uint    `gorm:"column:uom_entry;not null;default:0" json:"uom_entry"`

	Price     float64 `gorm:"column:unit_price;type:decimal(15,4);not null" json:"price"` // DB col: unit_price
	LineTotal float64 `gorm:"type:decimal(15,4);default:0" json:"line_total"`

	WarehouseCode string `gorm:"type:varchar(8)" json:"warehouse_code"`
	AccountCode   string `gorm:"type:varchar(15)" json:"account_code"`
	TaxCode       string `gorm:"type:varchar(8)" json:"tax_code"`
	ProjectCode   string `gorm:"type:varchar(20)" json:"project_code"`
	CostCenter    string `gorm:"type:varchar(8)" json:"cost_center"`
	LineStatus    string `gorm:"type:varchar(1);default:'O'" json:"line_status"`

	BaseDocEntry *uint `json:"base_doc_entry"` // DR.id or PurchaseHeader.id
	BaseLineNum  *int  `json:"base_line_num"`  // source line's LineNum
	DRLineID     *uint `gorm:"column:dr_line_id" json:"dr_line_id"` // FK to dr_line.id for open_qty tracking
}

func (APInvoiceItem) TableName() string { return "ap_invoice_line" }

func (i *APInvoiceItem) LineTotal_() float64 {
	return i.Quantity * i.Price
}

type APPayment struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	PaymentNumber        string    `gorm:"type:varchar(30);uniqueIndex;not null" json:"payment_number"`
	Status               string    `gorm:"type:varchar(20);default:'Posted'" json:"status"` // Posted | Cancelled
	Date                 time.Time `gorm:"type:date;not null" json:"date"`
	SupplierID           *uint     `gorm:"index" json:"supplier_id"`
	SupplierNameSnapshot string    `gorm:"type:varchar(150)" json:"supplier_name_snapshot"`
	TotalAmount          float64   `gorm:"type:decimal(15,4);default:0" json:"total_amount"`
	PaymentMethod        string    `gorm:"type:varchar(20);default:'Cash'" json:"payment_method"`
	ReferenceNumber      string    `gorm:"type:varchar(50)" json:"reference_number"`
	Notes                string    `gorm:"type:text" json:"notes"`
	CreatedAt            time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	CreatedByID          *uint     `json:"created_by_id"`
	UpdatedByID          *uint     `json:"updated_by_id"`

	Supplier *Supplier      `gorm:"foreignKey:SupplierID" json:"supplier,omitempty"`
	Lines    []APPaymentLine `gorm:"foreignKey:PaymentID;constraint:OnDelete:CASCADE" json:"lines,omitempty"`
}

func (APPayment) TableName() string { return "ap_payment" }

type APPaymentLine struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	PaymentID     uint    `gorm:"not null;index" json:"payment_id"`
	APInvoiceID   uint    `gorm:"not null;index" json:"ap_invoice_id"`
	AmountApplied float64 `gorm:"type:decimal(15,4);not null" json:"amount_applied"`

	Payment   *APPayment `gorm:"foreignKey:PaymentID" json:"payment,omitempty"`
	APInvoice *APInvoice `gorm:"foreignKey:APInvoiceID" json:"ap_invoice,omitempty"`
}

func (APPaymentLine) TableName() string { return "ap_payment_line" }
