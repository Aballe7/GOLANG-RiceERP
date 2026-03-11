package flocks

import (
	"errors"
	"fmt"
	"time"

	"egglayererp/app/constants"
	"egglayererp/app/db"
	"egglayererp/app/models"

	"gorm.io/gorm"
)

type FlockData struct {
	Flock       models.Flock `json:"flock"`
	AgeWeeks    *int         `json:"age_weeks"`
	AgeDays     *int         `json:"age_days"`
	WeeksInLay  *int         `json:"weeks_in_lay"`
	Stage       string       `json:"stage"`
	StageColor  string       `json:"stage_color"`
	LoggedToday bool         `json:"logged_today"`
}

func calcFlockData(f models.Flock, today time.Time, loggedLayerIDs, loggedGrowerIDs map[uint]bool) FlockData {
	fd := FlockData{Flock: f}

	if f.HatchDate != nil {
		ageDays := int(today.Sub(*f.HatchDate).Hours() / 24)
		ageWeeks := ageDays / 7
		fd.AgeDays = &ageDays
		fd.AgeWeeks = &ageWeeks

		if f.HouseType == "Layer" && ageWeeks >= constants.LayStartWeek {
			wil := ageWeeks - constants.LayStartWeek
			fd.WeeksInLay = &wil
		}

		stage, color := constants.GetFlockStage(f.HouseType, ageWeeks)
		fd.Stage = stage
		fd.StageColor = color
	} else {
		fd.Stage = "Unknown"
		fd.StageColor = "secondary"
	}

	if f.HouseType == "Layer" {
		fd.LoggedToday = loggedLayerIDs[f.ID]
	} else {
		fd.LoggedToday = loggedGrowerIDs[f.ID]
	}

	return fd
}

// ListFlocks returns all flocks with computed stage data.
func ListFlocks() (active, retired []FlockData, err error) {
	var flocks []models.Flock
	if err = db.DB.Order("house_type, house_number").Find(&flocks).Error; err != nil {
		return
	}

	today := time.Now().Truncate(24 * time.Hour)

	// Find which flocks were logged today
	type IDRow struct{ FlockID uint }
	var layerLogged, growerLogged []IDRow
	db.DB.Model(&models.DailyLog{}).Select("flock_id").Where("DATE(date) = ?", today.Format("2006-01-02")).Scan(&layerLogged)
	db.DB.Model(&models.GrowerLog{}).Select("flock_id").Where("DATE(date) = ?", today.Format("2006-01-02")).Scan(&growerLogged)

	layerIDs := make(map[uint]bool)
	for _, r := range layerLogged {
		layerIDs[r.FlockID] = true
	}
	growerIDs := make(map[uint]bool)
	for _, r := range growerLogged {
		growerIDs[r.FlockID] = true
	}

	for _, f := range flocks {
		fd := calcFlockData(f, today, layerIDs, growerIDs)
		if f.Status == "Retired" {
			retired = append(retired, fd)
		} else {
			active = append(active, fd)
		}
	}
	return
}

// GetFlock returns a single flock by ID.
func GetFlock(id uint) (*models.Flock, error) {
	var f models.Flock
	err := db.DB.First(&f, id).Error
	return &f, err
}

type CreateFlockRequest struct {
	Name         string `json:"name"`
	HouseNumber  string `json:"house_number"`
	HouseType    string `json:"house_type"`
	Breed        string `json:"breed"`
	BuildingName string `json:"building_name"`
	HatchDate    string `json:"hatch_date"` // YYYY-MM-DD
	InitialCount int    `json:"initial_count"`
	CreatedByID  *uint  `json:"created_by_id"`
}

// CreateFlock creates a new flock record.
func CreateFlock(req CreateFlockRequest) (*models.Flock, error) {
	flock := models.Flock{
		Name:         req.Name,
		HouseNumber:  req.HouseNumber,
		HouseType:    req.HouseType,
		Breed:        req.Breed,
		BuildingName: req.BuildingName,
		InitialCount: req.InitialCount,
		CurrentCount: req.InitialCount,
		CreatedByID:  req.CreatedByID,
	}
	if req.HouseType == "Grower" {
		flock.Status = "Grower"
	} else {
		flock.Status = "Layer"
	}
	if req.HatchDate != "" {
		t, err := time.Parse("2006-01-02", req.HatchDate)
		if err != nil {
			return nil, fmt.Errorf("invalid hatch date: %w", err)
		}
		flock.HatchDate = &t
	}
	if err := db.DB.Create(&flock).Error; err != nil {
		return nil, err
	}
	return &flock, nil
}

type UpdateFlockRequest struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	BuildingName string `json:"building_name"`
	HouseNumber  string `json:"house_number"`
	Breed        string `json:"breed"`
	HatchDate    string `json:"hatch_date"`
	CurrentCount int    `json:"current_count"`
}

