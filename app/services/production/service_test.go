package production

// Integration tests for the Production/Milling lifecycle.
//
// DB strategy: each test calls testutil.SetupDB, which opens a fresh in-memory
// SQLite instance with the production tables (owor/wor1/oitt/itt1) plus inventory
// movement/ledger tables (oign/ige1/oitw/oivl). Unlike the sales tests, these DO
// seed OITM rows and exercise the full stock + GL posting path, so they cover the
// exact atomicity / costing / GL bugs called out in the Part 1 code audit.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ricemill/app/db"
	"ricemill/app/testutil"
	"ricemill/app/models"
)

// jeSumDebit / jeSumCredit total every posted journal-entry line.
func jeSumDebit(t *testing.T) float64 {
	t.Helper()
	var v float64
	require.NoError(t, db.DB.Raw(`SELECT COALESCE(SUM(debit),0) FROM je_line`).Scan(&v).Error)
	return v
}
func jeSumCredit(t *testing.T) float64 {
	t.Helper()
	var v float64
	require.NoError(t, db.DB.Raw(`SELECT COALESCE(SUM(credit),0) FROM je_line`).Scan(&v).Error)
	return v
}
func onHand(t *testing.T, itemCode string) float64 {
	t.Helper()
	var v float64
	require.NoError(t, db.DB.Raw(`SELECT on_hand FROM oitm WHERE item_code = ?`, itemCode).Scan(&v).Error)
	return v
}
func statusOf(t *testing.T, docEntry uint) string {
	t.Helper()
	var v string
	require.NoError(t, db.DB.Raw(`SELECT status FROM owor WHERE doc_entry = ?`, docEntry).Scan(&v).Error)
	return v
}
func countRows(t *testing.T, table, where string, args ...interface{}) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.DB.Raw(`SELECT COUNT(*) FROM `+table+` WHERE `+where, args...).Scan(&n).Error)
	return n
}

// ─────────────────────────────────────────────────────────────────────────────
// Full milling cycle: balanced GL, correct stock movement
// ─────────────────────────────────────────────────────────────────────────────

func TestMilling_FullCycle_PostsBalancedGLAndMovesStock(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedProductionAccounting(t)
	grp := testutil.SeedOITB(t, "Grain", false)
	paddy := testutil.SeedProdItem(t, "PADDY", "Palay", 20.0, 1000, false, grp)
	rice := testutil.SeedProdItem(t, "RICE", "Milled Rice", 0, 0, false, grp)
	bran := testutil.SeedProdItem(t, "BRAN", "Rice Bran", 0, 0, false, grp)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "Draft", mo.Status)

	require.NoError(t, StartMillingOrder(mo.ID, nil))
	require.Equal(t, 900.0, onHand(t, "PADDY"), "input consumed by goods issue")

	res, err := CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		CompletionDate: "2026-01-06",
		Lines: []CompleteMillingLineInput{
			{OutputItemID: rice, ActualQty: 65},
			{OutputItemID: bran, ActualQty: 25},
		},
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "Completed", res.Order.Status)

	require.Equal(t, 65.0, onHand(t, "RICE"))
	require.Equal(t, 25.0, onHand(t, "BRAN"))

	// Start JE (Dr WIP 2000 / Cr Inventory 2000) + Complete JE (Dr Inventory 2000 /
	// Cr WIP 2000): total debits == total credits == 4000, and both balance.
	require.InDelta(t, jeSumDebit(t), jeSumCredit(t), 0.01)
	require.InDelta(t, 4000.0, jeSumDebit(t), 0.01)
}

// ─────────────────────────────────────────────────────────────────────────────
// Finding #1/#2: Start is atomic — a missing GL account leaves NO side effects
// ─────────────────────────────────────────────────────────────────────────────

