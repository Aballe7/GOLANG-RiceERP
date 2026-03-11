package models

import "time"

type Flock struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	Name                  string     `gorm:"type:varchar(100);not null" json:"name"`
	Status                string     `gorm:"type:varchar(20);default:'Grower'" json:"status"` // Grower | Layer | Retired
	BuildingName          string     `gorm:"type:varchar(50)" json:"building_name"`
	HouseNumber           string     `gorm:"type:varchar(10);not null" json:"house_number"`
	HouseType             string     `gorm:"type:varchar(20);not null" json:"house_type"` // Layer | Grower
	Breed                 string     `gorm:"type:varchar(50)" json:"breed"`
	InitialCount          int        `gorm:"not null" json:"initial_count"`
	CurrentCount          int        `gorm:"not null" json:"current_count"`
	HatchDate             *time.Time `gorm:"type:date" json:"hatch_date"`
	RetirementDate        *time.Time `gorm:"type:date" json:"retirement_date"`
	RetirementType        string     `gorm:"type:varchar(20)" json:"retirement_type"` // Sold | Culled | Molted
	BirdsRetired          int        `json:"birds_retired"`
	RetirementPricePerBird float64   `gorm:"type:decimal(15,4)" json:"retirement_price_per_bird"`
	RetirementBuyer       string     `gorm:"type:varchar(150)" json:"retirement_buyer"`
	RetirementNotes       string     `gorm:"type:text" json:"retirement_notes"`
	CreatedByID           *uint      `json:"created_by_id"`

	DailyLogs  []DailyLog        `gorm:"foreignKey:FlockID" json:"-"`
	GrowerLogs []GrowerLog       `gorm:"foreignKey:FlockID" json:"-"`
	Vaccines   []VaccineSchedule `gorm:"foreignKey:FlockID" json:"-"`
	WeightLogs []BodyWeightLog   `gorm:"foreignKey:FlockID" json:"-"`
}

func (Flock) TableName() string { return "flock" }

type DailyLog struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	FlockID        uint       `gorm:"not null;index" json:"flock_id"`
	Date           time.Time  `gorm:"type:date;not null" json:"date"`
	Pewee          int        `gorm:"default:0" json:"pewee"`
	Pullet         int        `gorm:"default:0" json:"pullet"`
	Small          int        `gorm:"default:0" json:"small"`
	Medium         int        `gorm:"default:0" json:"medium"`
	Large          int        `gorm:"default:0" json:"large"`
	ExtraLarge     int        `gorm:"default:0" json:"extra_large"`
	Jumbo          int        `gorm:"default:0" json:"jumbo"`
	DoubleYolk     int        `gorm:"default:0" json:"double_yolk"`
	CrackedDirty   int        `gorm:"default:0" json:"cracked_dirty"`
	FeedConsumedKg float64    `gorm:"type:decimal(10,3);default:0" json:"feed_consumed_kg"`
	Mortality      int        `gorm:"default:0" json:"mortality"`
	VaccineName    string     `gorm:"type:varchar(100)" json:"vaccine_name"`
	CreatedByID    *uint      `json:"created_by_id"`
	CreatedAt      *time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (DailyLog) TableName() string { return "daily_log" }

type GrowerLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	FlockID        uint      `gorm:"not null;index" json:"flock_id"`
	Date           time.Time `gorm:"type:date;not null" json:"date"`
	FeedConsumedKg float64   `gorm:"type:decimal(10,3);default:0" json:"feed_consumed_kg"`
	Mortality      int       `gorm:"default:0" json:"mortality"`
	VaccineName    string    `gorm:"type:varchar(100)" json:"vaccine_name"`
	Medication     string    `gorm:"type:varchar(100)" json:"medication"`
	Remarks        string    `gorm:"type:text" json:"remarks"`
	CreatedByID    *uint     `json:"created_by_id"`
}

func (GrowerLog) TableName() string { return "grower_log" }

type BodyWeightLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	FlockID     uint      `gorm:"not null;index" json:"flock_id"`
	Date        time.Time `gorm:"type:date;not null" json:"date"`
	WeekAge     int       `gorm:"not null" json:"week_age"`
	AvgWeightG  float64   `gorm:"type:decimal(10,2);not null" json:"avg_weight_g"`
	SampleSize  int       `gorm:"default:50" json:"sample_size"`
	Notes       string    `gorm:"type:text" json:"notes"`
	IsActive    bool      `gorm:"type:tinyint(1);default:1" json:"is_active"`
	CreatedByID *uint      `json:"created_by_id"`
	CreatedAt   *time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (BodyWeightLog) TableName() string { return "body_weight_log" }

type EggProduction struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Date       time.Time `gorm:"type:date;not null;index" json:"date"`
	FlockID    uint      `gorm:"not null;index" json:"flock_id"`
	GoodEggs   int       `gorm:"default:0" json:"good_eggs"`
	RejectEggs int       `gorm:"default:0" json:"reject_eggs"`
	TotalEggs  int       `json:"total_eggs"`
}

func (EggProduction) TableName() string { return "egg_production" }