// UpdateFlock updates flock details.
func UpdateFlock(req UpdateFlockRequest) error {
	updates := map[string]interface{}{
		"name":          req.Name,
		"building_name": req.BuildingName,
		"house_number":  req.HouseNumber,
		"breed":         req.Breed,
		"current_count": req.CurrentCount,
	}
	if req.HatchDate != "" {
		t, err := time.Parse("2006-01-02", req.HatchDate)
		if err == nil {
			updates["hatch_date"] = t
		}
	}
	return db.DB.Model(&models.Flock{}).Where("id = ?", req.ID).Updates(updates).Error
}

type TransferFlockRequest struct {
	ID             uint   `json:"id"`
	NewHouseNumber string `json:"new_house_number"`
	NewBuilding    string `json:"new_building"`
}

// TransferFlock moves a Grower flock to a Layer house.
func TransferFlock(req TransferFlockRequest) error {
	var f models.Flock
	if err := db.DB.First(&f, req.ID).Error; err != nil {
		return err
	}
	if f.HouseType != "Grower" {
		return errors.New("flock is already a Layer flock")
	}
	updates := map[string]interface{}{
		"house_type":   "Layer",
		"status":       "Layer",
		"house_number": req.NewHouseNumber,
	}
	if req.NewBuilding != "" {
		updates["building_name"] = req.NewBuilding
	}
	return db.DB.Model(&f).Updates(updates).Error
}

type RetireFlockRequest struct {
	ID                     uint    `json:"id"`
	RetirementDate         string  `json:"retirement_date"`
	RetirementType         string  `json:"retirement_type"`
	BirdsRetired           int     `json:"birds_retired"`
	RetirementPricePerBird float64 `json:"retirement_price_per_bird"`
	RetirementBuyer        string  `json:"retirement_buyer"`
	RetirementNotes        string  `json:"retirement_notes"`
	CreatedByID            *uint   `json:"created_by_id"`
}

// RetireFlock retires a flock.
func RetireFlock(req RetireFlockRequest) error {
	var f models.Flock
	if err := db.DB.First(&f, req.ID).Error; err != nil {
		return err
	}
	if f.Status == "Retired" {
		return errors.New("flock is already retired")
	}

	retDate := time.Now()
	if req.RetirementDate != "" {
		if t, err := time.Parse("2006-01-02", req.RetirementDate); err == nil {
			retDate = t
		}
	}

	birds := req.BirdsRetired
	if birds <= 0 {
		birds = f.CurrentCount
	}
	newCount := f.CurrentCount - birds
	if newCount < 0 {
		newCount = 0
	}

	return db.DB.Model(&f).Updates(map[string]interface{}{
		"status":                   "Retired",
		"retirement_date":          retDate,
		"retirement_type":          req.RetirementType,
		"birds_retired":            birds,
		"retirement_price_per_bird": req.RetirementPricePerBird,
		"retirement_buyer":         req.RetirementBuyer,
		"retirement_notes":         req.RetirementNotes,
		"current_count":            newCount,
	}).Error
}

// GetLayerHouses returns all Layer flock house numbers.
func GetLayerHouses() ([]string, error) {
	var flocks []models.Flock
	if err := db.DB.Where("house_type = 'Layer'").Find(&flocks).Error; err != nil {
		return nil, err
	}
	houses := make([]string, len(flocks))
	for i, f := range flocks {
		houses[i] = f.HouseNumber
	}
	return houses, nil
}

// GetHouseHistory returns logs for a flock (DailyLog or GrowerLog).
func GetHouseHistory(flockID uint) (interface{}, string, error) {
	var flock models.Flock
	if err := db.DB.First(&flock, flockID).Error; err != nil {
		return nil, "", err
	}
	if flock.HouseType == "Layer" {
		var logs []models.DailyLog
		err := db.DB.Where("flock_id = ?", flockID).Order("date DESC").Find(&logs).Error
		return logs, "layer", err
	}
	var logs []models.GrowerLog
	err := db.DB.Where("flock_id = ?", flockID).Order("date DESC").Find(&logs).Error
	return logs, "grower", err
}

// BodyWeightPageData holds body weight logs, standard, and chart data for a flock.
type BodyWeightPageData struct {
	Flock         models.Flock           `json:"flock"`
	Logs          []models.BodyWeightLog `json:"logs"`
	Standard      map[int]float64        `json:"standard"`
	AgeWeeks      *int                   `json:"age_weeks"`
	Latest        *models.BodyWeightLog  `json:"latest"`
	Status        string                 `json:"status"`
	StatusColor   string                 `json:"status_color"`
	DeviationPct  *float64               `json:"deviation_pct"`
	TransferReady bool                   `json:"transfer_ready"`
	ChartWeeks    []int                  `json:"chart_weeks"`
	ChartStandard []*float64             `json:"chart_standard"`
	ChartActual   []*float64             `json:"chart_actual"`
	NextWeek      *int                   `json:"next_week"`
	NextTarget    *float64               `json:"next_target"`
}