func TestMilling_StartIsAtomic_WhenInputAccountMissing(t *testing.T) {
	testutil.SetupDB(t)
	// Seed only WIP; deliberately omit MILLING_INPUT so Start's pre-flight fails.
	wip := testutil.GLAcct(t, "1-1350", "WIP", "ASSET", "DEBIT")
	testutil.AcctDet(t, "PRODUCTION", "MILLING_WIP", nil, wip)

	grp := testutil.SeedOITB(t, "Grain", false)
	paddy := testutil.SeedProdItem(t, "PADDY", "Palay", 20.0, 1000, false, grp)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)

	err = StartMillingOrder(mo.ID, nil)
	require.Error(t, err)

	// No stock consumed, no goods issue, order still Draft.
	require.Equal(t, 1000.0, onHand(t, "PADDY"))
	require.Zero(t, countRows(t, "oige", "1=1"))
	require.Equal(t, "P", statusOf(t, mo.ID))
}

// ─────────────────────────────────────────────────────────────────────────────
// Finding #3: starting an already-started order does not double-issue stock
// ─────────────────────────────────────────────────────────────────────────────

func TestMilling_StartTwice_DoesNotDoubleIssue(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedProductionAccounting(t)
	grp := testutil.SeedOITB(t, "Grain", false)
	paddy := testutil.SeedProdItem(t, "PADDY", "Palay", 20.0, 1000, false, grp)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)

	require.NoError(t, StartMillingOrder(mo.ID, nil))
	require.Error(t, StartMillingOrder(mo.ID, nil), "second start must be rejected")

	require.Equal(t, 900.0, onHand(t, "PADDY"), "stock decremented exactly once")
	require.Equal(t, int64(1), countRows(t, "oige", "1=1"), "exactly one goods issue")
}

// ─────────────────────────────────────────────────────────────────────────────
// Finding #6: joint cost is split by relative sales value, not uniform per-kg
// ─────────────────────────────────────────────────────────────────────────────

func TestMilling_Complete_AllocatesByRelativeSalesValue(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedProductionAccounting(t)
	grp := testutil.SeedOITB(t, "Grain", false)
	paddy := testutil.SeedProdItem(t, "PADDY", "Palay", 20.0, 1000, false, grp)
	rice := testutil.SeedProdItem(t, "RICE", "Milled Rice", 0, 0, false, grp)
	bran := testutil.SeedProdItem(t, "BRAN", "Rice Bran", 0, 0, false, grp)

	// Head rice sells for far more than bran, so it must absorb far more cost/kg.
	testutil.SeedItemPrice(t, "RICE", 1, 40.0)
	testutil.SeedItemPrice(t, "BRAN", 1, 5.0)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)
	require.NoError(t, StartMillingOrder(mo.ID, nil))

	_, err = CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		Lines: []CompleteMillingLineInput{
			{OutputItemID: rice, ActualQty: 65},
			{OutputItemID: bran, ActualQty: 25},
		},
	}, nil)
	require.NoError(t, err)

	var ricePrice, branPrice float64
	require.NoError(t, db.DB.Raw(`SELECT price FROM wor1 WHERE doc_entry = ? AND item_code = 'RICE'`, mo.ID).Scan(&ricePrice).Error)
	require.NoError(t, db.DB.Raw(`SELECT price FROM wor1 WHERE doc_entry = ? AND item_code = 'BRAN'`, mo.ID).Scan(&branPrice).Error)

	// Quantity-only allocation would give both 2000/90 ≈ 22.22. Relative-sales-value
	// must push rice well above and bran well below that.
	require.Greater(t, ricePrice, 25.0)
	require.Less(t, branPrice, 10.0)
	require.Greater(t, ricePrice, branPrice)
}

// ─────────────────────────────────────────────────────────────────────────────
// Finding #1: cancelling a Completed order reverses every step atomically
// ─────────────────────────────────────────────────────────────────────────────

func TestMilling_CancelCompleted_ReversesEverything(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedProductionAccounting(t)
	grp := testutil.SeedOITB(t, "Grain", false)
	paddy := testutil.SeedProdItem(t, "PADDY", "Palay", 20.0, 1000, false, grp)
	rice := testutil.SeedProdItem(t, "RICE", "Milled Rice", 0, 0, false, grp)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)
	require.NoError(t, StartMillingOrder(mo.ID, nil))
	_, err = CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		Lines: []CompleteMillingLineInput{{OutputItemID: rice, ActualQty: 65}},
	}, nil)
	require.NoError(t, err)

	require.NoError(t, CancelMillingOrder(mo.ID, nil))

	require.Equal(t, "L", statusOf(t, mo.ID))
	require.Equal(t, 1000.0, onHand(t, "PADDY"), "input restored")
	require.Equal(t, 0.0, onHand(t, "RICE"), "output removed")
	// Two reversal JEs (start + complete) posted, and the books still balance.
	require.Equal(t, int64(2), countRows(t, "journal_entry", "is_reversal = 1"))
	require.InDelta(t, jeSumDebit(t), jeSumCredit(t), 0.01)
}

