// Package wht implements Philippine BIR Expanded Withholding Tax (EWT)
// as required by NIRC Sections 57-58 and Revenue Regulations 2-98 (as amended by
// RR 11-2018 / TRAIN, RR 14-2023, and RR 5-2025).
//
// # WHT Categories and Rates
//
//	AGRICULTURAL_PRODUCER  — 1%  on VAT-exclusive amount, ONLY when cumulative
//	                           YTD purchases exceed ₱300,000 (threshold per
//	                           RR 6-2009; palay from individual farmers).  ATC: WA010
//	PALAY_TRADER           — 2%  on VAT-exclusive amount.  ATC: WC010
//	GOODS_SUPPLIER         — 2%  on VAT-exclusive amount.  ATC: WC158
//	SERVICE_PROVIDER       — 2%  on VAT-exclusive amount.  ATC: WC158
//	PROFESSIONAL           — 5%  on VAT-exclusive amount (annual income ≤ ₱3M). ATC: WM010
//	RENTAL                 — 5%  on VAT-exclusive amount.  ATC: WB010
//	NONE                   — Not subject to EWT.
//
// # Journal entry when WHT applies (AP Invoice)
//
//	Dr  GRNI / Inventory received     (full amount)
//	Cr  Accounts Payable              (DocTotal − WHTAmount)
//	Cr  Withholding Tax Payable 2-1300 (WHTAmount)
//
// # Remittance forms
//
//	0619-E   — Monthly remittance (months 1-2 of each quarter), due 10th of following month
//	1601-EQ  — Quarterly return (incl. month 3), due last day of month after quarter-end
//	1604-E   — Annual alphalist of payees, due March 1 of following year
//	2307     — Certificate issued to supplier quarterly, due 20th of month after quarter-end
package wht

import (
	"fmt"
	"math"
	"time"

	"ricemill/app/db"
	"ricemill/app/models"
)

// ─── Constants ────────────────────────────────────────────────────────────────

const (
	// WHT supplier categories — stored in Supplier.WHTCategory
	CategoryNone                 = "NONE"
	CategoryAgriculturalProducer = "AGRICULTURAL_PRODUCER" // palay from individual farmers
	CategoryPalayTrader          = "PALAY_TRADER"          // licensed palay trader/dealer
	CategoryGoodsSupplier        = "GOODS_SUPPLIER"        // general goods supplier
	CategoryServiceProvider      = "SERVICE_PROVIDER"      // repairs, trucking, milling service
	CategoryProfessional         = "PROFESSIONAL"          // accountant, engineer, consultant
	CategoryRental               = "RENTAL"                // land, equipment, warehouse rental

	// ATC codes (Alphanumeric Tax Codes) per BIR
	ATCWa010 = "WA010" // Agricultural products (palay, corn, etc.)
	ATCWc010 = "WC010" // Goods purchased from traders/dealers
	ATCWc158 = "WC158" // General goods and services from local suppliers
	ATCWm010 = "WM010" // Professional and consultancy fees
	ATCWb010 = "WB010" // Rentals, contractors, builders

	// EWT rates
	RateAgri       = 0.01 // 1% — agricultural products above threshold
	RateTrade      = 0.02 // 2% — goods from traders / general goods / services
	RateProfSmall  = 0.05 // 5% — professional fees (annual income ≤ ₱3,000,000)
	RateRental     = 0.05 // 5% — rentals (real property, equipment, warehouse)

	// Cumulative calendar-year threshold for agricultural producers (RR 6-2009)
	AgriThresholdPHP = 300_000.0
)

// WHTResult is returned by ComputeWHT.
type WHTResult struct {
	Applicable       bool    `json:"applicable"`
	Rate             float64 `json:"rate"`               // e.g. 0.02
	RatePct          string  `json:"rate_pct"`           // e.g. "2%"
	Amount           float64 `json:"amount"`             // PHP amount to withhold
	ATCCode          string  `json:"atc_code"`
	VATExclusiveBase float64 `json:"vat_exclusive_base"` // WHT computed on this
	GrossAmount      float64 `json:"gross_amount"`       // original invoice total (VAT-inclusive)
	NetPayable       float64 `json:"net_payable"`        // gross − WHT amount
	YTDPurchases     float64 `json:"ytd_purchases"`      // relevant for agri threshold
	Reason           string  `json:"reason"`
}