// GetBodyWeightData returns body weight logs, standard, and chart data for a flock.
func GetBodyWeightData(flockID uint) (*BodyWeightPageData, error) {
	var flock models.Flock
	if err := db.DB.First(&flock, flockID).Error; err != nil {
		return nil, err
	}

	var logs []models.BodyWeightLog
	db.DB.Where("flock_id = ? AND is_active = 1", flockID).Order("week_age").Find(&logs)

	standard := constants.BreedStandard(flock.Breed)
	today := time.Now()

	var ageWeeks *int
	if flock.HatchDate != nil {
		aw := int(today.Sub(*flock.HatchDate).Hours()/24) / 7
		ageWeeks = &aw
	}

	data := &BodyWeightPageData{
		Flock:    flock,
		Logs:     logs,
		Standard: standard,
		AgeWeeks: ageWeeks,
	}

	if len(logs) > 0 {
		latest := logs[len(logs)-1]
		data.Latest = &latest
		if target, ok := standard[latest.WeekAge]; ok {
			devPct := (latest.AvgWeightG - target) / target * 100
			rounded := float64(int(devPct*10)) / 10
			data.DeviationPct = &rounded
			switch {
			case devPct >= -3:
				data.Status, data.StatusColor = "On Track", "success"
			case devPct >= -8:
				data.Status, data.StatusColor = "Slightly Low", "warning"
			default:
				data.Status, data.StatusColor = "Underweight", "danger"
			}
		}
		if ageWeeks != nil {
			data.TransferReady = *ageWeeks >= 18 && latest.AvgWeightG >= 1350
		}
	}

	// Build chart data
	minWeek := 4
	maxWeek := 22
	if ageWeeks != nil {
		if *ageWeeks+2 > maxWeek {
			maxWeek = *ageWeeks + 2
		}
	}

	logMap := make(map[int]float64)
	recordedWeeks := make(map[int]bool)
	for _, l := range logs {
		logMap[l.WeekAge] = l.AvgWeightG
		recordedWeeks[l.WeekAge] = true
	}

	for w := minWeek; w <= maxWeek; w++ {
		data.ChartWeeks = append(data.ChartWeeks, w)
		if v, ok := standard[w]; ok {
			v2 := v
			data.ChartStandard = append(data.ChartStandard, &v2)
		} else {
			data.ChartStandard = append(data.ChartStandard, nil)
		}
		if v, ok := logMap[w]; ok {
			v2 := v
			data.ChartActual = append(data.ChartActual, &v2)
		} else {
			data.ChartActual = append(data.ChartActual, nil)
		}
	}

	// Next week to record
	if ageWeeks != nil {
		nw := *ageWeeks
		for recordedWeeks[nw] && nw <= maxWeek {
			nw++
		}
		data.NextWeek = &nw
		if t, ok := standard[nw]; ok {
			data.NextTarget = &t
		}
	}

	return data, nil
}

// AddBodyWeightLog records a body weight entry.
func AddBodyWeightLog(flockID uint, date string, weekAge int, avgWeightG float64, sampleSize int, notes string, createdByID *uint) error {
	// Check for duplicate week
	var existing models.BodyWeightLog
	err := db.DB.Where("flock_id = ? AND week_age = ? AND is_active = 1", flockID, weekAge).First(&existing).Error
	if err == nil {
		return fmt.Errorf("a record for Week %d already exists", weekAge)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}

	entry := models.BodyWeightLog{
		FlockID:     flockID,
		Date:        t,
		WeekAge:     weekAge,
		AvgWeightG:  avgWeightG,
		SampleSize:  sampleSize,
		Notes:       notes,
		IsActive:    true,
		CreatedByID: createdByID,
	}
	return db.DB.Create(&entry).Error
}

// UpdateBodyWeightLog edits an existing body weight log.
func UpdateBodyWeightLog(id uint, date string, avgWeightG float64, sampleSize int, notes string) error {
	updates := map[string]interface{}{
		"avg_weight_g": avgWeightG,
		"sample_size":  sampleSize,
		"notes":        notes,
	}
	if date != "" {
		if t, err := time.Parse("2006-01-02", date); err == nil {
			updates["date"] = t
		}
	}
	return db.DB.Model(&models.BodyWeightLog{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteBodyWeightLog soft-deletes a body weight log.
func DeleteBodyWeightLog(id uint) error {
	return db.DB.Model(&models.BodyWeightLog{}).Where("id = ?", id).Update("is_active", false).Error
}
