package importer

// Integration tests for the Excel master-data importer.
// Each test writes a real .xlsx into t.TempDir() via excelize, then runs the
// same Preview/Commit path the UI uses against in-memory SQLite.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"ricemill/app/db"
	"ricemill/app/testutil"
)

// writeXLSX builds a workbook with one "Data" sheet from headers + rows.
func writeXLSX(t *testing.T, headers []string, rows [][]string) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", "Data")
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue("Data", cell, h))
	}
	for r, row := range rows {
		for i, v := range row {
			cell, _ := excelize.CoordinatesToCellName(i+1, r+2)
			require.NoError(t, f.SetCellValue("Data", cell, v))
		}
	}
	path := filepath.Join(t.TempDir(), "import.xlsx")
	require.NoError(t, f.SaveAs(path))
	return path
}

func count(t *testing.T, query string, args ...interface{}) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.DB.Raw(query, args...).Scan(&n).Error)
	return n
}

// ─────────────────────────────────────────────────────────────────────────────
// Items
// ─────────────────────────────────────────────────────────────────────────────

func TestImportItems_CreateAndUpdate(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedUoM(t, "kg", "Kilogram") // "uom" cells below must resolve
	// Pre-existing item: the import must UPDATE it and must NOT touch avg_price.
	testutil.SeedOITM(t, "RICE-OLD", "Old Name", 33.33)

	path := writeXLSX(t,
		[]string{"item_code *", "item_name *", "category", "uom", "selling_price", "avg_cost", "sellable", "production_item"},
		[][]string{
			{"PALAY-001", "Palay Dinorado", "Palay", "kg", "", "18.50", "N", "N"},
			{"RICE-DIN", "Dinorado Premium", "Milled Rice", "kg", "52.00", "", "Y", "Y"},
			{"RICE-OLD", "Renamed Rice", "", "", "48.00", "99.99", "", ""},
		})

	preview, err := PreviewImport("items", path)
	require.NoError(t, err)
	require.Equal(t, 2, preview.CreateCount)
	require.Equal(t, 1, preview.UpdateCount)
	require.Zero(t, preview.ErrorCount)

	res, err := CommitImport("items", path, nil)
	require.NoError(t, err)
	require.Equal(t, 2, res.Created)
	require.Equal(t, 1, res.Updated)
	require.Zero(t, res.Skipped)

	// New item created with avg cost + flags.
	var avg float64
	require.NoError(t, db.DB.Raw(`SELECT avg_price FROM oitm WHERE item_code = 'PALAY-001'`).Scan(&avg).Error)
	require.Equal(t, 18.5, avg)
	require.Equal(t, int64(1), count(t, `SELECT COUNT(*) FROM oitm WHERE item_code = 'PALAY-001' AND sell_item = 'N'`))
	require.Equal(t, int64(1), count(t, `SELECT COUNT(*) FROM oitm WHERE item_code = 'RICE-DIN' AND mak_item = 'Y'`))

	// Categories auto-created by name.
	require.Equal(t, int64(1), count(t, `SELECT COUNT(*) FROM oitb WHERE itms_grp_nam = 'Palay'`))
	require.Equal(t, int64(1), count(t, `SELECT COUNT(*) FROM oitb WHERE itms_grp_nam = 'Milled Rice'`))

	// selling_price landed in Price List 1.
	var price float64
	require.NoError(t, db.DB.Raw(`SELECT price FROM itm1 WHERE item_code = 'RICE-DIN' AND price_list = 1`).Scan(&price).Error)
	require.Equal(t, 52.0, price)

	// Existing item renamed, but its moving average is untouched (99.99 ignored).
	var name string
	require.NoError(t, db.DB.Raw(`SELECT item_name FROM oitm WHERE item_code = 'RICE-OLD'`).Scan(&name).Error)
	require.Equal(t, "Renamed Rice", name)
	require.NoError(t, db.DB.Raw(`SELECT avg_price FROM oitm WHERE item_code = 'RICE-OLD'`).Scan(&avg).Error)
	require.InDelta(t, 33.33, avg, 0.001, "avg_cost must never overwrite an existing moving average")
}

