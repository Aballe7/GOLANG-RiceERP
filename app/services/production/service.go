package production

import (
	"errors"
	"fmt"
	"math"
	"time"

	"ricemill/app/db"
	"ricemill/app/models"
	accountingsvc "ricemill/app/services/accounting"
	"ricemill/app/services/docnumber"
	invsvc "ricemill/app/services/inventory"

	"gorm.io/gorm"
)

// ─────────────────────────────────────────────
// DTOs
// ─────────────────────────────────────────────

type CreateMillingOrderRequest struct {
	PostingDate   string  `json:"posting_date"`
	ExpectedDate  string  `json:"expected_date"`
	Remarks       string  `json:"remarks"`
	InputItemID   uint    `json:"input_item_id"`
	InputQty      float64 `json:"input_qty"`
	InputUomEntry uint    `json:"input_uom_entry"`
	WarehouseCode string  `json:"warehouse_code"` // warehouse for GI/GR movements
}

// CompleteMillingLineInput captures actual output recorded at completion time.
type CompleteMillingLineInput struct {
	OutputItemID uint    `json:"output_item_id"`
	ActualQty    float64 `json:"actual_qty"`
	ExpectedQty  float64 `json:"expected_qty"`  // optional: expected qty for variance reporting (0 = not specified)
	UomEntry     uint    `json:"uom_entry"`
}

type CompleteMillingRequest struct {
	Lines          []CompleteMillingLineInput `json:"lines"`
	CompletionNote string                     `json:"completion_note"`
	CompletionDate string                     `json:"completion_date"` // "YYYY-MM-DD"; defaults to today
}

// LineVariance is one row in the yield-variance report returned after completion.
type LineVariance struct {
	OutputItemCode string  `json:"output_item_code"`
	OutputItemName string  `json:"output_item_name"`
	ExpectedQty    float64 `json:"expected_qty"`
	ActualQty      float64 `json:"actual_qty"`
	VariancePct    float64 `json:"variance_pct"` // positive = over-yield, negative = under-yield
}

// CompleteMillingResult is returned by CompleteMillingOrder.
// It carries the refreshed order plus yield analytics.
type CompleteMillingResult struct {
	Order        *models.MillingOrder `json:"order"`
	RecoveryRate float64              `json:"recovery_rate"`  // totalOutputQty / inputQty × 100
	Variances    []LineVariance       `json:"variances"`      // only lines where ExpectedQty > 0
}

type CreateWORequest struct {
	ItemCode   string  `json:"item_code"`
	PlannedQty float64 `json:"planned_qty"`
	StartDate  string  `json:"start_date"`
	DueDate    string  `json:"due_date"`
	Warehouse  string  `json:"warehouse"`
	Notes      string  `json:"notes"`
}

// ─────────────────────────────────────────────
// Status helpers
// ─────────────────────────────────────────────

func woStatusToMilling(s string) string {
	switch s {
	case "P":
		return "Draft"
	case "R":
		return "In Progress"
	case "C":
		return "Completed"
	case "L":
		return "Cancelled"
	default:
		return s
	}
}

// woToMilling converts a WorkOrder + its WOR1 lines into the legacy MillingOrder shape
// so the frontend and handlers need no changes.
func woToMilling(wo *models.WorkOrder, inputLine models.WorkOrderLine, outputLines []models.WorkOrderLine) models.MillingOrder {
	mo := models.MillingOrder{
		ID:                 wo.DocEntry,
		MoNumber:           wo.DocNum,
		PostingDate:        wo.StartDate,
		ExpectedDate:       wo.DueDate,
		Status:             woStatusToMilling(wo.Status),
		InputItemID:        inputLine.ItemID,
		InputItemCode:      inputLine.ItemCode,
		InputItemName:      inputLine.ItemName,
		InputItemCategory:  inputLine.ItemCategory,
		InputQty:           inputLine.PlannedQty,
		InputUomEntry:      inputLine.UomEntry,
		InputUnit:          inputLine.UomCode,
		InputUnitCost:      inputLine.Price,
		DocTotal:           inputLine.PlannedQty * inputLine.Price,
		Remarks:            wo.Notes,
		GoodsIssueID:       wo.GoodsIssueID,
		GoodsIssueNumber:   wo.GoodsIssueNumber,
		StartJEID:          wo.StartJEID,
		GoodsReceiptID:     wo.GoodsReceiptID,
		GoodsReceiptNumber: wo.GoodsReceiptNumber,
		CompleteJEID:       wo.CompleteJEID,
	}
	if outputLines != nil {
		mo.Lines = make([]models.MillingOrderLine, 0, len(outputLines))
		for _, l := range outputLines {
			mo.Lines = append(mo.Lines, models.MillingOrderLine{
				ID:                 l.ID,
				MillingOrderID:     wo.DocEntry,
				LineNum:            l.LineNum,
				OutputItemID:       l.ItemID,
				OutputItemCode:     l.ItemCode,
				OutputItemName:     l.ItemName,
				OutputItemCategory: l.ItemCategory,
				ActualQty:          l.IssuedQty,
				UomEntry:           l.UomEntry,
				Unit:               l.UomCode,
			})
		}
	}
	return mo
}