// ─────────────────────────────────────────────────────────────────────────────
// Finding #7: BOM Work Orders post to the GL and roll up component cost
// ─────────────────────────────────────────────────────────────────────────────

func TestWorkOrder_ReleaseAndClose_PostGLAndRollUpCost(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedProductionAccounting(t)
	grp := testutil.SeedOITB(t, "Goods", false)
	testutil.SeedProdItem(t, "FG", "Finished Good", 0, 0, true, grp) // new item, no price history
	testutil.SeedProdItem(t, "COMP", "Component", 10.0, 500, false, grp)
	testutil.SeedOITW(t, "COMP", "WH01", 500)

	require.NoError(t, UpsertBOM(&models.ProductTree{
		Code: "FG", Quantity: 1,
		Lines: []models.ProductTreeLine{
			{ItemCode: "COMP", ItemName: "Component", Quantity: 2, IssueMethod: "B"},
		},
	}))

	wo, err := CreateWorkOrder(CreateWORequest{ItemCode: "FG", PlannedQty: 10, Warehouse: "WH01"}, nil)
	require.NoError(t, err)

	require.NoError(t, ReleaseWorkOrder(wo.DocEntry, nil))
	require.Equal(t, 480.0, onHand(t, "COMP"), "2×10 = 20 components issued")
	require.Positive(t, jeSumDebit(t), "release must post a WIP journal entry")

	require.NoError(t, CloseWorkOrder(wo.DocEntry, nil, 10, 0))
	require.Equal(t, 10.0, onHand(t, "FG"))

	var fgAvg float64
	require.NoError(t, db.DB.Raw(`SELECT avg_price FROM oitm WHERE item_code = 'FG'`).Scan(&fgAvg).Error)
	// 20 components × ₱10 = ₱200 rolled into 10 finished units → ₱20/unit (not zero).
	require.InDelta(t, 20.0, fgAvg, 0.01)
	require.InDelta(t, jeSumDebit(t), jeSumCredit(t), 0.01)
}

// ─────────────────────────────────────────────────────────────────────────────
// Finding (lower): BOM validation rejects self-reference and missing components
// ─────────────────────────────────────────────────────────────────────────────

func TestUpsertBOM_Validation(t *testing.T) {
	testutil.SetupDB(t)
	grp := testutil.SeedOITB(t, "Goods", false)
	testutil.SeedProdItem(t, "FG", "Finished Good", 0, 0, true, grp)
	testutil.SeedProdItem(t, "COMP", "Component", 10.0, 500, false, grp)

	// Self-reference (single-level circular).
	require.Error(t, UpsertBOM(&models.ProductTree{
		Code:  "FG",
		Lines: []models.ProductTreeLine{{ItemCode: "FG", Quantity: 1}},
	}))

	// Non-existent component.
	require.Error(t, UpsertBOM(&models.ProductTree{
		Code:  "FG",
		Lines: []models.ProductTreeLine{{ItemCode: "GHOST", Quantity: 1}},
	}))

	// Non-positive quantity.
	require.Error(t, UpsertBOM(&models.ProductTree{
		Code:  "FG",
		Lines: []models.ProductTreeLine{{ItemCode: "COMP", Quantity: 0}},
	}))

	// Valid BOM saves cleanly.
	require.NoError(t, UpsertBOM(&models.ProductTree{
		Code: "FG", Quantity: 1,
		Lines: []models.ProductTreeLine{{ItemCode: "COMP", ItemName: "Component", Quantity: 2}},
	}))
}

// ─────────────────────────────────────────────────────────────────────────────
// Part 2 #1: yield template auto-creates expected outputs; variance is automatic
// ─────────────────────────────────────────────────────────────────────────────

