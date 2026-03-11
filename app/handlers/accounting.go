package handlers

import (
	"egglayererp/app/middleware"
	"egglayererp/app/models"
	"egglayererp/app/services/accounting"
	"time"
)

// --- GL Accounts ---

func ListGLAccounts() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	accounts, err := accounting.ListGLAccounts(false)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", accounts)
}

func GetGLAccount(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	acct, err := accounting.GetGLAccount(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", acct)
}

type CreateGLAccountRequest struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Section       string `json:"section"`
	AccountType   string `json:"account_type"`
	NormalBalance string `json:"normal_balance"`
	ParentID      *uint  `json:"parent_id"`
	Description   string `json:"description"`
}

func CreateGLAccount(req CreateGLAccountRequest) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	acct := &models.GLAccount{
		Code:          req.Code,
		Name:          req.Name,
		Section:       req.Section,
		AccountType:   req.AccountType,
		NormalBalance: req.NormalBalance,
		ParentID:      req.ParentID,
		Description:   req.Description,
		IsActive:      true,
		CreatedAt:     time.Now(),
	}
	if err := accounting.CreateGLAccount(acct); err != nil {
		return errResponse(err)
	}
	return okResponse("GL Account created", acct)
}

func UpdateGLAccount(id uint, updates map[string]interface{}) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if err := accounting.UpdateGLAccount(id, updates); err != nil {
		return errResponse(err)
	}
	return okResponse("GL Account updated", nil)
}

func DeleteGLAccount(id uint) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if err := accounting.DeleteGLAccount(id); err != nil {
		return errResponse(err)
	}
	return okResponse("GL Account deactivated", nil)
}

// --- Account Determination ---

func ListAccountDeterminations() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	rules, err := accounting.ListAccountDeterminations()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", rules)
}

type UpsertAccountDeterminationRequest struct {
	Module       string  `json:"module"`
	PostingEvent string  `json:"posting_event"`
	ItemCategory *string `json:"item_category"`
	GLAccountID  uint    `json:"gl_account_id"`
	Notes        string  `json:"notes"`
}

func UpsertAccountDetermination(req UpsertAccountDeterminationRequest) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if err := accounting.UpsertAccountDetermination(req.Module, req.PostingEvent, req.ItemCategory, req.GLAccountID, req.Notes); err != nil {
		return errResponse(err)
	}
	return okResponse("Account determination saved", nil)
}

func DeleteAccountDetermination(id uint) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if err := accounting.DeleteAccountDetermination(id); err != nil {
		return errResponse(err)
	}
	return okResponse("Rule removed", nil)
}

// --- Payment Method Accounts ---

func ListPaymentMethodAccounts() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	recs, err := accounting.ListPaymentMethodAccounts()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", recs)
}

type UpsertPaymentMethodAccountRequest struct {
	PaymentMethod string `json:"payment_method"`
	BankName      string `json:"bank_name"`
	GLAccountID   uint   `json:"gl_account_id"`
	Direction     string `json:"direction"`
}

func UpsertPaymentMethodAccount(req UpsertPaymentMethodAccountRequest) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if err := accounting.UpsertPaymentMethodAccount(req.PaymentMethod, req.BankName, req.GLAccountID, req.Direction); err != nil {
		return errResponse(err)
	}
	return okResponse("Payment method account saved", nil)
}

// --- Journal Entries ---

func ListJournalEntries(module string, limit int) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	entries, err := accounting.ListJournalEntries(module, limit)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", entries)
}

func GetJournalEntry(id uint) Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	je, err := accounting.GetJournalEntry(id)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", je)
}

type ManualJELine struct {
	GLAccountID uint    `json:"gl_account_id"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
	Description string  `json:"description"`
}

type ManualJERequest struct {
	Date      string         `json:"date"`
	Narration string         `json:"narration"`
	Lines     []ManualJELine `json:"lines"`
}

func PostManualJournalEntry(req ManualJERequest) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}

	// Load GL accounts for each line
	glLines := make([]accounting.JELine, 0, len(req.Lines))
	for _, l := range req.Lines {
		acct, err := accounting.GetGLAccount(l.GLAccountID)
		if err != nil {
			return errMsg("GL Account not found: " + err.Error())
		}
		glLines = append(glLines, accounting.JELine{
			Account:     acct,
			Debit:       l.Debit,
			Credit:      l.Credit,
			Description: l.Description,
		})
	}

	var entryDate *time.Time
	if req.Date != "" {
		t, err := time.Parse("2006-01-02", req.Date)
		if err == nil {
			entryDate = &t
		}
	}

	userID := middleware.Store.UserID()
	je, err := accounting.PostManualJournalEntry(accounting.PostJEParams{
		Module:      "MANUAL",
		SourceType:  "MANUAL",
		SourceRef:   "MANUAL",
		Narration:   req.Narration,
		Lines:       glLines,
		EntryDate:   entryDate,
		CreatedByID: userID,
	})
	if err != nil {
		return errResponse(err)
	}
	return okResponse("Journal entry posted", je)
}

func ReverseJournalEntry(id uint) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	userID := middleware.Store.UserID()
	rev, err := accounting.ReverseJE(id, userID)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("Journal entry reversed", rev)
}