// loadWOLines loads WOR1 lines for a work order and splits into input/output.
func loadWOLines(docEntry uint) (input models.WorkOrderLine, outputs []models.WorkOrderLine) {
	var lines []models.WorkOrderLine
	db.DB.Where("doc_entry = ?", docEntry).Order("line_num").Find(&lines)
	for _, l := range lines {
		if l.LineDir == "I" {
			input = l
		} else {
			outputs = append(outputs, l)
		}
	}
	return
}

// ─────────────────────────────────────────────
// Milling Orders — List / Get
// ─────────────────────────────────────────────

func ListMillingOrders() ([]models.MillingOrder, error) {
	var wos []models.WorkOrder
	if err := db.DB.Where("order_type = 'M'").Order("created_at DESC").Find(&wos).Error; err != nil {
		return nil, err
	}
	if len(wos) == 0 {
		return []models.MillingOrder{}, nil
	}

	// Batch-load ALL lines (input + output) in a single query
	docEntries := make([]uint, len(wos))
	for i, wo := range wos {
		docEntries[i] = wo.DocEntry
	}
	var allLines []models.WorkOrderLine
	db.DB.Where("doc_entry IN ?", docEntries).Order("line_num").Find(&allLines)

	inputByDoc := make(map[uint]models.WorkOrderLine, len(wos))
	outputsByDoc := make(map[uint][]models.WorkOrderLine, len(wos))
	for _, l := range allLines {
		if l.LineDir == "I" {
			inputByDoc[l.DocEntry] = l
		} else {
			outputsByDoc[l.DocEntry] = append(outputsByDoc[l.DocEntry], l)
		}
	}

	result := make([]models.MillingOrder, len(wos))
	for i, wo := range wos {
		result[i] = woToMilling(&wo, inputByDoc[wo.DocEntry], outputsByDoc[wo.DocEntry])
	}
	return result, nil
}

func GetMillingOrder(id uint) (*models.MillingOrder, error) {
	var wo models.WorkOrder
	if err := db.DB.First(&wo, id).Error; err != nil {
		return nil, err
	}
	inputLine, outputLines := loadWOLines(wo.DocEntry)
	mo := woToMilling(&wo, inputLine, outputLines)
	return &mo, nil
}

// ─────────────────────────────────────────────
// Milling Orders — Create (Draft)
// ─────────────────────────────────────────────

func CreateMillingOrder(req CreateMillingOrderRequest, userID *uint) (*models.MillingOrder, error) {
	if req.InputItemID == 0 {
		return nil, errors.New("input item is required")
	}
	if req.InputQty <= 0 {
		return nil, errors.New("input quantity must be greater than zero")
	}
	if req.PostingDate == "" {
		return nil, errors.New("posting date is required")
	}

	var inputItem models.ItemMaster
	if err := db.DB.First(&inputItem, req.InputItemID).Error; err != nil {
		return nil, fmt.Errorf("input item not found: %w", err)
	}
	inputItem.CategoryName = invsvc.GetItemCategoryName(inputItem.ItmsGrpCod)

	// Validate category visibility for production
	var inputCat models.OITB
	if err := db.DB.First(&inputCat, inputItem.ItmsGrpCod).Error; err == nil && !inputCat.ForProduction {
		return nil, fmt.Errorf("item %s belongs to category '%s' which is not enabled for Production", inputItem.ItemCode, inputCat.ItmsGrpNam)
	}

	moNum, err := docnumber.Svc.NextMONumber()
	if err != nil {
		return nil, fmt.Errorf("generate MO number: %w", err)
	}

	uomEntry := req.InputUomEntry
	if uomEntry == 0 {
		uomEntry = inputItem.IUoMEntry
	}

	var createdByID uint
	if userID != nil {
		createdByID = *userID
	}

	wo := models.WorkOrder{
		DocNum:      moNum,
		OrderType:   "M",
		ItemCode:    inputItem.ItemCode,
		ItemName:    inputItem.ItemName,
		PlannedQty:  req.InputQty,
		StartDate:   req.PostingDate,
		DueDate:     req.ExpectedDate,
		Status:      "P",
		Warehouse:   req.WarehouseCode,
		Notes:       req.Remarks,
		CreatedByID: createdByID,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&wo).Error; err != nil {
			return err
		}
		inputLine := models.WorkOrderLine{
			DocEntry:     wo.DocEntry,
			LineNum:      0,
			LineDir:      "I",
			ItemID:       inputItem.ID,
			ItemCode:     inputItem.ItemCode,
			ItemName:     inputItem.ItemName,
			ItemCategory: inputItem.CategoryName,
			PlannedQty:   req.InputQty,
			UomEntry:     uomEntry,
			UomCode:      inputItem.InvntryUom,
			Price:        inputItem.AvgPrice,
		}
		return tx.Create(&inputLine).Error
	})
	if err != nil {
		return nil, err
	}
	return GetMillingOrder(wo.DocEntry)
}