// seedMillingScenario seeds the standard paddy → head rice / brokens / bran setup
// with a per-100-kg yield template (62 H / 8 B / 10 Y) and returns the item IDs.
func seedMillingScenario(t *testing.T) (paddy, rice, broken, bran uint) {
	t.Helper()
	testutil.SeedProductionAccounting(t)
	grain := testutil.SeedOITB(t, "Palay", false)
	milled := testutil.SeedOITB(t, "Milled Rice", false)
	byProd := testutil.SeedOITB(t, "By-Products", false)
	paddy = testutil.SeedProdItem(t, "PADDY", "Palay", 20.0, 1000, false, grain)
	rice = testutil.SeedProdItem(t, "RICE", "Head Rice", 0, 0, false, milled)
	broken = testutil.SeedProdItem(t, "BROKEN", "Broken Rice", 0, 0, false, milled)
	bran = testutil.SeedProdItem(t, "BRAN", "Rice Bran", 0, 0, false, byProd)

	require.NoError(t, UpsertBOM(&models.ProductTree{
		Code: "PADDY", TreeType: "M", Quantity: 100,
		Lines: []models.ProductTreeLine{
			{ItemCode: "RICE", ItemName: "Head Rice", Quantity: 62, OutputType: "H"},
			{ItemCode: "BROKEN", ItemName: "Broken Rice", Quantity: 8, OutputType: "B"},
			{ItemCode: "BRAN", ItemName: "Rice Bran", Quantity: 10, OutputType: "Y"},
		},
	}))
	return
}

func TestMilling_YieldTemplate_AutoCreatesExpectedLines(t *testing.T) {
	testutil.SetupDB(t)
	paddy, _, _, _ := seedMillingScenario(t)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 200, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)

	// Template scaled 200/100 = 2×: expected 124 rice, 16 brokens, 20 bran.
	require.Len(t, mo.Lines, 3)
	require.Equal(t, 124.0, mo.Lines[0].ExpectedQty)
	require.Equal(t, "H", mo.Lines[0].OutputType)
	require.Equal(t, 16.0, mo.Lines[1].ExpectedQty)
	require.Equal(t, "B", mo.Lines[1].OutputType)
	require.Equal(t, 20.0, mo.Lines[2].ExpectedQty)
	require.Equal(t, "Y", mo.Lines[2].OutputType)
	require.InDelta(t, 62.0, mo.Lines[0].YieldPct, 0.01)
}

func TestMilling_YieldTemplate_VarianceIsAutomatic(t *testing.T) {
	testutil.SetupDB(t)
	paddy, rice, broken, bran := seedMillingScenario(t)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)
	require.NoError(t, StartMillingOrder(mo.ID, nil))

	// Operator records actuals WITHOUT entering expected qty — variance must come
	// from the template: rice 60 vs 62 (−3.23%), brokens 10 vs 8 (+25%), bran missed.
	res, err := CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		Lines: []CompleteMillingLineInput{
			{OutputItemID: rice, ActualQty: 60},
			{OutputItemID: broken, ActualQty: 10},
		},
	}, nil)
	require.NoError(t, err)
	_ = bran

	require.Len(t, res.Variances, 3)
	byCode := map[string]LineVariance{}
	for _, v := range res.Variances {
		byCode[v.OutputItemCode] = v
	}
	require.InDelta(t, -3.23, byCode["RICE"].VariancePct, 0.01)
	require.InDelta(t, 25.0, byCode["BROKEN"].VariancePct, 0.01)
	require.InDelta(t, -100.0, byCode["BRAN"].VariancePct, 0.01, "missed template output reports -100%")

	// Template lines are UPDATED in place, not duplicated.
	require.Equal(t, int64(3), countRows(t, "wor1", "doc_entry = ? AND line_dir = 'O'", mo.ID))
	require.Equal(t, int64(1), countRows(t, "wor1", "doc_entry = ? AND item_code = 'RICE' AND issued_qty = 60", mo.ID))
}

// ─────────────────────────────────────────────────────────────────────────────
// Part 2 #1: recovery-band validation blocks suspicious completions
// ─────────────────────────────────────────────────────────────────────────────

