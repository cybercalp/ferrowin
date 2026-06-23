package domain_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"ferrowin/internal/billing/adapters"
	"ferrowin/internal/billing/domain"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Use a shared cache in-memory database to allow concurrent connections in tests.
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// SQLite busy timeout lets concurrent transactions wait rather than fail immediately
	_, err = db.Exec("PRAGMA busy_timeout = 5000")
	if err != nil {
		db.Close()
		t.Fatalf("failed to set busy_timeout: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE terminals (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			is_active BOOLEAN DEFAULT TRUE
		);
		CREATE TABLE invoicing_series (
			id TEXT PRIMARY KEY,
			terminal_id TEXT UNIQUE REFERENCES terminals(id) ON DELETE RESTRICT,
			prefix TEXT UNIQUE NOT NULL,
			next_sequence INTEGER NOT NULL DEFAULT 1
		);
	`)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create tables: %v", err)
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestBillingService_GenerateInvoiceNumber(t *testing.T) {
	ctx := context.Background()

	t.Run("Scenario: Sequence increment", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		termRepo := adapters.NewSQLTerminalRepository(db, true)
		seriesRepo := adapters.NewSQLInvoicingSeriesRepository(db, true)
		billingSvc := domain.NewBillingService(seriesRepo)

		terminalID := uuid.New()
		term := &domain.Terminal{
			ID:       terminalID,
			Name:     "T-01",
			IsActive: true,
		}
		if err := termRepo.Save(ctx, term); err != nil {
			t.Fatalf("failed to save terminal: %v", err)
		}

		seriesID := uuid.New()
		series := &domain.InvoicingSeries{
			ID:           seriesID,
			TerminalID:   terminalID,
			Prefix:       "S1",
			NextSequence: 16, // Configured "at sequence 15" -> next sequence is 16
		}
		if err := seriesRepo.Save(ctx, series); err != nil {
			t.Fatalf("failed to save invoicing series: %v", err)
		}

		// Generate invoice number
		invoiceNum, seq, err := billingSvc.GenerateInvoiceNumber(ctx, terminalID)
		if err != nil {
			t.Fatalf("failed to generate invoice number: %v", err)
		}

		if invoiceNum != "S1-16" {
			t.Errorf("expected invoice number 'S1-16', got '%s'", invoiceNum)
		}
		if seq != 16 {
			t.Errorf("expected sequence 16, got %d", seq)
		}

		// Verify database state updated next_sequence to 17
		updatedSeries, err := seriesRepo.GetByID(ctx, seriesID)
		if err != nil {
			t.Fatalf("failed to get invoicing series: %v", err)
		}
		if updatedSeries.NextSequence != 17 {
			t.Errorf("expected next sequence in db to update to 17, got %d", updatedSeries.NextSequence)
		}
	})

	t.Run("Scenario: Prefix isolation", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		termRepo := adapters.NewSQLTerminalRepository(db, true)
		seriesRepo := adapters.NewSQLInvoicingSeriesRepository(db, true)
		billingSvc := domain.NewBillingService(seriesRepo)

		// Create Terminal 1 and Series S1
		t1ID := uuid.New()
		t1 := &domain.Terminal{ID: t1ID, Name: "T-01", IsActive: true}
		if err := termRepo.Save(ctx, t1); err != nil {
			t.Fatalf("failed to save terminal 1: %v", err)
		}
		s1 := &domain.InvoicingSeries{ID: uuid.New(), TerminalID: t1ID, Prefix: "S1", NextSequence: 1}
		if err := seriesRepo.Save(ctx, s1); err != nil {
			t.Fatalf("failed to save invoicing series 1: %v", err)
		}

		// Create Terminal 2 and Series S2
		t2ID := uuid.New()
		t2 := &domain.Terminal{ID: t2ID, Name: "T-02", IsActive: true}
		if err := termRepo.Save(ctx, t2); err != nil {
			t.Fatalf("failed to save terminal 2: %v", err)
		}
		s2 := &domain.InvoicingSeries{ID: uuid.New(), TerminalID: t2ID, Prefix: "S2", NextSequence: 1}
		if err := seriesRepo.Save(ctx, s2); err != nil {
			t.Fatalf("failed to save invoicing series 2: %v", err)
		}

		// Run concurrently for T-01 and T-02
		var wg sync.WaitGroup
		wg.Add(2)

		var inv1, inv2 string
		var err1, err2 error

		go func() {
			defer wg.Done()
			inv1, _, err1 = billingSvc.GenerateInvoiceNumber(ctx, t1ID)
		}()

		go func() {
			defer wg.Done()
			inv2, _, err2 = billingSvc.GenerateInvoiceNumber(ctx, t2ID)
		}()

		wg.Wait()

		if err1 != nil {
			t.Errorf("error generating invoice for T-01: %v", err1)
		}
		if err2 != nil {
			t.Errorf("error generating invoice for T-02: %v", err2)
		}

		if inv1 != "S1-1" {
			t.Errorf("expected T-01 to assign prefix S1, got invoice %s", inv1)
		}
		if inv2 != "S2-1" {
			t.Errorf("expected T-02 to assign prefix S2, got invoice %s", inv2)
		}
	})

	t.Run("Scenario: Safe concurrent increments", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		termRepo := adapters.NewSQLTerminalRepository(db, true)
		seriesRepo := adapters.NewSQLInvoicingSeriesRepository(db, true)
		billingSvc := domain.NewBillingService(seriesRepo)

		terminalID := uuid.New()
		term := &domain.Terminal{ID: terminalID, Name: "T-01", IsActive: true}
		if err := termRepo.Save(ctx, term); err != nil {
			t.Fatalf("failed to save terminal: %v", err)
		}

		seriesID := uuid.New()
		series := &domain.InvoicingSeries{ID: seriesID, TerminalID: terminalID, Prefix: "S1", NextSequence: 1}
		if err := seriesRepo.Save(ctx, series); err != nil {
			t.Fatalf("failed to save invoicing series: %v", err)
		}

		const numRequests = 50
		var wg sync.WaitGroup
		wg.Add(numRequests)

		results := make(chan int, numRequests)
		errs := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				defer wg.Done()
				_, seq, err := billingSvc.GenerateInvoiceNumber(ctx, terminalID)
				if err != nil {
					errs <- err
					return
				}
				results <- seq
			}()
		}

		wg.Wait()
		close(results)
		close(errs)

		if len(errs) > 0 {
			for err := range errs {
				t.Errorf("concurrent generation error: %v", err)
			}
			t.FailNow()
		}

		// Verify we got unique sequences from 1 to numRequests
		seen := make(map[int]bool)
		for seq := range results {
			if seen[seq] {
				t.Errorf("duplicate sequence number generated: %d", seq)
			}
			seen[seq] = true
			if seq < 1 || seq > numRequests {
				t.Errorf("sequence number out of range: %d", seq)
			}
		}

		if len(seen) != numRequests {
			t.Errorf("expected %d unique sequences, got %d", numRequests, len(seen))
		}

		// Verify DB value has been updated to 51
		updatedSeries, err := seriesRepo.GetByID(ctx, seriesID)
		if err != nil {
			t.Fatalf("failed to get invoicing series: %v", err)
		}
		if updatedSeries.NextSequence != numRequests+1 {
			t.Errorf("expected final next sequence to be %d, got %d", numRequests+1, updatedSeries.NextSequence)
		}
	})
}