// ─────────────────────────────────────────────
// Milling Orders — Start (Draft → In Progress)
// ─────────────────────────────────────────────

func StartMillingOrder(id uint, userID *uint) error {
	var wo models.WorkOrder
	if err := db.DB.First(&wo, id).Error; err != nil {
		return err
	}
	if wo.Status != "P" {
		return fmt.Errorf("milling order is %s — can only start a Draft order", woStatusToMilling(wo.Status))
	}

	inputLine, _ := loadWOLines(wo.DocEntry)

	var createdByID uint
	if userID != nil {
		createdByID = *userID
	}

	// ── Pre-flight: validate required GL accounts before touching inventory ──────
	// Both accounts must exist; if either is missing the operator must configure
	// account determination first (Admin → Accounting → Account Determination).
	wipAccount, err := accountingsvc.GetAccount(nil, "PRODUCTION", "MILLING_WIP")
	if err != nil || wipAccount == nil {
		return fmt.Errorf("cannot start milling order: MILLING_WIP GL account is not configured — set it up under Account Determination before proceeding")
	}
	inputInvAccount, err := accountingsvc.GetAccount(nil, "PRODUCTION", "MILLING_INPUT", inputLine.ItemCategory)
	if err != nil || inputInvAccount == nil {
		return fmt.Errorf("cannot start milling order: MILLING_INPUT GL account for category %q is not configured — set it up under Account Determination before proceeding", inputLine.ItemCategory)
	}

	// Create Goods Issue to consume input paddy
	giReq := invsvc.CreateGoodsIssueRequest{
		PostingDate: wo.StartDate,
		Remarks:     fmt.Sprintf("Milling Order %s — Input consumption", wo.DocNum),
		Lines: []invsvc.GILineInput{
			{
				ItemID:        inputLine.ItemID,
				UomEntry:      inputLine.UomEntry,
				Quantity:      inputLine.PlannedQty,
				Price:         inputLine.Price,
				WarehouseCode: wo.Warehouse,
			},
		},
	}
	gi, err := invsvc.CreateGoodsIssue(giReq, userID)
	if err != nil {
		return fmt.Errorf("create goods issue: %w", err)
	}

	// Post journal entry: Dr WIP / Cr Inventory (input paddy)
	totalCost := inputLine.PlannedQty * inputLine.Price
	var startJEID *uint

	if totalCost > 0 {
		now := time.Now()
		jeParams := accountingsvc.PostJEParams{
			Module:      "PRODUCTION",
			SourceType:  "MILLING_START",
			SourceRef:   wo.DocNum,
			Narration:   fmt.Sprintf("Milling Order %s — input paddy to WIP", wo.DocNum),
			EntryDate:   &now,
			CreatedByID: &createdByID,
			Lines: []accountingsvc.JELine{
				{Account: wipAccount, Debit: totalCost, Description: "WIP — " + inputLine.ItemName},
				{Account: inputInvAccount, Credit: totalCost, Description: "Inventory — " + inputLine.ItemName},
			},
		}
		innerErr := db.DB.Transaction(func(tx *gorm.DB) error {
			je, jeErr := accountingsvc.PostJournalEntry(tx, jeParams)
			if jeErr != nil {
				return jeErr
			}
			startJEID = &je.ID
			return nil
		})
		if innerErr != nil {
			startJEID = nil
		}
	}

	updates := map[string]interface{}{
		"status":             "R",
		"goods_issue_id":     gi.ID,
		"goods_issue_number": gi.GINumber,
		"updated_by_id":      createdByID,
		"version":            gorm.Expr("version + 1"),
	}
	if startJEID != nil {
		updates["start_je_id"] = *startJEID
	}
	return db.DB.Model(&models.WorkOrder{}).Where("doc_entry = ?", id).Updates(updates).Error
}

// ─────────────────────────────────────────────
// Milling Orders — Complete (In Progress → Completed)
// ─────────────────────────────────────────────