func TestMilling_RecoveryBand_BlocksOutOfBandAndAllowsOverride(t *testing.T) {
	testutil.SetupDB(t)
	paddy, rice, _, _ := seedMillingScenario(t)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)
	require.NoError(t, StartMillingOrder(mo.ID, nil))

	// 40% milled-rice recovery — far below the 60% floor → shrinkage/theft or a
	// data-entry error; must be blocked with NO side effects.
	_, err = CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		Lines: []CompleteMillingLineInput{{OutputItemID: rice, ActualQty: 40}},
	}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "outside the acceptable band")
	require.Equal(t, "R", statusOf(t, mo.ID), "order must remain In Progress")
	require.Equal(t, 0.0, onHand(t, "RICE"), "no stock received")

	// Explicit operator override posts it and surfaces a warning.
	res, err := CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		Lines:                  []CompleteMillingLineInput{{OutputItemID: rice, ActualQty: 40}},
		AllowOutOfBandRecovery: true,
	}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, res.Warnings)
	require.Equal(t, 40.0, res.MilledRecovery)
	require.Equal(t, 40.0, onHand(t, "RICE"))
}

func TestMilling_RecoveryBand_ConfigurableViaSettings(t *testing.T) {
	testutil.SetupDB(t)
	paddy, rice, _, _ := seedMillingScenario(t)
	testutil.SeedFarmSetting(t, "milling_recovery_min", "30")

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)
	require.NoError(t, StartMillingOrder(mo.ID, nil))

	// 40% is fine once the floor is lowered to 30%.
	_, err = CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		Lines: []CompleteMillingLineInput{{OutputItemID: rice, ActualQty: 40}},
	}, nil)
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// Part 2 #2: head rice vs brokens KPIs + trend report
// ─────────────────────────────────────────────────────────────────────────────

func TestMilling_HeadRiceAndBrokenKPIs(t *testing.T) {
	testutil.SetupDB(t)
	paddy, rice, broken, bran := seedMillingScenario(t)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01", BatchNo: "LOT-01", MoisturePct: 14,
	}, nil)
	require.NoError(t, err)
	require.NoError(t, StartMillingOrder(mo.ID, nil))

	res, err := CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		OutMoisturePct: 12.5,
		Lines: []CompleteMillingLineInput{
			{OutputItemID: rice, ActualQty: 62},
			{OutputItemID: broken, ActualQty: 8},
			{OutputItemID: bran, ActualQty: 10},
		},
	}, nil)
	require.NoError(t, err)

	require.Equal(t, 62.0, res.HeadRicePct)
	require.Equal(t, 8.0, res.BrokenPct)
	require.Equal(t, 70.0, res.MilledRecovery)
	require.Equal(t, 80.0, res.RecoveryRate)

	rows, err := MillingYieldReport()
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "LOT-01", rows[0].BatchNo)
	require.Equal(t, 14.0, rows[0].MoisturePct)
	require.Equal(t, 12.5, rows[0].OutMoisturePct)
	require.Equal(t, 62.0, rows[0].HeadRicePct)
	require.Equal(t, 8.0, rows[0].BrokenPct)
	require.InDelta(t, 11.43, rows[0].BrokenShare, 0.01, "brokens / (head+brokens)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Part 2 #6: conversion cost is absorbed into WIP and output cost
// ─────────────────────────────────────────────────────────────────────────────

func TestMilling_ConversionCost_AbsorbedIntoOutputCost(t *testing.T) {
	testutil.SetupDB(t)
	paddy, rice, _, _ := seedMillingScenario(t)
	testutil.SeedFarmSetting(t, "milling_conv_cost_per_kg", "0.5")

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)
	require.NoError(t, StartMillingOrder(mo.ID, nil))

	// Input cost 100×20 = 2000; conversion 100×0.5 = 50 → 2050 flows to output.
	res, err := CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		Lines: []CompleteMillingLineInput{{OutputItemID: rice, ActualQty: 65}},
	}, nil)
	require.NoError(t, err)
	require.Equal(t, 50.0, res.ConversionCost)

	var ricePrice float64
	require.NoError(t, db.DB.Raw(`SELECT price FROM wor1 WHERE doc_entry = ? AND item_code = 'RICE'`, mo.ID).Scan(&ricePrice).Error)
	require.InDelta(t, 2050.0/65.0, ricePrice, 0.01, "unit cost includes conversion cost")

	// Overhead absorbed account credited 50; books balance.
	var ovhCredit float64
	require.NoError(t, db.DB.Raw(
		`SELECT COALESCE(SUM(l.credit),0) FROM je_line l
		 JOIN gl_account a ON a.id = l.gl_account_id WHERE a.code = '5-2100'`).Scan(&ovhCredit).Error)
	require.InDelta(t, 50.0, ovhCredit, 0.01)
	require.InDelta(t, jeSumDebit(t), jeSumCredit(t), 0.01)

	var convCost float64
	require.NoError(t, db.DB.Raw(`SELECT conv_cost FROM owor WHERE doc_entry = ?`, mo.ID).Scan(&convCost).Error)
	require.Equal(t, 50.0, convCost)
}