// ─── Computation ──────────────────────────────────────────────────────────────

// GetSupplierYTDPurchases returns the cumulative AP invoice total (doc_total / total_amount
// column) for the given supplier in the specified calendar year, excluding cancelled invoices.
func GetSupplierYTDPurchases(supplierID uint, year int) (float64, error) {
	var total float64
	err := db.DB.Model(&models.APInvoice{}).
		Where("supplier_id = ? AND YEAR(date) = ? AND status != 'Cancelled'", supplierID, year).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&total).Error
	return total, err
}

// ComputeWHT calculates the applicable EWT for an AP invoice.
//
//   - supplierID       — used to query cumulative YTD purchases (agri threshold)
//   - whtCategory      — from Supplier.WHTCategory
//   - vatExclusiveAmt  — invoice total excluding VAT (WHT basis per BIR rules)
//   - grossAmt         — invoice total INCLUDING VAT (used for net-payable & agri threshold)
//   - invoiceDate      — determines the calendar year for the agri cumulative check
func ComputeWHT(
	supplierID uint,
	whtCategory string,
	vatExclusiveAmt float64,
	grossAmt float64,
	invoiceDate time.Time,
) (WHTResult, error) {
	res := WHTResult{
		VATExclusiveBase: vatExclusiveAmt,
		GrossAmount:      grossAmt,
	}

	applyRate := func(rate float64, atc, reason string) WHTResult {
		amount := math.Round(vatExclusiveAmt*rate*100) / 100
		res.Applicable = true
		res.Rate = rate
		res.RatePct = fmt.Sprintf("%.0f%%", rate*100)
		res.Amount = amount
		res.ATCCode = atc
		res.NetPayable = grossAmt - amount
		res.Reason = reason
		return res
	}

	switch whtCategory {
	case CategoryNone, "":
		res.Applicable = false
		res.NetPayable = grossAmt
		res.Reason = "No WHT category — not subject to EWT"
		return res, nil

	case CategoryAgriculturalProducer:
		// 1% WHT only after cumulative calendar-year purchases exceed ₱300,000
		ytd, err := GetSupplierYTDPurchases(supplierID, invoiceDate.Year())
		if err != nil {
			return res, fmt.Errorf("compute WHT: query YTD: %w", err)
		}
		res.YTDPurchases = ytd
		if ytd >= AgriThresholdPHP {
			return applyRate(RateAgri, ATCWa010,
				fmt.Sprintf("1%% EWT on palay purchase — YTD ₱%.2f exceeds ₱300,000 threshold (RR 6-2009, WA010)", ytd)), nil
		} else if ytd+grossAmt > AgriThresholdPHP {
			// This invoice crosses the threshold — withhold on this invoice's VAT-exclusive amount
			return applyRate(RateAgri, ATCWa010,
				fmt.Sprintf("1%% EWT — threshold crossed with this invoice (YTD before: ₱%.2f) (RR 6-2009, WA010)", ytd)), nil
		}
		res.Applicable = false
		res.NetPayable = grossAmt
		res.Reason = fmt.Sprintf("YTD purchases ₱%.2f < ₱300,000 threshold — no WHT yet (RR 6-2009, WA010)", ytd)
		return res, nil

	case CategoryPalayTrader:
		return applyRate(RateTrade, ATCWc010, "2% EWT on goods from palay trader/dealer (RR 2-98 §2.57.2(B), WC010)"), nil

	case CategoryGoodsSupplier:
		return applyRate(RateTrade, ATCWc158, "2% EWT on goods from local supplier (RR 2-98 §2.57.2(B), WC158)"), nil

	case CategoryServiceProvider:
		return applyRate(RateTrade, ATCWc158, "2% EWT on service payment — repairs/transport/milling (RR 2-98 §2.57.2(C), WC158)"), nil

	case CategoryProfessional:
		return applyRate(RateProfSmall, ATCWm010, "5% EWT on professional fee (annual income ≤ ₱3M) (RR 2-98 §2.57.2(A), WM010)"), nil

	case CategoryRental:
		return applyRate(RateRental, ATCWb010, "5% EWT on rental payment — real property/equipment (RR 2-98 §2.57.2(D), WB010)"), nil

	default:
		res.Applicable = false
		res.NetPayable = grossAmt
		res.Reason = fmt.Sprintf("Unknown WHT category '%s'", whtCategory)
		return res, nil
	}
}

