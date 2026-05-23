package models

import "time"

// ─────────────────────────────────────────────
// Goods Receipt (OIGN / IGN1)
// ─────────────────────────────────────────────

// GoodsReceipt is a standalone inventory-increase document (no PO required).
// Equivalent to SAP B1 OIGN. Use for opening balances, stock corrections,
// free samples, and any unplanned stock increase.
type GoodsReceipt struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	GRNumber    string    `gorm:"column:gr_number;type:varchar(30);uniqueIndex;not null" json:"gr_number"`
	PostingDate string    `gorm:"column:posting_date;type:date;not null" json:"posting_date"`
	DocDueDate  *string   `gorm:"column:doc_due_date;type:date" json:"doc_due_date"`
	Status      string    `gorm:"column:status;type:varchar(20);default:'Open'" json:"status"` // Open | Cancelled
	Remarks     string    `gorm:"column:remarks;type:text" json:"remarks"`
	DocTotal    float64   `gorm:"column:doc_total;type:decimal(15,4);default:0" json:"doc_total"`
	CreatedByID uint      `gorm:"column:created_by_id" json:"created_by_id"`
	UpdatedByID *uint     `gorm:"column:updated_by_id" json:"updated_by_id"`
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`

	// Loaded at service layer — gorm:"-" prevents FK constraint generation
	Lines []GoodsReceiptItem `gorm:"-" json:"lines,omitempty"`
}

func (GoodsReceipt) TableName() string { return "oign" }

// GoodsReceiptItem is a single line on a GoodsReceipt.
// Equivalent to SAP B1 IGN1.
type GoodsReceiptItem struct {
	ID             uint    `gorm:"primaryKey" json:"id"`
	GoodsReceiptID uint    `gorm:"column:goods_receipt_id;not null;index" json:"goods_receipt_id"`
	LineNum        int     `gorm:"column:line_num;default:0" json:"line_num"`
	ItemID         uint    `gorm:"column:item_id;not null" json:"item_id"`
	ItemCode       string  `gorm:"column:item_code;type:varchar(50)" json:"item_code"`
	Description    string  `gorm:"column:description;type:varchar(200)" json:"description"`
	Quantity       float64 `gorm:"column:quantity;type:decimal(12,3);not null" json:"quantity"`
	Unit           string  `gorm:"column:unit;type:varchar(20)" json:"unit"`
	UomEntry       uint    `gorm:"column:uom_entry;not null;default:0" json:"uom_entry"`
	Price          float64 `gorm:"column:price;type:decimal(15,4);default:0" json:"price"`
	LineTotal      float64 `gorm:"column:line_total;type:decimal(15,4);default:0" json:"line_total"`
	WarehouseCode  string  `gorm:"column:warehouse_code;type:varchar(20)" json:"warehouse_code"`
	AccountCode    string  `gorm:"column:account_code;type:varchar(20)" json:"account_code"`
	Project        string  `gorm:"column:project;type:varchar(50)" json:"project"`
}

func (GoodsReceiptItem) TableName() string { return "ign1" }

// ─────────────────────────────────────────────
// Goods Issue (OIGE / IGE1)
// ─────────────────────────────────────────────

// GoodsIssue is a standalone inventory-decrease document (no SO required).
// Equivalent to SAP B1 OIGE. Use for internal use, breakage, samples, write-offs.
type GoodsIssue struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	GINumber    string    `gorm:"column:gi_number;type:varchar(30);uniqueIndex;not null" json:"gi_number"`
	PostingDate string    `gorm:"column:posting_date;type:date;not null" json:"posting_date"`
	DocDueDate  *string   `gorm:"column:doc_due_date;type:date" json:"doc_due_date"`
	Status      string    `gorm:"column:status;type:varchar(20);default:'Open'" json:"status"` // Open | Cancelled
	Remarks     string    `gorm:"column:remarks;type:text" json:"remarks"`
	DocTotal    float64   `gorm:"column:doc_total;type:decimal(15,4);default:0" json:"doc_total"`
	CreatedByID uint      `gorm:"column:created_by_id" json:"created_by_id"`
	UpdatedByID *uint     `gorm:"column:updated_by_id" json:"updated_by_id"`
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`

	// Loaded at service layer — gorm:"-" prevents FK constraint generation
	Lines []GoodsIssueItem `gorm:"-" json:"lines,omitempty"`
}