func CompleteMillingOrder(id uint, req CompleteMillingRequest, userID *uint) (*CompleteMillingResult, error) {
	var wo models.WorkOrder
	if err := db.DB.First(&wo, id).Error; err != nil {
		return nil, err
	}
	if wo.Status != "R" {
		return nil, fmt.Errorf("milling order is %s — can only complete an In Progress order", woStatusToMilling(wo.Status))
	}
	if len(req.Lines) == 0 {
		return nil, errors.New("at least one output line is required to complete the milling order")
	}

	inputLine, _ := loadWOLines(wo.DocEntry)

	// ── Pre-flight: validate MILLING_WIP account before touching inventory ───────
	// MILLING_WIP is required for both the debit and credit sides of the completion
	// JE. Output category accounts (MILLING_OUTPUT) fall back to WIP when missing,
	// so only WIP itself is a hard requirement here.
	wipAccount, err := accountingsvc.GetAccount(nil, "PRODUCTION", "MILLING_WIP")
	if err != nil || wipAccount == nil {
		return nil, fmt.Errorf("cannot complete milling order: MILLING_WIP GL account is not configured — set it up under Account Determination before proceeding")
	}

	// Fetch all output items referenced in the completion request
	outputIDs := make([]uint, 0, len(req.Lines))
	for _, l := range req.Lines {
		if l.OutputItemID > 0 && l.ActualQty > 0 {
			outputIDs = append(outputIDs, l.OutputItemID)
		}
	}
	if len(outputIDs) == 0 {
		return nil, errors.New("no output lines have valid item and quantity")
	}
	var outputItems []models.OITM
	if err := db.DB.Where("id IN ?", outputIDs).Find(&outputItems).Error; err != nil {
		return nil, fmt.Errorf("fetch output items: %w", err)
	}
	outputMap := make(map[uint]models.OITM, len(outputItems))
	for i := range outputItems {
		outputItems[i].CategoryName = invsvc.GetItemCategoryName(outputItems[i].ItmsGrpCod)
		outputMap[outputItems[i].ID] = outputItems[i]
	}

	// ── Circular item guard ──────────────────────────────────────────────────────
	// Prevent a user from recording the input item as one of the outputs, which
	// would create a circular stock movement (consume X kg paddy → produce X kg paddy).
	for _, outItem := range outputMap {
		if outItem.ItemCode == inputLine.ItemCode {
			return nil, fmt.Errorf("output item %q (%s) is the same as the input item — an item cannot be both consumed and produced in the same milling order", outItem.ItemCode, outItem.ItemName)
		}
	}

	var createdByID uint
	if userID != nil {
		createdByID = *userID
	}

	// ── Cost basis ──────────────────────────────────────────────────────────────
	// totalInputCost is the full cost of raw material consumed.
	// We distribute it across outputs proportional to each line's actual quantity
	// (quantity-weighted joint-cost allocation).
	totalInputCost := inputLine.PlannedQty * inputLine.Price

	// Pre-compute total actual output quantity across all valid lines.
	totalActualQty := 0.0
	for _, l := range req.Lines {
		if l.OutputItemID > 0 && l.ActualQty > 0 {
			totalActualQty += l.ActualQty
		}
	}

	// Unit cost applied to every output unit:  ₱/unit = totalInputCost / totalOutputUnits
	var unitCost float64
	if totalActualQty > 0 && totalInputCost > 0 {
		unitCost = math.Round((totalInputCost/totalActualQty)*10000) / 10000
	}

	// Completion date: use caller-supplied date or fall back to today.
	completionDate := req.CompletionDate
	if completionDate == "" {
		completionDate = time.Now().Format("2006-01-02")
	}

	// ── Build WOR1 output lines and GR lines ────────────────────────────────────
	newWORLines := make([]models.WorkOrderLine, 0, len(req.Lines))
	grLines := make([]invsvc.GRLineInput, 0, len(req.Lines))

	for i, l := range req.Lines {
		if l.OutputItemID == 0 || l.ActualQty <= 0 {
			continue
		}
		outItem, ok := outputMap[l.OutputItemID]
		if !ok {
			return nil, fmt.Errorf("output item ID %d not found", l.OutputItemID)
		}
		uom := l.UomEntry
		if uom == 0 {
			uom = outItem.IUoMEntry
		}
		newWORLines = append(newWORLines, models.WorkOrderLine{
			DocEntry:     wo.DocEntry,
			LineNum:      i + 1,
			LineDir:      "O",
			ItemID:       outItem.ID,
			ItemCode:     outItem.ItemCode,
			ItemName:     outItem.ItemName,
			ItemCategory: outItem.CategoryName,
			PlannedQty:   l.ActualQty,
			IssuedQty:    l.ActualQty, // actual qty produced
			UomEntry:     uom,
			UomCode:      outItem.InvntryUom,
			Warehouse:    wo.Warehouse,
			Price:        unitCost, // quantity-weighted unit cost (was hardcoded 0)
		})
		grLines = append(grLines, invsvc.GRLineInput{
			ItemID:        outItem.ID,
			UomEntry:      uom,
			Quantity:      l.ActualQty,
			Price:         unitCost, // was hardcoded 0 — now carries actual unit cost
			WarehouseCode: wo.Warehouse,
		})
	}

	if len(newWORLines) == 0 {
		return nil, errors.New("no valid output lines to record")
	}

	// Create Goods Receipt for all outputs.
	// PostingDate uses the actual completion date, not the milling start date.
	grReq := invsvc.CreateGoodsReceiptRequest{
		PostingDate: completionDate, // was wo.StartDate
		Remarks:     fmt.Sprintf("Milling Order %s — Output receipt", wo.DocNum),
		Lines:       grLines,
	}
	gr, err := invsvc.CreateGoodsReceipt(grReq, userID)
	if err != nil {
		return nil, fmt.Errorf("create goods receipt: %w", err)
	}

	// ── Post journal entry: Dr Inventory (outputs) / Cr WIP ─────────────────────
	// wipAccount was already validated above (pre-flight); use it directly.
	var completeJEID *uint

	if totalInputCost > 0 && len(newWORLines) > 0 {
		jeLines := make([]accountingsvc.JELine, 0, len(newWORLines)+1)

		// Quantity-weighted debit per output line.
		// Last line absorbs any rounding remainder to keep debits == credit exactly.
		allocatedSoFar := 0.0
		for idx, line := range newWORLines {
			var lineCost float64
			if idx == len(newWORLines)-1 {
				// Last line: use remainder to avoid rounding drift
				lineCost = math.Round((totalInputCost-allocatedSoFar)*100) / 100
			} else {
				lineCost = math.Round((totalInputCost*(line.PlannedQty/totalActualQty))*100) / 100
			}
			allocatedSoFar += lineCost

			outInvAccount, accErr := accountingsvc.GetAccount(nil, "PRODUCTION", "MILLING_OUTPUT", line.ItemCategory)
			if accErr != nil || outInvAccount == nil {
				outInvAccount = wipAccount
			}
			jeLines = append(jeLines, accountingsvc.JELine{
				Account:     outInvAccount,
				Debit:       lineCost,
				Description: fmt.Sprintf("Inventory — %s (%.3f %s @ %.4f)", line.ItemName, line.PlannedQty, line.UomCode, unitCost),
			})
		}
		jeLines = append(jeLines, accountingsvc.JELine{
			Account:     wipAccount,
			Credit:      math.Round(totalInputCost*100) / 100,
			Description: "WIP — " + wo.DocNum + " complete",
		})

		entryDate, _ := time.Parse("2006-01-02", completionDate)
		if entryDate.IsZero() {
			entryDate = time.Now()
		}
		jeParams := accountingsvc.PostJEParams{
			Module:      "PRODUCTION",
			SourceType:  "MILLING_COMPLETE",
			SourceRef:   wo.DocNum,
			Narration:   fmt.Sprintf("Milling Order %s — WIP to output inventory", wo.DocNum),
			EntryDate:   &entryDate,
			CreatedByID: &createdByID,
			Lines:       jeLines,
		}
		innerErr := db.DB.Transaction(func(tx *gorm.DB) error {
			je, jeErr := accountingsvc.PostJournalEntry(tx, jeParams)
			if jeErr != nil {
				return jeErr
			}
			completeJEID = &je.ID
			return nil
		})
		if innerErr != nil {
			completeJEID = nil
		}
	}

	// Persist output lines and update header
	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		for i := range newWORLines {
			if err := tx.Create(&newWORLines[i]).Error; err != nil {
				return err
			}
		}
		updates := map[string]interface{}{
			"status":               "C",
			"goods_receipt_id":     gr.ID,
			"goods_receipt_number": gr.GRNumber,
			"updated_by_id":        createdByID,
			"version":              gorm.Expr("version + 1"),
		}
		if req.CompletionNote != "" {
			notes := wo.Notes
			if notes != "" {
				notes += "\n"
			}
			updates["notes"] = notes + "[Completion] " + req.CompletionNote
		}
		if completeJEID != nil {
			updates["complete_je_id"] = *completeJEID
		}
		return tx.Model(&models.WorkOrder{}).Where("doc_entry = ?", id).Updates(updates).Error
	})
	if txErr != nil {
		return nil, txErr
	}

	// ── Build result: refreshed order + yield analytics ──────────────────────────
	mo, _ := GetMillingOrder(id)

	var recoveryRate float64
	if inputLine.PlannedQty > 0 {
		recoveryRate = math.Round((totalActualQty/inputLine.PlannedQty)*10000) / 100 // to 2 dp
	}

	var variances []LineVariance
	for _, l := range req.Lines {
		if l.ExpectedQty <= 0 || l.ActualQty <= 0 {
			continue
		}
		outItem, ok := outputMap[l.OutputItemID]
		if !ok {
			continue
		}
		varPct := math.Round(((l.ActualQty-l.ExpectedQty)/l.ExpectedQty)*10000) / 100
		variances = append(variances, LineVariance{
			OutputItemCode: outItem.ItemCode,
			OutputItemName: outItem.ItemName,
			ExpectedQty:    l.ExpectedQty,
			ActualQty:      l.ActualQty,
			VariancePct:    varPct,
		})
	}

	return &CompleteMillingResult{
		Order:        mo,
		RecoveryRate: recoveryRate,
		Variances:    variances,
	}, nil
}

