package models

import "time"

// ─────────────────────────────────────────────
// Milling Order (OMOR / MOR1)
// ─────────────────────────────────────────────

// MillingOrder is a production document that tracks the conversion of raw paddy
// into milled rice and by-products (bran, hull, broken rice).
// Status lifecycle: Draft → In Progress → Completed | Cancelled
type MillingOrder struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	MoNumber         string    `gorm:"column:mo_number;type:varchar(30);uniqueIndex;not null" json:"mo_number"`
	PostingDate      string    `gorm:"column:posting_date;type:date;not null" json:"posting_date"`
	ExpectedDate     string    `gorm:"column:expected_date;type:date" json:"expected_date"`
	Status           string    `gorm:"column:status;type:varchar(20);default:'Draft'" json:"status"` // Draft | In Progress | Completed | Cancelled
	InputItemID      uint      `gorm:"column:input_item_id;not null" json:"input_item_id"`
	InputItemCode    string    `gorm:"column:input_item_code;type:varchar(20)" json:"input_item_code"`
	InputItemName    string    `gorm:"column:input_item_name;type:varchar(150)" json:"input_item_name"`
	InputItemCategory string   `gorm:"column:input_item_category;type:varchar(50)" json:"input_item_category"`
	InputQty         float64   `gorm:"column:input_qty;type:decimal(12,3);not null" json:"input_qty"`
	InputUomEntry    uint      `gorm:"column:input_uom_entry;default:0" json:"input_uom_entry"`
	InputUnit        string    `gorm:"column:input_unit;type:varchar(20)" json:"input_unit"`
	InputUnitCost    float64   `gorm:"column:input_unit_cost;type:decimal(15,4);default:0" json:"input_unit_cost"`
	DocTotal         float64   `gorm:"column:doc_total;type:decimal(15,4);default:0" json:"doc_total"` // InputQty × InputUnitCost
	Remarks          string    `gorm:"column:remarks;type:text" json:"remarks"`

	// Set on Start
	GoodsIssueID     *uint  `gorm:"column:goods_issue_id" json:"goods_issue_id,omitempty"`
	GoodsIssueNumber string `gorm:"column:goods_issue_number;type:varchar(30)" json:"goods_issue_number"`
	StartJEID        *uint  `gorm:"column:start_je_id" json:"start_je_id,omitempty"`

	// Set on Complete
	GoodsReceiptID     *uint  `gorm:"column:goods_receipt_id" json:"goods_receipt_id,omitempty"`
	GoodsReceiptNumber string `gorm:"column:goods_receipt_number;type:varchar(30)" json:"goods_receipt_number"`
	CompleteJEID       *uint  `gorm:"column:complete_je_id" json:"complete_je_id,omitempty"`

	CreatedByID uint      `gorm:"column:created_by_id" json:"created_by_id"`
	UpdatedByID *uint    `gorm:"column:updated_by_id" json:"updated_by_id"`
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`

	// Loaded at service layer — gorm:"-" prevents FK constraint generation
	Lines []MillingOrderLine `gorm:"-" json:"lines,omitempty"`
}

func (MillingOrder) TableName() string { return "omor" }

// MillingOrderLine represents one expected/actual output product from a milling run.
// Equivalent to a BOM output line.
type MillingOrderLine struct {
	ID             uint    `gorm:"primaryKey" json:"id"`
	MillingOrderID uint    `gorm:"column:milling_order_id;not null;index" json:"milling_order_id"`
	LineNum        int     `gorm:"column:line_num;default:0" json:"line_num"`

	OutputItemID       uint    `gorm:"column:output_item_id;not null" json:"output_item_id"`
	OutputItemCode     string  `gorm:"column:output_item_code;type:varchar(20)" json:"output_item_code"`
	OutputItemName     string  `gorm:"column:output_item_name;type:varchar(150)" json:"output_item_name"`
	OutputItemCategory string  `gorm:"column:output_item_category;type:varchar(50)" json:"output_item_category"`

	YieldPct    float64 `gorm:"column:yield_pct;type:decimal(5,2);default:0" json:"yield_pct"`   // e.g. 65.00 for 65%
	ExpectedQty float64 `gorm:"column:expected_qty;type:decimal(12,3);default:0" json:"expected_qty"` // InputQty × YieldPct / 100
	ActualQty   float64 `gorm:"column:actual_qty;type:decimal(12,3);default:0" json:"actual_qty"`    // filled on Complete

	UomEntry uint   `gorm:"column:uom_entry;default:0" json:"uom_entry"`
	Unit     string `gorm:"column:unit;type:varchar(20)" json:"unit"`

	// Set on Complete
	GoodsReceiptLineID *uint `gorm:"column:goods_receipt_line_id" json:"goods_receipt_line_id,omitempty"`
}

func (MillingOrderLine) TableName() string { return "mor1" }

// ─── BOM / Product Tree (OITT / ITT1) ────────────────────────────────────────

// ProductTree is the BOM header. Code must match OITM.ItemCode where MakItem='Y'.
type ProductTree struct {
	Code      string           `gorm:"column:code;primaryKey;type:varchar(50)"       json:"code"`
	TreeType  string           `gorm:"column:tree_type;type:char(1);default:'P'"     json:"tree_type"` // P=Production
	Warehouse string           `gorm:"column:warehouse;type:varchar(10)"              json:"warehouse"`
	Quantity  float64          `gorm:"column:quantity;type:decimal(12,3);default:1"  json:"quantity"` // batch size
	Notes     string           `gorm:"column:notes;type:text"                         json:"notes"`
	UpdatedAt time.Time        `gorm:"column:updated_at"                              json:"updated_at"`
	Lines     []ProductTreeLine `gorm:"-" json:"lines,omitempty"`
}

func (ProductTree) TableName() string { return "oitt" }

// ProductTreeLine is one component row in a BOM.
type ProductTreeLine struct {
	ID          uint    `gorm:"primaryKey;autoIncrement"                               json:"id"`
	Code        string  `gorm:"column:code;type:varchar(50);not null;index"            json:"code"` // parent finished item
	ItemCode    string  `gorm:"column:item_code;type:varchar(50);not null"             json:"item_code"`
	ItemName    string  `gorm:"column:item_name;type:varchar(200)"                     json:"item_name"`
	Quantity    float64 `gorm:"column:quantity;type:decimal(12,3);not null;default:1"  json:"quantity"`
	Warehouse   string  `gorm:"column:warehouse;type:varchar(10)"                      json:"warehouse"`
	IssueMethod string  `gorm:"column:issue_method;type:char(1);default:'M'"           json:"issue_method"` // M=Manual B=Backflush
	UomCode     string  `gorm:"column:uom_code;type:varchar(20)"                       json:"uom_code"`
	UomEntry    uint    `gorm:"column:uom_entry;default:0"                             json:"uom_entry"`
	Price       float64 `gorm:"column:price;type:decimal(15,4);default:0"              json:"price"`
}

func (ProductTreeLine) TableName() string { return "itt1" }

// ─── Work Order (OWOR / WOR1) ─────────────────────────────────────────────────

// WorkOrder is the production execution document.
// order_type='P' (BOM Production): finished item is ItemCode; all WOR1 lines are inputs.
// order_type='M' (Milling): ItemCode is the paddy input; WOR1 line_dir='I' is input, 'O' are outputs.
type WorkOrder struct {
	DocEntry    uint      `gorm:"primaryKey;autoIncrement"                              json:"doc_entry"`
	DocNum      string    `gorm:"column:doc_num;type:varchar(30);uniqueIndex;not null"  json:"doc_num"`
	OrderType   string    `gorm:"column:order_type;type:char(1);default:'P'"           json:"order_type"` // P=Production M=Milling
	ItemCode    string    `gorm:"column:item_code;type:varchar(50);not null"            json:"item_code"`
	ItemName    string    `gorm:"column:item_name;type:varchar(200)"                    json:"item_name"`
	PlannedQty  float64   `gorm:"column:planned_qty;type:decimal(12,3);not null"        json:"planned_qty"`
	CmpltQty    float64   `gorm:"column:cmplt_qty;type:decimal(12,3);default:0"         json:"cmplt_qty"`
	RjctQty     float64   `gorm:"column:rjct_qty;type:decimal(12,3);default:0"          json:"rjct_qty"`
	Warehouse   string    `gorm:"column:warehouse;type:varchar(10)"                     json:"warehouse"`
	StartDate   string    `gorm:"column:start_date;type:date"                          json:"start_date"`
	DueDate     string    `gorm:"column:due_date;type:date"                            json:"due_date"`
	Status      string    `gorm:"column:status;type:char(1);default:'P'"               json:"status"` // P=Planned R=Released C=Closed L=Cancelled
	Notes       string    `gorm:"column:notes;type:text"                               json:"notes"`
	CreatedByID uint      `gorm:"column:created_by_id"                                 json:"created_by_id"`
	UpdatedByID *uint    `gorm:"column:updated_by_id"                                 json:"updated_by_id"`
	CreatedAt   time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)"      json:"created_at"`

	// GR/GI/JE linkage (used by Milling and optionally by Production)
	GoodsIssueID       *uint  `gorm:"column:goods_issue_id"                                json:"goods_issue_id,omitempty"`
	GoodsIssueNumber   string `gorm:"column:goods_issue_number;type:varchar(30)"           json:"goods_issue_number"`
	StartJEID          *uint  `gorm:"column:start_je_id"                                   json:"start_je_id,omitempty"`
	GoodsReceiptID     *uint  `gorm:"column:goods_receipt_id"                              json:"goods_receipt_id,omitempty"`
	GoodsReceiptNumber string `gorm:"column:goods_receipt_number;type:varchar(30)"         json:"goods_receipt_number"`
	CompleteJEID       *uint  `gorm:"column:complete_je_id"                                json:"complete_je_id,omitempty"`

	Lines []WorkOrderLine `gorm:"-" json:"lines,omitempty"`
}