func TestImportItems_InvalidRowsSkippedNotFatal(t *testing.T) {
	testutil.SetupDB(t)

	path := writeXLSX(t,
		[]string{"item_code", "item_name", "selling_price", "sellable"},
		[][]string{
			{"GOOD-1", "Valid Item", "10", "Y"},
			{"", "Missing key", "", ""},              // no item_code
			{"BAD-NUM", "Bad price", "abc", ""},      // unparseable number
			{"BAD-FLAG", "Bad flag", "", "maybe"},    // invalid Y/N
			{"GOOD-1", "Duplicate", "", ""},          // duplicate key in file
			{"NEW-NO-NAME", "", "", ""},              // new item without a name
		})

	preview, err := PreviewImport("items", path)
	require.NoError(t, err)
	require.Equal(t, 1, preview.CreateCount)
	require.Equal(t, 5, preview.ErrorCount)

	res, err := CommitImport("items", path, nil)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)
	require.Equal(t, 5, res.Skipped)
	require.Len(t, res.Errors, 5)

	require.Equal(t, int64(1), count(t, `SELECT COUNT(*) FROM oitm WHERE item_code = 'GOOD-1'`))
	require.Equal(t, int64(1), count(t, `SELECT COUNT(*) FROM oitm`), "only the valid row imported")
}

func TestImportItems_MissingKeyColumnRejected(t *testing.T) {
	testutil.SetupDB(t)
	path := writeXLSX(t,
		[]string{"name", "price"}, // no item_code column at all
		[][]string{{"Something", "10"}})
	_, err := PreviewImport("items", path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "item_code")
}

// ─────────────────────────────────────────────────────────────────────────────
// UoM columns: resolution, defaults, ITM12 sync, validation, template dropdowns
// ─────────────────────────────────────────────────────────────────────────────

func TestImportItems_UoMColumnsResolveEntries(t *testing.T) {
	testutil.SetupDB(t)
	kg := testutil.SeedUoM(t, "kg", "Kilogram")
	sack := testutil.SeedUoM(t, "sack", "Sack 50kg")
	grp := testutil.SeedUoMGroup(t, "WEIGHT", "Weight Group", kg)

	path := writeXLSX(t,
		[]string{"item_code", "item_name", "uom_group", "inventory_uom", "purchase_uom", "sales_uom"},
		[][]string{
			{"RICE-A", "Rice A", "WEIGHT", "kg", "sack", "sack"},
			{"RICE-B", "Rice B", "", "KG", "", ""}, // case-insensitive; blank P/S default to inventory
		})

	res, err := CommitImport("items", path, nil)
	require.NoError(t, err)
	require.Equal(t, 2, res.Created)
	require.Zero(t, res.Skipped)

	type uomState struct {
		InvntryUom   string
		IUomEntry    uint
		PurchaseUnit string
		PUomEntry    uint
		SalesUnit    string
		SUomEntry    uint
		UgpEntry     uint
	}
	var a uomState
	require.NoError(t, db.DB.Raw(`SELECT invntry_uom, i_uom_entry, purchase_unit, p_uom_entry,
		sales_unit, s_uom_entry, ugp_entry FROM oitm WHERE item_code = 'RICE-A'`).Scan(&a).Error)
	require.Equal(t, "kg", a.InvntryUom)
	require.Equal(t, kg, a.IUomEntry)
	require.Equal(t, "sack", a.PurchaseUnit)
	require.Equal(t, sack, a.PUomEntry)
	require.Equal(t, "sack", a.SalesUnit)
	require.Equal(t, sack, a.SUomEntry)
	require.Equal(t, grp, a.UgpEntry)

	var b uomState
	require.NoError(t, db.DB.Raw(`SELECT invntry_uom, i_uom_entry, purchase_unit, p_uom_entry,
		sales_unit, s_uom_entry, ugp_entry FROM oitm WHERE item_code = 'RICE-B'`).Scan(&b).Error)
	require.Equal(t, "kg", b.InvntryUom, "matched case-insensitively to the master code")
	require.Equal(t, kg, b.IUomEntry)
	require.Equal(t, kg, b.PUomEntry, "blank purchase UoM defaults to inventory UoM")
	require.Equal(t, kg, b.SUomEntry, "blank sales UoM defaults to inventory UoM")

	// ITM12 mapping synced like the UI create path (I + S + P roles).
	require.Equal(t, int64(2), count(t, `SELECT COUNT(*) FROM itm12 WHERE item_code = 'RICE-A'`),
		"kg(I) + sack(S/P collapse to one row per uom_entry)")
	require.Equal(t, int64(1), count(t, `SELECT COUNT(*) FROM itm12 WHERE item_code = 'RICE-B'`))
}

