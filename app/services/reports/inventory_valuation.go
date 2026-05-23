package reports

// inventory_valuation.go — Inventory Valuation Reports
//
// Report A — GetInventoryValuationSnapshot
//   Returns stock quantities and monetary values as of a given date.
//   Two query paths:
//     • Current (asOf = today): reads oitm.on_hand × oitm.avg_price (authoritative)
//     • Historical (asOf < today): reconstructs from oivl cumulative movements
//
// Report B — GetInventoryMovementReport
//   Paginated ledger of all OIVL entries matching the supplied filters.

import (
	"fmt"
	"math"
	"time"

	"ricemill/app/db"
)

// ─────────────────────────────────────────────────────────────────────────────
// DTOs — Report A: Inventory Valuation Snapshot
// ─────────────────────────────────────────────────────────────────────────────

// InvValuationRow is one item line in the valuation report.
type InvValuationRow struct {
	ItemCode     string  `json:"item_code"`
	ItemName     string  `json:"item_name"`
	CategoryName string  `json:"category_name"`
	Warehouse    string  `json:"warehouse"`   // "" = company-wide aggregate
	UoM          string  `json:"uom"`
	QtyOnHand    float64 `json:"qty_on_hand"`
	UnitCost     float64 `json:"unit_cost"`
	TotalValue   float64 `json:"total_value"`
	PctOfTotal   float64 `json:"pct_of_total"`
}

// InvValuationGroup groups rows by item category for subtotal display.
type InvValuationGroup struct {
	CategoryName string            `json:"category_name"`
	Rows         []InvValuationRow `json:"rows"`
	SubTotalQty  float64           `json:"sub_total_qty"`
	SubTotalValue float64          `json:"sub_total_value"`
}

// InvValuationReport is the full response for Report A.
type InvValuationReport struct {
	AsOf         string              `json:"as_of"`
	Warehouse    string              `json:"warehouse"`    // "" = all warehouses
	GroupCode    int                 `json:"group_code"`   // 0 = all groups
	IsHistorical bool                `json:"is_historical"` // true = OIVL-derived
	GeneratedAt  string              `json:"generated_at"`
	Groups       []InvValuationGroup `json:"groups"`
	TotalQty     float64             `json:"total_qty"`
	TotalValue   float64             `json:"total_value"`
}

// ─────────────────────────────────────────────────────────────────────────────
// DTOs — Report B: Inventory Movement Report
// ─────────────────────────────────────────────────────────────────────────────

// InvMovementRow is one ledger entry line.
type InvMovementRow struct {
	DocDate    string  `json:"doc_date"`
	TransType  string  `json:"trans_type"`
	TransLabel string  `json:"trans_label"` // human-readable label
	DocNum     int     `json:"doc_num"`
	ItemCode   string  `json:"item_code"`
	ItemName   string  `json:"item_name"`
	Warehouse  string  `json:"warehouse"`
	InQty      float64 `json:"in_qty"`
	OutQty     float64 `json:"out_qty"`
	Price      float64 `json:"price"`
	Value      float64 `json:"value"`
}

