package docnumber

import (
	"fmt"
	"sync"
	"time"

	"ricemill/app/db"
	"ricemill/app/models"

	"gorm.io/gorm"
)

// Service generates sequential document numbers with mutex protection.
// Format: PREFIX-YYYYMMDD-XXXX (e.g. INV-20250309-0001)
type Service struct {
	mu sync.Mutex
}

var Svc = &Service{}

func (s *Service) today() string {
	return time.Now().Format("20060102")
}

// pickConn returns the caller-supplied transaction if one was passed, else db.DB.
// Callers that generate a number *inside* a transaction must pass their tx so the
// MAX() read runs on the same connection — it then sees rows written earlier in the
// transaction (avoiding duplicate numbers) and can't deadlock against its own write
// lock on shared-cache SQLite.
func pickConn(conn []*gorm.DB) *gorm.DB {
	if len(conn) > 0 && conn[0] != nil {
		return conn[0]
	}
	return db.DB
}

func (s *Service) next(tableName, field, prefix string, conn *gorm.DB) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result struct {
		MaxNum string
	}
	query := fmt.Sprintf("SELECT MAX(%s) AS max_num FROM %s WHERE %s LIKE ?", field, tableName, field)
	conn.Raw(query, prefix+"%").Scan(&result)

	seq := 1
	if result.MaxNum != "" {
		// Parse last 4 digits after final '-'
		for i := len(result.MaxNum) - 1; i >= 0; i-- {
			if result.MaxNum[i] == '-' {
				fmt.Sscanf(result.MaxNum[i+1:], "%d", &seq)
				seq++
				break
			}
		}
	}
	return fmt.Sprintf("%s%04d", prefix, seq), nil
}

// Each Next*Number method optionally accepts the caller's transaction. Pass it when
// generating a number inside a transaction; omit it otherwise.

// NextInvoiceNumber generates an SO number: SO-YYYYMMDD-XXXX
func (s *Service) NextInvoiceNumber(conn ...*gorm.DB) (string, error) {
	prefix := "SO-" + s.today() + "-"
	return s.next(models.SalesOrder{}.TableName(), "sales_order_number", prefix, pickConn(conn))
}

// NextDeliveryNumber generates a DO number: DO-YYYYMMDD-XXXX
func (s *Service) NextDeliveryNumber(conn ...*gorm.DB) (string, error) {
	prefix := "DO-" + s.today() + "-"
	return s.next(models.DeliveryOrder{}.TableName(), "delivery_number", prefix, pickConn(conn))
}

// NextARNumber generates an AR invoice number: AR-YYYYMMDD-XXXX
func (s *Service) NextARNumber(conn ...*gorm.DB) (string, error) {
	prefix := "AR-" + s.today() + "-"
	return s.next(models.ARInvoice{}.TableName(), "invoice_number", prefix, pickConn(conn))
}

// NextCollectionNumber generates a collection number: CR-YYYYMMDD-XXXX
func (s *Service) NextCollectionNumber(conn ...*gorm.DB) (string, error) {
	prefix := "CR-" + s.today() + "-"
	return s.next(models.Collection{}.TableName(), "collection_number", prefix, pickConn(conn))
}

// NextDRNumber generates a delivery receipt number: DR-YYYYMMDD-XXXX
func (s *Service) NextDRNumber(conn ...*gorm.DB) (string, error) {
	prefix := "DR-" + s.today() + "-"
	return s.next(models.DeliveryReceipt{}.TableName(), "dr_number", prefix, pickConn(conn))
}

// NextAPNumber generates an AP invoice number: AP-YYYYMMDD-XXXX
func (s *Service) NextAPNumber(conn ...*gorm.DB) (string, error) {
	prefix := "AP-" + s.today() + "-"
	return s.next(models.APInvoice{}.TableName(), "invoice_number", prefix, pickConn(conn))
}

// NextPaymentNumber generates an AP payment number: PAY-YYYYMMDD-XXXX
func (s *Service) NextPaymentNumber(conn ...*gorm.DB) (string, error) {
	prefix := "PAY-" + s.today() + "-"
	return s.next(models.APPayment{}.TableName(), "payment_number", prefix, pickConn(conn))
}

// NextJENumber generates a journal entry number: JE-YYYYMMDD-XXXX
func (s *Service) NextJENumber(conn ...*gorm.DB) (string, error) {
	prefix := "JE-" + s.today() + "-"
	return s.next(models.JournalEntry{}.TableName(), "entry_number", prefix, pickConn(conn))
}

// NextGRNumber generates a goods receipt number: GR-YYYYMMDD-XXXX
func (s *Service) NextGRNumber(conn ...*gorm.DB) (string, error) {
	prefix := "GR-" + s.today() + "-"
	return s.next(models.GoodsReceipt{}.TableName(), "gr_number", prefix, pickConn(conn))
}

// NextGINumber generates a goods issue number: GI-YYYYMMDD-XXXX
func (s *Service) NextGINumber(conn ...*gorm.DB) (string, error) {
	prefix := "GI-" + s.today() + "-"
	return s.next(models.GoodsIssue{}.TableName(), "gi_number", prefix, pickConn(conn))
}

// NextMONumber generates a milling order number: MO-YYYYMMDD-XXXX
// Queries owor (WorkOrder) since milling orders are now stored there.
func (s *Service) NextMONumber(conn ...*gorm.DB) (string, error) {
	prefix := "MO-" + s.today() + "-"
	return s.next(models.WorkOrder{}.TableName(), "doc_num", prefix, pickConn(conn))
}

// NextWONumber generates a production work order number: WO-YYYYMMDD-XXXX
func (s *Service) NextWONumber(conn ...*gorm.DB) (string, error) {
	prefix := "WO-" + s.today() + "-"
	return s.next(models.WorkOrder{}.TableName(), "doc_num", prefix, pickConn(conn))
}
