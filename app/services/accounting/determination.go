package accounting

import (
	"fmt"

	"ricemill/app/db"
	"ricemill/app/models"

	"gorm.io/gorm"
)

// ErrAccountDetermination is raised when a GL account mapping is not configured.
type ErrAccountDetermination struct {
	Module       string
	PostingEvent string
	Category     string
}

func (e *ErrAccountDetermination) Error() string {
	cat := e.Category
	if cat == "" {
		cat = "catch-all"
	}
	return fmt.Sprintf(
		"No GL account configured for %s / %s / %s. "+
			"Please set up Account Determination in Accounting → Account Determination.",
		e.Module, e.PostingEvent, cat,
	)
}

// GetAccount looks up the GL account for a posting event.
// Priority: specific item_category first, then catch-all (nil).
func GetAccount(tx *gorm.DB, module, postingEvent string, itemCategory ...string) (*models.GLAccount, error) {
	if tx == nil {
		tx = db.DB
	}
	cat := ""
	if len(itemCategory) > 0 {
		cat = itemCategory[0]
	}

	var rule models.AccountDetermination

	// Try specific category first
	if cat != "" {
		err := tx.Preload("GLAccount").
			Where("module = ? AND posting_event = ? AND item_category = ? AND is_active = 1", module, postingEvent, cat).
			First(&rule).Error
		if err == nil {
			return rule.GLAccount, nil
		}
	}

	// Fall back to catch-all (item_category IS NULL)
	err := tx.Preload("GLAccount").
		Where("module = ? AND posting_event = ? AND item_category IS NULL AND is_active = 1", module, postingEvent).
		First(&rule).Error
	if err != nil {
		return nil, &ErrAccountDetermination{Module: module, PostingEvent: postingEvent, Category: cat}
	}
	return rule.GLAccount, nil
}

// GetPaymentAccount returns the GL account for a payment method and direction.
func GetPaymentAccount(tx *gorm.DB, paymentMethod, direction string) (*models.GLAccount, error) {
	if tx == nil {
		tx = db.DB
	}

	var rec models.PaymentMethodAccount

	// Try exact direction match
	err := tx.Preload("GLAccount").
		Where("payment_method = ? AND direction = ? AND is_active = 1", paymentMethod, direction).
		First(&rec).Error
	if err == nil {
		return rec.GLAccount, nil
	}

	// Fall back to BOTH
	err = tx.Preload("GLAccount").
		Where("payment_method = ? AND direction = 'BOTH' AND is_active = 1", paymentMethod).
		First(&rec).Error
	if err != nil {
		return nil, &ErrAccountDetermination{
			Module:       "BANKING",
			PostingEvent: paymentMethod,
			Category:     direction,
		}
	}
	return rec.GLAccount, nil
}
