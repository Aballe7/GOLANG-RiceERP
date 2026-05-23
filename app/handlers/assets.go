package handlers

import (
	"ricemill/app/middleware"
	assetSvc "ricemill/app/services/assets"
)

// ─── List / Get ───────────────────────────────────────────────────────────────

func ListFixedAssets() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	assets, err := assetSvc.ListFixedAssets()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", assets)
}

func GetFixedAsset(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	fa, err := assetSvc.GetFixedAsset(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", fa)
}

// ─── Create ───────────────────────────────────────────────────────────────────

type CreateFixedAssetRequest struct {
	AssetCode           string  `json:"asset_code"`
	AssetName           string  `json:"asset_name"`
	Category            string  `json:"category"`
	Description         string  `json:"description"`
	AcquisitionDate     string  `json:"acquisition_date"`
	AcquisitionCost     float64 `json:"acquisition_cost"`
	ResidualValue       float64 `json:"residual_value"`
	UsefulLifeMonths    int     `json:"useful_life_months"`
	DepreciationMethod  string  `json:"depreciation_method"`
	GLAssetAccountID    uint    `json:"gl_asset_account_id"`
	GLAccumDepAccountID uint    `json:"gl_accum_dep_account_id"`
	GLDepExpAccountID   uint    `json:"gl_dep_exp_account_id"`
	Notes               string  `json:"notes"`
}

func CreateFixedAsset(req CreateFixedAssetRequest) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	userID := middleware.Store.UserID()
	fa, err := assetSvc.CreateFixedAsset(assetSvc.CreateFixedAssetParams{
		AssetCode:           req.AssetCode,
		AssetName:           req.AssetName,
		Category:            req.Category,
		Description:         req.Description,
		AcquisitionDate:     req.AcquisitionDate,
		AcquisitionCost:     req.AcquisitionCost,
		ResidualValue:       req.ResidualValue,
		UsefulLifeMonths:    req.UsefulLifeMonths,
		DepreciationMethod:  req.DepreciationMethod,
		GLAssetAccountID:    req.GLAssetAccountID,
		GLAccumDepAccountID: req.GLAccumDepAccountID,
		GLDepExpAccountID:   req.GLDepExpAccountID,
		Notes:               req.Notes,
		CreatedByID:         userID,
	})
	if err != nil {
		return errResponse(err)
	}
	return okResponse("Fixed asset created", fa)
}

// ─── Update ───────────────────────────────────────────────────────────────────

func UpdateFixedAsset(id uint, updates map[string]interface{}) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if err := assetSvc.UpdateFixedAsset(id, updates); err != nil {
		return errResponse(err)
	}
	return okResponse("Fixed asset updated", nil)
}

// ─── Depreciation schedule ────────────────────────────────────────────────────

func GetDepreciationSchedule(assetID uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	schedule, err := assetSvc.GetDepreciationSchedule(assetID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", schedule)
}

// ─── Run depreciation ────────────────────────────────────────────────────────

type RunDepreciationRequest struct {
	PeriodDate string `json:"period_date"` // "YYYY-MM" or "YYYY-MM-DD"
}

func RunDepreciation(req RunDepreciationRequest) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	userID := middleware.Store.UserID()
	result, err := assetSvc.RunDepreciation(req.PeriodDate, userID)
	if err != nil {
		return errResponse(err)
	}
	msg := "No assets processed"
	if result.Posted > 0 {
		msg = "Depreciation posted successfully"
	}
	return okResponse(msg, result)
}

// ─── Dispose asset ────────────────────────────────────────────────────────────

type DisposeAssetRequest struct {
	AssetID           uint    `json:"asset_id"`
	DisposalDate      string  `json:"disposal_date"`
	Proceeds          float64 `json:"proceeds"`
	ProceedsAccountID uint    `json:"proceeds_account_id"`
	GainLossAccountID uint    `json:"gain_loss_account_id"`
	Notes             string  `json:"notes"`
}

func DisposeAsset(req DisposeAssetRequest) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	userID := middleware.Store.UserID()
	fa, err := assetSvc.DisposeAsset(assetSvc.DisposeAssetParams{
		AssetID:           req.AssetID,
		DisposalDate:      req.DisposalDate,
		Proceeds:          req.Proceeds,
		ProceedsAccountID: req.ProceedsAccountID,
		GainLossAccountID: req.GainLossAccountID,
		Notes:             req.Notes,
		CreatedByID:       userID,
	})
	if err != nil {
		return errResponse(err)
	}
	return okResponse("Asset disposed", fa)
}