func (GoodsIssue) TableName() string { return "oige" }

// GoodsIssueItem is a single line on a GoodsIssue.
// Equivalent to SAP B1 IGE1.
type GoodsIssueItem struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	GoodsIssueID  uint    `gorm:"column:goods_issue_id;not null;index" json:"goods_issue_id"`
	LineNum       int     `gorm:"column:line_num;default:0" json:"line_num"`
	ItemID        uint    `gorm:"column:item_id;not null" json:"item_id"`
	ItemCode      string  `gorm:"column:item_code;type:varchar(50)" json:"item_code"`
	Description   string  `gorm:"column:description;type:varchar(200)" json:"description"`
	Quantity      float64 `gorm:"column:quantity;type:decimal(12,3);not null" json:"quantity"`
	Unit          string  `gorm:"column:unit;type:varchar(20)" json:"unit"`
	UomEntry      uint    `gorm:"column:uom_entry;not null;default:0" json:"uom_entry"`
	Price         float64 `gorm:"column:price;type:decimal(15,4);default:0" json:"price"`
	LineTotal     float64 `gorm:"column:line_total;type:decimal(15,4);default:0" json:"line_total"`
	WarehouseCode string  `gorm:"column:warehouse_code;type:varchar(20)" json:"warehouse_code"`
	AccountCode   string  `gorm:"column:account_code;type:varchar(20)" json:"account_code"`
	Project       string  `gorm:"column:project;type:varchar(50)" json:"project"`
}

func (GoodsIssueItem) TableName() string { return "ige1" }

// ─────────────────────────────────────────────
// UoM Master Data (OUOM)
// ─────────────────────────────────────────────

type UoMMaster struct {
	UomEntry uint    `gorm:"primaryKey;autoIncrement" json:"uom_entry"`
	UomCode  string  `gorm:"type:varchar(20);not null;uniqueIndex" json:"uom_code"`
	UomName  string  `gorm:"type:varchar(100);not null" json:"uom_name"`
	Length   float64 `gorm:"type:decimal(15,6);default:0" json:"length"`
	LType    int     `gorm:"default:0" json:"l_type"`
	Width    float64 `gorm:"type:decimal(15,6);default:0" json:"width"`
	WType    int     `gorm:"default:0" json:"w_type"`
	Height   float64 `gorm:"type:decimal(15,6);default:0" json:"height"`
	HType    int     `gorm:"default:0" json:"h_type"`
	Volume   float64 `gorm:"type:decimal(15,6);default:0" json:"volume"`
	VType    int     `gorm:"default:0" json:"v_type"`
	Weight   float64 `gorm:"type:decimal(15,6);default:0" json:"weight"`
	WgtType  int     `gorm:"default:0" json:"wgt_type"`
	UserSign int     `gorm:"default:0" json:"user_sign"`
}

func (UoMMaster) TableName() string { return "ouom" }

// ─────────────────────────────────────────────
// UoM Groups (OUGP)
// ─────────────────────────────────────────────