func TestMilling_ConversionCost_RequiresOverheadAccount(t *testing.T) {
	testutil.SetupDB(t)
	// Seed WIP + input/output but NOT the overhead account.
	wip := testutil.GLAcct(t, "1-1350", "WIP", "ASSET", "DEBIT")
	inv := testutil.GLAcct(t, "1-1310", "Inventory", "ASSET", "DEBIT")
	testutil.AcctDet(t, "PRODUCTION", "MILLING_WIP", nil, wip)
	testutil.AcctDet(t, "PRODUCTION", "MILLING_INPUT", nil, inv)
	testutil.AcctDet(t, "PRODUCTION", "MILLING_OUTPUT", nil, inv)

	grp := testutil.SeedOITB(t, "Grain", false)
	paddy := testutil.SeedProdItem(t, "PADDY", "Palay", 20.0, 1000, false, grp)
	rice := testutil.SeedProdItem(t, "RICE", "Milled Rice", 0, 0, false, grp)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)
	require.NoError(t, StartMillingOrder(mo.ID, nil))

	_, err = CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		ConversionCost: 75,
		Lines:          []CompleteMillingLineInput{{OutputItemID: rice, ActualQty: 65}},
	}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "MILLING_OVERHEAD")
	require.Equal(t, "R", statusOf(t, mo.ID), "pre-flight failure leaves no side effects")
}

// ─────────────────────────────────────────────────────────────────────────────
// Part 2 #3/#4: batch flows to GR lines + OIVL; moisture persisted
// ─────────────────────────────────────────────────────────────────────────────

func TestMilling_BatchNo_FlowsToGRAndOIVL(t *testing.T) {
	testutil.SetupDB(t)
	paddy, rice, _, _ := seedMillingScenario(t)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01", BatchNo: "LOT-42",
	}, nil)
	require.NoError(t, err)
	require.NoError(t, StartMillingOrder(mo.ID, nil))
	_, err = CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		Lines: []CompleteMillingLineInput{{OutputItemID: rice, ActualQty: 65}},
	}, nil)
	require.NoError(t, err)

	require.Equal(t, int64(1), countRows(t, "ign1", "item_code = 'RICE' AND batch_no = 'LOT-42'"),
		"GR line carries the milling batch")
	require.Equal(t, int64(1), countRows(t, "oivl", "item_code = 'RICE' AND batch_no = 'LOT-42' AND trans_type = 'GR'"),
		"stock ledger carries the milling batch")
}

func TestMilling_BatchNo_DefaultsToMONumber(t *testing.T) {
	testutil.SetupDB(t)
	paddy, _, _, _ := seedMillingScenario(t)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, mo.MoNumber, mo.BatchNo)
}

// ─────────────────────────────────────────────────────────────────────────────
// Part 2 #8: rejected/spillage qty recorded as explicit process loss
// ─────────────────────────────────────────────────────────────────────────────

