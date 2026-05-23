package reports

// Statement of Cash Flows — PAS 7 (Philippine Accounting Standard 7)
// Indirect method for operating activities.
//
// Classification tags on gl_account.cash_flow_class:
//   CASH                   — Cash & cash equivalents (opening / closing balances)
//   OPERATING_NON_CASH     — Non-cash expenses added back: depreciation, impairment
//   OPERATING_WC_ASSET     — Working-capital assets: AR, inventory, GRNI, Input VAT
//   OPERATING_WC_LIABILITY — Working-capital liabilities: AP, Output VAT, WHT Payable
//   OPERATING_TAX          — Income tax payable (tracked separately)
//   INVESTING_PPE          — PPE accounts: debit = purchase; credit = disposal proceeds
//   FINANCING_DEBT         — Loans payable: credit = proceeds; debit = repayment
//   FINANCING_EQUITY       — Owner's capital: credit = contribution; debit = withdrawal

import (
	"fmt"
	"time"

	"ricemill/app/db"
)

// ─── Data types ──────────────────────────────────────────────────────────────

// CFLine is one line item in the cash flow statement.
type CFLine struct {
	Code        string  `json:"code"`         // GL account code (empty for computed lines)
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`       // positive = inflow; negative = outflow
	Indent      int     `json:"indent"`       // 0 = section heading, 1 = line item, 2 = sub-item
	IsSeparator bool    `json:"is_separator"` // render as a divider row
	IsTotal     bool    `json:"is_total"`     // render in bold
}

// CFSection holds one of the three sections plus its net total.
type CFSection struct {
	Title string   `json:"title"`
	Lines []CFLine `json:"lines"`
	Net   float64  `json:"net"` // net cash flow for this section
}

// CashCompositionItem is one component of cash & cash equivalents.
type CashCompositionItem struct {
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// CashFlowStatement is the complete PAS 7 report.
type CashFlowStatement struct {
	PeriodStart  string `json:"period_start"`
	PeriodEnd    string `json:"period_end"`
	GeneratedAt  string `json:"generated_at"`

	Operating CFSection `json:"operating"`
	Investing CFSection `json:"investing"`
	Financing CFSection `json:"financing"`

	NetCashChange float64 `json:"net_cash_change"` // sum of three sections
	OpeningCash   float64 `json:"opening_cash"`
	ClosingCash   float64 `json:"closing_cash"`

	// Note disclosure: composition of cash & cash equivalents (PAS 7 par. 48)
	CashComposition []CashCompositionItem `json:"cash_composition"`

	// Check: closing_cash should equal opening_cash + net_cash_change.
	// Any non-zero value indicates unclassified transactions.
	CheckDiff float64 `json:"check_diff"`
}

// ─── Main entry point ─────────────────────────────────────────────────────────

// GetCashFlowStatement generates a PAS 7 Statement of Cash Flows (indirect method)
// for the period [startDate, endDate] (ISO 8601 "2006-01-02").
func GetCashFlowStatement(startDate, endDate string) (*CashFlowStatement, error) {
	// Validate and parse dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("end date must be on or after start date")
	}

	// Day before start for opening-balance snapshots
	openingAsOf := start.AddDate(0, 0, -1).Format("2006-01-02")

	stmt := &CashFlowStatement{
		PeriodStart: startDate,
		PeriodEnd:   endDate,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// ── 1. Operating activities ─────────────────────────────────────────────
	operating, err := buildOperating(startDate, endDate, openingAsOf)
	if err != nil {
		return nil, fmt.Errorf("operating activities: %w", err)
	}
	stmt.Operating = operating

	// ── 2. Investing activities ─────────────────────────────────────────────
	investing, err := buildInvesting(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("investing activities: %w", err)
	}
	stmt.Investing = investing

	// ── 3. Financing activities ─────────────────────────────────────────────
	financing, err := buildFinancing(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("financing activities: %w", err)
	}
	stmt.Financing = financing

	// ── 4. Net change + opening/closing cash ───────────────────────────────
	stmt.NetCashChange = stmt.Operating.Net + stmt.Investing.Net + stmt.Financing.Net
	stmt.OpeningCash, err = cashBalance(openingAsOf)
	if err != nil {
		return nil, fmt.Errorf("opening cash: %w", err)
	}
	stmt.ClosingCash, err = cashBalance(endDate)
	if err != nil {
		return nil, fmt.Errorf("closing cash: %w", err)
	}
	stmt.CheckDiff = stmt.ClosingCash - (stmt.OpeningCash + stmt.NetCashChange)

	// ── 5. Cash composition note ────────────────────────────────────────────
	stmt.CashComposition, err = cashComposition(endDate)
	if err != nil {
		return nil, fmt.Errorf("cash composition: %w", err)
	}

	return stmt, nil
}

// ─── Operating Activities (Indirect Method) ───────────────────────────────────

func buildOperating(startDate, endDate, openingAsOf string) (CFSection, error) {
	sec := CFSection{Title: "Cash Flows from Operating Activities"}

	// (A) Profit before income tax — from P&L net income
	pnl, err := GetPnL(startDate, endDate)
	if err != nil {
		return sec, err
	}
	profitBeforeTax := pnl.NetIncome
	sec.Lines = append(sec.Lines, CFLine{
		Description: "Profit before income tax",
		Amount:      profitBeforeTax,
		Indent:      1,
	})

	// (B) Adjustments for non-cash items (add back depreciation & impairment)
	nonCashRows, err := nonCashAdjustments(startDate, endDate)
	if err != nil {
		return sec, err
	}
	if len(nonCashRows) > 0 {
		sec.Lines = append(sec.Lines, CFLine{Description: "Adjustments for non-cash items:", Indent: 1})
		for _, r := range nonCashRows {
			// Non-cash expense accounts are debit-normal; their balance is positive.
			// In the indirect method we add them back → positive amount = add-back.
			sec.Lines = append(sec.Lines, CFLine{Code: r.Code, Description: r.Name, Amount: r.Amount, Indent: 2})
		}
	}

	// Subtotal: operating profit adjusted for non-cash items
	nonCashTotal := 0.0
	for _, r := range nonCashRows {
		nonCashTotal += r.Amount
	}

	// (C) Working-capital changes
	wcLines, wcTotal, err := workingCapitalChanges(openingAsOf, endDate)
	if err != nil {
		return sec, err
	}
	if len(wcLines) > 0 {
		sec.Lines = append(sec.Lines, CFLine{Description: "Changes in working capital:", Indent: 1})
		sec.Lines = append(sec.Lines, wcLines...)
	}

	// (D) Cash generated from operations (before tax)
	cashFromOps := profitBeforeTax + nonCashTotal + wcTotal
	sec.Lines = append(sec.Lines, CFLine{IsSeparator: true})
	sec.Lines = append(sec.Lines, CFLine{
		Description: "Cash generated from operations",
		Amount:      cashFromOps,
		Indent:      1,
		IsTotal:     true,
	})

	// (E) Income tax paid (PAS 7 par. 35 — separately disclosed)
	taxPaid, err := incomeTaxPaid(startDate, endDate)
	if err != nil {
		return sec, err
	}
	sec.Lines = append(sec.Lines, CFLine{
		Description: "Income tax paid",
		Amount:      -taxPaid, // cash outflow
		Indent:      1,
	})

	sec.Net = cashFromOps - taxPaid
	sec.Lines = append(sec.Lines, CFLine{IsSeparator: true})
	sec.Lines = append(sec.Lines, CFLine{
		Description: "Net Cash from Operating Activities",
		Amount:      sec.Net,
		Indent:      0,
		IsTotal:     true,
	})
	return sec, nil
}

// ─── Investing Activities ─────────────────────────────────────────────────────

func buildInvesting(startDate, endDate string) (CFSection, error) {
	sec := CFSection{Title: "Cash Flows from Investing Activities"}

	type row struct {
		Code   string
		Name   string
		Debits float64
		Credits float64
	}
	var rows []row
	query := `
		SELECT ga.code, ga.name,
		       COALESCE(SUM(jel.debit),0)  AS debits,
		       COALESCE(SUM(jel.credit),0) AS credits
		FROM je_line jel
		JOIN journal_entry je ON je.id = jel.journal_entry_id
		JOIN gl_account ga   ON ga.id  = jel.gl_account_id
		WHERE je.status = 'POSTED'
		  AND je.date BETWEEN ? AND ?
		  AND ga.cash_flow_class = 'INVESTING_PPE'
		GROUP BY ga.id, ga.code, ga.name
		ORDER BY ga.code`
	if err := db.DB.Raw(query, startDate, endDate).Scan(&rows).Error; err != nil {
		return sec, err
	}

	var totalPurchases, totalProceeds float64
	if len(rows) > 0 {
		sec.Lines = append(sec.Lines, CFLine{Description: "Capital expenditures:", Indent: 1})
	}
	for _, r := range rows {
		if r.Debits > 0 {
			sec.Lines = append(sec.Lines, CFLine{
				Code:        r.Code,
				Description: "Purchase of " + r.Name,
				Amount:      -r.Debits, // outflow
				Indent:      2,
			})
			totalPurchases += r.Debits
		}
		if r.Credits > 0 {
			sec.Lines = append(sec.Lines, CFLine{
				Code:        r.Code,
				Description: "Proceeds from disposal of " + r.Name,
				Amount:      r.Credits, // inflow
				Indent:      2,
			})
			totalProceeds += r.Credits
		}
	}

	sec.Net = totalProceeds - totalPurchases
	sec.Lines = append(sec.Lines, CFLine{IsSeparator: true})
	sec.Lines = append(sec.Lines, CFLine{
		Description: "Net Cash used in Investing Activities",
		Amount:      sec.Net,
		Indent:      0,
		IsTotal:     true,
	})
	return sec, nil
}

// ─── Financing Activities ─────────────────────────────────────────────────────

func buildFinancing(startDate, endDate string) (CFSection, error) {
	sec := CFSection{Title: "Cash Flows from Financing Activities"}

	type row struct {
		Code    string
		Name    string
		Class   string
		Debits  float64
		Credits float64
	}
	var rows []row
	query := `
		SELECT ga.code, ga.name, ga.cash_flow_class,
		       COALESCE(SUM(jel.debit),0)  AS debits,
		       COALESCE(SUM(jel.credit),0) AS credits
		FROM je_line jel
		JOIN journal_entry je ON je.id = jel.journal_entry_id
		JOIN gl_account ga   ON ga.id  = jel.gl_account_id
		WHERE je.status = 'POSTED'
		  AND je.date BETWEEN ? AND ?
		  AND ga.cash_flow_class IN ('FINANCING_DEBT','FINANCING_EQUITY')
		GROUP BY ga.id, ga.code, ga.name, ga.cash_flow_class
		ORDER BY ga.cash_flow_class, ga.code`
	if err := db.DB.Raw(query, startDate, endDate).Scan(&rows).Error; err != nil {
		return sec, err
	}

	var netFinancing float64
	for _, r := range rows {
		switch r.Class {
		case "FINANCING_DEBT":
			// Credit on Loans Payable = proceeds (inflow); Debit = repayment (outflow)
			if r.Credits > 0 {
				sec.Lines = append(sec.Lines, CFLine{
					Code:        r.Code,
					Description: "Proceeds from " + r.Name,
					Amount:      r.Credits,
					Indent:      1,
				})
				netFinancing += r.Credits
			}
			if r.Debits > 0 {
				sec.Lines = append(sec.Lines, CFLine{
					Code:        r.Code,
					Description: "Repayment of " + r.Name,
					Amount:      -r.Debits,
					Indent:      1,
				})
				netFinancing -= r.Debits
			}
		case "FINANCING_EQUITY":
			// Credit on Owner's Capital = contribution (inflow); Debit = withdrawal/dividend (outflow)
			if r.Credits > 0 {
				sec.Lines = append(sec.Lines, CFLine{
					Code:        r.Code,
					Description: "Capital contribution — " + r.Name,
					Amount:      r.Credits,
					Indent:      1,
				})
				netFinancing += r.Credits
			}
			if r.Debits > 0 {
				sec.Lines = append(sec.Lines, CFLine{
					Code:        r.Code,
					Description: "Capital withdrawal / dividends paid",
					Amount:      -r.Debits,
					Indent:      1,
				})
				netFinancing -= r.Debits
			}
		}
	}

	sec.Net = netFinancing
	sec.Lines = append(sec.Lines, CFLine{IsSeparator: true})
	sec.Lines = append(sec.Lines, CFLine{
		Description: "Net Cash from Financing Activities",
		Amount:      sec.Net,
		Indent:      0,
		IsTotal:     true,
	})
	return sec, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

type glAmountRow struct {
	Code   string
	Name   string
	Amount float64
}

// nonCashAdjustments returns the period movement on OPERATING_NON_CASH accounts.
// These are add-backs to profit (depreciation, impairment, stock losses).
func nonCashAdjustments(startDate, endDate string) ([]glAmountRow, error) {
	var rows []glAmountRow
	// Non-cash expenses are debit-normal; net debit movement = add-back amount.
	query := `
		SELECT ga.code, ga.name,
		       COALESCE(SUM(jel.debit - jel.credit), 0) AS amount
		FROM je_line jel
		JOIN journal_entry je ON je.id = jel.journal_entry_id
		JOIN gl_account ga   ON ga.id  = jel.gl_account_id
		WHERE je.status = 'POSTED'
		  AND je.date BETWEEN ? AND ?
		  AND ga.cash_flow_class = 'OPERATING_NON_CASH'
		GROUP BY ga.id, ga.code, ga.name
		HAVING ABS(amount) > 0.001
		ORDER BY ga.code`
	return rows, db.DB.Raw(query, startDate, endDate).Scan(&rows).Error
}

// workingCapitalChanges computes the change in each WC account between
// openingAsOf and endDate. Returns the formatted lines and net WC impact.
func workingCapitalChanges(openingAsOf, endDate string) ([]CFLine, float64, error) {
	type accountBalance struct {
		ID            uint
		Code          string
		Name          string
		CashFlowClass string
		NormalBalance string
		Opening       float64
		Closing       float64
	}

	// Fetch all WC accounts
	var accounts []struct {
		ID            uint
		Code          string
		Name          string
		CashFlowClass string
		NormalBalance string
	}
	if err := db.DB.Raw(`
		SELECT id, code, name, cash_flow_class, normal_balance
		FROM gl_account
		WHERE cash_flow_class IN ('OPERATING_WC_ASSET','OPERATING_WC_LIABILITY')
		  AND account_type = 'POSTING'
		ORDER BY code`).Scan(&accounts).Error; err != nil {
		return nil, 0, err
	}
	if len(accounts) == 0 {
		return nil, 0, nil
	}

	// Build balance for each account at two dates
	acctBalances := make([]accountBalance, len(accounts))
	for i, a := range accounts {
		acctBalances[i] = accountBalance{
			ID:            a.ID,
			Code:          a.Code,
			Name:          a.Name,
			CashFlowClass: a.CashFlowClass,
			NormalBalance: a.NormalBalance,
		}
	}

	// Helper: get cumulative balance for all WC accounts up to a given date
	type balRow struct {
		GLID  uint
		Total float64 // SUM(debit) - SUM(credit)
	}
	getBalances := func(asOf string) (map[uint]float64, error) {
		var rows []balRow
		err := db.DB.Raw(`
			SELECT jel.gl_account_id AS gl_id,
			       COALESCE(SUM(jel.debit - jel.credit), 0) AS total
			FROM je_line jel
			JOIN journal_entry je ON je.id = jel.journal_entry_id
			WHERE je.status = 'POSTED'
			  AND je.date <= ?
			  AND jel.gl_account_id IN (
			        SELECT id FROM gl_account
			        WHERE cash_flow_class IN ('OPERATING_WC_ASSET','OPERATING_WC_LIABILITY')
			          AND account_type = 'POSTING'
			      )
			GROUP BY jel.gl_account_id`, asOf).Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		m := make(map[uint]float64, len(rows))
		for _, r := range rows {
			m[r.GLID] = r.Total
		}
		return m, nil
	}

	openBals, err := getBalances(openingAsOf)
	if err != nil {
		return nil, 0, err
	}
	closeBals, err := getBalances(endDate)
	if err != nil {
		return nil, 0, err
	}

	var lines []CFLine
	var netWC float64

	for _, a := range acctBalances {
		openBal := openBals[a.ID]
		closeBal := closeBals[a.ID]

		// Signed balance: positive = net debit balance (normal for assets)
		var openSigned, closeSigned float64
		if a.NormalBalance == "DEBIT" {
			openSigned = openBal
			closeSigned = closeBal
		} else {
			// CREDIT-normal (liabilities): credit > debit = positive balance
			openSigned = -openBal
			closeSigned = -closeBal
		}

		change := closeSigned - openSigned
		if change == 0 {
			continue
		}

		// Cash flow impact:
		//   WC asset increases  → uses cash → negative CF impact
		//   WC asset decreases  → frees cash → positive CF impact
		//   WC liability increases → saves cash → positive CF impact
		//   WC liability decreases → uses cash → negative CF impact
		var cfImpact float64
		switch a.CashFlowClass {
		case "OPERATING_WC_ASSET":
			cfImpact = -change // increase in asset = cash used
		case "OPERATING_WC_LIABILITY":
			cfImpact = change // increase in liability = cash saved
		}

		if cfImpact == 0 {
			continue
		}

		var label string
		if change > 0 {
			label = fmt.Sprintf("Increase in %s", a.Name)
		} else {
			label = fmt.Sprintf("Decrease in %s", a.Name)
		}

		lines = append(lines, CFLine{
			Code:        a.Code,
			Description: label,
			Amount:      cfImpact,
			Indent:      2,
		})
		netWC += cfImpact
	}

	return lines, netWC, nil
}

// incomeTaxPaid returns the actual income tax cash paid during the period.
// In the current COA there is no income-tax-payable account, so this returns 0
// unless the user has tagged an account as OPERATING_TAX and posted JEs to it.
func incomeTaxPaid(startDate, endDate string) (float64, error) {
	var total float64
	err := db.DB.Raw(`
		SELECT COALESCE(SUM(jel.debit - jel.credit), 0)
		FROM je_line jel
		JOIN journal_entry je ON je.id = jel.journal_entry_id
		JOIN gl_account ga   ON ga.id  = jel.gl_account_id
		WHERE je.status = 'POSTED'
		  AND je.date BETWEEN ? AND ?
		  AND ga.cash_flow_class = 'OPERATING_TAX'`, startDate, endDate).Scan(&total).Error
	// A net debit on tax-payable = tax paid (cash out)
	if total < 0 {
		total = -total
	}
	return total, err
}

// cashBalance returns the net balance of all cash & cash-equivalent GL accounts
// as of the given date (cumulative sum of all posted JEs).
func cashBalance(asOf string) (float64, error) {
	var total float64
	err := db.DB.Raw(`
		SELECT COALESCE(SUM(jel.debit - jel.credit), 0)
		FROM je_line jel
		JOIN journal_entry je ON je.id = jel.journal_entry_id
		JOIN gl_account ga   ON ga.id  = jel.gl_account_id
		WHERE je.status = 'POSTED'
		  AND je.date <= ?
		  AND ga.cash_flow_class = 'CASH'`, asOf).Scan(&total).Error
	return total, err
}

// cashComposition returns the balance of each individual cash account at endDate.
func cashComposition(asOf string) ([]CashCompositionItem, error) {
	var rows []struct {
		Code   string
		Name   string
		Amount float64
	}
	err := db.DB.Raw(`
		SELECT ga.code, ga.name,
		       COALESCE(SUM(jel.debit - jel.credit), 0) AS amount
		FROM je_line jel
		JOIN journal_entry je ON je.id = jel.journal_entry_id
		JOIN gl_account ga   ON ga.id  = jel.gl_account_id
		WHERE je.status = 'POSTED'
		  AND je.date <= ?
		  AND ga.cash_flow_class = 'CASH'
		  AND ga.account_type = 'POSTING'
		GROUP BY ga.id, ga.code, ga.name
		ORDER BY ga.code`, asOf).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]CashCompositionItem, len(rows))
	for i, r := range rows {
		items[i] = CashCompositionItem{Code: r.Code, Name: r.Name, Amount: r.Amount}
	}
	return items, nil
}
