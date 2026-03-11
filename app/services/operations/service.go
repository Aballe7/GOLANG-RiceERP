package operations

import (
	"fmt"
	"time"

	"egglayererp/app/db"
	"egglayererp/app/models"

	"gorm.io/gorm"
)

// RecordDailyLogRequest is the input for recording a daily (layer) log.
type RecordDailyLogRequest struct {
	FlockID        uint    `json:"flock_id"`
	Date           string  `json:"date"`
	Pewee          int     `json:"pewee"`
	Pullet         int     `json:"pullet"`
	Small          int     `json:"small"`
	Medium         int     `json:"medium"`
	Large          int     `json:"large"`
	ExtraLarge     int     `json:"extra_large"`
	Jumbo          int     `json:"jumbo"`
	DoubleYolk     int     `json:"double_yolk"`
	CrackedDirty   int     `json:"cracked_dirty"`
	FeedConsumedKg float64 `json:"feed_consumed_kg"`
	Mortality      int     `json:"mortality"`
	VaccineName    string  `json:"vaccine_name"`
	FeedTypeID     *uint   `json:"feed_type_id"`
	CreatedByID    *uint   `json:"created_by_id"`
}

// RecordDailyLog saves a layer house daily log and updates flock count + feed stock.
func RecordDailyLog(req RecordDailyLogRequest) error {
	t, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		// Deduct feed stock
		if req.FeedTypeID != nil && req.FeedConsumedKg > 0 {
			var feed models.FeedStock
			if err := tx.First(&feed, *req.FeedTypeID).Error; err == nil && feed.KgPerSack > 0 {
				sacksUsed := req.FeedConsumedKg / feed.KgPerSack
				tx.Model(&feed).UpdateColumn("total_sacks", gorm.Expr("total_sacks - ?", sacksUsed))
			}
		}

		// Deduct mortality from flock
		if req.Mortality > 0 {
			tx.Model(&models.Flock{}).Where("id = ?", req.FlockID).
				UpdateColumn("current_count", gorm.Expr("GREATEST(0, current_count - ?)", req.Mortality))
		}

		log := models.DailyLog{
			FlockID:        req.FlockID,
			Date:           t,
			Pewee:          req.Pewee,
			Pullet:         req.Pullet,
			Small:          req.Small,
			Medium:         req.Medium,
			Large:          req.Large,
			ExtraLarge:     req.ExtraLarge,
			Jumbo:          req.Jumbo,
			DoubleYolk:     req.DoubleYolk,
			CrackedDirty:   req.CrackedDirty,
			FeedConsumedKg: req.FeedConsumedKg,
			Mortality:      req.Mortality,
			VaccineName:    req.VaccineName,
			CreatedByID:    req.CreatedByID,
		}
		return tx.Create(&log).Error
	})
}

// RecordGrowerLogRequest is the input for recording a grower house daily log.
type RecordGrowerLogRequest struct {
	FlockID        uint    `json:"flock_id"`
	Date           string  `json:"date"`
	FeedConsumedKg float64 `json:"feed_consumed_kg"`
	Mortality      int     `json:"mortality"`
	VaccineName    string  `json:"vaccine_name"`
	Medication     string  `json:"medication"`
	Remarks        string  `json:"remarks"`
	FeedTypeID     *uint   `json:"feed_type_id"`
	CreatedByID    *uint   `json:"created_by_id"`
}

// RecordGrowerLog saves a grower house daily log.
func RecordGrowerLog(req RecordGrowerLogRequest) error {
	t, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		if req.FeedTypeID != nil && req.FeedConsumedKg > 0 {
			var feed models.FeedStock
			if err := tx.First(&feed, *req.FeedTypeID).Error; err == nil && feed.KgPerSack > 0 {
				sacksUsed := req.FeedConsumedKg / feed.KgPerSack
				tx.Model(&feed).UpdateColumn("total_sacks", gorm.Expr("GREATEST(0, total_sacks - ?)", sacksUsed))
			}
		}
		if req.Mortality > 0 {
			tx.Model(&models.Flock{}).Where("id = ?", req.FlockID).
				UpdateColumn("current_count", gorm.Expr("GREATEST(0, current_count - ?)", req.Mortality))
		}
		entry := models.GrowerLog{
			FlockID:        req.FlockID,
			Date:           t,
			FeedConsumedKg: req.FeedConsumedKg,
			Mortality:      req.Mortality,
			VaccineName:    req.VaccineName,
			Medication:     req.Medication,
			Remarks:        req.Remarks,
			CreatedByID:    req.CreatedByID,
		}
		return tx.Create(&entry).Error
	})
}

