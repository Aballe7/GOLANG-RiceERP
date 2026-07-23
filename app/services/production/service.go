package production

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
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
	OrderType     string  `json:"order_type"`     // "M"=Milling (default) | "D"=Drying
	BatchNo       string  `json:"batch_no"`       // lot number; defaults to the MO number
	MoisturePct   float64 `json:"moisture_pct"`   // paddy moisture % at intake (0 = not measured)
}

// CompleteMillingLineInput captures actual output recorded at completion time.
type CompleteMillingLineInput struct {
	OutputItemID uint    `json:"output_item_id"`
	ActualQty    float64 `json:"actual_qty"`
	ExpectedQty  float64 `json:"expected_qty"` // optional: expected qty for variance reporting (0 = use yield template)
	UomEntry     uint    `json:"uom_entry"`
	OutputType   string  `json:"output_type"` // H=head rice B=brokens Y=by-product; blank = use template / unclassified
}

type CompleteMillingRequest struct {
	Lines          []CompleteMillingLineInput `json:"lines"`
	CompletionNote string                     `json:"completion_note"`
	CompletionDate string                     `json:"completion_date"` // "YYYY-MM-DD"; defaults to today
	RejectedQty    float64                    `json:"rejected_qty"`    // spillage / rejected mass (explicit process loss)
	OutMoisturePct float64                    `json:"out_moisture_pct"` // output moisture % at completion (0 = not measured)
	ConversionCost float64                    `json:"conversion_cost"` // per-order override; 0 = use milling_conv_cost_per_kg × input qty
	// AllowOutOfBandRecovery lets the operator post a completion whose milled-rice
	// recovery falls outside the configured band after explicitly confirming it.
	AllowOutOfBandRecovery bool `json:"allow_out_of_band"`
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
	Order          *models.MillingOrder `json:"order"`
	RecoveryRate   float64              `json:"recovery_rate"`   // totalOutputQty / inputQty × 100 (mass balance)
	MilledRecovery float64              `json:"milled_recovery"` // (head rice + brokens) / inputQty × 100 — the MRR KPI
	HeadRicePct    float64              `json:"head_rice_pct"`   // head rice / inputQty × 100
	BrokenPct      float64              `json:"broken_pct"`      // brokens / inputQty × 100
	ProcessLossPct float64              `json:"process_loss_pct"` // unexplained mass gap after outputs + rejected qty
	ConversionCost float64              `json:"conversion_cost"` // absorbed into output cost this completion
	Variances      []LineVariance       `json:"variances"`       // lines with a known expected qty (template or explicit)
	Warnings       []string             `json:"warnings,omitempty"`
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
// Settings helpers
// ─────────────────────────────────────────────

// settingFloat reads a numeric FarmSettings value, falling back to def when the
// key is missing or not parseable.
func settingFloat(key string, def float64) float64 {
	var s models.FarmSettings
	if err := db.DB.Where("`key` = ?", key).First(&s).Error; err != nil {
		return def
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s.Value), 64)
	if err != nil {
		return def
	}
	return v
}