// ─── Reporting DTOs ───────────────────────────────────────────────────────────

// WHTEntry is a single AP invoice that has WHT applied — used for all reports.
type WHTEntry struct {
	InvoiceID      uint      `json:"invoice_id"`
	InvoiceNumber  string    `json:"invoice_number"`
	PostingDate    time.Time `json:"posting_date"`
	SupplierID     *uint     `json:"supplier_id"`
	SupplierName   string    `json:"supplier_name"`
	SupplierTIN    string    `json:"supplier_tin"`
	WHTCategory    string    `json:"wht_category"`
	ATCCode        string    `json:"atc_code"`
	GrossAmount    float64   `json:"gross_amount"`    // doc_total (VAT-inclusive)
	VATExclusive   float64   `json:"vat_exclusive"`   // WHT base
	WHTRate        float64   `json:"wht_rate"`
	WHTAmount      float64   `json:"wht_amount"`
	NetPayable     float64   `json:"net_payable"`
	Quarter        int       `json:"quarter"`
	Month          int       `json:"month"`
	Year           int       `json:"year"`
}

// Form2307Line is one row inside a BIR Form 2307 for a payee.
type Form2307Line struct {
	Month   int     `json:"month"`
	Payment float64 `json:"payment"` // income payment (VAT-exclusive)
	WHT     float64 `json:"wht"`     // amount withheld
}

// Form2307 represents the BIR Certificate of Creditable Tax Withheld (per payee per quarter).
type Form2307 struct {
	Quarter       int            `json:"quarter"`
	Year          int            `json:"year"`
	IssuedBy      string         `json:"issued_by"`   // rice mill name
	PayeeName     string         `json:"payee_name"`
	PayeeTIN      string         `json:"payee_tin"`
	ATCCode       string         `json:"atc_code"`
	WHTCategory   string         `json:"wht_category"`
	Lines         []Form2307Line `json:"lines"`        // one per month (max 3)
	TotalPayment  float64        `json:"total_payment"`
	TotalWHT      float64        `json:"total_wht"`
	IssueDueDate  string         `json:"issue_due_date"` // 20th of month after quarter-end
}

// ATCSummary is one row in the 0619-E / 1601-EQ totals.
type ATCSummary struct {
	ATCCode     string  `json:"atc_code"`
	WHTCategory string  `json:"wht_category"`
	TotalIncome float64 `json:"total_income"` // sum of VAT-exclusive amounts
	TotalWHT    float64 `json:"total_wht"`
}

// Form0619E is the monthly remittance return data (first 2 months of each quarter).
type Form0619E struct {
	Year       int          `json:"year"`
	Month      int          `json:"month"`
	DueDate    string       `json:"due_date"` // 10th of following month
	ATCSummary []ATCSummary `json:"atc_summary"`
	TotalWHT   float64      `json:"total_wht"`
	Entries    []WHTEntry   `json:"entries"`
}

// Form1601EQ is the quarterly EWT return data.
type Form1601EQ struct {
	Year       int          `json:"year"`
	Quarter    int          `json:"quarter"`
	DueDate    string       `json:"due_date"` // last day of month following quarter-end
	ATCSummary []ATCSummary `json:"atc_summary"`
	TotalWHT   float64      `json:"total_wht"`
	Entries    []WHTEntry   `json:"entries"`
}

// ─── List helpers ─────────────────────────────────────────────────────────────