func TestImportItems_UnknownUoMRejected(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedUoM(t, "kg", "Kilogram")

	path := writeXLSX(t,
		[]string{"item_code", "item_name", "inventory_uom", "uom_group"},
		[][]string{
			{"OK-1", "Fine", "kg", ""},
			{"BAD-UOM", "Bad UoM", "bucket", ""},   // no such uom
			{"BAD-GRP", "Bad Group", "kg", "XXXX"}, // no such group
		})

	preview, err := PreviewImport("items", path)
	require.NoError(t, err)
	require.Equal(t, 1, preview.CreateCount)
	require.Equal(t, 2, preview.ErrorCount)

	res, err := CommitImport("items", path, nil)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)
	require.Equal(t, 2, res.Skipped)
	require.Equal(t, int64(0), count(t, `SELECT COUNT(*) FROM oitm WHERE item_code IN ('BAD-UOM','BAD-GRP')`))
}

func TestImportItems_LegacyUomHeaderStillWorks(t *testing.T) {
	testutil.SetupDB(t)
	kg := testutil.SeedUoM(t, "kg", "Kilogram")

	// Old items template used a single "uom" column — it must map to inventory_uom.
	path := writeXLSX(t,
		[]string{"item_code", "item_name", "uom"},
		[][]string{{"OLD-TPL", "From old template", "kg"}})

	res, err := CommitImport("items", path, nil)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)

	var entry uint
	require.NoError(t, db.DB.Raw(`SELECT i_uom_entry FROM oitm WHERE item_code = 'OLD-TPL'`).Scan(&entry).Error)
	require.Equal(t, kg, entry)
}

func TestWriteTemplate_ItemsHasUoMDropdownsFromDB(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedUoM(t, "kg", "Kilogram")
	testutil.SeedUoM(t, "sack", "Sack 50kg")
	testutil.SeedUoMGroup(t, "WEIGHT", "Weight Group", 1)
	testutil.SeedOITB(t, "Milled Rice", true)
	require.NoError(t, db.DB.Exec(`INSERT INTO owhs (whs_code, whs_name) VALUES ('WH01', 'Main')`).Error)

	path := filepath.Join(t.TempDir(), "items.xlsx")
	require.NoError(t, WriteTemplate("items", path))

	f, err := excelize.OpenFile(path)
	require.NoError(t, err)
	defer f.Close()

	// Hidden Lists sheet carries the DB-sourced reference values.
	require.Contains(t, f.GetSheetList(), "Lists")
	visible, err := f.GetSheetVisible("Lists")
	require.NoError(t, err)
	require.False(t, visible, "Lists sheet must be hidden")

	rows, err := f.GetRows("Lists")
	require.NoError(t, err)
	flat := ""
	for _, r := range rows {
		for _, v := range r {
			flat += v + "|"
		}
	}
	require.Contains(t, flat, "kg")
	require.Contains(t, flat, "sack")
	require.Contains(t, flat, "WEIGHT")
	require.Contains(t, flat, "Milled Rice")
	require.Contains(t, flat, "WH01")

	// Data sheet carries dropdown validations pointing at the Lists sheet.
	dvs, err := f.GetDataValidations("Data")
	require.NoError(t, err)
	require.NotEmpty(t, dvs, "items template must have data-validation dropdowns")
	foundListRef := false
	for _, dv := range dvs {
		if dv.Formula1 != "" && strings.Contains(dv.Formula1, "Lists!") {
			foundListRef = true
		}
	}
	require.True(t, foundListRef, "at least one dropdown must reference the Lists sheet")
}

