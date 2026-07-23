// Package importer implements Excel (.xlsx) master-data import for Items,
// Customers, and Suppliers.
//
// Flow: WriteTemplate → operator fills the sheet → Preview (parse + validate,
// no writes) → Commit (transactional upsert). Rows are keyed by item_code
// (items) or name (customers/suppliers): existing records are updated with the
// non-empty cells provided, new records are created. Rows that fail validation
// are reported and skipped; they never abort the whole import.
package importer

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"ricemill/app/db"
	"ricemill/app/models"
	invsvc "ricemill/app/services/inventory"
)

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

// RowResult classifies one spreadsheet row during preview/commit.
type RowResult struct {
	RowNum int      `json:"row_num"` // 1-based Excel row number
	Key    string   `json:"key"`     // item_code or name
	Action string   `json:"action"`  // create | update | error
	Error  string   `json:"error,omitempty"`
	Values []string `json:"values"` // cells in template column order, for the preview table
}

// Preview is the dry-run result returned before committing.
type Preview struct {
	DataType    string      `json:"data_type"`
	FilePath    string      `json:"file_path"`
	FileName    string      `json:"file_name"`
	Columns     []string    `json:"columns"`
	Rows        []RowResult `json:"rows"`
	CreateCount int         `json:"create_count"`
	UpdateCount int         `json:"update_count"`
	ErrorCount  int         `json:"error_count"`
}