// ─────────────────────────────────────────────
// Milling Orders — Cancel
// ─────────────────────────────────────────────

func CancelMillingOrder(id uint, userID *uint) error {
	var wo models.WorkOrder
	if err := db.DB.First(&wo, id).Error; err != nil {
		return err
	}

	now := time.Now()

	switch wo.Status {
	case "L":
		return errors.New("milling order is already cancelled")

	case "P":
		// Draft — no inventory or accounting side-effects yet; just mark cancelled.
		return db.DB.Model(&models.WorkOrder{}).Where("doc_entry = ?", id).
			Updates(map[string]interface{}{
				"status":       "L",
				"updated_by_id": userID,
				"version":      gorm.Expr("version + 1"),
			}).Error

	case "R":
		// In Progress — one GI + one start JE exist; reverse both then cancel.
		if wo.GoodsIssueID != nil {
			if err := invsvc.CancelGoodsIssue(*wo.GoodsIssueID, userID); err != nil {
				return fmt.Errorf("cancel goods issue: %w", err)
			}
		}
		if wo.StartJEID != nil {
			var startJE models.JournalEntry
			if err := db.DB.First(&startJE, *wo.StartJEID).Error; err == nil {
				db.DB.Transaction(func(tx *gorm.DB) error {
					_, revErr := accountingsvc.ReverseJournalEntry(tx, &startJE, &now, userID)
					return revErr
				})
			}
		}
		return db.DB.Model(&models.WorkOrder{}).Where("doc_entry = ?", id).
			Updates(map[string]interface{}{
				"status":       "L",
				"updated_by_id": userID,
				"version":      gorm.Expr("version + 1"),
			}).Error

	case "C":
		// Completed — GR + complete JE + GI + start JE all exist.
		// Reverse in the correct order:
		//   1. Cancel GR      → removes output items (rice, bran, etc.) from stock
		//   2. Reverse complete JE → Dr WIP / Cr Inventory (undoes WIP→Inventory transfer)
		//   3. Cancel GI      → restores paddy stock
		//   4. Reverse start JE  → Dr Inventory / Cr WIP (undoes Inventory→WIP transfer)

		if wo.GoodsReceiptID != nil {
			if err := invsvc.CancelGoodsReceipt(*wo.GoodsReceiptID, userID); err != nil {
				return fmt.Errorf("cancel goods receipt: %w", err)
			}
		}
		if wo.CompleteJEID != nil {
			var completeJE models.JournalEntry
			if err := db.DB.First(&completeJE, *wo.CompleteJEID).Error; err == nil {
				db.DB.Transaction(func(tx *gorm.DB) error {
					_, revErr := accountingsvc.ReverseJournalEntry(tx, &completeJE, &now, userID)
					return revErr
				})
			}
		}
		if wo.GoodsIssueID != nil {
			if err := invsvc.CancelGoodsIssue(*wo.GoodsIssueID, userID); err != nil {
				return fmt.Errorf("cancel goods issue: %w", err)
			}
		}
		if wo.StartJEID != nil {
			var startJE models.JournalEntry
			if err := db.DB.First(&startJE, *wo.StartJEID).Error; err == nil {
				db.DB.Transaction(func(tx *gorm.DB) error {
					_, revErr := accountingsvc.ReverseJournalEntry(tx, &startJE, &now, userID)
					return revErr
				})
			}
		}
		// Clear GR/JE references and mark cancelled.
		return db.DB.Model(&models.WorkOrder{}).Where("doc_entry = ?", id).
			Updates(map[string]interface{}{
				"status":               "L",
				"goods_receipt_id":     nil,
				"goods_receipt_number": "",
				"complete_je_id":       nil,
				"updated_by_id":        userID,
				"version":              gorm.Expr("version + 1"),
			}).Error

	default:
		return fmt.Errorf("unknown milling order status: %s", wo.Status)
	}
}

