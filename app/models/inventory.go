package models

import "time"

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
	UgpEntry   uint      `gorm:"primaryKey;autoIncrement" json:"ugp_entry"`
	UgpCode    string    `gorm:"type:varchar(20);not null;uniqueIndex" json:"ugp_code"`
	UgpName    string    `gorm:"type:varchar(100);not null" json:"ugp_name"`
	BaseUom    int       `gorm:"default:0" json:"base_uom"`
	UserSign   int       `gorm:"default:0" json:"user_sign"`
	UserSign2  int       `gorm:"default:0" json:"user_sign2"`
	UpdateDate time.Time `gorm:"type:datetime" json:"update_date"`
	CreateDate time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"create_date"`
	DataSource string    `gorm:"type:char(1);default:'M'" json:"data_source"`
	Lines      []UoMGroupLine `gorm:"foreignKey:UgpEntry;references:UgpEntry" json:"lines,omitempty"`
	BaseUnit   *UoMMaster     `gorm:"foreignKey:UomEntry;references:BaseUom" json:"base_unit,omitempty"`
}

func (UoMGroup) TableName() string { return "ougp" }

// ─────────────────────────────────────────────
// UoM Group Lines / Definition (UGP1)
// ─────────────────────────────────────────────

type UoMGroupLine struct {
	UgpEntry   uint    `gorm:"primaryKey" json:"ugp_entry"`
	LineNum    int     `gorm:"primaryKey" json:"line_num"`
	UomEntry   uint    `gorm:"not null" json:"uom_entry"`
	AltQty     float64 `gorm:"type:decimal(15,6);default:1" json:"alt_qty"`
	BaseQty    float64 `gorm:"type:decimal(15,6);default:1" json:"base_qty"`
	LogInstanc int     `gorm:"default:0" json:"log_instanc"`
	ObjType    string  `gorm:"type:varchar(20);default:'173'" json:"obj_type"`
	Unit       *UoMMaster `gorm:"foreignKey:UomEntry;references:UomEntry" json:"unit,omitempty"`
}

func (UoMGroupLine) TableName() string { return "ugp1" }

// ─────────────────────────────────────────────
// Item Categories
// ─────────────────────────────────────────────

type ItemCategory struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	Description string `gorm:"type:varchar(255)" json:"description"`
	IsActive    bool   `gorm:"type:tinyint(1);default:1" json:"is_active"`
}

func (ItemCategory) TableName() string { return "item_categories" }

type Inventory struct {
	ID       uint    `gorm:"primaryKey" json:"id"`
	ItemName string  `gorm:"type:varchar(100);not null" json:"item_name"`
	Category string  `gorm:"type:varchar(50)" json:"category"` // Feed, Trays, Meds
	Quantity float64 `gorm:"type:decimal(12,3);default:0" json:"quantity"`
	Unit     string  `gorm:"type:varchar(20)" json:"unit"` // kg, bags, pcs
}

func (Inventory) TableName() string { return "inventory" }

type ItemMaster struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	ItemCode     string  `gorm:"type:varchar(20);uniqueIndex;not null" json:"item_code"`
	Name         string  `gorm:"type:varchar(150);not null" json:"name"`
	Category     string  `gorm:"type:varchar(50);not null" json:"category"` // Feeds, Vaccines, Medicine, Supplies, Equipment, Chicks
	Unit         string  `gorm:"type:varchar(20);not null" json:"unit"`
	UnitPrice    float64 `gorm:"type:decimal(15,4);default:0" json:"unit_price"`
	Description  string  `gorm:"type:text" json:"description"`
	ReorderLevel float64 `gorm:"type:decimal(12,3);default:0" json:"reorder_level"`
	IsActive     bool    `gorm:"type:tinyint(1);default:1" json:"is_active"`
}

func (ItemMaster) TableName() string { return "item_master" }
