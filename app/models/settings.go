package models

// FarmSettings stores key-value configuration for the farm.
type FarmSettings struct {
	Key   string `gorm:"type:varchar(50);primaryKey" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}

func (FarmSettings) TableName() string { return "farm_settings" }