// InvMovementReport is the full response for Report B.
type InvMovementReport struct {
	DateFrom      string           `json:"date_from"`
	DateTo        string           `json:"date_to"`
	ItemCode      string           `json:"item_code"`
	Warehouse     string           `json:"warehouse"`
	TransType     string           `json:"trans_type"`
	Rows          []InvMovementRow `json:"rows"`
	TotalInQty    float64          `json:"total_in_qty"`
	TotalOutQty   float64          `json:"total_out_qty"`
	TotalInValue  float64          `json:"total_in_value"`
	TotalOutValue float64          `json:"total_out_value"`
	Truncated     bool             `json:"truncated"` // true if result hit the 2 000-row cap
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// transTypeLabel returns a human-readable description for an OIVL trans_type code.
func transTypeLabel(t string) string {
	labels := map[string]string{
		"GR":         "Goods Receipt",
		"GR_CANCEL":  "GR Cancellation",
		"GI":         "Goods Issue",
		"GI_CANCEL":  "GI Cancellation",
		"DR":         "Purchase Receipt (PO)",
		"DR_CANCEL":  "Purchase Receipt Cancel",
		"DO":         "Sales Delivery",
		"DO_CANCEL":  "Sales Delivery Cancel",
		"WO_ISSUE":   "WO Component Issue",
		"WO_RECEIPT": "WO Completion Receipt",
		"WO_CANCEL":  "WO Cancellation",
	}
	if l, ok := labels[t]; ok {
		return l
	}
	return t
}

// round4 rounds a float to 4 decimal places.
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

// round3 rounds a float to 3 decimal places.
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

// groupAndAnnotate converts a flat slice of raw rows into category groups,
// computes subtotals, and calculates PctOfTotal for each row.
func groupAndAnnotate(rows []InvValuationRow, grandTotal float64) []InvValuationGroup {
	// Preserve insertion order of category names
	orderSeen := []string{}
	groupMap := map[string]*InvValuationGroup{}

	for _, r := range rows {
		cat := r.CategoryName
		if cat == "" {
			cat = "Uncategorized"
		}
		g, exists := groupMap[cat]
		if !exists {
			orderSeen = append(orderSeen, cat)
			groupMap[cat] = &InvValuationGroup{CategoryName: cat}
			g = groupMap[cat]
		}
		row := r
		row.CategoryName = cat
		if grandTotal > 0 {
			row.PctOfTotal = round4(row.TotalValue / grandTotal * 100)
		}
		g.Rows = append(g.Rows, row)
		g.SubTotalQty += row.QtyOnHand
		g.SubTotalValue += row.TotalValue
	}

	result := make([]InvValuationGroup, 0, len(orderSeen))
	for _, cat := range orderSeen {
		g := groupMap[cat]
		g.SubTotalQty = round3(g.SubTotalQty)
		g.SubTotalValue = round4(g.SubTotalValue)
		result = append(result, *g)
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Report A — GetInventoryValuationSnapshot
// ─────────────────────────────────────────────────────────────────────────────

// GetInventoryValuationSnapshot returns inventory valuation as of asOf.
//
//   - asOf="" or asOf=today → reads OITM (current, authoritative).
//   - asOf < today          → reconstructs from OIVL cumulative movements.
//   - whsCode=""            → all warehouses aggregated.
//   - itmGrpCod=0           → all item groups.
func GetInventoryValuationSnapshot(asOf, whsCode string, itmGrpCod int) (*InvValuationReport, error) {
	today := time.Now().Format("2006-01-02")
	if asOf == "" {
		asOf = today
	}

	report := &InvValuationReport{
		AsOf:        asOf,
		Warehouse:   whsCode,
		GroupCode:   itmGrpCod,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	var flatRows []InvValuationRow
	var err error

	if asOf >= today {
		// ── Path A: current stock from OITM / OITW ────────────────────────────
		report.IsHistorical = false
		if whsCode != "" {
			flatRows, err = currentValuationByWarehouse(whsCode, itmGrpCod)
		} else {
			flatRows, err = currentValuationAllWarehouses(itmGrpCod)
		}
	} else {
		// ── Path B: historical reconstruction from OIVL ───────────────────────
		report.IsHistorical = true
		flatRows, err = historicalValuation(asOf, whsCode, itmGrpCod)
	}
	if err != nil {
		return nil, err
	}

	// Compute grand total
	var grandTotal float64
	for _, r := range flatRows {
		grandTotal += r.TotalValue
	}
	grandTotal = round4(grandTotal)

	report.TotalValue = grandTotal
	for _, r := range flatRows {
		report.TotalQty += r.QtyOnHand
	}
	report.TotalQty = round3(report.TotalQty)

	report.Groups = groupAndAnnotate(flatRows, grandTotal)
	return report, nil
}

// currentValuationAllWarehouses queries OITM for company-wide totals.
func currentValuationAllWarehouses(itmGrpCod int) ([]InvValuationRow, error) {
	type rawRow struct {
		ItemCode     string
		ItemName     string
		CategoryName string
		UoM          string
		QtyOnHand    float64
		UnitCost     float64
		TotalValue   float64
	}

	var raws []rawRow
	q := db.DB.Raw(`
		SELECT
		    i.item_code,
		    i.item_name,
		    COALESCE(b.itms_grp_nam, 'Uncategorized') AS category_name,
		    i.invntry_uom                              AS uom,
		    ROUND(i.on_hand, 3)                        AS qty_on_hand,
		    ROUND(i.avg_price, 4)                      AS unit_cost,
		    ROUND(i.on_hand * i.avg_price, 4)          AS total_value
		FROM oitm i
		LEFT JOIN oitb b ON b.itms_grp_cod = i.itms_grp_cod
		WHERE i.valid_for = 'Y'
		  AND i.on_hand   > 0.001
		  AND (? = 0 OR i.itms_grp_cod = ?)
		ORDER BY category_name, i.item_name
	`, itmGrpCod, itmGrpCod)

	if err := q.Scan(&raws).Error; err != nil {
		return nil, fmt.Errorf("currentValuationAllWarehouses: %w", err)
	}

	rows := make([]InvValuationRow, len(raws))
	for i, r := range raws {
		rows[i] = InvValuationRow{
			ItemCode:     r.ItemCode,
			ItemName:     r.ItemName,
			CategoryName: r.CategoryName,
			Warehouse:    "",
			UoM:          r.UoM,
			QtyOnHand:    r.QtyOnHand,
			UnitCost:     r.UnitCost,
			TotalValue:   r.TotalValue,
		}
	}
	return rows, nil
}

// currentValuationByWarehouse queries OITW for a specific warehouse.
func currentValuationByWarehouse(whsCode string, itmGrpCod int) ([]InvValuationRow, error) {
	type rawRow struct {
		ItemCode     string
		ItemName     string
		CategoryName string
		Warehouse    string
		UoM          string
		QtyOnHand    float64
		UnitCost     float64
		TotalValue   float64
	}

	var raws []rawRow
	q := db.DB.Raw(`
		SELECT
		    w.item_code,
		    i.item_name,
		    COALESCE(b.itms_grp_nam, 'Uncategorized') AS category_name,
		    w.whs_code                                 AS warehouse,
		    i.invntry_uom                              AS uom,
		    ROUND(w.on_hand, 3)                        AS qty_on_hand,
		    ROUND(w.avg_price, 4)                      AS unit_cost,
		    ROUND(w.on_hand * w.avg_price, 4)          AS total_value
		FROM oitw w
		JOIN  oitm i ON i.item_code = w.item_code
		LEFT JOIN oitb b ON b.itms_grp_cod = i.itms_grp_cod
		WHERE i.valid_for = 'Y'
		  AND w.on_hand   > 0.001
		  AND (? = '' OR w.whs_code = ?)
		  AND (? = 0  OR i.itms_grp_cod = ?)
		ORDER BY category_name, i.item_name
	`, whsCode, whsCode, itmGrpCod, itmGrpCod)

	if err := q.Scan(&raws).Error; err != nil {
		return nil, fmt.Errorf("currentValuationByWarehouse: %w", err)
	}

	rows := make([]InvValuationRow, len(raws))
	for i, r := range raws {
		rows[i] = InvValuationRow{
			ItemCode:     r.ItemCode,
			ItemName:     r.ItemName,
			CategoryName: r.CategoryName,
			Warehouse:    r.Warehouse,
			UoM:          r.UoM,
			QtyOnHand:    r.QtyOnHand,
			UnitCost:     r.UnitCost,
			TotalValue:   r.TotalValue,
		}
	}
	return rows, nil
}

// historicalValuation reconstructs stock qty and value from OIVL up to asOf.
// unit_cost is derived as total_value / qty_on_hand (weighted average from ledger).
func historicalValuation(asOf, whsCode string, itmGrpCod int) ([]InvValuationRow, error) {
	type rawRow struct {
		ItemCode      string
		ItemName      string
		CategoryName  string
		Warehouse     string
		UoM           string
		QtyOnHand     float64
		InventoryValue float64
	}

	var raws []rawRow
	q := db.DB.Raw(`
		SELECT
		    v.item_code,
		    MAX(v.item_name)                                       AS item_name,
		    COALESCE(MAX(b.itms_grp_nam), 'Uncategorized')        AS category_name,
		    v.warehouse,
		    MAX(i.invntry_uom)                                     AS uom,
		    ROUND(SUM(v.in_qty)  - SUM(v.out_qty),  3)            AS qty_on_hand,
		    ROUND(SUM(v.in_qty * v.price) - SUM(v.out_qty * v.price), 4) AS inventory_value
		FROM oivl v
		JOIN  oitm i ON i.item_code = v.item_code
		LEFT JOIN oitb b ON b.itms_grp_cod = i.itms_grp_cod
		WHERE v.doc_date <= ?
		  AND (? = '' OR v.warehouse = ?)
		  AND (? = 0  OR i.itms_grp_cod = ?)
		GROUP BY v.item_code, v.warehouse
		HAVING qty_on_hand > 0.001
		ORDER BY category_name, item_name
	`, asOf, whsCode, whsCode, itmGrpCod, itmGrpCod)

	if err := q.Scan(&raws).Error; err != nil {
		return nil, fmt.Errorf("historicalValuation: %w", err)
	}

	rows := make([]InvValuationRow, len(raws))
	for i, r := range raws {
		unitCost := 0.0
		if r.QtyOnHand > 0 {
			unitCost = round4(r.InventoryValue / r.QtyOnHand)
		}
		rows[i] = InvValuationRow{
			ItemCode:     r.ItemCode,
			ItemName:     r.ItemName,
			CategoryName: r.CategoryName,
			Warehouse:    r.Warehouse,
			UoM:          r.UoM,
			QtyOnHand:    r.QtyOnHand,
			UnitCost:     unitCost,
			TotalValue:   round4(r.InventoryValue),
		}
	}
	return rows, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Report B — GetInventoryMovementReport
// ─────────────────────────────────────────────────────────────────────────────

// GetInventoryMovementReport returns OIVL ledger entries matching the filters.
//
//   - dateFrom / dateTo : required date range (YYYY-MM-DD)
//   - itemCode          : "" = all items; prefix search supported (e.g. "PALAY")
//   - whsCode           : "" = all warehouses
//   - transType         : "" = all types; exact match (e.g. "DR", "GR")
//
// Results are capped at 2 000 rows; Truncated=true if the cap was hit.
func GetInventoryMovementReport(dateFrom, dateTo, itemCode, whsCode, transType string) (*InvMovementReport, error) {
	today := time.Now().Format("2006-01-02")
	if dateFrom == "" {
		dateFrom = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if dateTo == "" {
		dateTo = today
	}

	// Build item_code match: allow prefix wildcard if no exact code given
	itemMatch := ""
	itemMatchArg := ""
	if itemCode != "" {
		itemMatch = "AND v.item_code LIKE ?"
		itemMatchArg = itemCode + "%"
	}

	// Build warehouse clause
	whsClause := ""
	if whsCode != "" {
		whsClause = "AND v.warehouse = ?"
	}

	// Build trans_type clause
	transClause := ""
	if transType != "" {
		transClause = "AND v.trans_type = ?"
	}

	// Collect variable args (GORM Raw requires positional args matching ? placeholders)
	args := []interface{}{dateFrom, dateTo}
	if itemCode != "" {
		args = append(args, itemMatchArg)
	}
	if whsCode != "" {
		args = append(args, whsCode)
	}
	if transType != "" {
		args = append(args, transType)
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
		    DATE_FORMAT(v.doc_date, '%%Y-%%m-%%d') AS doc_date,
		    v.trans_type,
		    v.doc_num,
		    v.item_code,
		    v.item_name,
		    v.warehouse,
		    v.in_qty,
		    v.out_qty,
		    v.price,
		    v.value
		FROM oivl v
		WHERE v.doc_date BETWEEN ? AND ?
		  %s
		  %s
		  %s
		ORDER BY v.doc_date ASC, v.doc_entry ASC
		LIMIT 2001
	`, itemMatch, whsClause, transClause)

	type rawRow struct {
		DocDate   string
		TransType string
		DocNum    int
		ItemCode  string
		ItemName  string
		Warehouse string
		InQty     float64
		OutQty    float64
		Price     float64
		Value     float64
	}

	var raws []rawRow
	if err := db.DB.Raw(sqlQuery, args...).Scan(&raws).Error; err != nil {
		return nil, fmt.Errorf("GetInventoryMovementReport: %w", err)
	}

	truncated := len(raws) > 2000
	if truncated {
		raws = raws[:2000]
	}

	report := &InvMovementReport{
		DateFrom:  dateFrom,
		DateTo:    dateTo,
		ItemCode:  itemCode,
		Warehouse: whsCode,
		TransType: transType,
		Truncated: truncated,
		Rows:      make([]InvMovementRow, len(raws)),
	}

	for i, r := range raws {
		report.Rows[i] = InvMovementRow{
			DocDate:    r.DocDate,
			TransType:  r.TransType,
			TransLabel: transTypeLabel(r.TransType),
			DocNum:     r.DocNum,
			ItemCode:   r.ItemCode,
			ItemName:   r.ItemName,
			Warehouse:  r.Warehouse,
			InQty:      r.InQty,
			OutQty:     r.OutQty,
			Price:      r.Price,
			Value:      r.Value,
		}
		report.TotalInQty += r.InQty
		report.TotalOutQty += r.OutQty
		report.TotalInValue += r.InQty * r.Price
		report.TotalOutValue += r.OutQty * r.Price
	}

	report.TotalInQty = round3(report.TotalInQty)
	report.TotalOutQty = round3(report.TotalOutQty)
	report.TotalInValue = round4(report.TotalInValue)
	report.TotalOutValue = round4(report.TotalOutValue)

	return report, nil
}