func TestMilling_RejectedQty_RecordedOnCompletion(t *testing.T) {
	testutil.SetupDB(t)
	paddy, rice, broken, bran := seedMillingScenario(t)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: paddy, InputQty: 100, WarehouseCode: "WH01",
	}, nil)
	require.NoError(t, err)
	require.NoError(t, StartMillingOrder(mo.ID, nil))

	res, err := CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		RejectedQty: 3,
		Lines: []CompleteMillingLineInput{
			{OutputItemID: rice, ActualQty: 62},
			{OutputItemID: broken, ActualQty: 8},
			{OutputItemID: bran, ActualQty: 10},
		},
	}, nil)
	require.NoError(t, err)

	var rjct float64
	require.NoError(t, db.DB.Raw(`SELECT rjct_qty FROM owor WHERE doc_entry = ?`, mo.ID).Scan(&rjct).Error)
	require.Equal(t, 3.0, rjct)
	// 100 in − 80 out − 3 rejected = 17% unexplained gap (moisture loss etc.).
	require.InDelta(t, 17.0, res.ProcessLossPct, 0.01)

	_, err = CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		RejectedQty: -1,
		Lines:       []CompleteMillingLineInput{{OutputItemID: rice, ActualQty: 62}},
	}, nil)
	require.Error(t, err, "negative rejected qty must be rejected")
}

// ─────────────────────────────────────────────────────────────────────────────
// Part 2 #7: drying orders ('D') reuse the milling machinery, band exempt
// ─────────────────────────────────────────────────────────────────────────────

func TestDryingOrder_ReusesMillingFlow_BandExempt(t *testing.T) {
	testutil.SetupDB(t)
	testutil.SeedProductionAccounting(t)
	grain := testutil.SeedOITB(t, "Palay", false)
	milled := testutil.SeedOITB(t, "Milled Rice", false)
	wet := testutil.SeedProdItem(t, "PADDY-WET", "Wet Palay", 15.0, 500, false, grain)
	dry := testutil.SeedProdItem(t, "PADDY-DRY", "Dried Palay", 0, 0, false, milled)

	mo, err := CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: wet, InputQty: 100, WarehouseCode: "WH01",
		OrderType: "D", MoisturePct: 22,
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "D", mo.OrderType)
	require.NoError(t, StartMillingOrder(mo.ID, nil))

	// 85% mass recovery would breach the milling ceiling (75%), but drying is a
	// separate yield-loss stage and is exempt from the milled-rice band.
	_, err = CompleteMillingOrder(mo.ID, CompleteMillingRequest{
		OutMoisturePct: 14,
		Lines:          []CompleteMillingLineInput{{OutputItemID: dry, ActualQty: 85}},
	}, nil)
	require.NoError(t, err)

	// Drying orders appear in the milling list.
	list, err := ListMillingOrders()
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "D", list[0].OrderType)

	// Invalid order types are rejected.
	_, err = CreateMillingOrder(CreateMillingOrderRequest{
		PostingDate: "2026-01-05", InputItemID: wet, InputQty: 10, WarehouseCode: "WH01", OrderType: "X",
	}, nil)
	require.Error(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// Part 2: yield-template validation
// ─────────────────────────────────────────────────────────────────────────────

func TestUpsertBOM_YieldTemplateValidation(t *testing.T) {
	testutil.SetupDB(t)
	grp := testutil.SeedOITB(t, "Palay", false)
	testutil.SeedProdItem(t, "PADDY", "Palay", 20.0, 1000, false, grp)
	testutil.SeedProdItem(t, "RICE", "Head Rice", 0, 0, false, grp)

	// Outputs exceeding 100% of input mass are rejected.
	require.Error(t, UpsertBOM(&models.ProductTree{
		Code: "PADDY", TreeType: "M", Quantity: 100,
		Lines: []models.ProductTreeLine{{ItemCode: "RICE", Quantity: 105}},
	}))

	// Invalid output classification is rejected.
	require.Error(t, UpsertBOM(&models.ProductTree{
		Code: "PADDY", TreeType: "M", Quantity: 100,
		Lines: []models.ProductTreeLine{{ItemCode: "RICE", Quantity: 62, OutputType: "Z"}},
	}))

	// Milling templates do NOT require mak_item='Y' on the input item.
	require.NoError(t, UpsertBOM(&models.ProductTree{
		Code: "PADDY", TreeType: "M", Quantity: 100,
		Lines: []models.ProductTreeLine{{ItemCode: "RICE", Quantity: 62, OutputType: "H"}},
	}))
}
