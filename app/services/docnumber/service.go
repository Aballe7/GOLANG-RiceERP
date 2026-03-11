package docnumber

import (
	"fmt"
	"sync"
	"time"

	"egglayererp/app/db"
	"egglayererp/app/models"
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

func (s *Service) next(tableName, field, prefix string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result struct {
		MaxNum string
	}
	query := fmt.Sprintf("SELECT MAX(%s) AS max_num FROM %s WHERE %s LIKE ?", field, tableName, field)
	db.DB.Raw(query, prefix+"%").Scan(&result)

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

// NextInvoiceNumber generates an SO number: SO-YYYYMMDD-XXXX
func (s *Service) NextInvoiceNumber() (string, error) {
	prefix := "SO-" + s.today() + "-"
	return s.next(models.SalesOrder{}.TableName(), "sales_order_number", prefix)
}

// NextDeliveryNumber generates a DO number: DO-YYYYMMDD-XXXX
func (s *Service) NextDeliveryNumber() (string, error) {
	prefix := "DO-" + s.today() + "-"
	return s.next(models.DeliveryOrder{}.TableName(), "delivery_number", prefix)
}

// NextARNumber generates an AR invoice number: AR-YYYYMMDD-XXXX
func (s *Service) NextARNumber() (string, error) {
	prefix := "AR-" + s.today() + "-"
	return s.next(models.ARInvoice{}.TableName(), "invoice_number", prefix)
}

// NextCollectionNumber generates a collection number: CR-YYYYMMDD-XXXX
func (s *Service) NextCollectionNumber() (string, error) {
	prefix := "CR-" + s.today() + "-"
	return s.next(models.Collection{}.TableName(), "collection_number", prefix)
}

// NextDRNumber generates a delivery receipt number: DR-YYYYMMDD-XXXX
func (s *Service) NextDRNumber() (string, error) {
	prefix := "DR-" + s.today() + "-"
	return s.next(models.DeliveryReceipt{}.TableName(), "dr_number", prefix)
}

// NextAPNumber generates an AP invoice number: AP-YYYYMMDD-XXXX
func (s *Service) NextAPNumber() (string, error) {
	prefix := "AP-" + s.today() + "-"
	return s.next(models.APInvoice{}.TableName(), "invoice_number", prefix)
}

// NextPaymentNumber generates an AP payment number: PAY-YYYYMMDD-XXXX
func (s *Service) NextPaymentNumber() (string, error) {
	prefix := "PAY-" + s.today() + "-"
	return s.next(models.APPayment{}.TableName(), "payment_number", prefix)
}

// NextJENumber generates a journal entry number: JE-YYYYMMDD-XXXX
func (s *Service) NextJENumber() (string, error) {
	prefix := "JE-" + s.today() + "-"
	return s.next(models.JournalEntry{}.TableName(), "entry_number", prefix)
}
