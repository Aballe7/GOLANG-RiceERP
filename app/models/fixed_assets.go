package models

import "time"

// FixedAsset tracks a physical asset subject to depreciation (PAS 16).
// DepreciationMethod: STRAIGHT_LINE | DECLINING_BALANCE
// Status:            ACTIVE | FULLY_DEPRECIATED | DISPOSED
type FixedAsset struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	AssetCode           string     `gorm:"type:varchar(30);uniqueIndex;not null" json:"asset_code"`
	AssetName           string     `gorm:"type:varchar(150);not null" json:"asset_name"`
	Category            string     `gorm:"type:varchar(50);not null" json:"category"` // Land|Building|Equipment|Vehicle|Office Equipment
	Description         string     `gorm:"type:text" json:"description"`
	AcquisitionDate     time.Time  `gorm:"type:date;not null" json:"acquisition_date"`
	AcquisitionCost     float64    `gorm:"type:decimal(15,4);not null" json:"acquisition_cost"`
	ResidualValue       float64    `gorm:"type:decimal(15,4);default:0" json:"residual_value"`
	UsefulLifeMonths    int        `gorm:"not null" json:"useful_life_months"`
	DepreciationMethod  string     `gorm:"type:varchar(20);not null;default:'STRAIGHT_LINE'" json:"depreciation_method"`
	GLAssetAccountID    uint       `gorm:"not null" json:"gl_asset_account_id"`
	GLAccumDepAccountID uint       `gorm:"not null" json:"gl_accum_dep_account_id"`
	GLDepExpAccountID   uint       `gorm:"not null" json:"gl_dep_exp_account_id"`
	AccumDepreciation   float64    `gorm:"type:decimal(15,4);default:0" json:"accum_depreciation"`
	Status              string     `gorm:"type:varchar(20);not null;default:'ACTIVE'" json:"status"`
	DisposalDate        *time.Time `gorm:"type:date" json:"disposal_date"`
	DisposalProceeds    *float64   `gorm:"type:decimal(15,4)" json:"disposal_proceeds"`
	DisposalJEID        *uint      `json:"disposal_je_id"`
	Notes               string     `gorm:"type:text" json:"notes"`
	CreatedByID         *uint      `json:"created_by_id"`
	UpdatedByID         *uint      `json:"updated_by_id"`
	CreatedAt           time.Time  `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`

	AssetAccount    *GLAccount         `gorm:"foreignKey:GLAssetAccountID" json:"asset_account,omitempty"`
	AccumDepAccount *GLAccount         `gorm:"foreignKey:GLAccumDepAccountID" json:"accum_dep_account,omitempty"`
	DepExpAccount   *GLAccount         `gorm:"foreignKey:GLDepExpAccountID" json:"dep_exp_account,omitempty"`
	DepLogs         []DepreciationEntry `gorm:"foreignKey:FixedAssetID" json:"dep_logs,omitempty"`
}

func (FixedAsset) TableName() string { return "fixed_asset" }

func (fa *FixedAsset) BookValue() float64 {
	return fa.AcquisitionCost - fa.AccumDepreciation
}

func (fa *FixedAsset) DepreciableAmount() float64 {
	return fa.AcquisitionCost - fa.ResidualValue
}

// DepreciationEntry is the log of every posted depreciation period for an asset.
type DepreciationEntry struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	FixedAssetID   uint      `gorm:"not null;index" json:"fixed_asset_id"`
	PeriodDate     time.Time `gorm:"type:date;not null" json:"period_date"` // first day of the month
	Amount         float64   `gorm:"type:decimal(15,4);not null" json:"amount"`
	AccumDepAfter  float64   `gorm:"type:decimal(15,4);not null" json:"accum_dep_after"`
	BookValueAfter float64   `gorm:"type:decimal(15,4);not null" json:"book_value_after"`
	JournalEntryID *uint     `json:"journal_entry_id"`
	CreatedAt      time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`

	FixedAsset FixedAsset `gorm:"foreignKey:FixedAssetID" json:"-"`
}

func (DepreciationEntry) TableName() string { return "depreciation_entry" }
