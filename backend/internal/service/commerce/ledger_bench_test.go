package commerce_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
)

type benchmarkLedgerStore struct {
	mu      sync.Mutex
	entries []*domainCommerce.LedgerEntry
	balance int64
}

func (s *benchmarkLedgerStore) Append(entry *domainCommerce.LedgerEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
	if entry.Direction == domainCommerce.DirectionCredit {
		s.balance += entry.AmountMinor
	} else {
		s.balance -= entry.AmountMinor
	}
}

func BenchmarkConcurrentLedgerPosting(b *testing.B) {
	store := &benchmarkLedgerStore{
		entries: make([]*domainCommerce.LedgerEntry, 0, b.N),
	}

	walletID := uuid.New()
	txID := uuid.New()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			entry := &domainCommerce.LedgerEntry{
				ID:            uuid.New(),
				TransactionID: txID,
				AccountType:   domainCommerce.AccountUserWallet,
				AccountID:     walletID,
				Direction:     domainCommerce.DirectionCredit,
				AmountMinor:   100,
				Currency:      "USD",
				CreatedAt:     time.Now().UTC(),
			}
			store.Append(entry)
		}
	})

	b.StopTimer()
	expectedTotal := int64(b.N) * 100
	if store.balance != expectedTotal {
		b.Fatalf("expected ledger projected balance %d, got %d", expectedTotal, store.balance)
	}
}

func TestConcurrentLedgerIntegrityUnderLoad(t *testing.T) {
	store := &benchmarkLedgerStore{
		entries: make([]*domainCommerce.LedgerEntry, 0, 1000),
	}

	walletID := uuid.New()
	txID := uuid.New()

	var wg sync.WaitGroup
	concurrency := 50
	entriesPerWorker := 20

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < entriesPerWorker; j++ {
				amount := int64(100)
				direction := domainCommerce.DirectionCredit
				if j%2 == 1 {
					amount = 50
					direction = domainCommerce.DirectionDebit
				}
				entry := &domainCommerce.LedgerEntry{
					ID:            uuid.New(),
					TransactionID: txID,
					AccountType:   domainCommerce.AccountUserWallet,
					AccountID:     walletID,
					Direction:     direction,
					AmountMinor:   amount,
					Currency:      "USD",
					CreatedAt:     time.Now().UTC(),
				}
				store.Append(entry)
			}
		}(i)
	}

	wg.Wait()

	expectedBalance := int64(concurrency) * int64(entriesPerWorker/2) * (100 - 50)
	if store.balance != expectedBalance {
		t.Fatalf("expected ledger balance %d, got %d", expectedBalance, store.balance)
	}
	if len(store.entries) != concurrency*entriesPerWorker {
		t.Fatalf("expected %d entries, got %d", concurrency*entriesPerWorker, len(store.entries))
	}
}
