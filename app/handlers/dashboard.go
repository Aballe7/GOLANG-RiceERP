package handlers

import (
	"math"
	"time"

	"egglayererp/app/db"
	"egglayererp/app/middleware"
	"egglayererp/app/models"
)

// DashboardData aggregates all KPIs for the main dashboard.
type DashboardData struct {
	Today           string              `json:"today"`
	ActiveFlocks    int                 `json:"active_flocks"`
	TotalBirds      int                 `json:"total_birds"`
	LayerFlocks     int                 `json:"layer_flocks"`
	GrowerFlocks    int                 `json:"grower_flocks"`
	LayerBirds      int                 `json:"layer_birds"`
	GrowerBirds     int                 `json:"grower_birds"`
	TodayEggs       int                 `json:"today_eggs"`
	TodayMortality  int                 `json:"today_mortality"`
	OpenARBalance   float64             `json:"open_ar_balance"`
	OpenAPBalance   float64             `json:"open_ap_balance"`
	EggInventory    map[string]int      `json:"egg_inventory"`
	RecentLogs      []RecentLog         `json:"recent_logs"`
	LowFeedAlerts   []FeedAlert         `json:"low_feed_alerts"`
	UnloggedHouses  []string            `json:"unlogged_houses"`
	FlockTiles      []FlockTile         `json:"flock_tiles"`
	RecentOrders    []RecentOrderItem   `json:"recent_orders"`
	HDP             float64             `json:"hdp"`
	FCR             float64             `json:"fcr"`
	MortalityRate   float64             `json:"mortality_rate"`
	RejectRate      float64             `json:"reject_rate"`
}

type RecentLog struct {
	FlockName   string `json:"flock_name"`
	HouseNumber string `json:"house_number"`
	Date        string `json:"date"`
	TotalEggs   int    `json:"total_eggs"`
	Mortality   int    `json:"mortality"`
}

type FeedAlert struct {
	Name       string  `json:"name"`
	TotalSacks float64 `json:"total_sacks"`
}

type FlockTile struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	HouseType    string `json:"house_type"`
	HouseNumber  string `json:"house_number"`
	CurrentCount int    `json:"current_count"`
}

type RecentOrderItem struct {
	InvoiceNumber        string  `json:"invoice_number"`
	CustomerNameSnapshot string  `json:"customer_name_snapshot"`
	GrandTotal           float64 `json:"grand_total"`
	PaymentStatus        string  `json:"payment_status"`
	Date                 string  `json:"date"`
}

