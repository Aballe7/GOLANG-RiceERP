package models

import "time"

type VaccineSchedule struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	FlockID       uint      `gorm:"not null;index" json:"flock_id"`
	VaccineName   string    `gorm:"type:varchar(100);not null" json:"vaccine_name"`
	ScheduledDate string    `gorm:"type:date;not null" json:"scheduled_date"`
	AdminMethod   string    `gorm:"type:varchar(50)" json:"admin_method"`
	IsCompleted   bool      `gorm:"type:tinyint(1);default:0" json:"is_completed"`
	IsActive      bool      `gorm:"type:tinyint(1);default:1" json:"is_active"`
	CreatedByID   *uint     `json:"created_by_id"`
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (VaccineSchedule) TableName() string { return "vaccine_schedule" }

type FeedStock struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	Name      string  `gorm:"type:varchar(50);not null" json:"name"`
	TotalSacks float64 `gorm:"type:decimal(12,3);default:0" json:"total_sacks"`
	KgPerSack float64 `gorm:"type:decimal(10,2);default:50" json:"kg_per_sack"`
}

func (FeedStock) TableName() string { return "feed_stock" }

func (f *FeedStock) TotalKg() float64 {
	return f.TotalSacks * f.KgPerSack
}

type VaccineStock struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"type:varchar(50);not null" json:"name"`
	Quantity int    `gorm:"default:0" json:"quantity"`
}

func (VaccineStock) TableName() string { return "vaccine_stock" }