// ─────────────────────────────────────────────
// BOM (OITT / ITT1) — CRUD
// ─────────────────────────────────────────────

func ListBOMs() ([]models.ProductTree, error) {
	var boms []models.ProductTree
	if err := db.DB.Order("code").Find(&boms).Error; err != nil {
		return nil, err
	}
	if len(boms) == 0 {
		return boms, nil
	}
	codes := make([]string, len(boms))
	for i, b := range boms {
		codes[i] = b.Code
	}
	var lines []models.ProductTreeLine
	db.DB.Where("code IN ?", codes).Find(&lines)
	linesByCode := make(map[string][]models.ProductTreeLine)
	for _, l := range lines {
		linesByCode[l.Code] = append(linesByCode[l.Code], l)
	}
	for i := range boms {
		boms[i].Lines = linesByCode[boms[i].Code]
	}
	return boms, nil
}

func GetBOM(code string) (*models.ProductTree, error) {
	var bom models.ProductTree
	if err := db.DB.Where("code = ?", code).First(&bom).Error; err != nil {
		return nil, err
	}
	db.DB.Where("code = ?", code).Find(&bom.Lines)
	return &bom, nil
}

func UpsertBOM(bom *models.ProductTree) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		// Delete existing component lines
		if err := tx.Where("code = ?", bom.Code).Delete(&models.ProductTreeLine{}).Error; err != nil {
			return err
		}
		// Save (insert or update) the header
		bom.UpdatedAt = time.Now()
		if err := tx.Save(bom).Error; err != nil {
			return err
		}
		// Insert fresh component lines
		for i := range bom.Lines {
			bom.Lines[i].ID = 0 // let DB assign new ID
			bom.Lines[i].Code = bom.Code
			if err := tx.Create(&bom.Lines[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func DeleteBOM(code string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("code = ?", code).Delete(&models.ProductTreeLine{}).Error; err != nil {
			return err
		}
		return tx.Where("code = ?", code).Delete(&models.ProductTree{}).Error
	})
}

// ─────────────────────────────────────────────
// Work Orders (OWOR / WOR1) — List / Get
// ─────────────────────────────────────────────

func ListWorkOrders() ([]models.WorkOrder, error) {
	var wos []models.WorkOrder
	err := db.DB.Where("order_type = 'P'").Order("created_at DESC").Find(&wos).Error
	return wos, err
}

func GetWorkOrder(docEntry uint) (*models.WorkOrder, error) {
	var wo models.WorkOrder
	if err := db.DB.First(&wo, docEntry).Error; err != nil {
		return nil, err
	}
	db.DB.Where("doc_entry = ?", docEntry).Order("line_num").Find(&wo.Lines)
	return &wo, nil
}

// ─────────────────────────────────────────────
// Work Orders — Create (Planned)
// ─────────────────────────────────────────────

func CreateWorkOrder(req CreateWORequest, userID *uint) (*models.WorkOrder, error) {
	if req.ItemCode == "" {
		return nil, errors.New("item code is required")
	}
	if req.PlannedQty <= 0 {
		return nil, errors.New("planned quantity must be greater than zero")
	}

	var item models.OITM
	if err := db.DB.Where("item_code = ? AND mak_item = 'Y'", req.ItemCode).First(&item).Error; err != nil {
		return nil, fmt.Errorf("item not found or not a production item: %w", err)
	}

	// Validate category visibility for production
	var woCat models.OITB
	if err := db.DB.First(&woCat, item.ItmsGrpCod).Error; err == nil && !woCat.ForProduction {
		return nil, fmt.Errorf("item %s belongs to category '%s' which is not enabled for Production", item.ItemCode, woCat.ItmsGrpNam)
	}

	bom, err := GetBOM(req.ItemCode)
	if err != nil {
		return nil, fmt.Errorf("BOM not found for item %s: %w", req.ItemCode, err)
	}
	if len(bom.Lines) == 0 {
		return nil, fmt.Errorf("BOM for %s has no component lines", req.ItemCode)
	}

	woNum, err := docnumber.Svc.NextWONumber()
	if err != nil {
		return nil, fmt.Errorf("generate WO number: %w", err)
	}

	var createdByID uint
	if userID != nil {
		createdByID = *userID
	}

	whs := req.Warehouse
	if whs == "" {
		whs = item.DfltWh
	}
	if whs == "" {
		whs = "WH01"
	}

	// Scale component quantities proportionally to PlannedQty / BOM.Quantity
	scale := req.PlannedQty
	if bom.Quantity > 0 {
		scale = req.PlannedQty / bom.Quantity
	}

	wo := models.WorkOrder{
		DocNum:      woNum,
		OrderType:   "P",
		ItemCode:    item.ItemCode,
		ItemName:    item.ItemName,
		PlannedQty:  req.PlannedQty,
		Warehouse:   whs,
		StartDate:   req.StartDate,
		DueDate:     req.DueDate,
		Status:      "P",
		Notes:       req.Notes,
		CreatedByID: createdByID,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&wo).Error; err != nil {
			return err
		}
		for i, bl := range bom.Lines {
			// Look up component item to get price and default warehouse
			var compItem models.OITM
			tx.Where("item_code = ?", bl.ItemCode).First(&compItem)
			lineWhs := bl.Warehouse
			if lineWhs == "" {
				lineWhs = compItem.DfltWh
			}
			if lineWhs == "" {
				lineWhs = whs
			}
			line := models.WorkOrderLine{
				DocEntry:    wo.DocEntry,
				LineNum:     i,
				LineDir:     "I",
				ItemID:      compItem.ID,
				ItemCode:    bl.ItemCode,
				ItemName:    bl.ItemName,
				PlannedQty:  bl.Quantity * scale,
				Warehouse:   lineWhs,
				IssueMethod: bl.IssueMethod,
				UomEntry:    bl.UomEntry,
				UomCode:     bl.UomCode,
				Price:       compItem.AvgPrice,
			}
			if err := tx.Create(&line).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return GetWorkOrder(wo.DocEntry)
}

// ─────────────────────────────────────────────
// Work Orders — Release (Planned → Released)
// ─────────────────────────────────────────────

// ReleaseWorkOrder issues Backflush components from stock (IssueMethod='B').
// Manual components (IssueMethod='M') are left for the user to issue separately.
func ReleaseWorkOrder(docEntry uint, userID *uint) error {
	wo, err := GetWorkOrder(docEntry)
	if err != nil {
		return err
	}
	if wo.Status != "P" {
		return errors.New("work order must be in Planned status to release")
	}

	var uid uint
	if userID != nil {
		uid = *userID
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, line := range wo.Lines {
			if line.LineDir != "I" || line.IssueMethod != "B" {
				continue
			}
			var item models.OITM
			if err := tx.Where("item_code = ?", line.ItemCode).First(&item).Error; err != nil {
				return fmt.Errorf("item %s not found: %w", line.ItemCode, err)
			}
			if item.OnHand < line.PlannedQty {
				return fmt.Errorf("insufficient stock for %s: have %.3f, need %.3f",
					line.ItemCode, item.OnHand, line.PlannedQty)
			}
			if err := tx.Model(&models.OITM{}).Where("item_code = ?", line.ItemCode).
				UpdateColumn("on_hand", gorm.Expr("on_hand - ?", line.PlannedQty)).Error; err != nil {
				return err
			}
			if err := invsvc.UpsertOITWOnHand(tx, line.ItemCode, line.Warehouse, item.DfltWh, -line.PlannedQty); err != nil {
				return err
			}
			if err := invsvc.AppendOIVL(tx, line.ItemCode, item.ItemName, line.Warehouse, item.DfltWh,
				"WO_ISSUE", wo.StartDate, int(wo.DocEntry), 0, line.PlannedQty, item.AvgPrice, uid); err != nil {
				return err
			}
			if err := tx.Model(&models.WorkOrderLine{}).Where("id = ?", line.ID).
				UpdateColumn("issued_qty", line.PlannedQty).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.WorkOrder{}).Where("doc_entry = ?", docEntry).
			Updates(map[string]interface{}{"status": "R", "updated_by_id": uid}).Error
	})
}

// ─────────────────────────────────────────────
// Work Orders — Close (Released → Closed)
// ─────────────────────────────────────────────

// CloseWorkOrder receives the finished goods into stock and marks the WO closed.
func CloseWorkOrder(docEntry uint, userID *uint, cmpltQty, rjctQty float64) error {
	wo, err := GetWorkOrder(docEntry)
	if err != nil {
		return err
	}
	if wo.Status != "R" {
		return errors.New("work order must be in Released status to close")
	}
	if cmpltQty <= 0 {
		return errors.New("completed quantity must be greater than zero")
	}

	var item models.OITM
	if err := db.DB.Where("item_code = ?", wo.ItemCode).First(&item).Error; err != nil {
		return fmt.Errorf("finished item %s not found: %w", wo.ItemCode, err)
	}

	var uid uint
	if userID != nil {
		uid = *userID
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		// Receive finished goods into stock
		if err := tx.Model(&models.OITM{}).Where("item_code = ?", wo.ItemCode).
			UpdateColumn("on_hand", gorm.Expr("on_hand + ?", cmpltQty)).Error; err != nil {
			return err
		}
		if err := invsvc.UpsertOITWOnHand(tx, wo.ItemCode, wo.Warehouse, item.DfltWh, cmpltQty); err != nil {
			return err
		}
		if err := invsvc.AppendOIVL(tx, wo.ItemCode, item.ItemName, wo.Warehouse, item.DfltWh,
			"WO_RECEIPT", wo.DueDate, int(wo.DocEntry), cmpltQty, 0, item.AvgPrice, uid); err != nil {
			return err
		}
		return tx.Model(&models.WorkOrder{}).Where("doc_entry = ?", docEntry).Updates(map[string]interface{}{
			"status":        "C",
			"cmplt_qty":     cmpltQty,
			"rjct_qty":      rjctQty,
			"updated_by_id": uid,
		}).Error
	})
}