// GetDashboard returns all dashboard KPIs in a single call.
func GetDashboard() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}

	today := time.Now().Format("2006-01-02")
	data := DashboardData{
		Today:        today,
		EggInventory: make(map[string]int),
	}

	// Flock counts
	var flocks []models.Flock
	db.DB.Where("status != 'Retired'").Find(&flocks)
	data.ActiveFlocks = len(flocks)
	for _, f := range flocks {
		data.TotalBirds += f.CurrentCount
		tile := FlockTile{ID: f.ID, Name: f.Name, HouseType: f.HouseType, HouseNumber: f.HouseNumber, CurrentCount: f.CurrentCount}
		data.FlockTiles = append(data.FlockTiles, tile)
		if f.HouseType == "Layer" {
			data.LayerFlocks++
			data.LayerBirds += f.CurrentCount
		} else {
			data.GrowerFlocks++
			data.GrowerBirds += f.CurrentCount
		}
	}

	// Today's egg production and mortality from DailyLog
	type DailyAgg struct {
		TotalEggs      int
		TotalMortality int
	}
	var agg DailyAgg
	db.DB.Raw(`
		SELECT
			COALESCE(SUM(pewee+pullet+small+medium+large+extra_large+jumbo+double_yolk), 0) AS total_eggs,
			COALESCE(SUM(mortality), 0) AS total_mortality
		FROM daily_log WHERE DATE(date) = ?
	`, today).Scan(&agg)
	data.TodayEggs = agg.TotalEggs
	data.TodayMortality = agg.TotalMortality

	// Open AR balance
	db.DB.Model(&models.ARInvoice{}).
		Where("status IN ('Open', 'Partial')").
		Select("COALESCE(SUM(total_amount - amount_collected), 0)").
		Scan(&data.OpenARBalance)

	// Open AP balance
	db.DB.Model(&models.APInvoice{}).
		Where("status IN ('Open', 'Partial')").
		Select("COALESCE(SUM(total_amount - amount_paid_stored), 0)").
		Scan(&data.OpenAPBalance)

	// Low feed alerts (< 10 sacks)
	var feedStocks []models.FeedStock
	db.DB.Where("total_sacks < 10").Order("total_sacks").Find(&feedStocks)
	for _, f := range feedStocks {
		data.LowFeedAlerts = append(data.LowFeedAlerts, FeedAlert{
			Name: f.Name, TotalSacks: f.TotalSacks,
		})
	}

	// Unlogged houses today (Layer houses without DailyLog, Grower without GrowerLog)
	type LoggedID struct{ FlockID uint }
	var loggedLayer, loggedGrower []LoggedID
	db.DB.Model(&models.DailyLog{}).Select("flock_id").Where("DATE(date) = ?", today).Scan(&loggedLayer)
	db.DB.Model(&models.GrowerLog{}).Select("flock_id").Where("DATE(date) = ?", today).Scan(&loggedGrower)

	loggedLayerIDs := make(map[uint]bool)
	for _, r := range loggedLayer {
		loggedLayerIDs[r.FlockID] = true
	}
	loggedGrowerIDs := make(map[uint]bool)
	for _, r := range loggedGrower {
		loggedGrowerIDs[r.FlockID] = true
	}

	for _, f := range flocks {
		if f.HouseType == "Layer" && !loggedLayerIDs[f.ID] {
			data.UnloggedHouses = append(data.UnloggedHouses, "Layer H"+f.HouseNumber)
		} else if f.HouseType == "Grower" && !loggedGrowerIDs[f.ID] {
			data.UnloggedHouses = append(data.UnloggedHouses, "Grower H"+f.HouseNumber)
		}
	}

	// Egg inventory (simplified — top 5 sizes by volume)
	type EggSizeSum struct {
		Pewee, Pullet, Small, Medium, Large, ExtraLarge, Jumbo, DoubleYolk, CrackedDirty int
	}
	var sums EggSizeSum
	db.DB.Raw(`
		SELECT COALESCE(SUM(pewee),0) AS pewee, COALESCE(SUM(pullet),0) AS pullet,
		       COALESCE(SUM(small),0) AS small, COALESCE(SUM(medium),0) AS medium,
		       COALESCE(SUM(large),0) AS large, COALESCE(SUM(extra_large),0) AS extra_large,
		       COALESCE(SUM(jumbo),0) AS jumbo, COALESCE(SUM(double_yolk),0) AS double_yolk,
		       COALESCE(SUM(cracked_dirty),0) AS cracked_dirty
		FROM daily_log
	`).Scan(&sums)
	data.EggInventory = map[string]int{
		"Pewee": sums.Pewee, "Pullet": sums.Pullet, "Small": sums.Small,
		"Medium": sums.Medium, "Large": sums.Large, "Extra Large": sums.ExtraLarge,
		"Jumbo": sums.Jumbo, "Double Yolk": sums.DoubleYolk, "Cracked/Dirty": sums.CrackedDirty,
	}

	// KPIs from last 30 days
	thirty := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	type KPIAgg struct {
		TotalEggs      int
		TotalRejects   int
		TotalMortality int
		TotalFeedKg    float64
	}
	var kpi KPIAgg
	db.DB.Raw(`
		SELECT
			COALESCE(SUM(pewee+pullet+small+medium+large+extra_large+jumbo+double_yolk+cracked_dirty), 0) AS total_eggs,
			COALESCE(SUM(cracked_dirty+double_yolk), 0) AS total_rejects,
			COALESCE(SUM(mortality), 0) AS total_mortality,
			COALESCE(SUM(feed_consumed_kg), 0) AS total_feed_kg
		FROM daily_log WHERE DATE(date) >= ?
	`, thirty).Scan(&kpi)

	if data.LayerBirds > 0 && kpi.TotalEggs > 0 {
		data.HDP = math.Round(float64(kpi.TotalEggs)/float64(data.LayerBirds*30)*10000) / 100
	}
	if kpi.TotalEggs > 0 {
		data.RejectRate = math.Round(float64(kpi.TotalRejects)/float64(kpi.TotalEggs)*10000) / 100
		// FCR: feed kg / (eggs * avg weight 0.06 kg)
		if kpi.TotalFeedKg > 0 {
			eggWeightKg := float64(kpi.TotalEggs) * 0.06
			data.FCR = math.Round(kpi.TotalFeedKg/eggWeightKg*100) / 100
		}
	}
	if data.TotalBirds > 0 {
		data.MortalityRate = math.Round(float64(kpi.TotalMortality)/float64(data.TotalBirds)*10000) / 100
	}

	// Recent sales orders (last 5)
	type OrderRow struct {
		InvoiceNumber        string
		CustomerNameSnapshot string
		GrandTotal           float64
		PaymentStatus        string
		Date                 time.Time
	}
	var orders []OrderRow
	db.DB.Raw(`SELECT invoice_number, customer_name_snapshot, grand_total, payment_status, date
		FROM sales_order ORDER BY id DESC LIMIT 5`).Scan(&orders)
	for _, o := range orders {
		data.RecentOrders = append(data.RecentOrders, RecentOrderItem{
			InvoiceNumber:        o.InvoiceNumber,
			CustomerNameSnapshot: o.CustomerNameSnapshot,
			GrandTotal:           o.GrandTotal,
			PaymentStatus:        o.PaymentStatus,
			Date:                 o.Date.Format("Jan 2"),
		})
	}

	return okResponse("", data)
}
