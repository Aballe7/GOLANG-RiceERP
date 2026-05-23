package assets

import (
	"fmt"
	"math"
	"time"

	"ricemill/app/db"
	"ricemill/app/models"
	acctSvc "ricemill/app/services/accounting"

	"gorm.io/gorm"
)

// CreateFixedAssetParams holds inputs for creating a new fixed asset.
type CreateFixedAssetParams struct {
	AssetCode           string  `json:"asset_code"`
	AssetName           string  `json:"asset_name"`
	Category            string  `json:"category"`
	Description         string  `json:"description"`
	AcquisitionDate     string  `json:"acquisition_date"` // "YYYY-MM-DD"
	AcquisitionCost     float64 `json:"acquisition_cost"`
	ResidualValue       float64 `json:"residual_value"`
	UsefulLifeMonths    int     `json:"useful_life_months"`
	DepreciationMethod  string  `json:"depreciation_method"` // STRAIGHT_LINE|DECLINING_BALANCE
	GLAssetAccountID    uint    `json:"gl_asset_account_id"`
	GLAccumDepAccountID uint    `json:"gl_accum_dep_account_id"`
	GLDepExpAccountID   uint    `json:"gl_dep_exp_account_id"`
	Notes               string  `json:"notes"`
	CreatedByID         *uint
}

// DisposeAssetParams holds inputs for asset disposal.
type DisposeAssetParams struct {
	AssetID           uint    `json:"asset_id"`
	DisposalDate      string  `json:"disposal_date"` // "YYYY-MM-DD"
	Proceeds          float64 `json:"proceeds"`
	ProceedsAccountID uint    `json:"proceeds_account_id"`  // Cash/Bank GL account
	GainLossAccountID uint    `json:"gain_loss_account_id"` // Other Income/Expense GL account
	Notes             string  `json:"notes"`
	CreatedByID       *uint
}

// DepreciationScheduleLine is one row in the projected or posted depreciation table.
type DepreciationScheduleLine struct {
	Period         string  `json:"period"`       // "YYYY-MM"
	PeriodDate     string  `json:"period_date"`  // "YYYY-MM-01"
	MonthlyAmount  float64 `json:"monthly_amount"`
	AccumDep       float64 `json:"accum_dep"`
	BookValue      float64 `json:"book_value"`
	IsPosted       bool    `json:"is_posted"`
	JournalEntryID *uint   `json:"journal_entry_id"`
}

// DepreciationRunDetail is the per-asset result from RunDepreciation.
type DepreciationRunDetail struct {
	AssetCode string  `json:"asset_code"`
	AssetName string  `json:"asset_name"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"` // posted|skipped|error
	Message   string  `json:"message,omitempty"`
	JEID      *uint   `json:"je_id,omitempty"`
}

// DepreciationRunResult summarises a RunDepreciation call.
type DepreciationRunResult struct {
	Period  string                  `json:"period"`
	Posted  int                     `json:"posted"`
	Skipped int                     `json:"skipped"`
	Total   float64                 `json:"total"`
	Details []DepreciationRunDetail `json:"details"`
}

// ─────────────────────────────────────────────────────────────────────────────
// CRUD
// ─────────────────────────────────────────────────────────────────────────────

func ListFixedAssets() ([]models.FixedAsset, error) {
	var assets []models.FixedAsset
	err := db.DB.
		Preload("AssetAccount").
		Preload("AccumDepAccount").
		Preload("DepExpAccount").
		Order("asset_code").
		Find(&assets).Error
	return assets, err
}

func GetFixedAsset(id uint) (*models.FixedAsset, error) {
	var fa models.FixedAsset
	err := db.DB.
		Preload("AssetAccount").
		Preload("AccumDepAccount").
		Preload("DepExpAccount").
		Preload("DepLogs", func(db *gorm.DB) *gorm.DB { return db.Order("period_date asc") }).
		First(&fa, id).Error
	return &fa, err
}