// ─────────────────────────────────────────────
// Work Orders — Cancel
// ─────────────────────────────────────────────

// CancelWorkOrder cancels a Planned (P) or Released (R) work order.
// If Released, issued stock is restored for Backflush lines.
func CancelWorkOrder(docEntry uint, userID *uint) error {
	wo, err := GetWorkOrder(docEntry)
	if err != nil {
		return err
	}

	var uid uint
	if userID != nil {
		uid = *userID
	}

	switch wo.Status {
	case "C":
		return errors.New("closed work orders cannot be cancelled")
	case "L":
		return errors.New("work order is already cancelled")
	case "P":
		return db.DB.Model(&models.WorkOrder{}).Where("doc_entry = ?", docEntry).
			Updates(map[string]interface{}{"status": "L", "updated_by_id": uid}).Error
	case "R":
		// Restore issued quantities for Backflush lines
		return db.DB.Transaction(func(tx *gorm.DB) error {
			for _, line := range wo.Lines {
				if line.LineDir != "I" || line.IssuedQty <= 0 {
					continue
				}
				var item models.OITM
				if err := tx.Where("item_code = ?", line.ItemCode).First(&item).Error; err != nil {
					continue
				}
				if err := tx.Model(&models.OITM{}).Where("item_code = ?", line.ItemCode).
					UpdateColumn("on_hand", gorm.Expr("on_hand + ?", line.IssuedQty)).Error; err != nil {
					return err
				}
				if err := invsvc.UpsertOITWOnHand(tx, line.ItemCode, line.Warehouse, item.DfltWh, line.IssuedQty); err != nil {
					return err
				}
				if err := invsvc.AppendOIVL(tx, line.ItemCode, item.ItemName, line.Warehouse, item.DfltWh,
					"WO_CANCEL", wo.StartDate, int(wo.DocEntry), line.IssuedQty, 0, item.AvgPrice, uid); err != nil {
					return err
				}
			}
			return tx.Model(&models.WorkOrder{}).Where("doc_entry = ?", docEntry).
				Updates(map[string]interface{}{"status": "L", "updated_by_id": uid}).Error
		})
	default:
		return fmt.Errorf("unknown status: %s", wo.Status)
	}
}