// recoveryBand returns the configured acceptable milled-rice recovery band
// (percent of paddy input). Industry reference: total milled-rice recovery of
// 68–72% (IRRI); the default band of 60–75 gives operating headroom while still
// catching shrinkage/theft or data-entry errors at posting time.
func recoveryBand() (min, max float64) {
	min = settingFloat("milling_recovery_min", 60)
	max = settingFloat("milling_recovery_max", 75)
	if max <= min { // nonsense configuration — fall back to defaults
		return 60, 75
	}
	return min, max
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
		OrderType:          wo.OrderType,
		BatchNo:            wo.BatchNo,
		MoisturePct:        wo.MoisturePct,
		OutMoisturePct:     wo.OutMoisturePct,
		RejectedQty:        wo.RjctQty,
		ConvCost:           wo.ConvCost,
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
			ml := models.MillingOrderLine{
				ID:                 l.ID,
				MillingOrderID:     wo.DocEntry,
				LineNum:            l.LineNum,
				OutputItemID:       l.ItemID,
				OutputItemCode:     l.ItemCode,
				OutputItemName:     l.ItemName,
				OutputItemCategory: l.ItemCategory,
				ExpectedQty:        l.PlannedQty, // yield-template expectation (0 when no template)
				ActualQty:          l.IssuedQty,
				UomEntry:           l.UomEntry,
				Unit:               l.UomCode,
				OutputType:         l.OutputType,
			}
			if inputLine.PlannedQty > 0 && l.PlannedQty > 0 {
				ml.YieldPct = math.Round((l.PlannedQty/inputLine.PlannedQty)*10000) / 100
			}
			mo.Lines = append(mo.Lines, ml)
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

// outputSaleUnitValues returns a per-unit "sale value" for each output item, keyed
// by OITM ID, used to allocate joint milling cost by relative sales value (finding
// #6). It prefers the item's price-list price (lowest price_list with a positive
// price), falling back to the item's moving-average cost. Items with neither get 0,
// which makes the caller fall back to a quantity-weighted split.
func outputSaleUnitValues(items map[uint]models.OITM) map[uint]float64 {
	weights := make(map[uint]float64, len(items))
	codes := make([]string, 0, len(items))
	for _, it := range items {
		codes = append(codes, it.ItemCode)
	}
	priceByCode := make(map[string]float64)
	if len(codes) > 0 {
		var prices []models.ITM1
		// Ordered by price_list so the first row seen per item is the lowest list.
		if err := db.DB.Where("item_code IN ? AND price > 0", codes).Order("price_list").Find(&prices).Error; err == nil {
			for _, p := range prices {
				if _, ok := priceByCode[p.ItemCode]; !ok {
					priceByCode[p.ItemCode] = p.Price
				}
			}
		}
	}
	for id, it := range items {
		switch {
		case priceByCode[it.ItemCode] > 0:
			weights[id] = priceByCode[it.ItemCode]
		case it.AvgPrice > 0:
			weights[id] = it.AvgPrice
		default:
			weights[id] = 0
		}
	}
	return weights
}

// ─────────────────────────────────────────────
// Milling Orders — List / Get
// ─────────────────────────────────────────────

func ListMillingOrders() ([]models.MillingOrder, error) {
	var wos []models.WorkOrder
	// 'M' = milling, 'D' = drying — both run through the same input/output machinery.
	if err := db.DB.Where("order_type IN ('M','D')").Order("created_at DESC").Find(&wos).Error; err != nil {
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
	orderType := req.OrderType
	if orderType == "" {
		orderType = "M"
	}
	if orderType != "M" && orderType != "D" {
		return nil, fmt.Errorf("unknown order type %q — must be 'M' (milling) or 'D' (drying)", req.OrderType)
	}
	if req.MoisturePct < 0 || req.MoisturePct > 100 {
		return nil, errors.New("moisture % must be between 0 and 100")
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

	// Batch/lot number: default to the MO number so every order is traceable even
	// when the operator doesn't assign an explicit lot.
	batchNo := strings.TrimSpace(req.BatchNo)
	if batchNo == "" {
		batchNo = moNum
	}

	wo := models.WorkOrder{
		DocNum:      moNum,
		OrderType:   orderType,
		ItemCode:    inputItem.ItemCode,
		ItemName:    inputItem.ItemName,
		PlannedQty:  req.InputQty,
		StartDate:   req.PostingDate,
		DueDate:     req.ExpectedDate,
		Status:      "P",
		Warehouse:   req.WarehouseCode,
		Notes:       req.Remarks,
		BatchNo:     batchNo,
		MoisturePct: req.MoisturePct,
		CreatedByID: createdByID,
	}

	// Yield template (finding #1): a ProductTree whose tree_type matches the order
	// type and whose code is the INPUT item describes the expected outputs
	// (e.g. 100 kg paddy → 62 head rice / 8 brokens / 10 bran / 20 husk).
	recipe, err := GetMillingRecipe(orderType, inputItem.ItemCode)
	if err != nil {
		return nil, err
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
		if err := tx.Create(&inputLine).Error; err != nil {
			return err
		}

		// Auto-create expected output lines from the yield template so the variance
		// report at completion is automatic instead of operator-entered.
		if recipe != nil {
			base := recipe.Quantity
			if base <= 0 {
				base = 1
			}
			for i, rl := range recipe.Lines {
				var outItem models.OITM
				if err := tx.Where("item_code = ?", rl.ItemCode).First(&outItem).Error; err != nil {
					return fmt.Errorf("yield template output %s not found: %w", rl.ItemCode, err)
				}
				outLine := models.WorkOrderLine{
					DocEntry:     wo.DocEntry,
					LineNum:      i + 1,
					LineDir:      "O",
					ItemID:       outItem.ID,
					ItemCode:     outItem.ItemCode,
					ItemName:     outItem.ItemName,
					ItemCategory: invsvc.GetItemCategoryName(outItem.ItmsGrpCod, tx),
					PlannedQty:   math.Round((req.InputQty*rl.Quantity/base)*1000) / 1000,
					UomEntry:     rl.UomEntry,
					UomCode:      outItem.InvntryUom,
					Warehouse:    req.WarehouseCode,
					OutputType:   rl.OutputType,
				}
				if outLine.UomEntry == 0 {
					outLine.UomEntry = outItem.IUoMEntry
				}
				if err := tx.Create(&outLine).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return GetMillingOrder(wo.DocEntry)
}

// GetMillingRecipe loads the yield template for an input item, if one exists.
// treeType is 'M' (milling) or 'D' (drying). A missing template is not an error.
func GetMillingRecipe(treeType, inputItemCode string) (*models.ProductTree, error) {
	var tree models.ProductTree
	err := db.DB.Where("code = ? AND tree_type = ?", inputItemCode, treeType).First(&tree).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := db.DB.Where("code = ?", tree.Code).Find(&tree.Lines).Error; err != nil {
		return nil, err
	}
	return &tree, nil
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

	// Everything below — status claim, goods issue, and journal entry — runs in ONE
	// transaction. If any step fails the whole thing rolls back, so we can never
	// consume paddy without advancing the order (no double-issue on retry) and the
	// stock ledger can never diverge from the general ledger.
	return db.DB.Transaction(func(tx *gorm.DB) error {
		// ── Claim the order: flip Draft(P) → In Progress(R) atomically. ──────────
		// This is the concurrency guard (finding #3): if another Start already
		// advanced this order, RowsAffected == 0 and we abort before issuing stock.
		claim := tx.Model(&models.WorkOrder{}).
			Where("doc_entry = ? AND status = 'P'", id).
			Updates(map[string]interface{}{
				"status":        "R",
				"updated_by_id": createdByID,
				"version":       gorm.Expr("version + 1"),
			})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			return errors.New("milling order was already started by another operation")
		}

		// ── Re-read input cost at issue time (finding #5). ───────────────────────
		// The moving-average cost may have moved since the order was created; issue
		// at the current cost and persist it back onto the input line so the
		// completion JE and WIP balance agree.
		var freshItem models.OITM
		if err := tx.First(&freshItem, inputLine.ItemID).Error; err == nil && freshItem.AvgPrice > 0 {
			inputLine.Price = freshItem.AvgPrice
			if err := tx.Model(&models.WorkOrderLine{}).Where("id = ?", inputLine.ID).
				UpdateColumn("price", inputLine.Price).Error; err != nil {
				return err
			}
		}

		// ── Goods Issue to consume input paddy (same transaction). ───────────────
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
		gi, err := invsvc.CreateGoodsIssueTx(tx, giReq, userID)
		if err != nil {
			return fmt.Errorf("create goods issue: %w", err)
		}

		updates := map[string]interface{}{
			"goods_issue_id":     gi.ID,
			"goods_issue_number": gi.GINumber,
		}

		// ── Journal entry: Dr WIP / Cr Inventory (input paddy). ──────────────────
		// A failure here now aborts the transaction (finding #2) rather than being
		// silently swallowed and leaving inventory and the GL out of sync.
		totalCost := inputLine.PlannedQty * inputLine.Price
		if totalCost > 0 {
			now := time.Now()
			je, err := accountingsvc.PostJournalEntry(tx, accountingsvc.PostJEParams{
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
			})
			if err != nil {
				return fmt.Errorf("post start journal entry: %w", err)
			}
			updates["start_je_id"] = je.ID
		}

		return tx.Model(&models.WorkOrder{}).Where("doc_entry = ?", id).Updates(updates).Error
	})
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
	if req.RejectedQty < 0 {
		return nil, errors.New("rejected/spillage quantity cannot be negative")
	}
	if req.OutMoisturePct < 0 || req.OutMoisturePct > 100 {
		return nil, errors.New("moisture % must be between 0 and 100")
	}

	inputLine, templateLines := loadWOLines(wo.DocEntry)

	// ── Pre-flight: validate MILLING_WIP account before touching inventory ───────
	// MILLING_WIP is required for both the debit and credit sides of the completion
	// JE. Output category accounts (MILLING_OUTPUT) fall back to WIP when missing,
	// so only WIP itself is a hard requirement here.
	wipAccount, err := accountingsvc.GetAccount(nil, "PRODUCTION", "MILLING_WIP")
	if err != nil || wipAccount == nil {
		return nil, fmt.Errorf("cannot complete milling order: MILLING_WIP GL account is not configured — set it up under Account Determination before proceeding")
	}

	// ── Conversion cost (finding: absorb conversion costs into WIP) ──────────────
	// Electricity, labor, and machine overhead are significant in milling; leaving
	// them out understates rice cost. Use the per-order override when supplied,
	// otherwise the per-kg standard from settings × input quantity. The absorbed
	// amount is credited to MILLING_OVERHEAD in the completion JE.
	convCost := req.ConversionCost
	if convCost < 0 {
		return nil, errors.New("conversion cost cannot be negative")
	}
	if convCost == 0 {
		convCost = settingFloat("milling_conv_cost_per_kg", 0) * inputLine.PlannedQty
	}
	convCost = math.Round(convCost*100) / 100
	var overheadAccount *models.GLAccount
	if convCost > 0 {
		overheadAccount, err = accountingsvc.GetAccount(nil, "PRODUCTION", "MILLING_OVERHEAD")
		if err != nil || overheadAccount == nil {
			return nil, fmt.Errorf("cannot absorb conversion cost of %.2f: MILLING_OVERHEAD GL account is not configured — set it up under Account Determination, or set the conversion cost to zero", convCost)
		}
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

	// ── Cost basis (finding #6: relative-sales-value joint-cost allocation) ──────
	// totalInputCost is the full cost of raw material consumed. It was issued at
	// Start using the re-read moving-average cost and persisted onto the input line,
	// so it agrees with the WIP debit. totalCost adds the absorbed conversion cost —
	// the full amount that flows into output inventory.
	//
	// We split it across outputs by RELATIVE SALES VALUE — each output's share is
	// proportional to (qty × unit sale price) — so low-value by-products (husk,
	// bran) carry a fraction of head rice's per-kg cost instead of the same uniform
	// rate. When no sale prices are known we fall back to a quantity-weighted split.
	totalInputCost := inputLine.PlannedQty * inputLine.Price
	totalCost := totalInputCost + convCost

	// Yield-template lines (LineDir='O', created with the order) keyed by item ID.
	// They supply expected qty and output classification when the request doesn't.
	templateByItem := make(map[uint]*models.WorkOrderLine, len(templateLines))
	maxLineNum := 0
	for i := range templateLines {
		if templateLines[i].LineNum > maxLineNum {
			maxLineNum = templateLines[i].LineNum
		}
		if _, dup := templateByItem[templateLines[i].ItemID]; !dup {
			templateByItem[templateLines[i].ItemID] = &templateLines[i]
		}
	}

	// Collect valid output lines in request order.
	type validOutput struct {
		item     models.OITM
		qty      float64
		uom      uint
		outType  string                // H | B | Y | ""
		expected float64               // request > template > 0
		tmpl     *models.WorkOrderLine // matched template line, nil when ad-hoc
	}
	valid := make([]validOutput, 0, len(req.Lines))
	claimedTmpl := make(map[uint]bool)
	totalActualQty := 0.0
	for _, l := range req.Lines {
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
		v := validOutput{item: outItem, qty: l.ActualQty, uom: uom, outType: l.OutputType, expected: l.ExpectedQty}
		if tmpl, ok := templateByItem[l.OutputItemID]; ok && !claimedTmpl[l.OutputItemID] {
			claimedTmpl[l.OutputItemID] = true
			v.tmpl = tmpl
			if v.expected <= 0 {
				v.expected = tmpl.PlannedQty
			}
			if v.outType == "" {
				v.outType = tmpl.OutputType
			}
		}
		// Classification fallback: anything in the Milled Rice category that isn't
		// explicitly typed counts as head rice for the recovery KPIs.
		if v.outType == "" && outItem.CategoryName == "Milled Rice" {
			v.outType = "H"
		}
		if v.outType != "" && v.outType != "H" && v.outType != "B" && v.outType != "Y" {
			return nil, fmt.Errorf("invalid output type %q for %s — must be H (head rice), B (brokens), or Y (by-product)", v.outType, outItem.ItemCode)
		}
		valid = append(valid, v)
		totalActualQty += l.ActualQty
	}
	if len(valid) == 0 {
		return nil, errors.New("no valid output lines to record")
	}

	// ── Recovery-band validation (finding #1) ─────────────────────────────────────
	// Milled-rice recovery (head rice + brokens, per PSA/IRRI convention) outside
	// the configured band signals shrinkage/theft or a data-entry error; catch it at
	// posting time. The operator can override explicitly. Drying orders and
	// completions with no classified rice output are exempt.
	var headQty, brokenQty float64
	for _, v := range valid {
		switch v.outType {
		case "H":
			headQty += v.qty
		case "B":
			brokenQty += v.qty
		}
	}
	var warnings []string
	milledRecovery := 0.0
	if inputLine.PlannedQty > 0 {
		milledRecovery = math.Round(((headQty+brokenQty)/inputLine.PlannedQty)*10000) / 100
	}
	if wo.OrderType != "D" && headQty+brokenQty > 0 {
		bandMin, bandMax := recoveryBand()
		if milledRecovery < bandMin || milledRecovery > bandMax {
			if !req.AllowOutOfBandRecovery {
				return nil, fmt.Errorf(
					"milled-rice recovery of %.2f%% is outside the acceptable band (%.0f%%–%.0f%%) — verify the recorded quantities; if they are correct, confirm the override to post anyway",
					milledRecovery, bandMin, bandMax)
			}
			warnings = append(warnings, fmt.Sprintf(
				"milled-rice recovery of %.2f%% is outside the acceptable band (%.0f%%–%.0f%%) — posted with operator override",
				milledRecovery, bandMin, bandMax))
		}
	}

	// Relative-sales-value weight per line = qty × unit sale value.
	saleUnitVals := outputSaleUnitValues(outputMap)
	weights := make([]float64, len(valid))
	totalWeight := 0.0
	for i, v := range valid {
		w := v.qty * saleUnitVals[v.item.ID]
		weights[i] = w
		totalWeight += w
	}
	if totalWeight <= 0 {
		// No sale prices anywhere — fall back to quantity weighting.
		totalWeight = 0
		for i, v := range valid {
			weights[i] = v.qty
			totalWeight += v.qty
		}
	}

	// Allocate totalCost (input + conversion) across lines; the last line absorbs
	// the rounding remainder so the sum of debits equals the credits exactly.
	allocatedCost := make([]float64, len(valid))
	unitCost := make([]float64, len(valid))
	allocatedSoFar := 0.0
	for i := range valid {
		var lineCost float64
		if i == len(valid)-1 {
			lineCost = math.Round((totalCost-allocatedSoFar)*100) / 100
		} else if totalWeight > 0 {
			lineCost = math.Round((totalCost*(weights[i]/totalWeight))*100) / 100
		}
		if lineCost < 0 {
			lineCost = 0
		}
		allocatedSoFar += lineCost
		allocatedCost[i] = lineCost
		if valid[i].qty > 0 {
			unitCost[i] = math.Round((lineCost/valid[i].qty)*10000) / 10000
		}
	}

	// Completion date: use caller-supplied date or fall back to today.
	completionDate := req.CompletionDate
	if completionDate == "" {
		completionDate = time.Now().Format("2006-01-02")
	}
	entryDate, _ := time.Parse("2006-01-02", completionDate)
	if entryDate.IsZero() {
		entryDate = time.Now()
	}

	// ── Build WOR1 output lines and GR lines with per-line unit cost. ───────────
	// Lines that match a yield-template line UPDATE it in place (actual against
	// expected); ad-hoc outputs append new lines after the template block.
	newWORLines := make([]models.WorkOrderLine, len(valid))
	grLines := make([]invsvc.GRLineInput, len(valid))
	nextLineNum := maxLineNum
	for i, v := range valid {
		if v.tmpl != nil {
			line := *v.tmpl // PlannedQty keeps the template expectation for variance history
			line.IssuedQty = v.qty
			line.UomEntry = v.uom
			line.Warehouse = wo.Warehouse
			line.Price = unitCost[i]
			line.OutputType = v.outType
			newWORLines[i] = line
		} else {
			nextLineNum++
			newWORLines[i] = models.WorkOrderLine{
				DocEntry:     wo.DocEntry,
				LineNum:      nextLineNum,
				LineDir:      "O",
				ItemID:       v.item.ID,
				ItemCode:     v.item.ItemCode,
				ItemName:     v.item.ItemName,
				ItemCategory: v.item.CategoryName,
				PlannedQty:   v.expected, // 0 when no expectation was given
				IssuedQty:    v.qty,      // actual qty produced
				UomEntry:     v.uom,
				UomCode:      v.item.InvntryUom,
				Warehouse:    wo.Warehouse,
				Price:        unitCost[i], // relative-sales-value unit cost
				OutputType:   v.outType,
			}
		}
		grLines[i] = invsvc.GRLineInput{
			ItemID:        v.item.ID,
			UomEntry:      v.uom,
			Quantity:      v.qty,
			Price:         unitCost[i],
			WarehouseCode: wo.Warehouse,
			BatchNo:       wo.BatchNo, // lot traceability: paddy intake → milled outputs
		}
	}

	// ── Single atomic transaction: status claim, GR, WOR1 lines, JE, header. ─────
	// If any step fails the whole completion rolls back, so output stock and the GL
	// can never diverge and a retry can never double-receive (findings #1, #2, #3).
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// Concurrency guard: only one completion may advance In Progress(R) → Completed(C).
		claim := tx.Model(&models.WorkOrder{}).
			Where("doc_entry = ? AND status = 'R'", id).
			Updates(map[string]interface{}{
				"status":           "C",
				"rjct_qty":         req.RejectedQty,   // explicit spillage / process loss (finding #8)
				"out_moisture_pct": req.OutMoisturePct, // quality capture at completion (finding #4)
				"conv_cost":        convCost,
				"updated_by_id":    createdByID,
				"version":          gorm.Expr("version + 1"),
			})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			return errors.New("milling order was already completed by another operation")
		}

		// Goods Receipt for all outputs (same transaction).
		grReq := invsvc.CreateGoodsReceiptRequest{
			PostingDate: completionDate,
			Remarks:     fmt.Sprintf("Milling Order %s — Output receipt", wo.DocNum),
			Lines:       grLines,
		}
		gr, err := invsvc.CreateGoodsReceiptTx(tx, grReq, userID)
		if err != nil {
			return fmt.Errorf("create goods receipt: %w", err)
		}

		// Persist WOR1 output lines: template-matched lines are updated in place
		// (ID != 0), ad-hoc outputs are inserted.
		for i := range newWORLines {
			if newWORLines[i].ID != 0 {
				if err := tx.Save(&newWORLines[i]).Error; err != nil {
					return err
				}
			} else if err := tx.Create(&newWORLines[i]).Error; err != nil {
				return err
			}
		}

		// Completion JE: Dr Inventory (per output, relative-value split) /
		// Cr WIP (input cost) + Cr Overhead Absorbed (conversion cost).
		// A failure here aborts the whole transaction (finding #2).
		var completeJEID *uint
		if totalCost > 0 {
			jeLines := make([]accountingsvc.JELine, 0, len(valid)+2)
			for i, v := range valid {
				outInvAccount, accErr := accountingsvc.GetAccount(tx, "PRODUCTION", "MILLING_OUTPUT", v.item.CategoryName)
				if accErr != nil || outInvAccount == nil {
					outInvAccount = wipAccount
				}
				jeLines = append(jeLines, accountingsvc.JELine{
					Account:     outInvAccount,
					Debit:       allocatedCost[i],
					Description: fmt.Sprintf("Inventory — %s (%.3f %s @ %.4f)", v.item.ItemName, v.qty, v.item.InvntryUom, unitCost[i]),
				})
			}
			if totalInputCost > 0 {
				jeLines = append(jeLines, accountingsvc.JELine{
					Account:     wipAccount,
					Credit:      math.Round(totalInputCost*100) / 100,
					Description: "WIP — " + wo.DocNum + " complete",
				})
			}
			if convCost > 0 {
				jeLines = append(jeLines, accountingsvc.JELine{
					Account:     overheadAccount,
					Credit:      convCost,
					Description: "Conversion cost absorbed — " + wo.DocNum,
				})
			}
			je, err := accountingsvc.PostJournalEntry(tx, accountingsvc.PostJEParams{
				Module:      "PRODUCTION",
				SourceType:  "MILLING_COMPLETE",
				SourceRef:   wo.DocNum,
				Narration:   fmt.Sprintf("Milling Order %s — WIP to output inventory", wo.DocNum),
				EntryDate:   &entryDate,
				CreatedByID: &createdByID,
				Lines:       jeLines,
			})
			if err != nil {
				return fmt.Errorf("post completion journal entry: %w", err)
			}
			completeJEID = &je.ID
		}

		// Finalize header fields (status already flipped by the claim above).
		updates := map[string]interface{}{
			"goods_receipt_id":     gr.ID,
			"goods_receipt_number": gr.GRNumber,
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
	if err != nil {
		return nil, err
	}

	// ── Build result: refreshed order + yield analytics ──────────────────────────
	mo, _ := GetMillingOrder(id)

	var recoveryRate, headRicePct, brokenPct, processLossPct float64
	if inputLine.PlannedQty > 0 {
		recoveryRate = math.Round((totalActualQty/inputLine.PlannedQty)*10000) / 100 // to 2 dp
		headRicePct = math.Round((headQty/inputLine.PlannedQty)*10000) / 100
		brokenPct = math.Round((brokenQty/inputLine.PlannedQty)*10000) / 100
		// Mass that went in and came out neither as product nor as recorded
		// spillage — the unexplained recovery gap (moisture loss, shrinkage, …).
		processLossPct = math.Round(((inputLine.PlannedQty-totalActualQty-req.RejectedQty)/inputLine.PlannedQty)*10000) / 100
	}

	// Variances are automatic: every line with a known expectation (template or
	// operator-entered) reports actual-vs-expected, and template outputs that were
	// never produced show up as -100%.
	var variances []LineVariance
	for _, v := range valid {
		if v.expected <= 0 {
			continue
		}
		varPct := math.Round(((v.qty-v.expected)/v.expected)*10000) / 100
		variances = append(variances, LineVariance{
			OutputItemCode: v.item.ItemCode,
			OutputItemName: v.item.ItemName,
			ExpectedQty:    v.expected,
			ActualQty:      v.qty,
			VariancePct:    varPct,
		})
	}
	for _, tl := range templateLines {
		if claimedTmpl[tl.ItemID] || tl.PlannedQty <= 0 {
			continue
		}
		variances = append(variances, LineVariance{
			OutputItemCode: tl.ItemCode,
			OutputItemName: tl.ItemName,
			ExpectedQty:    tl.PlannedQty,
			ActualQty:      0,
			VariancePct:    -100,
		})
	}

	return &CompleteMillingResult{
		Order:          mo,
		RecoveryRate:   recoveryRate,
		MilledRecovery: milledRecovery,
		HeadRicePct:    headRicePct,
		BrokenPct:      brokenPct,
		ProcessLossPct: processLossPct,
		ConversionCost: convCost,
		Variances:      variances,
		Warnings:       warnings,
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
	if wo.Status == "L" {
		return errors.New("milling order is already cancelled")
	}
	if wo.Status != "P" && wo.Status != "R" && wo.Status != "C" {
		return fmt.Errorf("unknown milling order status: %s", wo.Status)
	}

	now := time.Now()

	// All reversals run in ONE transaction (finding #1): cancelling a Completed order
	// chains GR reversal + completion-JE reversal + GI reversal + start-JE reversal,
	// and every step must succeed together or none of them do — no more half-reversed
	// books. Reversal/JE errors now propagate instead of being discarded (finding #2).
	return db.DB.Transaction(func(tx *gorm.DB) error {
		// Claim: mark cancelled only from the status we observed. If the order moved
		// on concurrently, RowsAffected == 0 and we abort before reversing anything.
		claim := tx.Model(&models.WorkOrder{}).
			Where("doc_entry = ? AND status = ?", id, wo.Status).
			Updates(map[string]interface{}{
				"status":        "L",
				"updated_by_id": userID,
				"version":       gorm.Expr("version + 1"),
			})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			return errors.New("milling order changed status concurrently — cancel aborted")
		}

		reverseJE := func(jeID *uint, label string) error {
			if jeID == nil {
				return nil
			}
			var je models.JournalEntry
			if err := tx.First(&je, *jeID).Error; err != nil {
				return fmt.Errorf("load %s journal entry: %w", label, err)
			}
			if _, err := accountingsvc.ReverseJournalEntry(tx, &je, &now, userID); err != nil {
				return fmt.Errorf("reverse %s journal entry: %w", label, err)
			}
			return nil
		}

		switch wo.Status {
		case "P":
			// Draft — no inventory or accounting side-effects yet.
			return nil

		case "R":
			// In Progress — one GI + one start JE exist; reverse both.
			if wo.GoodsIssueID != nil {
				if err := invsvc.CancelGoodsIssueTx(tx, *wo.GoodsIssueID, userID); err != nil {
					return fmt.Errorf("cancel goods issue: %w", err)
				}
			}
			return reverseJE(wo.StartJEID, "start")

		case "C":
			// Completed — reverse in order:
			//   1. Cancel GR         → removes output items (rice, bran, …) from stock
			//   2. Reverse complete JE → undoes WIP → Inventory transfer
			//   3. Cancel GI         → restores paddy stock
			//   4. Reverse start JE  → undoes Inventory → WIP transfer
			if wo.GoodsReceiptID != nil {
				if err := invsvc.CancelGoodsReceiptTx(tx, *wo.GoodsReceiptID, userID); err != nil {
					return fmt.Errorf("cancel goods receipt: %w", err)
				}
			}
			if err := reverseJE(wo.CompleteJEID, "completion"); err != nil {
				return err
			}
			if wo.GoodsIssueID != nil {
				if err := invsvc.CancelGoodsIssueTx(tx, *wo.GoodsIssueID, userID); err != nil {
					return fmt.Errorf("cancel goods issue: %w", err)
				}
			}
			if err := reverseJE(wo.StartJEID, "start"); err != nil {
				return err
			}
			// Clear GR/completion-JE references now that they are reversed.
			return tx.Model(&models.WorkOrder{}).Where("doc_entry = ?", id).
				Updates(map[string]interface{}{
					"goods_receipt_id":     nil,
					"goods_receipt_number": "",
					"complete_je_id":       nil,
				}).Error
		}
		return nil
	})
}

// ─────────────────────────────────────────────
// Milling Yield Report (per-order + trending KPIs)
// ─────────────────────────────────────────────

// MillingYieldRow is one completed milling/drying order in the yield KPI report.
// Head-rice % and brokens % are the economically decisive numbers in milling
// (brokens sell at roughly half price; modern mills target 10–15% brokens).
type MillingYieldRow struct {
	DocEntry       uint    `json:"doc_entry"`
	DocNum         string  `json:"doc_num"`
	OrderType      string  `json:"order_type"`
	PostingDate    string  `json:"posting_date"`
	BatchNo        string  `json:"batch_no"`
	InputItemName  string  `json:"input_item_name"`
	InputQty       float64 `json:"input_qty"`
	MoisturePct    float64 `json:"moisture_pct"`
	OutMoisturePct float64 `json:"out_moisture_pct"`
	TotalOutputQty float64 `json:"total_output_qty"`
	RejectedQty    float64 `json:"rejected_qty"`
	RecoveryPct    float64 `json:"recovery_pct"`     // total mass out / mass in
	MilledPct      float64 `json:"milled_pct"`       // (head + brokens) / mass in — the MRR KPI
	HeadRicePct    float64 `json:"head_rice_pct"`    // head rice / mass in
	BrokenPct      float64 `json:"broken_pct"`       // brokens / mass in
	BrokenShare    float64 `json:"broken_share"`     // brokens / (head + brokens) — quality KPI
	ProcessLossPct float64 `json:"process_loss_pct"` // unexplained gap after outputs + rejected
}

// MillingYieldReport returns per-order yield KPIs for all completed milling and
// drying orders, newest first, for trending head-rice % and recovery over time.
func MillingYieldReport() ([]MillingYieldRow, error) {
	var wos []models.WorkOrder
	if err := db.DB.Where("order_type IN ('M','D') AND status = 'C'").
		Order("created_at DESC").Find(&wos).Error; err != nil {
		return nil, err
	}
	if len(wos) == 0 {
		return []MillingYieldRow{}, nil
	}

	docEntries := make([]uint, len(wos))
	for i, wo := range wos {
		docEntries[i] = wo.DocEntry
	}
	var lines []models.WorkOrderLine
	if err := db.DB.Where("doc_entry IN ?", docEntries).Find(&lines).Error; err != nil {
		return nil, err
	}
	linesByDoc := make(map[uint][]models.WorkOrderLine, len(wos))
	for _, l := range lines {
		linesByDoc[l.DocEntry] = append(linesByDoc[l.DocEntry], l)
	}

	pct := func(part, whole float64) float64 {
		if whole <= 0 {
			return 0
		}
		return math.Round((part/whole)*10000) / 100
	}

	rows := make([]MillingYieldRow, 0, len(wos))
	for _, wo := range wos {
		inputQty, inputName := 0.0, ""
		var totalOut, headQty, brokenQty float64
		for _, l := range linesByDoc[wo.DocEntry] {
			if l.LineDir == "I" {
				inputQty = l.PlannedQty
				inputName = l.ItemName
				continue
			}
			totalOut += l.IssuedQty
			// Classification: explicit output type first, Milled Rice category as
			// head-rice fallback (mirrors CompleteMillingOrder).
			switch {
			case l.OutputType == "H", l.OutputType == "" && l.ItemCategory == "Milled Rice":
				headQty += l.IssuedQty
			case l.OutputType == "B":
				brokenQty += l.IssuedQty
			}
		}
		milled := headQty + brokenQty
		rows = append(rows, MillingYieldRow{
			DocEntry:       wo.DocEntry,
			DocNum:         wo.DocNum,
			OrderType:      wo.OrderType,
			PostingDate:    wo.StartDate,
			BatchNo:        wo.BatchNo,
			InputItemName:  inputName,
			InputQty:       inputQty,
			MoisturePct:    wo.MoisturePct,
			OutMoisturePct: wo.OutMoisturePct,
			TotalOutputQty: totalOut,
			RejectedQty:    wo.RjctQty,
			RecoveryPct:    pct(totalOut, inputQty),
			MilledPct:      pct(milled, inputQty),
			HeadRicePct:    pct(headQty, inputQty),
			BrokenPct:      pct(brokenQty, inputQty),
			BrokenShare:    pct(brokenQty, milled),
			ProcessLossPct: pct(inputQty-totalOut-wo.RjctQty, inputQty),
		})
	}
	return rows, nil
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
	// ── Validation (finding: UpsertBOM had none) ────────────────────────────────
	if bom.Code == "" {
		return errors.New("BOM code is required")
	}
	if bom.TreeType == "" {
		bom.TreeType = "P"
	}
	switch bom.TreeType {
	case "P":
		// Production BOM: the parent must be a real production item (mak_item = 'Y').
		var parent models.OITM
		if err := db.DB.Where("item_code = ? AND mak_item = 'Y'", bom.Code).First(&parent).Error; err != nil {
			return fmt.Errorf("BOM code %s is not a valid production item (mak_item='Y'): %w", bom.Code, err)
		}
	case "M", "D":
		// Milling/drying yield template: the code is the INPUT item (paddy); it only
		// needs to exist. Lines are expected outputs per Quantity units of input.
		var input models.OITM
		if err := db.DB.Where("item_code = ?", bom.Code).First(&input).Error; err != nil {
			return fmt.Errorf("yield template input item %s does not exist: %w", bom.Code, err)
		}
		if bom.Quantity <= 0 {
			return errors.New("yield template batch quantity must be greater than zero (e.g. 100 for per-100-kg yields)")
		}
	default:
		return fmt.Errorf("unknown tree type %q — must be P (BOM), M (milling template), or D (drying template)", bom.TreeType)
	}
	if len(bom.Lines) == 0 {
		return errors.New("a BOM must have at least one component line")
	}
	totalTemplateQty := 0.0
	for _, l := range bom.Lines {
		if l.ItemCode == "" {
			return errors.New("every BOM component must have an item code")
		}
		// Single-level circular reference: a BOM cannot contain itself.
		if l.ItemCode == bom.Code {
			return fmt.Errorf("BOM %s cannot contain itself as a component", bom.Code)
		}
		if l.Quantity <= 0 {
			return fmt.Errorf("component %s must have a positive quantity", l.ItemCode)
		}
		if l.OutputType != "" && l.OutputType != "H" && l.OutputType != "B" && l.OutputType != "Y" {
			return fmt.Errorf("component %s has invalid output type %q — must be H (head rice), B (brokens), or Y (by-product)", l.ItemCode, l.OutputType)
		}
		var comp models.OITM
		if err := db.DB.Where("item_code = ?", l.ItemCode).First(&comp).Error; err != nil {
			return fmt.Errorf("BOM component %s does not exist: %w", l.ItemCode, err)
		}
		totalTemplateQty += l.Quantity
	}
	// A yield template cannot expect more mass out than goes in.
	if (bom.TreeType == "M" || bom.TreeType == "D") && totalTemplateQty > bom.Quantity {
		return fmt.Errorf("yield template outputs total %.3f which exceeds the input batch quantity %.3f — total yield cannot exceed 100%%", totalTemplateQty, bom.Quantity)
	}

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
	// Block deletion while open (Planned/Released) Work Orders still reference it.
	var openWOs int64
	if err := db.DB.Model(&models.WorkOrder{}).
		Where("order_type = 'P' AND item_code = ? AND status IN ('P','R')", code).
		Count(&openWOs).Error; err != nil {
		return fmt.Errorf("check open work orders for %s: %w", code, err)
	}
	if openWOs > 0 {
		return fmt.Errorf("cannot delete BOM %s: %d open work order(s) still use it — close or cancel them first", code, openWOs)
	}
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
			// Look up component item to get price and default warehouse.
			// A BOM line pointing at a deleted/renamed item must fail loudly rather
			// than silently create a WO line with ItemID 0 and price 0.
			var compItem models.OITM
			if err := tx.Where("item_code = ?", bl.ItemCode).First(&compItem).Error; err != nil {
				return fmt.Errorf("BOM component %s (referenced by BOM %s) not found: %w", bl.ItemCode, req.ItemCode, err)
			}
			lineWhs := bl.Warehouse
			if lineWhs == "" {
				lineWhs = compItem.DfltWh
			}
			if lineWhs == "" {
				lineWhs = whs
			}
			line := models.WorkOrderLine{
				DocEntry:     wo.DocEntry,
				LineNum:      i,
				LineDir:      "I",
				ItemID:       compItem.ID,
				ItemCode:     bl.ItemCode,
				ItemName:     bl.ItemName,
				ItemCategory: invsvc.GetItemCategoryName(compItem.ItmsGrpCod, tx),
				PlannedQty:   bl.Quantity * scale,
				Warehouse:    lineWhs,
				IssueMethod:  bl.IssueMethod,
				UomEntry:     bl.UomEntry,
				UomCode:      bl.UomCode,
				Price:        compItem.AvgPrice,
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
//
// Issuing components moves their value into WIP, so — like the Milling path — the
// release now posts Dr WIP / Cr Inventory and requires the WIP GL account (finding
// #7). Stock is checked and decremented atomically, both company-wide AND for the
// specific warehouse, so no warehouse can be driven negative (finding #4). All of
// this runs in one transaction.
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

	// Any backflush lines to issue now?
	hasBackflush := false
	for _, line := range wo.Lines {
		if line.LineDir == "I" && line.IssueMethod == "B" && line.PlannedQty > 0 {
			hasBackflush = true
			break
		}
	}

	// Pre-flight WIP account when there is component value to move into WIP.
	var wipAccount *models.GLAccount
	if hasBackflush {
		wipAccount, err = accountingsvc.GetAccount(nil, "PRODUCTION", "MILLING_WIP")
		if err != nil || wipAccount == nil {
			return fmt.Errorf("cannot release work order: MILLING_WIP GL account is not configured — set it up under Account Determination before proceeding")
		}
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		// Concurrency guard: only one release may advance Planned(P) → Released(R).
		claim := tx.Model(&models.WorkOrder{}).
			Where("doc_entry = ? AND status = 'P'", docEntry).
			Updates(map[string]interface{}{"status": "R", "updated_by_id": uid})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			return errors.New("work order was already released by another operation")
		}

		creditLines := make([]accountingsvc.JELine, 0)
		wipTotal := 0.0

		for _, line := range wo.Lines {
			if line.LineDir != "I" || line.IssueMethod != "B" || line.PlannedQty <= 0 {
				continue
			}
			var item models.OITM
			if err := tx.Where("item_code = ?", line.ItemCode).First(&item).Error; err != nil {
				return fmt.Errorf("item %s not found: %w", line.ItemCode, err)
			}
			// Re-read component cost at issue time (finding #5).
			unitCost := item.AvgPrice
			whs := line.Warehouse
			if whs == "" {
				whs = item.DfltWh
			}
			if whs == "" {
				whs = "WH01"
			}
			// Company-wide guard: atomic conditional decrement.
			res := tx.Model(&models.OITM{}).
				Where("item_code = ? AND on_hand >= ?", line.ItemCode, line.PlannedQty).
				UpdateColumn("on_hand", gorm.Expr("on_hand - ?", line.PlannedQty))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("insufficient stock for %s: need %.3f", line.ItemCode, line.PlannedQty)
			}
			// Per-warehouse guard (finding #4): the issuing warehouse must hold enough.
			wres := tx.Exec(
				`UPDATE oitw SET on_hand = on_hand - ? WHERE item_code = ? AND whs_code = ? AND on_hand >= ?`,
				line.PlannedQty, line.ItemCode, whs, line.PlannedQty)
			if wres.Error != nil {
				return wres.Error
			}
			if wres.RowsAffected == 0 {
				return fmt.Errorf("insufficient stock for %s in warehouse %s: need %.3f", line.ItemCode, whs, line.PlannedQty)
			}
			if err := invsvc.AppendOIVL(tx, line.ItemCode, item.ItemName, whs, item.DfltWh,
				"WO_ISSUE", wo.StartDate, int(wo.DocEntry), 0, line.PlannedQty, unitCost, uid); err != nil {
				return err
			}
			if err := tx.Model(&models.WorkOrderLine{}).Where("id = ?", line.ID).
				Updates(map[string]interface{}{"issued_qty": line.PlannedQty, "price": unitCost}).Error; err != nil {
				return err
			}

			// Accumulate WIP value: credit the component's inventory account.
			lineVal := math.Round(line.PlannedQty*unitCost*100) / 100
			if lineVal > 0 {
				wipTotal += lineVal
				invAcct, accErr := accountingsvc.GetAccount(tx, "PRODUCTION", "MILLING_INPUT", invsvc.GetItemCategoryName(item.ItmsGrpCod, tx))
				if accErr != nil || invAcct == nil {
					invAcct = wipAccount
				}
				creditLines = append(creditLines, accountingsvc.JELine{
					Account:     invAcct,
					Credit:      lineVal,
					Description: "Inventory — " + item.ItemName,
				})
			}
		}

		// Post the WIP journal entry: Dr WIP / Cr Inventory(components).
		if wipTotal > 0 {
			jeLines := append([]accountingsvc.JELine{{
				Account:     wipAccount,
				Debit:       math.Round(wipTotal*100) / 100,
				Description: "WIP — " + wo.DocNum,
			}}, creditLines...)
			now := time.Now()
			je, err := accountingsvc.PostJournalEntry(tx, accountingsvc.PostJEParams{
				Module:      "PRODUCTION",
				SourceType:  "WO_RELEASE",
				SourceRef:   wo.DocNum,
				Narration:   fmt.Sprintf("Work Order %s — components to WIP", wo.DocNum),
				EntryDate:   &now,
				CreatedByID: &uid,
				Lines:       jeLines,
			})
			if err != nil {
				return fmt.Errorf("post release journal entry: %w", err)
			}
			// Reuse start_je_id to link the release WIP entry (mirrors the Milling path).
			if err := tx.Model(&models.WorkOrder{}).Where("doc_entry = ?", docEntry).
				UpdateColumn("start_je_id", je.ID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ─────────────────────────────────────────────
// Work Orders — Close (Released → Closed)
// ─────────────────────────────────────────────

// CloseWorkOrder receives the finished goods into stock and marks the WO closed.
//
// The finished good is received at its ROLLED-UP component cost (finding #7): the
// value of components actually issued (issued_qty × issue-time cost) divided by the
// completed quantity — so a brand-new item with no price history no longer books in
// at zero. The moving average is updated accordingly and the close posts
// Dr Finished-Goods Inventory / Cr WIP, all in one transaction.
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

	// Roll up the cost of components actually consumed into WIP.
	wipTotal := 0.0
	for _, line := range wo.Lines {
		if line.LineDir == "I" && line.IssuedQty > 0 {
			wipTotal += line.IssuedQty * line.Price
		}
	}
	wipTotal = math.Round(wipTotal*100) / 100
	unitFGCost := item.AvgPrice
	if wipTotal > 0 && cmpltQty > 0 {
		unitFGCost = math.Round((wipTotal/cmpltQty)*10000) / 10000
	}

	// Pre-flight WIP account when there is WIP value to transfer to finished goods.
	var wipAccount *models.GLAccount
	if wipTotal > 0 {
		wipAccount, err = accountingsvc.GetAccount(nil, "PRODUCTION", "MILLING_WIP")
		if err != nil || wipAccount == nil {
			return fmt.Errorf("cannot close work order: MILLING_WIP GL account is not configured — set it up under Account Determination before proceeding")
		}
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		// Concurrency guard: only one close may advance Released(R) → Closed(C).
		claim := tx.Model(&models.WorkOrder{}).
			Where("doc_entry = ? AND status = 'R'", docEntry).
			Updates(map[string]interface{}{
				"status":        "C",
				"cmplt_qty":     cmpltQty,
				"rjct_qty":      rjctQty,
				"updated_by_id": uid,
			})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			return errors.New("work order was already closed by another operation")
		}

		// Receive finished goods, rolling the moving average toward the component cost.
		// RecalcMovingAvgPrice must run before the on_hand increment.
		newAvg := invsvc.RecalcMovingAvgPrice(tx, &item, cmpltQty, unitFGCost)
		if err := tx.Model(&models.OITM{}).Where("item_code = ?", wo.ItemCode).
			UpdateColumn("on_hand", gorm.Expr("on_hand + ?", cmpltQty)).Error; err != nil {
			return err
		}
		if err := invsvc.UpsertOITWOnHand(tx, wo.ItemCode, wo.Warehouse, item.DfltWh, cmpltQty); err != nil {
			return err
		}
		if err := invsvc.AppendOIVL(tx, wo.ItemCode, item.ItemName, wo.Warehouse, item.DfltWh,
			"WO_RECEIPT", wo.DueDate, int(wo.DocEntry), cmpltQty, 0, newAvg, uid); err != nil {
			return err
		}

		// Post the completion JE: Dr Finished-Goods Inventory / Cr WIP.
		if wipTotal > 0 {
			fgAcct, accErr := accountingsvc.GetAccount(tx, "PRODUCTION", "MILLING_OUTPUT", invsvc.GetItemCategoryName(item.ItmsGrpCod, tx))
			if accErr != nil || fgAcct == nil {
				fgAcct = wipAccount
			}
			now := time.Now()
			je, err := accountingsvc.PostJournalEntry(tx, accountingsvc.PostJEParams{
				Module:      "PRODUCTION",
				SourceType:  "WO_CLOSE",
				SourceRef:   wo.DocNum,
				Narration:   fmt.Sprintf("Work Order %s — WIP to finished goods", wo.DocNum),
				EntryDate:   &now,
				CreatedByID: &uid,
				Lines: []accountingsvc.JELine{
					{Account: fgAcct, Debit: wipTotal, Description: "Finished goods — " + item.ItemName},
					{Account: wipAccount, Credit: wipTotal, Description: "WIP — " + wo.DocNum + " close"},
				},
			})
			if err != nil {
				return fmt.Errorf("post close journal entry: %w", err)
			}
			if err := tx.Model(&models.WorkOrder{}).Where("doc_entry = ?", docEntry).
				UpdateColumn("complete_je_id", je.ID).Error; err != nil {
				return err
			}
		}
		return nil
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
		res := db.DB.Model(&models.WorkOrder{}).Where("doc_entry = ? AND status = 'P'", docEntry).
			Updates(map[string]interface{}{"status": "L", "updated_by_id": uid})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("work order changed status concurrently — cancel aborted")
		}
		return nil
	case "R":
		// Restore issued quantities for Backflush lines and reverse the WIP JE.
		now := time.Now()
		return db.DB.Transaction(func(tx *gorm.DB) error {
			claim := tx.Model(&models.WorkOrder{}).Where("doc_entry = ? AND status = 'R'", docEntry).
				Updates(map[string]interface{}{"status": "L", "updated_by_id": uid})
			if claim.Error != nil {
				return claim.Error
			}
			if claim.RowsAffected == 0 {
				return errors.New("work order changed status concurrently — cancel aborted")
			}
			for _, line := range wo.Lines {
				if line.LineDir != "I" || line.IssuedQty <= 0 {
					continue
				}
				var item models.OITM
				// A missing item must abort — silently skipping it would leave stock
				// half-restored with the WO already marked cancelled.
				if err := tx.Where("item_code = ?", line.ItemCode).First(&item).Error; err != nil {
					return fmt.Errorf("cannot restore stock for %s: %w", line.ItemCode, err)
				}
				whs := line.Warehouse
				if whs == "" {
					whs = item.DfltWh
				}
				if err := tx.Model(&models.OITM{}).Where("item_code = ?", line.ItemCode).
					UpdateColumn("on_hand", gorm.Expr("on_hand + ?", line.IssuedQty)).Error; err != nil {
					return err
				}
				if err := invsvc.UpsertOITWOnHand(tx, line.ItemCode, whs, item.DfltWh, line.IssuedQty); err != nil {
					return err
				}
				if err := invsvc.AppendOIVL(tx, line.ItemCode, item.ItemName, whs, item.DfltWh,
					"WO_CANCEL", wo.StartDate, int(wo.DocEntry), line.IssuedQty, 0, line.Price, uid); err != nil {
					return err
				}
			}
			// Reverse the release WIP journal entry, if one was posted.
			if wo.StartJEID != nil {
				var je models.JournalEntry
				if err := tx.First(&je, *wo.StartJEID).Error; err != nil {
					return fmt.Errorf("load release journal entry: %w", err)
				}
				if _, err := accountingsvc.ReverseJournalEntry(tx, &je, &now, userID); err != nil {
					return fmt.Errorf("reverse release journal entry: %w", err)
				}
			}
			return nil
		})
	default:
		return fmt.Errorf("unknown status: %s", wo.Status)
	}
}
