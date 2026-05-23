package reports

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// ExportBalanceSheetExcel builds an .xlsx workbook from a BalanceSheet and writes it to destPath.
func ExportBalanceSheetExcel(bs *BalanceSheet, destPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Balance Sheet"
	f.SetSheetName("Sheet1", sheet)

	// ── Column widths ──
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 40)
	f.SetColWidth(sheet, "C", "C", 20)

	// ── Styles ──
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	subTitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Color: "666666"},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	sectionStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8E8E8"}},
	})
	currencyFmt := `_(₱* #,##0.00_);_(₱* (#,##0.00);_(₱* "-"??_);_(@_)`
	amountStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt:    0,
		CustomNumFmt: &currencyFmt,
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		NumFmt:       0,
		CustomNumFmt: &currencyFmt,
		Alignment:    &excelize.Alignment{Horizontal: "right"},
		Border: []excelize.Border{
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 6},
		},
	})
	balancedStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D4EDDA"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	unbalancedStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "842029"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F8D7DA"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// ── Title rows ──
	f.MergeCell(sheet, "A1", "C1")
	f.SetCellValue(sheet, "A1", "Balance Sheet")
	f.SetCellStyle(sheet, "A1", "C1", titleStyle)

	f.MergeCell(sheet, "A2", "C2")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("As of %s", bs.AsOf))
	f.SetCellStyle(sheet, "A2", "C2", subTitleStyle)

	row := 4

	writeSection := func(sec BSSection) {
		f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), sec.Name)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), sectionStyle)
		row++

		for _, acct := range sec.Accounts {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), acct.Code)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), acct.Name)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), acct.Amount)
			f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), amountStyle)
			row++
		}
		if len(sec.Accounts) == 0 {
			f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "No records")
			row++
		}

		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("Total %s", sec.Name))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), sec.Total)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), totalStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), totalStyle)
		row += 2
	}

	writeSection(bs.Assets)
	writeSection(bs.Liabilities)
	writeSection(bs.Equity)

	// ── Balance check ──
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
	isBalanced := bs.Check < 0.01 && bs.Check > -0.01
	if isBalanced {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Balance Check: Balanced")
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), balancedStyle)
	} else {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("Balance Check: Off by ₱%.2f", bs.Check))
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), unbalancedStyle)
	}

	return f.SaveAs(destPath)
}

// ExportPnLExcel builds an .xlsx workbook from a PnLReport and writes it to destPath.
func ExportPnLExcel(pnl *PnLReport, destPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Profit & Loss"
	f.SetSheetName("Sheet1", sheet)

	f.SetColWidth(sheet, "A", "A", 40)
	f.SetColWidth(sheet, "B", "B", 20)

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	subTitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Color: "666666"},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	sectionStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8E8E8"}},
	})
	currencyFmt := `_(₱* #,##0.00_);_(₱* (#,##0.00);_(₱* "-"??_);_(@_)`
	amountStyle, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &currencyFmt,
		Alignment:    &excelize.Alignment{Horizontal: "right"},
	})
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		CustomNumFmt: &currencyFmt,
		Alignment:    &excelize.Alignment{Horizontal: "right"},
		Border: []excelize.Border{
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 6},
		},
	})
	profitStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 12},
		CustomNumFmt: &currencyFmt,
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D4EDDA"}},
		Alignment:    &excelize.Alignment{Horizontal: "right"},
	})
	profitLabelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D4EDDA"}},
	})
	lossStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 12, Color: "842029"},
		CustomNumFmt: &currencyFmt,
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F8D7DA"}},
		Alignment:    &excelize.Alignment{Horizontal: "right"},
	})
	lossLabelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "842029"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F8D7DA"}},
	})

	f.MergeCell(sheet, "A1", "B1")
	f.SetCellValue(sheet, "A1", "Profit & Loss Statement")
	f.SetCellStyle(sheet, "A1", "B1", titleStyle)

	f.MergeCell(sheet, "A2", "B2")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("%s to %s", pnl.PeriodStart, pnl.PeriodEnd))
	f.SetCellStyle(sheet, "A2", "B2", subTitleStyle)

	row := 4

	writeRows := func(label string, items []PnLRow, total float64, totalLabel string) {
		f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row))
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), label)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), sectionStyle)
		row++

		for _, item := range items {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.AccountName)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.Amount)
			f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), amountStyle)
			row++
		}
		if len(items) == 0 {
			f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row))
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "No records")
			row++
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), totalLabel)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), total)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), totalStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), totalStyle)
		row += 2
	}

	writeRows("Revenue", pnl.Revenue, pnl.TotalRevenue, "Total Revenue")
	writeRows("Expenses", pnl.Expenses, pnl.TotalExpense, "Total Expenses")

	// Net Income row
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Net Income")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), pnl.NetIncome)
	if pnl.NetIncome >= 0 {
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), profitLabelStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), profitStyle)
	} else {
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), lossLabelStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), lossStyle)
	}

	return f.SaveAs(destPath)
}