// ListWHTEntries returns all AP invoices that have WHT applied (wht_amount > 0),
// optionally filtered by year and quarter (0 = all).
func ListWHTEntries(year, quarter int) ([]WHTEntry, error) {
	type row struct {
		InvoiceID     uint      `gorm:"column:id"`
		InvoiceNumber string    `gorm:"column:invoice_number"`
		PostingDate   time.Time `gorm:"column:date"`
		SupplierID    *uint     `gorm:"column:supplier_id"`
		SupplierName  string    `gorm:"column:supplier_name"`
		TINNumber     string    `gorm:"column:tin_number"`
		WHTCategory   string    `gorm:"column:wht_category"`
		ATCCode       string    `gorm:"column:wht_atc_code"`
		GrossAmount   float64   `gorm:"column:total_amount"`
		VATExclusive  float64   `gorm:"column:vat_exclusive_amount"`
		WHTRate       float64   `gorm:"column:wht_rate"`
		WHTAmount     float64   `gorm:"column:wht_amount"`
		NetPayable    float64   `gorm:"column:net_payable"`
	}

	q := db.DB.Table("ap_invoice").
		Select(`ap_invoice.id, ap_invoice.invoice_number, ap_invoice.date,
			ap_invoice.supplier_id, ap_invoice.supplier_name,
			COALESCE(supplier.tin_number,'') AS tin_number,
			COALESCE(supplier.wht_category,'NONE') AS wht_category,
			ap_invoice.wht_atc_code, ap_invoice.total_amount,
			ap_invoice.vat_exclusive_amount, ap_invoice.wht_rate,
			ap_invoice.wht_amount, ap_invoice.net_payable`).
		Joins("LEFT JOIN supplier ON supplier.id = ap_invoice.supplier_id").
		Where("ap_invoice.wht_amount > 0 AND ap_invoice.status != 'Cancelled'")

	if year > 0 {
		q = q.Where("YEAR(ap_invoice.date) = ?", year)
	}
	if quarter > 0 {
		q = q.Where("QUARTER(ap_invoice.date) = ?", quarter)
	}

	var rows []row
	if err := q.Order("ap_invoice.date").Scan(&rows).Error; err != nil {
		return nil, err
	}

	entries := make([]WHTEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, WHTEntry{
			InvoiceID:     r.InvoiceID,
			InvoiceNumber: r.InvoiceNumber,
			PostingDate:   r.PostingDate,
			SupplierID:    r.SupplierID,
			SupplierName:  r.SupplierName,
			SupplierTIN:   r.TINNumber,
			WHTCategory:   r.WHTCategory,
			ATCCode:       r.ATCCode,
			GrossAmount:   r.GrossAmount,
			VATExclusive:  r.VATExclusive,
			WHTRate:       r.WHTRate,
			WHTAmount:     r.WHTAmount,
			NetPayable:    r.NetPayable,
			Quarter:       int((r.PostingDate.Month() + 2) / 3),
			Month:         int(r.PostingDate.Month()),
			Year:          r.PostingDate.Year(),
		})
	}
	return entries, nil
}

// ─── Form generators ──────────────────────────────────────────────────────────

// quarterDueDate returns the due date for Form 0619-E (10th of following month)
// and Form 1601-EQ (last day of month following quarter-end).
func lastDayOfMonth(year, month int) time.Time {
	first := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
	return first.AddDate(0, 0, -1)
}

// GenerateForm0619E builds the data for a monthly remittance (months 1-2 of each quarter).
// Use quarter=0 to allow any month.
func GenerateForm0619E(year, month int) (*Form0619E, error) {
	entries, err := ListWHTEntries(year, 0)
	if err != nil {
		return nil, err
	}

	// Filter to requested month
	var filtered []WHTEntry
	for _, e := range entries {
		if e.Month == month {
			filtered = append(filtered, e)
		}
	}

	// Build ATC summary
	atcMap := map[string]*ATCSummary{}
	total := 0.0
	for _, e := range filtered {
		key := e.ATCCode
		if _, ok := atcMap[key]; !ok {
			atcMap[key] = &ATCSummary{ATCCode: key, WHTCategory: e.WHTCategory}
		}
		atcMap[key].TotalIncome += e.VATExclusive
		atcMap[key].TotalWHT += e.WHTAmount
		total += e.WHTAmount
	}
	atcList := make([]ATCSummary, 0, len(atcMap))
	for _, v := range atcMap {
		atcList = append(atcList, *v)
	}

	dueDate := time.Date(year, time.Month(month+1), 10, 0, 0, 0, 0, time.UTC)

	return &Form0619E{
		Year:       year,
		Month:      month,
		DueDate:    dueDate.Format("January 10, 2006"),
		ATCSummary: atcList,
		TotalWHT:   math.Round(total*100) / 100,
		Entries:    filtered,
	}, nil
}