type UoMGroup struct {
	UgpEntry   uint           `gorm:"primaryKey;autoIncrement" json:"ugp_entry"`
	UgpCode    string         `gorm:"type:varchar(20);not null;uniqueIndex" json:"ugp_code"`
	UgpName    string         `gorm:"type:varchar(100);not null" json:"ugp_name"`
	BaseUom    int            `gorm:"default:0" json:"base_uom"`
	UserSign   int            `gorm:"default:0" json:"user_sign"`
	UserSign2  int            `gorm:"default:0" json:"user_sign2"`
	UpdateDate time.Time      `gorm:"type:datetime" json:"update_date"`
	CreateDate time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"create_date"`
	DataSource string         `gorm:"type:char(1);default:'M'" json:"data_source"`
	Lines      []UoMGroupLine `gorm:"foreignKey:UgpEntry;references:UgpEntry" json:"lines,omitempty"`
	BaseUnit   *UoMMaster     `gorm:"foreignKey:UomEntry;references:BaseUom" json:"base_unit,omitempty"`
}

func (UoMGroup) TableName() string { return "ougp" }

// ─────────────────────────────────────────────
// UoM Group Lines / Definition (UGP1)
// ─────────────────────────────────────────────

type UoMGroupLine struct {
	UgpEntry   uint       `gorm:"primaryKey" json:"ugp_entry"`
	LineNum    int        `gorm:"primaryKey" json:"line_num"`
	UomEntry   uint       `gorm:"not null" json:"uom_entry"`
	AltQty     float64    `gorm:"type:decimal(15,6);default:1" json:"alt_qty"`
	BaseQty    float64    `gorm:"type:decimal(15,6);default:1" json:"base_qty"`
	LogInstanc int        `gorm:"default:0" json:"log_instanc"`
	ObjType    string     `gorm:"type:varchar(20);default:'173'" json:"obj_type"`
	Unit       *UoMMaster `gorm:"foreignKey:UomEntry;references:UomEntry" json:"unit,omitempty"`
}

func (UoMGroupLine) TableName() string { return "ugp1" }

// ─────────────────────────────────────────────
// ItemUoMPrice — SAP B1 ITM9
// Stores an explicit price per unit of measure for an item.
// Composite PK: (item_code, uom_entry).
// FK migrated from item_id (uint) → item_code (string) to align with OITM.
// ─────────────────────────────────────────────

type ItemUoMPrice struct {
	ItemCode string  `gorm:"column:item_code;primaryKey;type:varchar(50);not null" json:"item_code"`
	UomEntry uint    `gorm:"column:uom_entry;primaryKey;not null" json:"uom_entry"`
	Price    float64 `gorm:"type:decimal(15,4);not null;default:0" json:"price"`
	Factor   float64 `gorm:"type:decimal(15,6);not null;default:0" json:"factor"` // "Reduce By %" discount

	Unit *UoMMaster `gorm:"foreignKey:UomEntry;references:UomEntry" json:"unit,omitempty"`
}

func (ItemUoMPrice) TableName() string { return "itm9" }

// ─────────────────────────────────────────────
// OITB — Item Groups (SAP B1 OITB)
// Replaces the old ItemCategory struct.
// Table renamed: item_categories → oitb
// ─────────────────────────────────────────────

type OITB struct {
	ItmsGrpCod  int    `gorm:"column:itms_grp_cod;primaryKey;autoIncrement" json:"itms_grp_cod"`
	ItmsGrpNam  string `gorm:"column:itms_grp_nam;type:varchar(100);not null;uniqueIndex" json:"itms_grp_nam"`
	Description string `gorm:"column:description;type:varchar(255)" json:"description"`
	CostingMeth string `gorm:"column:costing_meth;type:char(1);default:'A'" json:"costing_meth"` // A=MovAvg, S=Standard, F=FIFO, B=LIFO
	IsActive      bool   `gorm:"column:is_active;type:tinyint(1);default:1" json:"is_active"`
	ForSales      bool   `gorm:"column:for_sales;type:tinyint(1);default:1" json:"for_sales"`
	ForPurchasing bool   `gorm:"column:for_purchasing;type:tinyint(1);default:1" json:"for_purchasing"`
	ForInventory  bool   `gorm:"column:for_inventory;type:tinyint(1);default:1" json:"for_inventory"`
	ForProduction bool   `gorm:"column:for_production;type:tinyint(1);default:1" json:"for_production"`
}

func (OITB) TableName() string { return "oitb" }

// ItemCategory is a backward-compatibility alias for OITB.
// Existing handlers and services referencing ItemCategory continue to compile.
// Remove once all callers are updated to use OITB directly.
type ItemCategory = OITB

// ─────────────────────────────────────────────
// OITM — Item Master Data (SAP B1 OITM)
// Replaces the old ItemMaster struct.
// Table renamed: item_master → oitm
// ─────────────────────────────────────────────

type OITM struct {
	// Internal PK — retained for GORM update efficiency; not part of SAP B1 OITM
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// ── Core identification ───────────────────────────────────────────────
	ItemCode string `gorm:"column:item_code;type:varchar(50);uniqueIndex;not null" json:"item_code"`
	ItemName string `gorm:"column:item_name;type:varchar(150);not null" json:"item_name"`
	FrgnName string `gorm:"column:frgn_name;type:varchar(150)" json:"frgn_name"` // Alternate / foreign description

	// ── Classification ───────────────────────────────────────────────────
	ItmsGrpCod  int    `gorm:"column:itms_grp_cod;not null;default:0" json:"itms_grp_cod"` // FK → oitb.itms_grp_cod
	CodeBars    string `gorm:"column:code_bars;type:varchar(50)" json:"code_bars"`          // Barcode

	// ── Item type flags (CHAR 'Y'/'N' — SAP B1 convention) ───────────────
	InvntItem  string `gorm:"column:invnt_item;type:char(1);default:'Y'" json:"invnt_item"`    // Track inventory
	SellItem   string `gorm:"column:sell_item;type:char(1);default:'Y'" json:"sell_item"`      // Can be sold
	PrchseItem string `gorm:"column:prchse_item;type:char(1);default:'Y'" json:"prchse_item"`  // Can be purchased
	MakItem    string `gorm:"column:mak_item;type:char(1);default:'N'" json:"mak_item"`         // Manufactured (Y = has BOM)

	// ── Units of Measure ─────────────────────────────────────────────────
	InvntryUom   string  `gorm:"column:invntry_uom;type:varchar(20);not null" json:"invntry_uom"`  // Base / inventory UoM code
	PurchaseUnit string  `gorm:"column:purchase_unit;type:varchar(20)" json:"purchase_unit"`       // Default purchasing UoM code
	SalesUnit    string  `gorm:"column:sales_unit;type:varchar(20)" json:"sales_unit"`             // Default sales UoM code
	IUoMEntry    uint    `gorm:"column:i_uom_entry;not null;default:0" json:"i_uom_entry"`         // FK → ouom.uom_entry (inventory)
	SUoMEntry    uint    `gorm:"column:s_uom_entry;not null;default:0" json:"s_uom_entry"`         // FK → ouom.uom_entry (sales)
	PUoMEntry    uint    `gorm:"column:p_uom_entry;not null;default:0" json:"p_uom_entry"`         // FK → ouom.uom_entry (purchase)
	UgpEntry     uint    `gorm:"column:ugp_entry;not null;default:0" json:"ugp_entry"`             // FK → ougp.ugp_entry (UoM group)
	NumInBuy     float64 `gorm:"column:num_in_buy;type:decimal(15,6);default:1" json:"num_in_buy"`  // Items per purchase unit
	NumInSale    float64 `gorm:"column:num_in_sale;type:decimal(15,6);default:1" json:"num_in_sale"` // Items per sales unit

	// ── Stock quantities (system-maintained) ─────────────────────────────
	OnHand     float64 `gorm:"column:on_hand;type:decimal(12,3);default:0" json:"on_hand"`      // Total on-hand (all warehouses)
	IsCommited float64 `gorm:"column:is_commited;type:decimal(12,3);default:0" json:"is_commited"` // Committed (open SOs)
	OnOrder    float64 `gorm:"column:on_order;type:decimal(12,3);default:0" json:"on_order"`     // On order (open POs)

	// ── Reorder planning ─────────────────────────────────────────────────
	MinLevel float64 `gorm:"column:min_level;type:decimal(12,3);default:0" json:"min_level"` // Reorder point
	MaxLevel float64 `gorm:"column:max_level;type:decimal(12,3);default:0" json:"max_level"` // Maximum stock
	LeadTime int     `gorm:"column:lead_time;type:int;default:0" json:"lead_time"`           // Procurement lead time (days)

	// ── Pricing & cost ───────────────────────────────────────────────────
	AvgPrice   float64 `gorm:"column:avg_price;type:decimal(15,4);default:0" json:"avg_price"`     // Moving average / standard cost
	LstEvlPric float64 `gorm:"column:lst_evl_pric;type:decimal(15,4);default:0" json:"lst_evl_pric"` // Last evaluated price (FIFO/LIFO)
	LastPurPrc float64 `gorm:"column:last_pur_prc;type:decimal(15,4);default:0" json:"last_pur_prc"` // Last purchase price
	EvalSystem string  `gorm:"column:eval_system;type:char(1);default:'A'" json:"eval_system"`      // A=MovAvg, S=Std, F=FIFO, B=LIFO
	PricingCod string  `gorm:"column:pricing_cod;type:varchar(20)" json:"pricing_cod"`              // Default price list → OPLN

	// ── Warehouse & vendor ───────────────────────────────────────────────
	DfltWh     string `gorm:"column:dflt_wh;type:varchar(10)" json:"dflt_wh"`          // Default warehouse → owhs.whs_code
	CardCode   string `gorm:"column:card_code;type:varchar(15)" json:"card_code"`       // Preferred vendor → OCRD.CardCode
	SuppCatNum string `gorm:"column:supp_cat_num;type:varchar(20)" json:"supp_cat_num"` // Vendor catalog number

	// ── Tax & compliance ─────────────────────────────────────────────────
	VatGourpSa string `gorm:"column:vat_gourp_sa;type:varchar(10)" json:"vat_gourp_sa"` // Sales tax group → OVTG
	WTLiable   string `gorm:"column:wt_liable;type:char(1);default:'N'" json:"wt_liable"` // Withholding tax liable

	// ── Batch / Serial tracking ──────────────────────────────────────────
	ManBtchNum string `gorm:"column:man_btch_num;type:char(1);default:'N'" json:"man_btch_num"` // Manage by batch
	ManSerNum  string `gorm:"column:man_ser_num;type:char(1);default:'N'" json:"man_ser_num"`  // Manage by serial number

	// ── Status & audit ───────────────────────────────────────────────────
	ValidFor    string     `gorm:"column:valid_for;type:char(1);default:'Y'" json:"valid_for"` // Active: Y/N
	Description string     `gorm:"column:description;type:text" json:"description"`
	CreateDate  *time.Time `gorm:"column:create_date;type:date" json:"create_date"`
	UpdateDate  *time.Time `gorm:"column:update_date;type:date" json:"update_date"`

	// ── GORM associations ────────────────────────────────────────────────
	UoMPrices []ItemUoMPrice `gorm:"foreignKey:ItemCode;references:ItemCode;constraint:OnDelete:CASCADE" json:"uom_prices,omitempty"`
	ItemGroup *OITB          `gorm:"foreignKey:ItmsGrpCod;references:ItmsGrpCod" json:"item_group,omitempty"`

	// ── Virtual fields (populated at service layer, no DB column) ────────
	CategoryName string     `gorm:"-" json:"category_name,omitempty"` // Populated from OITB.ItmsGrpNam
	BaseUoM      *UoMMaster `gorm:"-" json:"base_uom,omitempty"`
	SalesUoM     *UoMMaster `gorm:"-" json:"sales_uom,omitempty"`
	PurchUoM     *UoMMaster `gorm:"-" json:"purch_uom,omitempty"`
}

func (OITM) TableName() string { return "oitm" }

// ItemMaster is a backward-compatibility alias for OITM.
// Existing handlers, services, and production code referencing ItemMaster continue to compile.
// Remove once all callers are updated to use OITM directly.
type ItemMaster = OITM

// ─────────────────────────────────────────────
// Legacy Inventory (keep for migration safety)
// ─────────────────────────────────────────────

type Inventory struct {
	ID       uint    `gorm:"primaryKey" json:"id"`
	ItemName string  `gorm:"type:varchar(100);not null" json:"item_name"`
	Category string  `gorm:"type:varchar(50)" json:"category"`
	Quantity float64 `gorm:"type:decimal(12,3);default:0" json:"quantity"`
	Unit     string  `gorm:"type:varchar(20)" json:"unit"`
}

func (Inventory) TableName() string { return "inventory" }

// ─────────────────────────────────────────────
// OWHS — Warehouse Master (SAP B1 OWHS)
// ─────────────────────────────────────────────

type OWHS struct {
	WhsCode  string `gorm:"column:whs_code;primaryKey;type:varchar(10)" json:"whs_code"`
	WhsName  string `gorm:"column:whs_name;type:varchar(100);not null" json:"whs_name"`
	Location string `gorm:"column:location;type:varchar(200)" json:"location"`
	Street   string `gorm:"column:street;type:varchar(200)" json:"street"`
	ZipCode  string `gorm:"column:zip_code;type:varchar(20)" json:"zip_code"`
	City     string `gorm:"column:city;type:varchar(100)" json:"city"`
	State    string `gorm:"column:state;type:varchar(100)" json:"state"`
	Phone    string `gorm:"column:phone;type:varchar(50)" json:"phone"`
	Inactive string `gorm:"column:inactive;type:char(1);default:'N'" json:"inactive"` // N=Active, Y=Inactive
}

func (OWHS) TableName() string { return "owhs" }

// ─────────────────────────────────────────────
// OITW — Item Warehouse Info (SAP B1 OITW)
// Per-warehouse stock levels for each item.
// Composite PK: (item_code, whs_code)
// ─────────────────────────────────────────────

type OITW struct {
	ItemCode   string  `gorm:"column:item_code;primaryKey;type:varchar(50)" json:"item_code"`
	WhsCode    string  `gorm:"column:whs_code;primaryKey;type:varchar(10)" json:"whs_code"`
	OnHand     float64 `gorm:"column:on_hand;type:decimal(12,3);default:0" json:"on_hand"`
	IsCommited float64 `gorm:"column:is_commited;type:decimal(12,3);default:0" json:"is_commited"`
	OnOrder    float64 `gorm:"column:on_order;type:decimal(12,3);default:0" json:"on_order"`
	MinStock   float64 `gorm:"column:min_stock;type:decimal(12,3);default:0" json:"min_stock"`
	MaxStock   float64 `gorm:"column:max_stock;type:decimal(12,3);default:0" json:"max_stock"`
	AvgPrice   float64 `gorm:"column:avg_price;type:decimal(15,4);default:0" json:"avg_price"`
	DfltBin    string  `gorm:"column:dflt_bin;type:varchar(20)" json:"dflt_bin"`
	Locked     string  `gorm:"column:locked;type:char(1);default:'N'" json:"locked"`
}

func (OITW) TableName() string { return "oitw" }

// ─────────────────────────────────────────────
// OIVL — Inventory Valuation Ledger (SAP B1 OIVL)
// Audit trail for every inventory movement (GR, GI, SO delivery, PO receipt, production).
// ─────────────────────────────────────────────

type OIVL struct {
	DocEntry    uint      `gorm:"column:doc_entry;primaryKey;autoIncrement" json:"doc_entry"`
	ItemCode    string    `gorm:"column:item_code;type:varchar(50);not null;index" json:"item_code"`
	ItemName    string    `gorm:"column:item_name;type:varchar(150)" json:"item_name"`
	Warehouse   string    `gorm:"column:warehouse;type:varchar(10)" json:"warehouse"`
	DocDate     string    `gorm:"column:doc_date;type:date" json:"doc_date"`
	TransType   string    `gorm:"column:trans_type;type:varchar(20)" json:"trans_type"` // GR|GI|PO|SO|MO|CANCEL
	DocNum      int       `gorm:"column:doc_num;type:int;default:0" json:"doc_num"`
	InQty       float64   `gorm:"column:in_qty;type:decimal(12,3);default:0" json:"in_qty"`
	OutQty      float64   `gorm:"column:out_qty;type:decimal(12,3);default:0" json:"out_qty"`
	Price       float64   `gorm:"column:price;type:decimal(15,4);default:0" json:"price"`
	Value       float64   `gorm:"column:value;type:decimal(15,4);default:0" json:"value"`
	CreatedByID uint      `gorm:"column:created_by_id;default:0;index" json:"created_by_id"`
	CreatedAt   time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
}

func (OIVL) TableName() string { return "oivl" }

// ─────────────────────────────────────────────
// ITM1 — Item Prices per Price List (SAP B1 ITM1)
// One row per item per price list.
// Composite PK: (item_code, price_list)
// ─────────────────────────────────────────────

type ITM1 struct {
	ItemCode  string  `gorm:"column:item_code;primaryKey;type:varchar(50)" json:"item_code"`
	PriceList int     `gorm:"column:price_list;primaryKey" json:"price_list"` // FK → OPLN.ListNum
	Price     float64 `gorm:"column:price;type:decimal(15,4);not null;default:0" json:"price"`
	Currency  string  `gorm:"column:currency;type:varchar(3);default:'PHP'" json:"currency"`
	PriceDec  int     `gorm:"column:price_dec;type:int;default:2" json:"price_dec"`
}

func (ITM1) TableName() string { return "itm1" }

// ─────────────────────────────────────────────
// ITM2 — Item Preferred Vendors (SAP B1 ITM2)
// Multiple preferred vendors per item.
// Composite PK: (item_code, line_num)
// ─────────────────────────────────────────────

type ITM2 struct {
	ItemCode   string `gorm:"column:item_code;primaryKey;type:varchar(50)" json:"item_code"`
	LineNum    int    `gorm:"column:line_num;primaryKey" json:"line_num"`
	CardCode   string `gorm:"column:card_code;type:varchar(15);index" json:"card_code"` // FK → OCRD.CardCode
	VendorName string `gorm:"column:vendor_name;type:varchar(150)" json:"vendor_name"`
	SuppCatNum string `gorm:"column:supp_cat_num;type:varchar(20)" json:"supp_cat_num"` // Vendor catalog number
}

func (ITM2) TableName() string { return "itm2" }

// ─────────────────────────────────────────────
// ITM12 — UoM per Item Mapping (SAP B1 ITM12)
// Defines which UoMs are valid for each item and their barcodes.
// Composite PK: (item_code, uom_entry)
// ─────────────────────────────────────────────

type ITM12 struct {
	ItemCode string `gorm:"column:item_code;primaryKey;type:varchar(50)" json:"item_code"`
	UomEntry uint   `gorm:"column:uom_entry;primaryKey" json:"uom_entry"` // FK → ouom.uom_entry
	UomType  string `gorm:"column:uom_type;type:char(1)" json:"uom_type"` // I=Inventory, S=Sales, P=Purchase
	BarCode  string `gorm:"column:bar_code;type:varchar(50)" json:"bar_code"`
}

func (ITM12) TableName() string { return "itm12" }

// ─────────────────────────────────────────────
// OSPP — Special Prices per Business Partner (SAP B1 OSPP)
// BP-specific price overrides for items.
// Composite PK: (card_code, item_code)
// ─────────────────────────────────────────────

type OSPP struct {
	CardCode   string  `gorm:"column:card_code;primaryKey;type:varchar(15)" json:"card_code"` // FK → OCRD.CardCode
	ItemCode   string  `gorm:"column:item_code;primaryKey;type:varchar(50)" json:"item_code"` // FK → oitm.item_code
	Price      float64 `gorm:"column:price;type:decimal(15,4);not null;default:0" json:"price"`
	Currency   string  `gorm:"column:currency;type:varchar(3);default:'PHP'" json:"currency"`
	Discount   float64 `gorm:"column:discount;type:decimal(5,2);default:0" json:"discount"`   // % discount (alt to fixed price)
	PriceList  int     `gorm:"column:price_list;default:0" json:"price_list"`                 // Base price list this overrides
	AutoUpdate string  `gorm:"column:auto_update;type:char(1);default:'Y'" json:"auto_update"` // Auto-recalculate on list change
}

func (OSPP) TableName() string { return "ospp" }