// Result summarizes a committed import.
type Result struct {
	DataType string   `json:"data_type"`
	Created  int      `json:"created"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"` // rows with validation errors
	Errors   []string `json:"errors,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Column specifications
// ─────────────────────────────────────────────────────────────────────────────

type colSpec struct {
	header   string // template header text
	required bool   // must be non-empty on CREATE (updates may leave blank = keep)
}

var specs = map[string][]colSpec{
	"items": {
		{"item_code", true}, {"item_name", true}, {"category", false},
		{"uom_group", false}, {"inventory_uom", false}, {"purchase_uom", false},
		{"sales_uom", false}, {"selling_price", false}, {"avg_cost", false},
		{"min_level", false}, {"max_level", false}, {"default_warehouse", false},
		{"sellable", false}, {"purchasable", false}, {"production_item", false},
		{"active", false}, {"description", false},
	},
	"customers": {
		{"name", true}, {"customer_type", false}, {"address", false},
		{"delivery_address", false}, {"contact_number", false},
		{"email", false}, {"notes", false},
	},
	"suppliers": {
		{"name", true}, {"supplier_type", false}, {"contact_person", false},
		{"contact_number", false}, {"email", false}, {"address", false},
		{"tin_number", false}, {"payment_terms", false}, {"notes", false},
	},
}

var sampleRows = map[string][][]string{
	"items": {
		{"PALAY-001", "Palay — Dinorado (wet)", "Palay", "", "kg", "kg", "kg", "", "18.50", "500", "20000", "WH01", "N", "Y", "N", "Y", "Freshly harvested paddy"},
		{"RICE-DIN-25", "Dinorado Premium 25kg", "Milled Rice", "", "kg", "sack", "sack", "1450", "", "50", "2000", "WH01", "Y", "N", "Y", "Y", ""},
	},
	"customers": {
		{"Aling Nena's Store", "Account", "123 Rizal St, Poblacion", "", "0917-123-4567", "nena@example.com", "Weekly buyer"},
		{"Walk-in Cash Sale", "Walk-in", "", "", "", "", ""},
	},
	"suppliers": {
		{"Farmer's Coop — San Isidro", "Regular", "Mang Tomas", "0918-765-4321", "coop@example.com", "San Isidro, Nueva Ecija", "123-456-789-000", "COD", "Palay supplier"},
	},
}

var templateNotes = map[string][]string{
	"items": {
		"item_code is the KEY: an existing code updates that item; a new code creates one.",
		"item_name is required when creating a new item.",
		"category is matched by name (case-insensitive) and CREATED automatically if it does not exist.",
		"inventory_uom / purchase_uom / sales_uom must be UoM codes that already exist in the system (see the dropdowns) — unknown codes are rejected.",
		"uom_group must be an existing UoM Group code. Blank purchase/sales UoM defaults to the inventory UoM on new items.",
		"selling_price fills Price List 1 (itm1). avg_cost is applied to NEW items only — the moving average of existing items is never overwritten.",
		"sellable / purchasable / production_item / active accept Y or N (blank = keep current / default).",
		"Stock quantities are NOT imported here — use Goods Receipts for opening stock so the ledger and GL stay consistent.",
	},
	"customers": {
		"name is the KEY: an existing customer (exact name) is updated; otherwise created.",
		"customer_type must be Account or Walk-in (blank = Walk-in for new customers).",
	},
	"suppliers": {
		"name is the KEY: an existing supplier (exact name) is updated; otherwise created.",
		"Imported suppliers are created as ACTIVE (approved) — import only vetted suppliers.",
		"payment_terms examples: COD, Net 7, Net 15, Net 30.",
	},
}

// DataTypeLabel returns a human label for audit/messages.
func DataTypeLabel(dataType string) string {
	switch dataType {
	case "items":
		return "Item Master"
	case "customers":
		return "Customers"
	case "suppliers":
		return "Suppliers"
	}
	return dataType
}

func specFor(dataType string) ([]colSpec, error) {
	s, ok := specs[dataType]
	if !ok {
		return nil, fmt.Errorf("unknown import type %q — must be items, customers, or suppliers", dataType)
	}
	return s, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Template
// ─────────────────────────────────────────────────────────────────────────────

// dropListRows is how many data rows get a validation dropdown in templates.
const dropListRows = 1000

// lookupLists pulls the reference lists used for Excel data validation from the
// database. Empty lists simply produce no dropdown for that column.
func lookupLists(dataType string) map[string][]string {
	lists := map[string][]string{}
	pluck := func(sql string) []string {
		var vals []string
		db.DB.Raw(sql).Scan(&vals)
		return vals
	}
	switch dataType {
	case "items":
		uoms := pluck(`SELECT uom_code FROM ouom ORDER BY uom_code`)
		lists["uom_group"] = pluck(`SELECT ugp_code FROM ougp ORDER BY ugp_code`)
		lists["inventory_uom"] = uoms
		lists["purchase_uom"] = uoms
		lists["sales_uom"] = uoms
		lists["category"] = pluck(`SELECT itms_grp_nam FROM oitb WHERE is_active = 1 ORDER BY itms_grp_nam`)
		lists["default_warehouse"] = pluck(`SELECT whs_code FROM owhs WHERE inactive = 'N' ORDER BY whs_code`)
		for _, yn := range []string{"sellable", "purchasable", "production_item", "active"} {
			lists[yn] = []string{"Y", "N"}
		}
	case "customers":
		lists["customer_type"] = []string{"Account", "Walk-in"}
	case "suppliers":
		lists["supplier_type"] = pluck(`SELECT value FROM purchasing_lookup WHERE category = 'supplier_type' AND is_active = 1 ORDER BY sort_order`)
		lists["payment_terms"] = pluck(`SELECT value FROM purchasing_lookup WHERE category = 'payment_terms' AND is_active = 1 ORDER BY sort_order`)
	}
	return lists
}

// WriteTemplate writes an .xlsx template for the given data type to destPath.
// Columns backed by a reference list (UoMs, UoM groups, categories, warehouses,
// Y/N flags, …) get an Excel data-validation dropdown sourced from a hidden
// "Lists" sheet, so operators pick valid values instead of typing them.
func WriteTemplate(dataType, destPath string) error {
	spec, err := specFor(dataType)
	if err != nil {
		return err
	}

	f := excelize.NewFile()
	defer f.Close()
	sheet := "Data"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4472C4"}},
	})
	for i, c := range spec {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		header := c.header
		if c.required {
			header += " *"
		}
		f.SetCellValue(sheet, cell, header)
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 20)
	}
	lastCol, _ := excelize.ColumnNumberToName(len(spec))
	f.SetCellStyle(sheet, "A1", lastCol+"1", headerStyle)

	for r, row := range sampleRows[dataType] {
		for i, v := range row {
			cell, _ := excelize.CoordinatesToCellName(i+1, r+2)
			f.SetCellValue(sheet, cell, v)
		}
	}

	// ── Data-validation dropdowns from a hidden Lists sheet ──────────────────
	// A hidden sheet holds one reference list per column; the Data sheet's
	// validations point at those ranges (SetSqrefDropList), which avoids the
	// 255-character limit of inline drop lists.
	lists := lookupLists(dataType)
	listSheet := "Lists"
	listCol := 0
	for specIdx, c := range spec {
		values := lists[c.header]
		if len(values) == 0 {
			continue
		}
		if listCol == 0 {
			if _, err := f.NewSheet(listSheet); err != nil {
				return err
			}
		}
		listCol++
		colName, _ := excelize.ColumnNumberToName(listCol)
		f.SetCellValue(listSheet, colName+"1", c.header)
		for i, v := range values {
			f.SetCellValue(listSheet, fmt.Sprintf("%s%d", colName, i+2), v)
		}

		dataCol, _ := excelize.ColumnNumberToName(specIdx + 1)
		dv := excelize.NewDataValidation(true)
		dv.Sqref = fmt.Sprintf("%s2:%s%d", dataCol, dataCol, dropListRows+1)
		dv.SetSqrefDropList(fmt.Sprintf("%s!$%s$2:$%s$%d", listSheet, colName, colName, len(values)+1))
		dv.SetError(excelize.DataValidationErrorStyleStop, "Invalid value",
			fmt.Sprintf("Pick a %s from the dropdown list.", c.header))
		if err := f.AddDataValidation(sheet, dv); err != nil {
			return fmt.Errorf("add %s dropdown: %w", c.header, err)
		}
	}
	if listCol > 0 {
		if err := f.SetSheetVisible(listSheet, false); err != nil {
			return err
		}
	}

	// Instructions sheet
	instr := "Instructions"
	f.NewSheet(instr)
	f.SetCellValue(instr, "A1", "How to use this template — "+DataTypeLabel(dataType))
	titleStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 13}})
	f.SetCellStyle(instr, "A1", "A1", titleStyle)
	f.SetColWidth(instr, "A", "A", 110)
	notes := append([]string{
		"Fill the Data sheet, one record per row, keeping the header row intact.",
		"The sample rows are examples — replace them with your data.",
		"Columns marked * are required for new records.",
	}, templateNotes[dataType]...)
	for i, n := range notes {
		f.SetCellValue(instr, fmt.Sprintf("A%d", i+3), "• "+n)
	}

	return f.SaveAs(destPath)
}

// ─────────────────────────────────────────────────────────────────────────────
// Parsing
// ─────────────────────────────────────────────────────────────────────────────

// normalizeHeader makes header matching tolerant: case, spaces, underscores,
// and the required-marker asterisk are all ignored.
func normalizeHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer(" ", "", "_", "", "*", "", "-", "").Replace(s)
	return s
}

// headerAliases maps legacy/alternate (normalized) header names onto spec headers,
// so older template files keep importing after a column rename.
var headerAliases = map[string]string{
	"uom":  "inventory_uom", // pre-UoM-columns items template
	"unit": "inventory_uom",
}

type parsedRow struct {
	rowNum int
	cells  map[string]string // spec header → trimmed cell value
	values []string          // in spec order, for preview display
}

// parseFile reads the first sheet and maps rows onto the column spec.
func parseFile(dataType, filePath string) ([]parsedRow, error) {
	spec, err := specFor(dataType)
	if err != nil {
		return nil, err
	}

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("open Excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("the workbook has no sheets")
	}
	// Prefer a sheet named "Data" (our template); otherwise use the first sheet.
	sheet := sheets[0]
	for _, s := range sheets {
		if normalizeHeader(s) == "data" {
			sheet = s
			break
		}
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read sheet %s: %w", sheet, err)
	}
	if len(rows) < 2 {
		return nil, errors.New("the sheet has no data rows below the header")
	}

	// Map header positions → spec headers.
	colIdx := map[string]int{} // spec header → column index
	for i, h := range rows[0] {
		n := normalizeHeader(h)
		if target, ok := headerAliases[n]; ok {
			n = normalizeHeader(target)
		}
		for _, c := range spec {
			if normalizeHeader(c.header) == n {
				colIdx[c.header] = i
				break
			}
		}
	}
	// The key column (first spec column) must be present.
	if _, ok := colIdx[spec[0].header]; !ok {
		return nil, fmt.Errorf("required column %q not found in the header row — download the template for the expected layout", spec[0].header)
	}

	var parsed []parsedRow
	for r := 1; r < len(rows); r++ {
		cells := map[string]string{}
		values := make([]string, len(spec))
		empty := true
		for i, c := range spec {
			if idx, ok := colIdx[c.header]; ok && idx < len(rows[r]) {
				v := strings.TrimSpace(rows[r][idx])
				cells[c.header] = v
				values[i] = v
				if v != "" {
					empty = false
				}
			}
		}
		if empty {
			continue // skip fully blank rows
		}
		parsed = append(parsed, parsedRow{rowNum: r + 1, cells: cells, values: values})
	}
	if len(parsed) == 0 {
		return nil, errors.New("the sheet has no data rows below the header")
	}
	return parsed, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Field parsing helpers
// ─────────────────────────────────────────────────────────────────────────────

func parseYN(v, field string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "":
		return "", nil
	case "Y", "YES", "TRUE", "1":
		return "Y", nil
	case "N", "NO", "FALSE", "0":
		return "N", nil
	}
	return "", fmt.Errorf("%s must be Y or N (got %q)", field, v)
}

func parseNum(v, field string) (float64, error) {
	if v == "" {
		return 0, nil
	}
	// Tolerate thousand separators and currency symbols users leave in cells.
	clean := strings.NewReplacer(",", "", "₱", "", " ", "").Replace(v)
	n, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number (got %q)", field, v)
	}
	if n < 0 {
		return 0, fmt.Errorf("%s cannot be negative", field)
	}
	return n, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Preview + Commit
// ─────────────────────────────────────────────────────────────────────────────

// PreviewImport parses and validates the file without writing anything.
func PreviewImport(dataType, filePath string) (*Preview, error) {
	spec, err := specFor(dataType)
	if err != nil {
		return nil, err
	}
	parsed, err := parseFile(dataType, filePath)
	if err != nil {
		return nil, err
	}

	p := &Preview{
		DataType: dataType,
		FilePath: filePath,
		FileName: filepath.Base(filePath),
	}
	for _, c := range spec {
		p.Columns = append(p.Columns, c.header)
	}

	seen := map[string]bool{}
	for _, row := range parsed {
		res := classifyRow(db.DB, dataType, row, seen)
		switch res.Action {
		case "create":
			p.CreateCount++
		case "update":
			p.UpdateCount++
		default:
			p.ErrorCount++
		}
		p.Rows = append(p.Rows, res)
	}
	return p, nil
}

// CommitImport applies the file inside one transaction. Valid rows are
// upserted; invalid rows are skipped and reported. The transaction only rolls
// back on unexpected DB errors, never on row-level validation problems.
func CommitImport(dataType, filePath string, userID *uint) (*Result, error) {
	if _, err := specFor(dataType); err != nil {
		return nil, err
	}
	parsed, err := parseFile(dataType, filePath)
	if err != nil {
		return nil, err
	}

	result := &Result{DataType: dataType}
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		seen := map[string]bool{}
		for _, row := range parsed {
			res := classifyRow(tx, dataType, row, seen)
			if res.Action == "error" {
				result.Skipped++
				result.Errors = append(result.Errors, fmt.Sprintf("row %d: %s", row.rowNum, res.Error))
				continue
			}
			if err := applyRow(tx, dataType, row, userID); err != nil {
				return fmt.Errorf("row %d (%s): %w", row.rowNum, res.Key, err)
			}
			if res.Action == "create" {
				result.Created++
			} else {
				result.Updated++
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// classifyRow validates one row and decides create/update/error. Uses conn for
// existence checks so Commit sees rows created earlier in its own transaction.
func classifyRow(conn *gorm.DB, dataType string, row parsedRow, seen map[string]bool) RowResult {
	spec := specs[dataType]
	key := row.cells[spec[0].header]
	res := RowResult{RowNum: row.rowNum, Key: key, Values: row.values}

	fail := func(msg string) RowResult {
		res.Action = "error"
		res.Error = msg
		return res
	}

	if key == "" {
		return fail(spec[0].header + " is required")
	}
	dupKey := strings.ToLower(key)
	if seen[dupKey] {
		return fail(fmt.Sprintf("duplicate %s %q earlier in this file", spec[0].header, key))
	}
	seen[dupKey] = true

	exists := false
	switch dataType {
	case "items":
		var n int64
		conn.Model(&models.OITM{}).Where("item_code = ?", key).Count(&n)
		exists = n > 0
		if !exists && row.cells["item_name"] == "" {
			return fail("item_name is required for new items")
		}
		for _, f := range []string{"selling_price", "avg_cost", "min_level", "max_level"} {
			if _, err := parseNum(row.cells[f], f); err != nil {
				return fail(err.Error())
			}
		}
		for _, f := range []string{"sellable", "purchasable", "production_item", "active"} {
			if _, err := parseYN(row.cells[f], f); err != nil {
				return fail(err.Error())
			}
		}
		// UoM codes and the UoM group must already exist — they carry FK entry
		// IDs that drive conversions, so they cannot be auto-created from text.
		for _, f := range []string{"inventory_uom", "purchase_uom", "sales_uom"} {
			if v := row.cells[f]; v != "" && findUoM(conn, v) == nil {
				return fail(fmt.Sprintf("%s %q is not a defined unit of measure — add it under Inventory → UoM first", f, v))
			}
		}
		if g := row.cells["uom_group"]; g != "" && findUoMGroup(conn, g) == nil {
			return fail(fmt.Sprintf("uom_group %q is not a defined UoM group — add it under Inventory → UoM first", g))
		}
	case "customers":
		var n int64
		conn.Model(&models.Customer{}).Where("name = ?", key).Count(&n)
		exists = n > 0
		if t := row.cells["customer_type"]; t != "" && t != "Account" && t != "Walk-in" {
			return fail(fmt.Sprintf("customer_type must be Account or Walk-in (got %q)", t))
		}
	case "suppliers":
		var n int64
		conn.Model(&models.Supplier{}).Where("name = ?", key).Count(&n)
		exists = n > 0
	}

	if exists {
		res.Action = "update"
	} else {
		res.Action = "create"
	}
	return res
}

// applyRow upserts one validated row. Validation already ran in classifyRow.
func applyRow(tx *gorm.DB, dataType string, row parsedRow, userID *uint) error {
	switch dataType {
	case "items":
		return applyItem(tx, row.cells)
	case "customers":
		return applyCustomer(tx, row.cells, userID)
	case "suppliers":
		return applySupplier(tx, row.cells)
	}
	return fmt.Errorf("unknown data type %q", dataType)
}

// findUoM looks up a UoM master row by code (case-insensitive). Nil = not found.
func findUoM(conn *gorm.DB, code string) *models.UoMMaster {
	if code == "" {
		return nil
	}
	var u models.UoMMaster
	if err := conn.Where("LOWER(uom_code) = LOWER(?)", code).First(&u).Error; err != nil {
		return nil
	}
	return &u
}

// findUoMGroup looks up a UoM group by code or name (case-insensitive). Nil = not found.
func findUoMGroup(conn *gorm.DB, code string) *models.UoMGroup {
	if code == "" {
		return nil
	}
	var g models.UoMGroup
	if err := conn.Where("LOWER(ugp_code) = LOWER(?) OR LOWER(ugp_name) = LOWER(?)", code, code).First(&g).Error; err != nil {
		return nil
	}
	return &g
}

// resolveCategory finds an OITB by name (case-insensitive) or creates it.
func resolveCategory(tx *gorm.DB, name string) (int, error) {
	var cat models.OITB
	if err := tx.Where("LOWER(itms_grp_nam) = LOWER(?)", name).First(&cat).Error; err == nil {
		return cat.ItmsGrpCod, nil
	}
	cat = models.OITB{
		ItmsGrpNam: name, IsActive: true,
		ForSales: true, ForPurchasing: true, ForInventory: true, ForProduction: true,
	}
	if err := tx.Create(&cat).Error; err != nil {
		return 0, fmt.Errorf("create category %q: %w", name, err)
	}
	return cat.ItmsGrpCod, nil
}

func applyItem(tx *gorm.DB, c map[string]string) error {
	sellingPrice, _ := parseNum(c["selling_price"], "selling_price")
	avgCost, _ := parseNum(c["avg_cost"], "avg_cost")
	minLevel, _ := parseNum(c["min_level"], "min_level")
	maxLevel, _ := parseNum(c["max_level"], "max_level")
	sellable, _ := parseYN(c["sellable"], "sellable")
	purchasable, _ := parseYN(c["purchasable"], "purchasable")
	makItem, _ := parseYN(c["production_item"], "production_item")
	active, _ := parseYN(c["active"], "active")

	grpCod := 0
	if c["category"] != "" {
		var err error
		grpCod, err = resolveCategory(tx, c["category"])
		if err != nil {
			return err
		}
	}

	// UoM resolution — classifyRow already rejected unknown codes, so a non-nil
	// cell value here always resolves.
	invUoM := findUoM(tx, c["inventory_uom"])
	purUoM := findUoM(tx, c["purchase_uom"])
	salUoM := findUoM(tx, c["sales_uom"])
	uomGrp := findUoMGroup(tx, c["uom_group"])

	var item models.OITM
	err := tx.Where("item_code = ?", c["item_code"]).First(&item).Error
	isNew := errors.Is(err, gorm.ErrRecordNotFound)
	if err != nil && !isNew {
		return err
	}

	if isNew {
		item = models.OITM{
			ItemCode:   c["item_code"],
			ItemName:   c["item_name"],
			ItmsGrpCod: grpCod,
			InvntryUom: "kg", // fallback when no inventory UoM given
			// avg_cost is only honoured on CREATE — the moving average of an
			// existing item is system-maintained and must not be overwritten.
			AvgPrice:    avgCost,
			MinLevel:    minLevel,
			MaxLevel:    maxLevel,
			DfltWh:      c["default_warehouse"],
			Description: c["description"],
			InvntItem:   "Y",
			SellItem:    orDefault(sellable, "Y"),
			PrchseItem:  orDefault(purchasable, "Y"),
			MakItem:     orDefault(makItem, "N"),
			ValidFor:    orDefault(active, "Y"),
		}
		if invUoM == nil {
			// No inventory UoM in the row: try to attach the entry for the
			// default code so conversions still work.
			invUoM = findUoM(tx, item.InvntryUom)
		}
		if invUoM != nil {
			item.InvntryUom = invUoM.UomCode
			item.IUoMEntry = invUoM.UomEntry
		}
		// Blank purchase/sales UoM defaults to the inventory UoM on new items.
		if purUoM == nil {
			purUoM = invUoM
		}
		if salUoM == nil {
			salUoM = invUoM
		}
		if purUoM != nil {
			item.PurchaseUnit = purUoM.UomCode
			item.PUoMEntry = purUoM.UomEntry
		}
		if salUoM != nil {
			item.SalesUnit = salUoM.UomCode
			item.SUoMEntry = salUoM.UomEntry
		}
		if uomGrp != nil {
			item.UgpEntry = uomGrp.UgpEntry
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
	} else {
		// Update only the fields the row actually provides — blank cells keep
		// the current value.
		updates := map[string]interface{}{}
		if c["item_name"] != "" {
			updates["item_name"] = c["item_name"]
		}
		if grpCod != 0 {
			updates["itms_grp_cod"] = grpCod
		}
		if invUoM != nil {
			updates["invntry_uom"] = invUoM.UomCode
			updates["i_uom_entry"] = invUoM.UomEntry
		}
		if purUoM != nil {
			updates["purchase_unit"] = purUoM.UomCode
			updates["p_uom_entry"] = purUoM.UomEntry
		}
		if salUoM != nil {
			updates["sales_unit"] = salUoM.UomCode
			updates["s_uom_entry"] = salUoM.UomEntry
		}
		if uomGrp != nil {
			updates["ugp_entry"] = uomGrp.UgpEntry
		}
		if c["min_level"] != "" {
			updates["min_level"] = minLevel
		}
		if c["max_level"] != "" {
			updates["max_level"] = maxLevel
		}
		if c["default_warehouse"] != "" {
			updates["dflt_wh"] = c["default_warehouse"]
		}
		if c["description"] != "" {
			updates["description"] = c["description"]
		}
		if sellable != "" {
			updates["sell_item"] = sellable
		}
		if purchasable != "" {
			updates["prchse_item"] = purchasable
		}
		if makItem != "" {
			updates["mak_item"] = makItem
		}
		if active != "" {
			updates["valid_for"] = active
		}
		if len(updates) > 0 {
			if err := tx.Model(&models.OITM{}).Where("item_code = ?", c["item_code"]).
				Updates(updates).Error; err != nil {
				return err
			}
		}
	}

	// Mirror the UI create/update path: keep the ITM12 (UoM-per-item) mapping
	// rows in sync with the item's I/S/P UoM entries.
	var fresh models.OITM
	if err := tx.Where("item_code = ?", c["item_code"]).First(&fresh).Error; err == nil {
		invsvc.UpsertITM12ForItemTx(tx, fresh.ItemCode, fresh.IUoMEntry, fresh.SUoMEntry, fresh.PUoMEntry)
	}

	// selling_price → Price List 1 (itm1) upsert.
	if c["selling_price"] != "" && sellingPrice > 0 {
		var itm1 models.ITM1
		err := tx.Where("item_code = ? AND price_list = 1", c["item_code"]).First(&itm1).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&models.ITM1{ItemCode: c["item_code"], PriceList: 1, Price: sellingPrice}).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&models.ITM1{}).
			Where("item_code = ? AND price_list = 1", c["item_code"]).
			Update("price", sellingPrice).Error
	}
	return nil
}

func applyCustomer(tx *gorm.DB, c map[string]string, userID *uint) error {
	var cust models.Customer
	err := tx.Where("name = ?", c["name"]).First(&cust).Error
	isNew := errors.Is(err, gorm.ErrRecordNotFound)
	if err != nil && !isNew {
		return err
	}
	if isNew {
		cust = models.Customer{
			Name:            c["name"],
			Address:         c["address"],
			DeliveryAddress: c["delivery_address"],
			ContactNumber:   c["contact_number"],
			Email:           c["email"],
			CustomerType:    orDefault(c["customer_type"], "Walk-in"),
			Notes:           c["notes"],
			IsActive:        true,
			CreatedByID:     userID,
		}
		return tx.Create(&cust).Error
	}
	updates := map[string]interface{}{}
	for field, col := range map[string]string{
		"customer_type": "customer_type", "address": "address",
		"delivery_address": "delivery_address", "contact_number": "contact_number",
		"email": "email", "notes": "notes",
	} {
		if c[field] != "" {
			updates[col] = c[field]
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return tx.Model(&models.Customer{}).Where("id = ?", cust.ID).Updates(updates).Error
}

func applySupplier(tx *gorm.DB, c map[string]string) error {
	var sup models.Supplier
	err := tx.Where("name = ?", c["name"]).First(&sup).Error
	isNew := errors.Is(err, gorm.ErrRecordNotFound)
	if err != nil && !isNew {
		return err
	}
	if isNew {
		sup = models.Supplier{
			Name:          c["name"],
			SupplierType:  orDefault(c["supplier_type"], "Regular"),
			ContactPerson: c["contact_person"],
			ContactNumber: c["contact_number"],
			Email:         c["email"],
			Address:       c["address"],
			TINNumber:     c["tin_number"],
			PaymentTerms:  orDefault(c["payment_terms"], "COD"),
			Notes:         c["notes"],
			IsActive:      true,
			// Master-data import implies the supplier is already vetted.
			Status: "active",
		}
		return tx.Create(&sup).Error
	}
	updates := map[string]interface{}{}
	for field, col := range map[string]string{
		"supplier_type": "supplier_type", "contact_person": "contact_person",
		"contact_number": "contact_number", "email": "email", "address": "address",
		"tin_number": "tin_number", "payment_terms": "payment_terms", "notes": "notes",
	} {
		if c[field] != "" {
			updates[col] = c[field]
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return tx.Model(&models.Supplier{}).Where("id = ?", sup.ID).Updates(updates).Error
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
