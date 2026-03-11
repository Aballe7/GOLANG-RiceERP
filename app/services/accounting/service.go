package accounting

import (
	"fmt"
	"time"

	"egglayererp/app/db"
	"egglayererp/app/models"

	"gorm.io/gorm"
)

// ListGLAccounts returns all GL accounts ordered by code.
func ListGLAccounts(activeOnly bool) ([]models.GLAccount, error) {
	var accounts []models.GLAccount
	q := db.DB.Order("code")
	if activeOnly {
		q = q.Where("is_active = 1")
	}
	return accounts, q.Find(&accounts).Error
}

// GetGLAccount fetches a single account by ID with children preloaded.
func GetGLAccount(id uint) (*models.GLAccount, error) {
	var acct models.GLAccount
	err := db.DB.Preload("Children").First(&acct, id).Error
	return &acct, err
}

// CreateGLAccount creates a new GL account.
func CreateGLAccount(acct *models.GLAccount) error {
	acct.CreatedAt = time.Now()
	return db.DB.Create(acct).Error
}

// UpdateGLAccount updates an existing GL account.
func UpdateGLAccount(id uint, updates map[string]interface{}) error {
	return db.DB.Model(&models.GLAccount{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteGLAccount soft-deletes (deactivates) a GL account.
func DeleteGLAccount(id uint) error {
	var acct models.GLAccount
	if err := db.DB.First(&acct, id).Error; err != nil {
		return err
	}
	if acct.IsSystem {
		return fmt.Errorf("cannot delete system account %s", acct.Code)
	}
	return db.DB.Model(&acct).Update("is_active", false).Error
}

// ListAccountDeterminations returns all account determination rules.
func ListAccountDeterminations() ([]models.AccountDetermination, error) {
	var rules []models.AccountDetermination
	err := db.DB.Preload("GLAccount").Order("module, posting_event").Find(&rules).Error
	return rules, err
}

// DeleteAccountDetermination removes a determination rule by ID.
func DeleteAccountDetermination(id uint) error {
	return db.DB.Delete(&models.AccountDetermination{}, id).Error
}

// UpsertAccountDetermination creates or updates a determination rule.
func UpsertAccountDetermination(module, postingEvent string, itemCategory *string, glAccountID uint, notes string) error {
	var rule models.AccountDetermination
	q := db.DB.Where("module = ? AND posting_event = ?", module, postingEvent)
	if itemCategory != nil {
		q = q.Where("item_category = ?", *itemCategory)
	} else {
		q = q.Where("item_category IS NULL")
	}

	if err := q.First(&rule).Error; err == gorm.ErrRecordNotFound {
		rule = models.AccountDetermination{
			Module:       module,
			PostingEvent: postingEvent,
			ItemCategory: itemCategory,
			GLAccountID:  glAccountID,
			IsActive:     true,
			Notes:        notes,
		}
		return db.DB.Create(&rule).Error
	}
	return db.DB.Model(&rule).Updates(map[string]interface{}{
		"gl_account_id": glAccountID,
		"is_active":     true,
		"notes":         notes,
	}).Error
}

// ListPaymentMethodAccounts returns all payment method account mappings.
func ListPaymentMethodAccounts() ([]models.PaymentMethodAccount, error) {
	var recs []models.PaymentMethodAccount
	err := db.DB.Preload("GLAccount").Order("payment_method, direction").Find(&recs).Error
	return recs, err
}

// UpsertPaymentMethodAccount creates or updates a payment method account mapping.
func UpsertPaymentMethodAccount(paymentMethod, bankName string, glAccountID uint, direction string) error {
	var rec models.PaymentMethodAccount
	q := db.DB.Where("payment_method = ? AND direction = ?", paymentMethod, direction)
	if bankName != "" {
		q = q.Where("bank_name = ?", bankName)
	}
	if err := q.First(&rec).Error; err == gorm.ErrRecordNotFound {
		rec = models.PaymentMethodAccount{
			PaymentMethod: paymentMethod,
			BankName:      bankName,
			GLAccountID:   glAccountID,
			Direction:     direction,
			IsActive:      true,
		}
		return db.DB.Create(&rec).Error
	}
	return db.DB.Model(&rec).Updates(map[string]interface{}{
		"gl_account_id": glAccountID,
		"is_active":     true,
	}).Error
}

// ListJournalEntries returns journal entries with optional filters.
func ListJournalEntries(module string, limit int) ([]models.JournalEntry, error) {
	var entries []models.JournalEntry
	q := db.DB.Preload("Lines.GLAccount").Order("created_at DESC")
	if module != "" {
		q = q.Where("module = ?", module)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	return entries, q.Find(&entries).Error
}

// GetJournalEntry fetches a single journal entry with all lines.
func GetJournalEntry(id uint) (*models.JournalEntry, error) {
	var je models.JournalEntry
	err := db.DB.Preload("Lines.GLAccount").First(&je, id).Error
	return &je, err
}

// PostManualJournalEntry creates a manual journal entry.
func PostManualJournalEntry(params PostJEParams) (*models.JournalEntry, error) {
	var je *models.JournalEntry
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		je, err = PostJournalEntry(tx, params)
		return err
	})
	return je, err
}

// ReverseJE reverses an existing journal entry.
func ReverseJE(id uint, userID *uint) (*models.JournalEntry, error) {
	var rev *models.JournalEntry
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var original models.JournalEntry
		if err := tx.Preload("Lines").First(&original, id).Error; err != nil {
			return err
		}
		var err error
		rev, err = ReverseJournalEntry(tx, &original, nil, userID)
		return err
	})
	return rev, err
}