func CreateFixedAsset(params CreateFixedAssetParams) (*models.FixedAsset, error) {
	if params.AssetCode == "" {
		return nil, fmt.Errorf("asset code is required")
	}
	if params.AcquisitionCost <= 0 {
		return nil, fmt.Errorf("acquisition cost must be greater than zero")
	}
	if params.UsefulLifeMonths <= 0 {
		return nil, fmt.Errorf("useful life months must be greater than zero")
	}

	acqDate, err := time.Parse("2006-01-02", params.AcquisitionDate)
	if err != nil {
		return nil, fmt.Errorf("invalid acquisition date: %w", err)
	}

	method := params.DepreciationMethod
	if method == "" {
		method = "STRAIGHT_LINE"
	}

	fa := &models.FixedAsset{
		AssetCode:           params.AssetCode,
		AssetName:           params.AssetName,
		Category:            params.Category,
		Description:         params.Description,
		AcquisitionDate:     acqDate,
		AcquisitionCost:     params.AcquisitionCost,
		ResidualValue:       params.ResidualValue,
		UsefulLifeMonths:    params.UsefulLifeMonths,
		DepreciationMethod:  method,
		GLAssetAccountID:    params.GLAssetAccountID,
		GLAccumDepAccountID: params.GLAccumDepAccountID,
		GLDepExpAccountID:   params.GLDepExpAccountID,
		Status:              "ACTIVE",
		Notes:               params.Notes,
		CreatedByID:         params.CreatedByID,
		CreatedAt:           time.Now(),
	}

	if err := db.DB.Create(fa).Error; err != nil {
		return nil, fmt.Errorf("failed to create fixed asset: %w", err)
	}
	return GetFixedAsset(fa.ID)
}

func UpdateFixedAsset(id uint, updates map[string]interface{}) error {
	// Prevent updating status through the generic update path
	delete(updates, "status")
	delete(updates, "accum_depreciation")
	return db.DB.Model(&models.FixedAsset{}).Where("id = ?", id).Updates(updates).Error
}

// ─────────────────────────────────────────────────────────────────────────────
// Depreciation schedule (projected)
// ─────────────────────────────────────────────────────────────────────────────

