package accounting

import (
	"fmt"
	"math"
	"time"

	"ricemill/app/models"
	"ricemill/app/services/docnumber"

	"gorm.io/gorm"
)

// JELine is a single debit or credit line passed to PostJournalEntry.
type JELine struct {
	Account     *models.GLAccount
	Debit       float64
	Credit      float64
	Description string
}

// PostJEParams holds all inputs needed to post a journal entry.
type PostJEParams struct {
	Module      string
	SourceType  string
	SourceID    *uint
	SourceRef   string
	Narration   string
	Lines       []JELine
	EntryDate   *time.Time
	CreatedByID *uint
}

// PostJournalEntry creates and immediately posts a JournalEntry within the given transaction.
// Must be called inside the same DB transaction as the triggering business event.
func PostJournalEntry(tx *gorm.DB, params PostJEParams) (*models.JournalEntry, error) {
	postDate := time.Now()
	if params.EntryDate != nil {
		postDate = *params.EntryDate
	}

	// Validate balance
	totalDr := 0.0
	totalCr := 0.0
	for _, l := range params.Lines {
		totalDr += l.Debit
		totalCr += l.Credit
	}
	totalDr = math.Round(totalDr*100) / 100
	totalCr = math.Round(totalCr*100) / 100
	if math.Abs(totalDr-totalCr) > 0.005 {
		return nil, fmt.Errorf(
			"journal entry for %s does not balance: Dr=%.2f Cr=%.2f",
			params.SourceRef, totalDr, totalCr,
		)
	}

	// Generate JE number (mutex-protected) on the caller's transaction.
	jeNum, err := docnumber.Svc.NextJENumber(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JE number: %w", err)
	}

	je := models.JournalEntry{
		EntryNumber: jeNum,
		Date:        postDate,
		Module:      params.Module,
		SourceType:  params.SourceType,
		SourceID:    params.SourceID,
		SourceRef:   params.SourceRef,
		Narration:   params.Narration,
		Status:      "POSTED",
		IsReversal:  false,
		CreatedByID: params.CreatedByID,
		CreatedAt:   time.Now(),
	}

	if err := tx.Create(&je).Error; err != nil {
		return nil, fmt.Errorf("failed to create journal entry: %w", err)
	}

	for idx, line := range params.Lines {
		var debit, credit *float64
		if line.Debit > 0 {
			v := math.Round(line.Debit*100) / 100
			debit = &v
		}
		if line.Credit > 0 {
			v := math.Round(line.Credit*100) / 100
			credit = &v
		}
		jel := models.JournalEntryLine{
			JournalEntryID: je.ID,
			LineNumber:     idx + 1,
			GLAccountID:    line.Account.ID,
			Debit:          debit,
			Credit:         credit,
			Description:    line.Description,
		}
		if err := tx.Create(&jel).Error; err != nil {
			return nil, fmt.Errorf("failed to create journal entry line %d: %w", idx+1, err)
		}
	}

	return &je, nil
}

// ReverseJournalEntry auto-generates a reversal JournalEntry for originalJE.
// Swaps all debits↔credits and marks original as REVERSED.
// Must be called inside the same DB transaction.
func ReverseJournalEntry(tx *gorm.DB, originalJE *models.JournalEntry, reversalDate *time.Time, createdByID *uint) (*models.JournalEntry, error) {
	if originalJE.Status == "REVERSED" {
		return nil, fmt.Errorf("journal entry %s is already reversed", originalJE.EntryNumber)
	}

	// Load lines if not already loaded
	if len(originalJE.Lines) == 0 {
		if err := tx.Preload("Lines").First(originalJE, originalJE.ID).Error; err != nil {
			return nil, fmt.Errorf("failed to load original JE lines: %w", err)
		}
	}

	revDate := time.Now()
	if reversalDate != nil {
		revDate = *reversalDate
	}

	jeNum, err := docnumber.Svc.NextJENumber(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate reversal JE number: %w", err)
	}

	origID := originalJE.ID
	rev := models.JournalEntry{
		EntryNumber:     jeNum,
		Date:            revDate,
		Module:          originalJE.Module,
		SourceType:      "REVERSAL",
		SourceID:        originalJE.SourceID,
		SourceRef:       "REV-" + originalJE.SourceRef,
		Narration:       fmt.Sprintf("Reversal of %s: %s", originalJE.EntryNumber, originalJE.Narration),
		Status:          "POSTED",
		IsReversal:      true,
		ReversedEntryID: &origID,
		CreatedByID:     createdByID,
		CreatedAt:       time.Now(),
	}

	if err := tx.Create(&rev).Error; err != nil {
		return nil, fmt.Errorf("failed to create reversal journal entry: %w", err)
	}

	for idx, origLine := range originalJE.Lines {
		jel := models.JournalEntryLine{
			JournalEntryID: rev.ID,
			LineNumber:     idx + 1,
			GLAccountID:    origLine.GLAccountID,
			Debit:          origLine.Credit, // swap
			Credit:         origLine.Debit,  // swap
			Description:    "Reversal: " + origLine.Description,
		}
		if err := tx.Create(&jel).Error; err != nil {
			return nil, fmt.Errorf("failed to create reversal line %d: %w", idx+1, err)
		}
	}

	// Mark original as reversed
	revID := rev.ID
	if err := tx.Model(originalJE).Updates(map[string]interface{}{
		"status":            "REVERSED",
		"reversed_entry_id": revID,
		"updated_by_id":     createdByID,
		"version":           gorm.Expr("version + 1"),
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to mark original JE as reversed: %w", err)
	}

	return &rev, nil
}
