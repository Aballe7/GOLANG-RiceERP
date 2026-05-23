package reports

import (
	"math"
	"sort"
	"time"

	"ricemill/app/db"
)

// PnLRow represents a single line in the P&L report.
type PnLRow struct {
	AccountCode string  `json:"account_code"`
	AccountName string  `json:"account_name"`
	Amount      float64 `json:"amount"`
}

// PnLReport holds the full profit & loss report.
type PnLReport struct {
	PeriodStart  string   `json:"period_start"`
	PeriodEnd    string   `json:"period_end"`
	Revenue      []PnLRow `json:"revenue"`
	Expenses     []PnLRow `json:"expenses"`
	TotalRevenue float64  `json:"total_revenue"`
	TotalExpense float64  `json:"total_expense"`
	NetIncome    float64  `json:"net_income"`
}

// GetPnL generates a Profit & Loss report for the given date range.
func GetPnL(startDate, endDate string) (*PnLReport, error) {
	type GLSumRow struct {
		Code    string
		Name    string
		Section string
		Total   float64
	}

	var rows []GLSumRow
	query := `
		SELECT ga.code, ga.name, ga.section,
		       COALESCE(SUM(jel.credit), 0) - COALESCE(SUM(jel.debit), 0) AS total
	FROM je_line jel
		JOIN journal_entry je ON je.id = jel.journal_entry_id
		JOIN gl_account ga ON ga.id = jel.gl_account_id
		WHERE je.status = 'POSTED'
		  AND je.date BETWEEN ? AND ?
		  AND ga.section IN ('REVENUE', 'EXPENSE')
		  AND ga.account_type = 'POSTING'
		GROUP BY ga.id, ga.code, ga.name, ga.section
		ORDER BY ga.section DESC, ga.code
	`

	if err := db.DB.Raw(query, startDate, endDate).Scan(&rows).Error; err != nil {
		return nil, err
	}

	report := &PnLReport{
		PeriodStart: startDate,
		PeriodEnd:   endDate,
	}

	for _, r := range rows {
		row := PnLRow{AccountCode: r.Code, AccountName: r.Name, Amount: r.Total}
		if r.Section == "REVENUE" {
			report.Revenue = append(report.Revenue, row)
			report.TotalRevenue += r.Total
		} else {
			// Expenses: negate (positive = expense)
			row.Amount = -r.Total
			report.Expenses = append(report.Expenses, row)
			report.TotalExpense += row.Amount
		}
	}
	report.NetIncome = report.TotalRevenue - report.TotalExpense
	return report, nil
}

// BalanceSheetSection holds one section of the Balance Sheet.
type BSSection struct {
	Name     string     `json:"name"`
	Accounts []BSRow    `json:"accounts"`
	Total    float64    `json:"total"`
}

