package models

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