// ─────────────────────────────────────────────────────────────────────────────
// Customers / Suppliers
// ─────────────────────────────────────────────────────────────────────────────

func TestImportCustomers_CreateUpdateAndTypeValidation(t *testing.T) {
	testutil.SetupDB(t)
	require.NoError(t, db.DB.Exec(
		`INSERT INTO customer (name, address, customer_type, is_active) VALUES ('Aling Nena', 'Old Addr', 'Walk-in', 1)`).Error)

	path := writeXLSX(t,
		[]string{"name", "customer_type", "address", "contact_number"},
		[][]string{
			{"Aling Nena", "Account", "New Addr", "0917-000-1111"}, // update
			{"New Buyer", "", "", ""},                              // create, defaults to Walk-in
			{"Bad Type", "VIP", "", ""},                            // invalid type
		})

	preview, err := PreviewImport("customers", path)
	require.NoError(t, err)
	require.Equal(t, 1, preview.CreateCount)
	require.Equal(t, 1, preview.UpdateCount)
	require.Equal(t, 1, preview.ErrorCount)

	res, err := CommitImport("customers", path, nil)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)
	require.Equal(t, 1, res.Updated)
	require.Equal(t, 1, res.Skipped)

	require.Equal(t, int64(1), count(t,
		`SELECT COUNT(*) FROM customer WHERE name = 'Aling Nena' AND customer_type = 'Account' AND address = 'New Addr'`))
	require.Equal(t, int64(1), count(t,
		`SELECT COUNT(*) FROM customer WHERE name = 'New Buyer' AND customer_type = 'Walk-in'`))
	require.Equal(t, int64(0), count(t, `SELECT COUNT(*) FROM customer WHERE name = 'Bad Type'`))
}

func TestImportSuppliers_CreatedActive(t *testing.T) {
	testutil.SetupDB(t)

	path := writeXLSX(t,
		[]string{"name", "supplier_type", "payment_terms", "tin_number"},
		[][]string{{"Farmer's Coop", "Regular", "Net 15", "123-456-789-000"}})

	res, err := CommitImport("suppliers", path, nil)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)

	require.Equal(t, int64(1), count(t,
		`SELECT COUNT(*) FROM supplier WHERE name = 'Farmer''s Coop' AND status = 'active' AND payment_terms = 'Net 15'`))
}

// ─────────────────────────────────────────────────────────────────────────────
// Template round-trip
// ─────────────────────────────────────────────────────────────────────────────

func TestWriteTemplate_RoundTripsThroughPreview(t *testing.T) {
	testutil.SetupDB(t)
	// The items sample rows reference the standard rice-mill UoMs.
	testutil.SeedUoM(t, "kg", "Kilogram")
	testutil.SeedUoM(t, "sack", "Sack 50kg")
	for _, dataType := range []string{"items", "customers", "suppliers"} {
		path := filepath.Join(t.TempDir(), dataType+".xlsx")
		require.NoError(t, WriteTemplate(dataType, path))

		// The generated template (with its sample rows) must parse cleanly.
		preview, err := PreviewImport(dataType, path)
		require.NoError(t, err, dataType)
		require.Zero(t, preview.ErrorCount, "template sample rows must be valid (%s)", dataType)
		require.Positive(t, preview.CreateCount, dataType)
	}
	// Unknown type is rejected.
	require.Error(t, WriteTemplate("nonsense", filepath.Join(t.TempDir(), "x.xlsx")))
}