func GetDepreciationSchedule(assetID uint) ([]DepreciationScheduleLine, error) {
	fa, err := GetFixedAsset(assetID)
	if err != nil {
		return nil, err
	}

	// Build a lookup map of already-posted periods
	posted := make(map[string]*models.DepreciationEntry)
	for i := range fa.DepLogs {
		key := fa.DepLogs[i].PeriodDate.Format("2006-01")
		posted[key] = &fa.DepLogs[i]
	}

	depreciable := fa.DepreciableAmount()
	accumDep := 0.0
	bookValue := fa.AcquisitionCost
	start := time.Date(fa.AcquisitionDate.Year(), fa.AcquisitionDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	var lines []DepreciationScheduleLine
	for i := 0; i < fa.UsefulLifeMonths; i++ {
		period := start.AddDate(0, i, 0)
		periodKey := period.Format("2006-01")

		var monthlyAmount float64
		if fa.DepreciationMethod == "STRAIGHT_LINE" {
			monthlyAmount = depreciable / float64(fa.UsefulLifeMonths)
		} else {
			// Double declining balance
			years := float64(fa.UsefulLifeMonths) / 12.0
			monthlyRate := (2.0 / years) / 12.0
			monthlyAmount = bookValue * monthlyRate
		}

		remaining := depreciable - accumDep
		if monthlyAmount > remaining {
			monthlyAmount = remaining
		}
		monthlyAmount = math.Round(monthlyAmount*100) / 100
		if monthlyAmount < 0.005 {
			break
		}

		accumDep = math.Round((accumDep+monthlyAmount)*100) / 100
		bookValue = math.Round((fa.AcquisitionCost-accumDep)*100) / 100

		isPosted := false
		var jeID *uint
		if entry, ok := posted[periodKey]; ok {
			isPosted = true
			jeID = entry.JournalEntryID
		}

		lines = append(lines, DepreciationScheduleLine{
			Period:         periodKey,
			PeriodDate:     period.Format("2006-01-02"),
			MonthlyAmount:  monthlyAmount,
			AccumDep:       accumDep,
			BookValue:      bookValue,
			IsPosted:       isPosted,
			JournalEntryID: jeID,
		})
	}

	return lines, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Run depreciation for a given period (YYYY-MM or YYYY-MM-DD)
// ─────────────────────────────────────────────────────────────────────────────

func RunDepreciation(periodStr string, createdByID *uint) (*DepreciationRunResult, error) {
	// Accept either "YYYY-MM" or "YYYY-MM-DD"
	var periodDate time.Time
	var parseErr error
	if len(periodStr) == 7 {
		periodDate, parseErr = time.Parse("2006-01", periodStr)
	} else {
		periodDate, parseErr = time.Parse("2006-01-02", periodStr)
	}
	if parseErr != nil {
		return nil, fmt.Errorf("invalid period: %w", parseErr)
	}
	// Normalise to first day of month UTC
	periodDate = time.Date(periodDate.Year(), periodDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	assets, err := ListFixedAssets()
	if err != nil {
		return nil, err
	}

	result := &DepreciationRunResult{Period: periodDate.Format("January 2006")}

	for _, fa := range assets {
		detail := DepreciationRunDetail{AssetCode: fa.AssetCode, AssetName: fa.AssetName}

		if fa.Status != "ACTIVE" {
			detail.Status = "skipped"
			detail.Message = "asset is not active"
			result.Skipped++
			result.Details = append(result.Details, detail)
			continue
		}

		acqPeriod := time.Date(fa.AcquisitionDate.Year(), fa.AcquisitionDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		if periodDate.Before(acqPeriod) {
			detail.Status = "skipped"
			detail.Message = "period is before acquisition date"
			result.Skipped++
			result.Details = append(result.Details, detail)
			continue
		}

		// Already posted for this period?
		var existing models.DepreciationEntry
		if db.DB.Where("fixed_asset_id = ? AND period_date = ?", fa.ID, periodDate).First(&existing).Error == nil {
			detail.Status = "skipped"
			detail.Amount = existing.Amount
			detail.Message = "already posted for this period"
			detail.JEID = existing.JournalEntryID
			result.Skipped++
			result.Details = append(result.Details, detail)
			continue
		}

		depreciable := fa.DepreciableAmount()
		remaining := depreciable - fa.AccumDepreciation
		if remaining <= 0.005 {
			db.DB.Model(&fa).Updates(map[string]interface{}{
				"status":        "FULLY_DEPRECIATED",
				"updated_by_id": createdByID,
			})
			detail.Status = "skipped"
			detail.Message = "fully depreciated"
			result.Skipped++
			result.Details = append(result.Details, detail)
			continue
		}

		var monthlyAmount float64
		if fa.DepreciationMethod == "STRAIGHT_LINE" {
			monthlyAmount = depreciable / float64(fa.UsefulLifeMonths)
		} else {
			years := float64(fa.UsefulLifeMonths) / 12.0
			monthlyRate := (2.0 / years) / 12.0
			monthlyAmount = fa.BookValue() * monthlyRate
		}
		if monthlyAmount > remaining {
			monthlyAmount = remaining
		}
		monthlyAmount = math.Round(monthlyAmount*100) / 100
		if monthlyAmount < 0.005 {
			detail.Status = "skipped"
			detail.Message = "amount rounds to zero"
			result.Skipped++
			result.Details = append(result.Details, detail)
			continue
		}

		var postedJEID *uint
		txErr := db.DB.Transaction(func(tx *gorm.DB) error {
			var assetAcct, accumAcct, expAcct models.GLAccount
			if err := tx.First(&assetAcct, fa.GLAssetAccountID).Error; err != nil {
				return fmt.Errorf("asset GL account not found")
			}
			if err := tx.First(&accumAcct, fa.GLAccumDepAccountID).Error; err != nil {
				return fmt.Errorf("accum dep GL account not found")
			}
			if err := tx.First(&expAcct, fa.GLDepExpAccountID).Error; err != nil {
				return fmt.Errorf("dep expense GL account not found")
			}

			narration := fmt.Sprintf("Depreciation — %s %s (%s)", fa.AssetCode, fa.AssetName, periodDate.Format("Jan 2006"))
			je, err := acctSvc.PostJournalEntry(tx, acctSvc.PostJEParams{
				Module:      "ASSETS",
				SourceType:  "DEPRECIATION",
				SourceRef:   fa.AssetCode,
				Narration:   narration,
				EntryDate:   &periodDate,
				CreatedByID: createdByID,
				Lines: []acctSvc.JELine{
					{Account: &expAcct, Debit: monthlyAmount, Description: narration},
					{Account: &accumAcct, Credit: monthlyAmount, Description: narration},
				},
			})
			if err != nil {
				return err
			}
			postedJEID = &je.ID

			newAccum := math.Round((fa.AccumDepreciation+monthlyAmount)*100) / 100
			newBook := math.Round((fa.AcquisitionCost-newAccum)*100) / 100

			entry := models.DepreciationEntry{
				FixedAssetID:   fa.ID,
				PeriodDate:     periodDate,
				Amount:         monthlyAmount,
				AccumDepAfter:  newAccum,
				BookValueAfter: newBook,
				JournalEntryID: &je.ID,
				CreatedAt:      time.Now(),
			}
			if err := tx.Create(&entry).Error; err != nil {
				return fmt.Errorf("failed to create depreciation log: %w", err)
			}

			newStatus := "ACTIVE"
			if newAccum >= depreciable-0.005 {
				newStatus = "FULLY_DEPRECIATED"
			}
			return tx.Model(&models.FixedAsset{}).Where("id = ?", fa.ID).Updates(map[string]interface{}{
				"accum_depreciation": newAccum,
				"status":             newStatus,
				"updated_by_id":      createdByID,
			}).Error
		})

		if txErr != nil {
			detail.Status = "error"
			detail.Message = txErr.Error()
		} else {
			detail.Status = "posted"
			detail.Amount = monthlyAmount
			detail.JEID = postedJEID
			result.Posted++
			result.Total = math.Round((result.Total+monthlyAmount)*100) / 100
		}
		result.Details = append(result.Details, detail)
	}

	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Dispose asset
// ─────────────────────────────────────────────────────────────────────────────

func DisposeAsset(params DisposeAssetParams) (*models.FixedAsset, error) {
	fa, err := GetFixedAsset(params.AssetID)
	if err != nil {
		return nil, fmt.Errorf("asset not found: %w", err)
	}
	if fa.Status == "DISPOSED" {
		return nil, fmt.Errorf("asset %s is already disposed", fa.AssetCode)
	}

	dispDate, err := time.Parse("2006-01-02", params.DisposalDate)
	if err != nil {
		return nil, fmt.Errorf("invalid disposal date: %w", err)
	}

	bookValue := fa.BookValue()
	gainLoss := params.Proceeds - bookValue // positive = gain, negative = loss

	var assetAcct, accumAcct, proceedsAcct, gainLossAcct models.GLAccount
	if err := db.DB.First(&assetAcct, fa.GLAssetAccountID).Error; err != nil {
		return nil, fmt.Errorf("asset GL account not found")
	}
	if err := db.DB.First(&accumAcct, fa.GLAccumDepAccountID).Error; err != nil {
		return nil, fmt.Errorf("accum dep GL account not found")
	}
	if err := db.DB.First(&proceedsAcct, params.ProceedsAccountID).Error; err != nil {
		return nil, fmt.Errorf("proceeds GL account not found")
	}
	if err := db.DB.First(&gainLossAcct, params.GainLossAccountID).Error; err != nil {
		return nil, fmt.Errorf("gain/loss GL account not found")
	}

	narration := fmt.Sprintf("Disposal of fixed asset %s — %s", fa.AssetCode, fa.AssetName)

	var jeID uint
	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		lines := []acctSvc.JELine{
			// Remove accumulated depreciation (Dr Accum Dep)
			{Account: &accumAcct, Debit: math.Round(fa.AccumDepreciation*100) / 100, Description: "Remove accumulated depreciation"},
			// Remove asset at cost (Cr Asset Account)
			{Account: &assetAcct, Credit: math.Round(fa.AcquisitionCost*100) / 100, Description: "Remove asset at cost"},
		}

		if params.Proceeds > 0.005 {
			proceeds := math.Round(params.Proceeds*100) / 100
			lines = append(lines, acctSvc.JELine{
				Account:     &proceedsAcct,
				Debit:       proceeds,
				Description: "Disposal proceeds",
			})
		}

		absGL := math.Round(math.Abs(gainLoss)*100) / 100
		if absGL > 0.005 {
			if gainLoss > 0 {
				lines = append(lines, acctSvc.JELine{
					Account:     &gainLossAcct,
					Credit:      absGL,
					Description: "Gain on asset disposal",
				})
			} else {
				lines = append(lines, acctSvc.JELine{
					Account:     &gainLossAcct,
					Debit:       absGL,
					Description: "Loss on asset disposal",
				})
			}
		}

		je, err := acctSvc.PostJournalEntry(tx, acctSvc.PostJEParams{
			Module:      "ASSETS",
			SourceType:  "DISPOSAL",
			SourceRef:   fa.AssetCode,
			Narration:   narration,
			EntryDate:   &dispDate,
			CreatedByID: params.CreatedByID,
			Lines:       lines,
		})
		if err != nil {
			return err
		}
		jeID = je.ID

		proceeds := params.Proceeds
		return tx.Model(&models.FixedAsset{}).Where("id = ?", fa.ID).Updates(map[string]interface{}{
			"status":            "DISPOSED",
			"disposal_date":     dispDate,
			"disposal_proceeds": proceeds,
			"disposal_je_id":    jeID,
			"updated_by_id":     params.CreatedByID,
		}).Error
	})

	if txErr != nil {
		return nil, txErr
	}
	return GetFixedAsset(fa.ID)
}