func (WorkOrder) TableName() string { return "owor" }

// WorkOrderLine is one line on a Work Order.
// line_dir='I': input component consumed. line_dir='O': output co-product received (Milling only).
type WorkOrderLine struct {
	ID           uint    `gorm:"primaryKey;autoIncrement"                               json:"id"`
	DocEntry     uint    `gorm:"column:doc_entry;not null;index"                        json:"doc_entry"`
	LineNum      int     `gorm:"column:line_num;default:0"                              json:"line_num"`
	LineDir      string  `gorm:"column:line_dir;type:char(1);default:'I'"              json:"line_dir"` // I=Input O=Output
	ItemID       uint    `gorm:"column:item_id;default:0"                               json:"item_id"`
	ItemCode     string  `gorm:"column:item_code;type:varchar(50);not null"             json:"item_code"`
	ItemName     string  `gorm:"column:item_name;type:varchar(200)"                     json:"item_name"`
	ItemCategory string  `gorm:"column:item_category;type:varchar(50)"                  json:"item_category"`
	PlannedQty   float64 `gorm:"column:planned_qty;type:decimal(12,3);not null"         json:"planned_qty"`
	IssuedQty    float64 `gorm:"column:issued_qty;type:decimal(12,3);default:0"         json:"issued_qty"`
	Warehouse    string  `gorm:"column:warehouse;type:varchar(10)"                      json:"warehouse"`
	IssueMethod  string  `gorm:"column:issue_method;type:char(1);default:'M'"          json:"issue_method"` // M=Manual B=Backflush
	UomEntry     uint    `gorm:"column:uom_entry;default:0"                             json:"uom_entry"`
	UomCode      string  `gorm:"column:uom_code;type:varchar(20)"                       json:"uom_code"`
	Price        float64 `gorm:"column:price;type:decimal(15,4);default:0"              json:"price"`
}

func (WorkOrderLine) TableName() string { return "wor1" }
