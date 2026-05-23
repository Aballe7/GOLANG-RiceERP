package models

import "time"

// GLAccount — Chart of Accounts.
// account_type: HEADER (group node) or POSTING (leaf, transactions post here)
// CashFlowClass values for PAS 7 Statement of Cash Flows classification:
//   CASH                  — Cash & cash equivalents (opening/closing balance)
//   OPERATING_NON_CASH    — Non-cash expense/income: depreciation, impairment, provisions
//   OPERATING_WC_ASSET    — Working-capital assets: AR, inventory, prepaid, GRNI, VAT receivable
//   OPERATING_WC_LIABILITY — Working-capital liabilities: AP, VAT payable, WHT payable, accrued
//   OPERATING_TAX         — Income tax payable
//   INVESTING_PPE         — PPE assets: land, buildings, equipment, vehicles
//   FINANCING_DEBT        — Loans payable (long/short term)
//   FINANCING_EQUITY      — Owner's capital / paid-in capital
//   "" (empty)            — Not classified (header accounts, retained earnings, P&L accounts)
type GLAccount struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Code          string    `gorm:"type:varchar(20);uniqueIndex;not null" json:"code"`
	Name          string    `gorm:"type:varchar(150);not null" json:"name"`
	Section       string    `gorm:"type:varchar(20);not null" json:"section"` // ASSET|LIABILITY|EQUITY|REVENUE|EXPENSE
	AccountType   string    `gorm:"type:varchar(10);not null;default:'POSTING'" json:"account_type"` // HEADER|POSTING
	NormalBalance string    `gorm:"type:varchar(6);not null" json:"normal_balance"` // DEBIT|CREDIT
	CashFlowClass string    `gorm:"type:varchar(30);not null;default:''" json:"cash_flow_class"` // PAS 7 classification
	ParentID      *uint     `gorm:"index" json:"parent_id"`
	IsActive      bool      `gorm:"type:tinyint(1);default:1" json:"is_active"`
	IsSystem      bool      `gorm:"type:tinyint(1);default:0" json:"is_system"`
	Description   string    `gorm:"type:text" json:"description"`
	CreatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`

	Parent   *GLAccount  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []GLAccount `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (GLAccount) TableName() string { return "gl_account" }

func (a *GLAccount) FullName() string {
	return a.Code + "  " + a.Name
}

// AccountDetermination maps posting_event + item_category → gl_account.
// item_category=nil means catch-all.
type AccountDetermination struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Module       string `gorm:"type:varchar(20);not null" json:"module"` // SALES|PURCHASING|INVENTORY|BANKING
	PostingEvent string `gorm:"type:varchar(40);not null" json:"posting_event"`
	ItemCategory *string `gorm:"type:varchar(50)" json:"item_category"` // null = catch-all
	GLAccountID  uint   `gorm:"not null;index" json:"gl_account_id"`
	IsActive     bool   `gorm:"type:tinyint(1);default:1" json:"is_active"`
	Notes        string `gorm:"type:text" json:"notes"`

	GLAccount *GLAccount `gorm:"foreignKey:GLAccountID" json:"gl_account,omitempty"`
}

func (AccountDetermination) TableName() string { return "account_determination" }

// PaymentMethodAccount maps payment_method + direction → gl_account (cash/bank).
// direction: INFLOW | OUTFLOW | BOTH
type PaymentMethodAccount struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	PaymentMethod string `gorm:"type:varchar(30);not null" json:"payment_method"`
	BankName      string `gorm:"type:varchar(100)" json:"bank_name"`
	GLAccountID   uint   `gorm:"not null;index" json:"gl_account_id"`
	Direction     string `gorm:"type:varchar(10);not null;default:'BOTH'" json:"direction"` // INFLOW|OUTFLOW|BOTH
	IsActive      bool   `gorm:"type:tinyint(1);default:1" json:"is_active"`

	GLAccount *GLAccount `gorm:"foreignKey:GLAccountID" json:"gl_account,omitempty"`
}

func (PaymentMethodAccount) TableName() string { return "payment_method_account" }

// JournalEntry — header of every accounting transaction.
type JournalEntry struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	EntryNumber     string    `gorm:"type:varchar(30);uniqueIndex;not null" json:"entry_number"`
	Date            time.Time `gorm:"type:date;not null" json:"date"`
	Module          string    `gorm:"type:varchar(20);not null" json:"module"` // SALES|PURCHASING|INVENTORY|MANUAL
	SourceType      string    `gorm:"type:varchar(30);not null" json:"source_type"` // AR_INVOICE|COLLECTION|AP_INVOICE|AP_PAYMENT|DR_CONFIRM|DAILY_LOG|MANUAL|REVERSAL
	SourceID        *uint     `json:"source_id"`
	SourceRef       string    `gorm:"type:varchar(50)" json:"source_ref"`
	Narration       string    `gorm:"type:text" json:"narration"`
	Status          string    `gorm:"type:varchar(10);not null;default:'POSTED'" json:"status"` // POSTED|REVERSED
	IsReversal      bool      `gorm:"type:tinyint(1);default:0" json:"is_reversal"`
	ReversedEntryID *uint     `gorm:"index" json:"reversed_entry_id"`
	CreatedByID     *uint     `json:"created_by_id"`
	UpdatedByID     *uint     `json:"updated_by_id"`
	Version         uint      `gorm:"default:1" json:"version"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`

	Lines         []JournalEntryLine `gorm:"foreignKey:JournalEntryID;constraint:OnDelete:CASCADE" json:"lines,omitempty"`
	ReversedEntry *JournalEntry      `gorm:"foreignKey:ReversedEntryID" json:"reversed_entry,omitempty"`
}

func (JournalEntry) TableName() string { return "journal_entry" }

func (je *JournalEntry) TotalDebit() float64 {
	total := 0.0
	for _, l := range je.Lines {
		if l.Debit != nil {
			total += *l.Debit
		}
	}
	return total
}

func (je *JournalEntry) TotalCredit() float64 {
	total := 0.0
	for _, l := range je.Lines {
		if l.Credit != nil {
			total += *l.Credit
		}
	}
	return total
}

func (je *JournalEntry) IsBalanced() bool {
	diff := je.TotalDebit() - je.TotalCredit()
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.005
}

// JournalEntryLine — one debit or credit line within a JournalEntry.
type JournalEntryLine struct {
	ID             uint     `gorm:"primaryKey" json:"id"`
	JournalEntryID uint     `gorm:"not null;index" json:"journal_entry_id"`
	LineNumber     int      `gorm:"not null" json:"line_number"`
	GLAccountID    uint     `gorm:"not null;index" json:"gl_account_id"`
	Debit          *float64 `gorm:"type:decimal(15,4)" json:"debit"`
	Credit         *float64 `gorm:"type:decimal(15,4)" json:"credit"`
	Description    string   `gorm:"type:varchar(200)" json:"description"`

	GLAccount *GLAccount `gorm:"foreignKey:GLAccountID" json:"gl_account,omitempty"`
}

func (JournalEntryLine) TableName() string { return "je_line" }