// UpdateDailyLog edits an existing layer log.
func UpdateDailyLog(id uint, req RecordDailyLogRequest) error {
	t, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var old models.DailyLog
		if err := tx.First(&old, id).Error; err != nil {
			return err
		}
		diff := req.Mortality - old.Mortality
		if diff != 0 {
			tx.Model(&models.Flock{}).Where("id = ?", old.FlockID).
				UpdateColumn("current_count", gorm.Expr("GREATEST(0, current_count - ?)", diff))
		}
		return tx.Model(&old).Updates(map[string]interface{}{
			"date": t, "pewee": req.Pewee, "pullet": req.Pullet,
			"small": req.Small, "medium": req.Medium, "large": req.Large,
			"extra_large": req.ExtraLarge, "jumbo": req.Jumbo,
			"double_yolk": req.DoubleYolk, "cracked_dirty": req.CrackedDirty,
			"feed_consumed_kg": req.FeedConsumedKg, "mortality": req.Mortality,
			"vaccine_name": req.VaccineName,
		}).Error
	})
}

// UpdateGrowerLog edits an existing grower log.
func UpdateGrowerLog(id uint, req RecordGrowerLogRequest) error {
	t, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var old models.GrowerLog
		if err := tx.First(&old, id).Error; err != nil {
			return err
		}
		diff := req.Mortality - old.Mortality
		if diff != 0 {
			tx.Model(&models.Flock{}).Where("id = ?", old.FlockID).
				UpdateColumn("current_count", gorm.Expr("GREATEST(0, current_count - ?)", diff))
		}
		return tx.Model(&old).Updates(map[string]interface{}{
			"date": t, "feed_consumed_kg": req.FeedConsumedKg, "mortality": req.Mortality,
			"vaccine_name": req.VaccineName, "medication": req.Medication, "remarks": req.Remarks,
		}).Error
	})
}

// GetDailyLog returns a single daily log by ID.
func GetDailyLog(id uint) (*models.DailyLog, error) {
	var log models.DailyLog
	err := db.DB.First(&log, id).Error
	return &log, err
}

// GetGrowerLog returns a single grower log by ID.
func GetGrowerLog(id uint) (*models.GrowerLog, error) {
	var log models.GrowerLog
	err := db.DB.First(&log, id).Error
	return &log, err
}

// ListFeedStocks returns all feed stocks.
func ListFeedStocks() ([]models.FeedStock, error) {
	var stocks []models.FeedStock
	err := db.DB.Order("name").Find(&stocks).Error
	return stocks, err
}

// UpsertFeedStock creates or updates a feed stock entry.
func UpsertFeedStock(name string, totalSacks, kgPerSack float64) error {
	var stock models.FeedStock
	if err := db.DB.Where("LOWER(name) = LOWER(?)", name).First(&stock).Error; err != nil {
		// Create new
		return db.DB.Create(&models.FeedStock{Name: name, TotalSacks: totalSacks, KgPerSack: kgPerSack}).Error
	}
	return db.DB.Model(&stock).Updates(map[string]interface{}{
		"total_sacks": gorm.Expr("total_sacks + ?", totalSacks),
	}).Error
}

// ListVaccineStocks returns all vaccine stocks.
func ListVaccineStocks() ([]models.VaccineStock, error) {
	var stocks []models.VaccineStock
	err := db.DB.Order("name").Find(&stocks).Error
	return stocks, err
}

// ListVaccineSchedules returns vaccine schedules. If flockID is 0, all active schedules are returned.
func ListVaccineSchedules(flockID uint) ([]models.VaccineSchedule, error) {
	var schedules []models.VaccineSchedule
	q := db.DB.Where("is_active = 1").Order("scheduled_date")
	if flockID != 0 {
		q = q.Where("flock_id = ?", flockID)
	}
	err := q.Find(&schedules).Error
	return schedules, err
}

// CreateVaccineSchedule creates a new vaccine schedule entry.
func CreateVaccineSchedule(flockID uint, vaccineName, scheduledDate, adminMethod string, createdByID *uint) error {
	sched := models.VaccineSchedule{
		FlockID:       flockID,
		VaccineName:   vaccineName,
		ScheduledDate: scheduledDate,
		AdminMethod:   adminMethod,
		IsCompleted:   false,
		IsActive:      true,
		CreatedByID:   createdByID,
	}
	return db.DB.Create(&sched).Error
}

// MarkVaccineComplete marks a vaccine schedule as completed.
func MarkVaccineComplete(id uint) error {
	return db.DB.Model(&models.VaccineSchedule{}).Where("id = ?", id).
		Update("is_completed", true).Error
}