// GenerateForm1601EQ builds the data for a quarterly EWT return.
func GenerateForm1601EQ(year, quarter int) (*Form1601EQ, error) {
	entries, err := ListWHTEntries(year, quarter)
	if err != nil {
		return nil, err
	}

	// Build ATC summary
	atcMap := map[string]*ATCSummary{}
	total := 0.0
	for _, e := range entries {
		key := e.ATCCode
		if _, ok := atcMap[key]; !ok {
			atcMap[key] = &ATCSummary{ATCCode: key, WHTCategory: e.WHTCategory}
		}
		atcMap[key].TotalIncome += e.VATExclusive
		atcMap[key].TotalWHT += e.WHTAmount
		total += e.WHTAmount
	}
	atcList := make([]ATCSummary, 0, len(atcMap))
	for _, v := range atcMap {
		atcList = append(atcList, *v)
	}

	// Due date: last day of month following quarter-end
	quarterEndMonth := quarter * 3
	dueDateMonth := quarterEndMonth + 1
	dueYear := year
	if dueDateMonth > 12 {
		dueDateMonth -= 12
		dueYear++
	}
	due := lastDayOfMonth(dueYear, dueDateMonth)

	return &Form1601EQ{
		Year:       year,
		Quarter:    quarter,
		DueDate:    due.Format("January 2, 2006"),
		ATCSummary: atcList,
		TotalWHT:   math.Round(total*100) / 100,
		Entries:    entries,
	}, nil
}

// GenerateForm2307 builds BIR Form 2307 data for a single supplier in a given quarter.
// company is the rice mill's registered name (printed as "Issued By").
func GenerateForm2307(supplierID uint, year, quarter int, company string) (*Form2307, error) {
	entries, err := ListWHTEntries(year, quarter)
	if err != nil {
		return nil, err
	}

	// Filter to this supplier
	var sup models.Supplier
	if err := db.DB.First(&sup, supplierID).Error; err != nil {
		return nil, fmt.Errorf("supplier not found: %w", err)
	}

	var filtered []WHTEntry
	for _, e := range entries {
		if e.SupplierID != nil && *e.SupplierID == supplierID {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		return &Form2307{
			Quarter: quarter, Year: year,
			IssuedBy: company, PayeeName: sup.Name, PayeeTIN: sup.TINNumber,
		}, nil
	}

	// Build monthly lines (3 months in quarter)
	startMonth := (quarter-1)*3 + 1
	lineMap := map[int]*Form2307Line{}
	for m := startMonth; m < startMonth+3; m++ {
		lineMap[m] = &Form2307Line{Month: m}
	}
	atcCode := ""
	whtCat := ""
	totPay, totWHT := 0.0, 0.0
	for _, e := range filtered {
		ln := lineMap[e.Month]
		if ln != nil {
			ln.Payment += e.VATExclusive
			ln.WHT += e.WHTAmount
		}
		atcCode = e.ATCCode
		whtCat = e.WHTCategory
		totPay += e.VATExclusive
		totWHT += e.WHTAmount
	}
	lines := []Form2307Line{
		*lineMap[startMonth],
		*lineMap[startMonth+1],
		*lineMap[startMonth+2],
	}

	// Issue due date: 20th of month after quarter-end
	quarterEndMonth := quarter * 3
	issueDueDateMonth := quarterEndMonth + 1
	issueDueYear := year
	if issueDueDateMonth > 12 {
		issueDueDateMonth -= 12
		issueDueYear++
	}
	issueDate := time.Date(issueDueYear, time.Month(issueDueDateMonth), 20, 0, 0, 0, 0, time.UTC)

	return &Form2307{
		Quarter:      quarter,
		Year:         year,
		IssuedBy:     company,
		PayeeName:    sup.Name,
		PayeeTIN:     sup.TINNumber,
		ATCCode:      atcCode,
		WHTCategory:  whtCat,
		Lines:        lines,
		TotalPayment: math.Round(totPay*100) / 100,
		TotalWHT:     math.Round(totWHT*100) / 100,
		IssueDueDate: issueDate.Format("January 20, 2006"),
	}, nil
}

// ListSuppliersWithWHT returns all distinct suppliers that have at least one
// WHT entry in the given year (for the 2307 payee selector).
func ListSuppliersWithWHT(year int) ([]models.Supplier, error) {
	var ids []uint
	if err := db.DB.Model(&models.APInvoice{}).
		Select("DISTINCT supplier_id").
		Where("YEAR(date) = ? AND wht_amount > 0 AND supplier_id IS NOT NULL AND status != 'Cancelled'", year).
		Pluck("supplier_id", &ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []models.Supplier{}, nil
	}
	var suppliers []models.Supplier
	return suppliers, db.DB.Where("id IN ?", ids).Order("name").Find(&suppliers).Error
}