type BSRow struct {
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// BalanceSheet holds the full balance sheet report.
type BalanceSheet struct {
	AsOf        string    `json:"as_of"`
	Assets      BSSection `json:"assets"`
	Liabilities BSSection `json:"liabilities"`
	Equity      BSSection `json:"equity"`
	Check       float64   `json:"check"` // Assets - (Liabilities + Equity), should be 0
}

// GetBalanceSheet generates a balance sheet as of the given date.
func GetBalanceSheet(asOf string) (*BalanceSheet, error) {
	type GLSumRow struct {
		Code          string
		Name          string
		Section       string
		NormalBalance string
		Total         float64
	}

	var rows []GLSumRow
	query := `
		SELECT ga.code, ga.name, ga.section, ga.normal_balance,
		       COALESCE(SUM(jel.debit), 0) - COALESCE(SUM(jel.credit), 0) AS total
	FROM je_line jel
		JOIN journal_entry je ON je.id = jel.journal_entry_id
		JOIN gl_account ga ON ga.id = jel.gl_account_id
		WHERE je.status = 'POSTED'
		  AND je.date <= ?
		  AND ga.section IN ('ASSET', 'LIABILITY', 'EQUITY')
		  AND ga.account_type = 'POSTING'
		GROUP BY ga.id, ga.code, ga.name, ga.section, ga.normal_balance
		ORDER BY ga.section, ga.code
	`

	if err := db.DB.Raw(query, asOf).Scan(&rows).Error; err != nil {
		return nil, err
	}

	bs := &BalanceSheet{AsOf: asOf}
	bs.Assets.Name = "Assets"
	bs.Liabilities.Name = "Liabilities"
	bs.Equity.Name = "Equity"

	for _, r := range rows {
		// For assets: debit-normal → positive = balance
		// For liabilities/equity: credit-normal → negative total = positive balance
		amount := r.Total
		if r.NormalBalance == "CREDIT" {
			amount = -r.Total
		}
		row := BSRow{Code: r.Code, Name: r.Name, Amount: amount}
		switch r.Section {
		case "ASSET":
			bs.Assets.Accounts = append(bs.Assets.Accounts, row)
			bs.Assets.Total += amount
		case "LIABILITY":
			bs.Liabilities.Accounts = append(bs.Liabilities.Accounts, row)
			bs.Liabilities.Total += amount
		case "EQUITY":
			bs.Equity.Accounts = append(bs.Equity.Accounts, row)
			bs.Equity.Total += amount
		}
	}
	bs.Check = bs.Assets.Total - (bs.Liabilities.Total + bs.Equity.Total)
	return bs, nil
}

// ARAgingRow holds a single customer's aging breakdown.
type ARAgingRow struct {
	CustomerName string  `json:"customer_name"`
	InvoiceNum   string  `json:"invoice_number"`
	InvoiceDate  string  `json:"invoice_date"`
	DueDate      string  `json:"due_date"`
	TotalAmount  float64 `json:"total_amount"`
	Balance      float64 `json:"balance"`
	Current      float64 `json:"current"`
	Days30       float64 `json:"days_30"`
	Days60       float64 `json:"days_60"`
	Days90       float64 `json:"days_90"`
	Over90       float64 `json:"over_90"`
}

// GetARAgingReport generates the AR Aging report as of today.
func GetARAgingReport() ([]ARAgingRow, error) {
	today := time.Now().Format("2006-01-02")

	type RawRow struct {
		CustomerName string
		InvoiceNum   string
		InvoiceDate  string
		DueDate      string
		TotalAmount  float64
		Balance      float64
		DaysOverdue  int
	}

	var rawRows []RawRow
	query := `
		SELECT
			COALESCE(c.name, ar.customer_name_snapshot, 'Walk-in') AS customer_name,
			ar.invoice_number AS invoice_num,
			DATE_FORMAT(ar.date, '%Y-%m-%d') AS invoice_date,
			DATE_FORMAT(ar.due_date, '%Y-%m-%d') AS due_date,
			ar.total_amount,
			ar.total_amount - ar.amount_collected AS balance,
			DATEDIFF(?, COALESCE(ar.due_date, ar.date)) AS days_overdue
		FROM ar_invoice ar
		LEFT JOIN customer c ON c.id = ar.customer_id
		WHERE ar.status IN ('Open', 'Partial')
		  AND (ar.total_amount - ar.amount_collected) > 0.005
		ORDER BY customer_name, ar.date
	`
	if err := db.DB.Raw(query, today).Scan(&rawRows).Error; err != nil {
		return nil, err
	}

	result := make([]ARAgingRow, 0, len(rawRows))
	for _, r := range rawRows {
		row := ARAgingRow{
			CustomerName: r.CustomerName,
			InvoiceNum:   r.InvoiceNum,
			InvoiceDate:  r.InvoiceDate,
			DueDate:      r.DueDate,
			TotalAmount:  r.TotalAmount,
			Balance:      r.Balance,
		}
		d := r.DaysOverdue
		switch {
		case d <= 0:
			row.Current = r.Balance
		case d <= 30:
			row.Days30 = r.Balance
		case d <= 60:
			row.Days60 = r.Balance
		case d <= 90:
			row.Days90 = r.Balance
		default:
			row.Over90 = r.Balance
		}
		result = append(result, row)
	}
	return result, nil
}

// TrialBalanceRow represents a single account line in the Trial Balance.
type TrialBalanceRow struct {
	AccountCode   string  `json:"account_code"`
	AccountName   string  `json:"account_name"`
	Section       string  `json:"section"`
	NormalBalance string  `json:"normal_balance"`
	DebitBalance  float64 `json:"debit_balance"`
	CreditBalance float64 `json:"credit_balance"`
}

// TrialBalanceReport holds the full Trial Balance report.
type TrialBalanceReport struct {
	PeriodStart string            `json:"period_start"`
	PeriodEnd   string            `json:"period_end"`
	GeneratedAt string            `json:"generated_at"`
	Rows        []TrialBalanceRow `json:"rows"`
	TotalDebit  float64           `json:"total_debit"`
	TotalCredit float64           `json:"total_credit"`
}

// GetTrialBalance generates an adjusted Trial Balance for the given date range.
// Each account's net movement (Debit - Credit) is placed in the Debit column when
// positive, or the Credit column when negative, following PFRS convention.
// Total Debit must equal Total Credit (double-entry verification).
func GetTrialBalance(startDate, endDate string) (*TrialBalanceReport, error) {
	type RawRow struct {
		Code          string
		Name          string
		Section       string
		NormalBalance string
		TotalDebit    float64
		TotalCredit   float64
	}

	var rawRows []RawRow
	query := `
		SELECT
			ga.code          AS code,
			ga.name          AS name,
			ga.section       AS section,
			ga.normal_balance AS normal_balance,
			COALESCE(SUM(jel.debit), 0)  AS total_debit,
			COALESCE(SUM(jel.credit), 0) AS total_credit
		FROM je_line jel
		JOIN journal_entry je ON je.id = jel.journal_entry_id
		JOIN gl_account ga    ON ga.id  = jel.gl_account_id
		WHERE je.status        = 'POSTED'
		  AND je.date          BETWEEN ? AND ?
		  AND ga.account_type  = 'POSTING'
		  AND ga.is_active      = 1
		GROUP BY ga.id, ga.code, ga.name, ga.section, ga.normal_balance
		HAVING (total_debit + total_credit) > 0.005
		ORDER BY ga.code
	`
	if err := db.DB.Raw(query, startDate, endDate).Scan(&rawRows).Error; err != nil {
		return nil, err
	}

	report := &TrialBalanceReport{
		PeriodStart: startDate,
		PeriodEnd:   endDate,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Rows:        make([]TrialBalanceRow, 0, len(rawRows)),
	}

	for _, r := range rawRows {
		net := r.TotalDebit - r.TotalCredit
		row := TrialBalanceRow{
			AccountCode:   r.Code,
			AccountName:   r.Name,
			Section:       r.Section,
			NormalBalance: r.NormalBalance,
		}
		if net >= 0 {
			row.DebitBalance = math.Round(net*100) / 100
		} else {
			row.CreditBalance = math.Round(-net*100) / 100
		}
		report.Rows = append(report.Rows, row)
		report.TotalDebit += row.DebitBalance
		report.TotalCredit += row.CreditBalance
	}

	report.TotalDebit = math.Round(report.TotalDebit*100) / 100
	report.TotalCredit = math.Round(report.TotalCredit*100) / 100

	return report, nil
}

// APAgingRow holds a single supplier's AP aging breakdown.
type APAgingRow struct {
	SupplierName string  `json:"supplier_name"`
	InvoiceNum   string  `json:"invoice_number"`
	InvoiceDate  string  `json:"invoice_date"`
	DueDate      string  `json:"due_date"`
	TotalAmount  float64 `json:"total_amount"`
	Balance      float64 `json:"balance"`
	Current      float64 `json:"current"`
	Days30       float64 `json:"days_30"`
	Days60       float64 `json:"days_60"`
	Days90       float64 `json:"days_90"`
	Over90       float64 `json:"over_90"`
}

// GetAPAgingReport generates the AP Aging report as of today.
func GetAPAgingReport() ([]APAgingRow, error) {
	today := time.Now().Format("2006-01-02")

	type RawRow struct {
		SupplierName string
		InvoiceNum   string
		InvoiceDate  string
		DueDate      string
		TotalAmount  float64
		Balance      float64
		DaysOverdue  int
	}

	var rawRows []RawRow
	query := `
		SELECT
			COALESCE(s.name, ap.supplier_name, 'Unknown') AS supplier_name,
			ap.invoice_number AS invoice_num,
			DATE_FORMAT(ap.date, '%Y-%m-%d') AS invoice_date,
			DATE_FORMAT(ap.due_date, '%Y-%m-%d') AS due_date,
			ap.total_amount,
			ap.total_amount - ap.amount_paid_stored AS balance,
			DATEDIFF(?, COALESCE(ap.due_date, ap.date)) AS days_overdue
		FROM ap_invoice ap
		LEFT JOIN supplier s ON s.id = ap.supplier_id
		WHERE ap.status IN ('Open', 'Partial')
		  AND (ap.total_amount - ap.amount_paid_stored) > 0.005
		ORDER BY supplier_name, ap.date
	`
	if err := db.DB.Raw(query, today).Scan(&rawRows).Error; err != nil {
		return nil, err
	}

	result := make([]APAgingRow, 0, len(rawRows))
	for _, r := range rawRows {
		row := APAgingRow{
			SupplierName: r.SupplierName,
			InvoiceNum:   r.InvoiceNum,
			InvoiceDate:  r.InvoiceDate,
			DueDate:      r.DueDate,
			TotalAmount:  r.TotalAmount,
			Balance:      r.Balance,
		}
		d := r.DaysOverdue
		switch {
		case d <= 0:
			row.Current = r.Balance
		case d <= 30:
			row.Days30 = r.Balance
		case d <= 60:
			row.Days60 = r.Balance
		case d <= 90:
			row.Days90 = r.Balance
		default:
			row.Over90 = r.Balance
		}
		result = append(result, row)
	}
	return result, nil
}

// MonthlyCPEPoint holds cost-per-egg analysis for a single month.
type MonthlyCPEPoint struct {
	Month         string  `json:"month"`
	FeedCost      float64 `json:"feed_cost"`
	FeedKgBought  float64 `json:"feed_kg_bought"`
	FeedKgUsed    float64 `json:"feed_kg_used"`
	GoodEggs      int     `json:"good_eggs"`
	Revenue       float64 `json:"revenue"`
	CostPerEgg    float64 `json:"cost_per_egg"`
	RevenuePerEgg float64 `json:"revenue_per_egg"`
	Margin        float64 `json:"margin"`
}

// AnalyticsData holds dashboard-level analytics.
type AnalyticsData struct {
	Period      string          `json:"period"`
	MonthlySales []MonthlyTotal `json:"monthly_sales"`
	TopCustomers []CustomerTotal `json:"top_customers"`
	EggSizeMix  map[string]int  `json:"egg_size_mix"`

	// Farm performance KPIs
	HDP           float64     `json:"hdp"`
	FCR           float64     `json:"fcr"`
	MortalityRate float64     `json:"mortality_rate"`
	RejectRate    float64     `json:"reject_rate"`
	AvgEggsPerHen float64     `json:"avg_eggs_per_hen"`
	TotalGood     int         `json:"total_good"`
	TotalCracked  int         `json:"total_cracked"`
	TotalFeedKg   float64     `json:"total_feed_kg"`
	TotalMortality int        `json:"total_mortality"`
	TotalBirds    int         `json:"total_birds"`
	DaysLogged    int         `json:"days_logged"`
	TrendLabels   []string    `json:"trend_labels"`
	TrendGood     []int       `json:"trend_good"`
	TrendFeed     []float64   `json:"trend_feed"`
	HouseStats    []HouseStat `json:"house_stats"`

	// Cost Per Egg Analysis
	FeedCostTotal     float64           `json:"feed_cost_total"`
	FeedCostPerKg     float64           `json:"feed_cost_per_kg"`
	CostPerEgg        float64           `json:"cost_per_egg"`
	CostPerDozen      float64           `json:"cost_per_dozen"`
	RevenuePerEgg     float64           `json:"revenue_per_egg"`
	GrossMarginPerEgg float64           `json:"gross_margin_per_egg"`
	MonthlyCPE        []MonthlyCPEPoint `json:"monthly_cpe"`
}

type MonthlyTotal struct {
	Month string  `json:"month"`
	Total float64 `json:"total"`
}

type CustomerTotal struct {
	Name  string  `json:"name"`
	Total float64 `json:"total"`
}

type HouseStat struct {
	Name     string  `json:"name"`
	Birds    int     `json:"birds"`
	Days     int     `json:"days"`
	GoodEggs int     `json:"good_eggs"`
	Cracked  int     `json:"cracked"`
	FeedKg   float64 `json:"feed_kg"`
	HDP      float64 `json:"hdp"`
	FCR      float64 `json:"fcr"`
}

// GetAnalytics generates analytics data for the given date range.
// from and to are "YYYY-MM-DD" strings. An empty from means all-time.
func GetAnalytics(from, to string) (*AnalyticsData, error) {
	data := &AnalyticsData{Period: from + " – " + to}

	// Build date filter clause (reused across queries)
	// dateFilter returns a condition string and args slice for dl.date
	filterCond := func(col string) (string, []interface{}) {
		if from == "" {
			return col + " <= ?", []interface{}{to}
		}
		return col + " BETWEEN ? AND ?", []interface{}{from, to}
	}

	// ── Sales analytics ──────────────────────────────────────────────────────

	cond, args := filterCond("date")
	var monthly []MonthlyTotal
	db.DB.Raw(`
		SELECT DATE_FORMAT(date, '%Y-%m') AS month, SUM(total_amount) AS total
		FROM ar_invoice
		WHERE `+cond+` AND status != 'Cancelled'
		GROUP BY month ORDER BY month
	`, args...).Scan(&monthly)
	data.MonthlySales = monthly

	cond, args = filterCond("ar.date")
	var topCust []CustomerTotal
	db.DB.Raw(`
		SELECT COALESCE(c.name, ar.customer_name_snapshot, 'Walk-in') AS name,
		       SUM(ar.total_amount) AS total
		FROM ar_invoice ar
		LEFT JOIN customer c ON c.id = ar.customer_id
		WHERE `+cond+` AND ar.status != 'Cancelled'
		GROUP BY name ORDER BY total DESC LIMIT 10
	`, args...).Scan(&topCust)
	data.TopCustomers = topCust

	cond, args = filterCond("date")
	type SizeTotals struct {
		Pewee, Pullet, Small, Medium, Large, ExtraLarge, Jumbo, DoubleYolk, CrackedDirty int
	}
	var sizes SizeTotals
	db.DB.Raw(`
		SELECT SUM(pewee) AS pewee, SUM(pullet) AS pullet, SUM(small) AS small,
		       SUM(medium) AS medium, SUM(large) AS large, SUM(extra_large) AS extra_large,
		       SUM(jumbo) AS jumbo, SUM(double_yolk) AS double_yolk, SUM(cracked_dirty) AS cracked_dirty
		FROM daily_log WHERE `+cond, args...).Scan(&sizes)
	data.EggSizeMix = map[string]int{
		"Pewee": sizes.Pewee, "Pullet": sizes.Pullet, "Small": sizes.Small,
		"Medium": sizes.Medium, "Large": sizes.Large, "Extra Large": sizes.ExtraLarge,
		"Jumbo": sizes.Jumbo, "Double Yolk": sizes.DoubleYolk, "Cracked/Dirty": sizes.CrackedDirty,
	}

	// ── Farm performance KPIs (layer flocks, selected period) ────────────────

	cond, args = filterCond("dl.date")
	type FarmTotals struct {
		DaysLogged     int
		TotalGood      int
		TotalCracked   int
		TotalFeedKg    float64
		TotalMortality int
	}
	var ft FarmTotals
	db.DB.Raw(`
		SELECT
			COUNT(DISTINCT dl.date) AS days_logged,
			COALESCE(SUM(dl.pewee+dl.pullet+dl.small+dl.medium+dl.large+dl.extra_large+dl.jumbo+dl.double_yolk), 0) AS total_good,
			COALESCE(SUM(dl.cracked_dirty), 0) AS total_cracked,
			COALESCE(SUM(dl.feed_consumed_kg), 0) AS total_feed_kg,
			COALESCE(SUM(dl.mortality), 0) AS total_mortality
		FROM daily_log dl
		JOIN flock f ON f.id = dl.flock_id AND f.house_type = 'Layer'
		WHERE `+cond, args...).Scan(&ft)
	data.DaysLogged = ft.DaysLogged
	data.TotalGood = ft.TotalGood
	data.TotalCracked = ft.TotalCracked
	data.TotalFeedKg = ft.TotalFeedKg
	data.TotalMortality = ft.TotalMortality

	var totalBirds int
	db.DB.Raw(`SELECT COALESCE(SUM(current_count), 0) FROM flock WHERE house_type = 'Layer' AND status != 'Retired'`).Scan(&totalBirds)
	data.TotalBirds = totalBirds

	totalEggs := ft.TotalGood + ft.TotalCracked
	if totalBirds > 0 && ft.DaysLogged > 0 {
		data.HDP = math.Round(float64(totalEggs)/float64(totalBirds*ft.DaysLogged)*10000) / 100
		data.AvgEggsPerHen = math.Round(float64(totalEggs)/float64(totalBirds*ft.DaysLogged)*1000) / 1000
	}
	if totalEggs > 0 {
		data.FCR = math.Round(ft.TotalFeedKg/(float64(totalEggs)/12.0)*100) / 100
		data.RejectRate = math.Round(float64(ft.TotalCracked)/float64(totalEggs)*10000) / 100
	}
	if totalBirds > 0 {
		data.MortalityRate = math.Round(float64(ft.TotalMortality)/float64(totalBirds)*10000) / 100
	}

	// ── Daily trend — within selected period (capped at 90 points) ──────────

	type TrendRow struct {
		Day      string
		GoodEggs int
		FeedKg   float64
	}
	var trend []TrendRow
	cond, args = filterCond("dl.date")
	db.DB.Raw(`
		SELECT
			DATE_FORMAT(dl.date, '%Y-%m-%d') AS day,
			COALESCE(SUM(dl.pewee+dl.pullet+dl.small+dl.medium+dl.large+dl.extra_large+dl.jumbo+dl.double_yolk), 0) AS good_eggs,
			COALESCE(SUM(dl.feed_consumed_kg), 0) AS feed_kg
		FROM daily_log dl
		JOIN flock f ON f.id = dl.flock_id AND f.house_type = 'Layer'
		WHERE `+cond+`
		GROUP BY day ORDER BY day
	`, args...).Scan(&trend)
	for _, t := range trend {
		data.TrendLabels = append(data.TrendLabels, t.Day)
		data.TrendGood = append(data.TrendGood, t.GoodEggs)
		data.TrendFeed = append(data.TrendFeed, t.FeedKg)
	}

	// ── Per-house breakdown ──────────────────────────────────────────────────

	type HouseRow struct {
		Name     string
		Birds    int
		Days     int
		GoodEggs int
		Cracked  int
		FeedKg   float64
	}
	var houses []HouseRow
	cond, args = filterCond("dl.date")
	db.DB.Raw(`
		SELECT
			f.name,
			f.current_count AS birds,
			COUNT(DISTINCT dl.date) AS days,
			COALESCE(SUM(dl.pewee+dl.pullet+dl.small+dl.medium+dl.large+dl.extra_large+dl.jumbo+dl.double_yolk), 0) AS good_eggs,
			COALESCE(SUM(dl.cracked_dirty), 0) AS cracked,
			COALESCE(SUM(dl.feed_consumed_kg), 0) AS feed_kg
		FROM flock f
		LEFT JOIN daily_log dl ON dl.flock_id = f.id AND `+cond+`
		WHERE f.house_type = 'Layer'
		GROUP BY f.id, f.name, f.current_count
		ORDER BY f.name
	`, args...).Scan(&houses)

	for _, h := range houses {
		stat := HouseStat{Name: h.Name, Birds: h.Birds, Days: h.Days, GoodEggs: h.GoodEggs, Cracked: h.Cracked, FeedKg: h.FeedKg}
		eggs := h.GoodEggs + h.Cracked
		if h.Birds > 0 && h.Days > 0 {
			stat.HDP = math.Round(float64(eggs)/float64(h.Birds*h.Days)*10000) / 100
		}
		if eggs > 0 {
			stat.FCR = math.Round(h.FeedKg/(float64(eggs)/12.0)*100) / 100
		}
		data.HouseStats = append(data.HouseStats, stat)
	}

	// ── Cost Per Egg Analysis ────────────────────────────────────────────────

	// Query A — total feed cost + kg purchased from AP Invoice lines
	type FeedCostAgg struct {
		TotalCost float64
		TotalKg   float64
	}
	var fc FeedCostAgg
	cond, args = filterCond("ai.date")
	db.DB.Raw(`
		SELECT COALESCE(SUM(apl.line_total), 0) AS total_cost,
		       COALESCE(SUM(apl.quantity), 0)   AS total_kg
		FROM ap_invoice_line apl
		JOIN ap_invoice ai ON ai.id = apl.ap_invoice_id
		WHERE `+cond+`
		  AND apl.category = 'Feeds'
		  AND ai.status != 'Cancelled'
	`, args...).Scan(&fc)
	data.FeedCostTotal = fc.TotalCost
	if fc.TotalKg > 0 {
		data.FeedCostPerKg = math.Round(fc.TotalCost/fc.TotalKg*100) / 100
	}

	// Query B — monthly feed cost + kg bought
	type MonthlyFeedRow struct {
		Month    string
		FeedCost float64
		FeedKg   float64
	}
	var monthlyFeed []MonthlyFeedRow
	cond, args = filterCond("ai.date")
	db.DB.Raw(`
		SELECT DATE_FORMAT(ai.date, '%Y-%m') AS month,
		       COALESCE(SUM(apl.line_total), 0) AS feed_cost,
		       COALESCE(SUM(apl.quantity), 0)   AS feed_kg
		FROM ap_invoice_line apl
		JOIN ap_invoice ai ON ai.id = apl.ap_invoice_id
		WHERE `+cond+`
		  AND apl.category = 'Feeds'
		  AND ai.status != 'Cancelled'
		GROUP BY month ORDER BY month
	`, args...).Scan(&monthlyFeed)

	// Query C — monthly good eggs + feed kg used from daily_log
	type MonthlyProdRow struct {
		Month      string
		GoodEggs   int
		FeedKgUsed float64
	}
	var monthlyProd []MonthlyProdRow
	cond, args = filterCond("dl.date")
	db.DB.Raw(`
		SELECT DATE_FORMAT(dl.date, '%Y-%m') AS month,
		       COALESCE(SUM(dl.pewee+dl.pullet+dl.small+dl.medium+dl.large
		                   +dl.extra_large+dl.jumbo+dl.double_yolk), 0) AS good_eggs,
		       COALESCE(SUM(dl.feed_consumed_kg), 0) AS feed_kg_used
		FROM daily_log dl
		JOIN flock f ON f.id = dl.flock_id AND f.house_type = 'Layer'
		WHERE `+cond+`
		GROUP BY month ORDER BY month
	`, args...).Scan(&monthlyProd)

	// In-memory monthly merge
	cpeMap := make(map[string]*MonthlyCPEPoint)
	for _, r := range monthlyFeed {
		cpeMap[r.Month] = &MonthlyCPEPoint{Month: r.Month, FeedCost: r.FeedCost, FeedKgBought: r.FeedKg}
	}
	for _, r := range monthlyProd {
		if p := cpeMap[r.Month]; p != nil {
			p.GoodEggs = r.GoodEggs
			p.FeedKgUsed = r.FeedKgUsed
		} else {
			cpeMap[r.Month] = &MonthlyCPEPoint{Month: r.Month, GoodEggs: r.GoodEggs, FeedKgUsed: r.FeedKgUsed}
		}
	}
	for _, m := range data.MonthlySales {
		if p := cpeMap[m.Month]; p != nil {
			p.Revenue = m.Total
		} else {
			cpeMap[m.Month] = &MonthlyCPEPoint{Month: m.Month, Revenue: m.Total}
		}
	}
	var cpeMonths []string
	for k := range cpeMap {
		cpeMonths = append(cpeMonths, k)
	}
	sort.Strings(cpeMonths)
	for _, mo := range cpeMonths {
		p := cpeMap[mo]
		if p.GoodEggs > 0 {
			p.CostPerEgg = math.Round(p.FeedCost/float64(p.GoodEggs)*10000) / 10000
			p.RevenuePerEgg = math.Round(p.Revenue/float64(p.GoodEggs)*10000) / 10000
		}
		p.Margin = p.Revenue - p.FeedCost
		data.MonthlyCPE = append(data.MonthlyCPE, *p)
	}

	// Overall CPE KPIs
	totalRevenue := 0.0
	for _, m := range data.MonthlySales {
		totalRevenue += m.Total
	}
	if data.TotalGood > 0 {
		data.CostPerEgg = math.Round(fc.TotalCost/float64(data.TotalGood)*10000) / 10000
		data.CostPerDozen = math.Round(data.CostPerEgg*12*100) / 100
		data.RevenuePerEgg = math.Round(totalRevenue/float64(data.TotalGood)*10000) / 10000
		data.GrossMarginPerEgg = math.Round((data.RevenuePerEgg-data.CostPerEgg)*10000) / 10000
	}

	return data, nil
}
