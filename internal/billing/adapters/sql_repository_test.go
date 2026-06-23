package adapters_test

import (
	"context"
	"database/sql"
	"testing"

	"ferrowin/internal/billing/adapters"
	"ferrowin/internal/billing/domain"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test SQLite DB: %v", err)
	}

	queries := []string{
		`CREATE TABLE terminals (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE invoicing_series (
			id TEXT PRIMARY KEY,
			terminal_id TEXT NOT NULL,
			prefix TEXT NOT NULL,
			next_sequence INTEGER NOT NULL DEFAULT 1
		)`,
	}

	for _, q := range queries {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			t.Fatalf("failed to run query %q: %v", q, err)
		}
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestSQLTerminalRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := adapters.NewSQLTerminalRepository(db, true)

	t.Run("Save and GetByID", func(t *testing.T) {
		terminalID := uuid.New()
		terminal := &domain.Terminal{
			ID:       terminalID,
			Name:     "TPV-001",
			IsActive: true,
		}

		err := repo.Save(ctx, terminal)
		if err != nil {
			t.Fatalf("failed to save terminal: %v", err)
		}

		fetched, err := repo.GetByID(ctx, terminalID)
		if err != nil {
			t.Fatalf("failed to get terminal: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected terminal to be found, got nil")
		}
		if fetched.ID != terminal.ID || fetched.Name != terminal.Name || fetched.IsActive != terminal.IsActive {
			t.Errorf("expected terminal %+v, got %+v", terminal, fetched)
		}
	})

	t.Run("GetByID returns nil for non-existent", func(t *testing.T) {
		fetched, err := repo.GetByID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fetched != nil {
			t.Fatal("expected nil for non-existent terminal")
		}
	})

	t.Run("Update existing terminal (upsert)", func(t *testing.T) {
		terminalID := uuid.New()

		original := &domain.Terminal{
			ID:       terminalID,
			Name:     "Original",
			IsActive: true,
		}
		err := repo.Save(ctx, original)
		if err != nil {
			t.Fatalf("failed to save original terminal: %v", err)
		}

		updated := &domain.Terminal{
			ID:       terminalID,
			Name:     "Updated",
			IsActive: false,
		}
		err = repo.Save(ctx, updated)
		if err != nil {
			t.Fatalf("failed to update terminal: %v", err)
		}

		fetched, err := repo.GetByID(ctx, terminalID)
		if err != nil {
			t.Fatalf("failed to get terminal: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected terminal to be found after upsert")
		}
		if fetched.Name != "Updated" || fetched.IsActive != false {
			t.Errorf("expected updated terminal {Name: Updated, IsActive: false}, got %+v", fetched)
		}
	})
}

func TestSQLInvoicingSeriesRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := adapters.NewSQLInvoicingSeriesRepository(db, true)

	t.Run("Save and GetByID", func(t *testing.T) {
		seriesID := uuid.New()
		terminalID := uuid.New()
		series := &domain.InvoicingSeries{
			ID:           seriesID,
			TerminalID:   terminalID,
			Prefix:       "FAC",
			NextSequence: 1,
		}

		err := repo.Save(ctx, series)
		if err != nil {
			t.Fatalf("failed to save invoicing series: %v", err)
		}

		fetched, err := repo.GetByID(ctx, seriesID)
		if err != nil {
			t.Fatalf("failed to get invoicing series: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected invoicing series to be found, got nil")
		}
		if fetched.ID != series.ID || fetched.TerminalID != series.TerminalID ||
			fetched.Prefix != series.Prefix || fetched.NextSequence != series.NextSequence {
			t.Errorf("expected series %+v, got %+v", series, fetched)
		}
	})

	t.Run("GetByTerminalID returns correct series", func(t *testing.T) {
		terminalID := uuid.New()
		seriesID := uuid.New()
		series := &domain.InvoicingSeries{
			ID:           seriesID,
			TerminalID:   terminalID,
			Prefix:       "FAC",
			NextSequence: 5,
		}

		err := repo.Save(ctx, series)
		if err != nil {
			t.Fatalf("failed to save series: %v", err)
		}

		fetched, err := repo.GetByTerminalID(ctx, terminalID)
		if err != nil {
			t.Fatalf("failed to get series by terminal ID: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected series to be found by terminal ID")
		}
		if fetched.Prefix != "FAC" || fetched.NextSequence != 5 {
			t.Errorf("expected prefix FAC and next_sequence 5, got prefix %s and next_sequence %d", fetched.Prefix, fetched.NextSequence)
		}
	})

	t.Run("GetByTerminalID returns nil for non-existent terminal", func(t *testing.T) {
		fetched, err := repo.GetByTerminalID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fetched != nil {
			t.Fatal("expected nil for non-existent terminal")
		}
	})

	t.Run("IncrementSequence returns prefix + current seq and increments", func(t *testing.T) {
		terminalID := uuid.New()
		series := &domain.InvoicingSeries{
			ID:           uuid.New(),
			TerminalID:   terminalID,
			Prefix:       "FAC",
			NextSequence: 1,
		}
		err := repo.Save(ctx, series)
		if err != nil {
			t.Fatalf("failed to save series: %v", err)
		}

		prefix, seq, err := repo.IncrementSequence(ctx, terminalID)
		if err != nil {
			t.Fatalf("failed to increment sequence: %v", err)
		}
		if prefix != "FAC" {
			t.Errorf("expected prefix FAC, got %s", prefix)
		}
		if seq != 1 {
			t.Errorf("expected sequence 1, got %d", seq)
		}

		fetched, err := repo.GetByTerminalID(ctx, terminalID)
		if err != nil {
			t.Fatalf("failed to get series after increment: %v", err)
		}
		if fetched.NextSequence != 2 {
			t.Errorf("expected next_sequence 2 after increment, got %d", fetched.NextSequence)
		}
	})

	t.Run("IncrementSequence called twice increments correctly (1 -> 2 -> 3)", func(t *testing.T) {
		terminalID := uuid.New()
		series := &domain.InvoicingSeries{
			ID:           uuid.New(),
			TerminalID:   terminalID,
			Prefix:       "FAC",
			NextSequence: 1,
		}
		err := repo.Save(ctx, series)
		if err != nil {
			t.Fatalf("failed to save series: %v", err)
		}

		_, seq1, err := repo.IncrementSequence(ctx, terminalID)
		if err != nil {
			t.Fatalf("first increment failed: %v", err)
		}
		if seq1 != 1 {
			t.Errorf("expected first sequence 1, got %d", seq1)
		}

		_, seq2, err := repo.IncrementSequence(ctx, terminalID)
		if err != nil {
			t.Fatalf("second increment failed: %v", err)
		}
		if seq2 != 2 {
			t.Errorf("expected second sequence 2, got %d", seq2)
		}

		_, seq3, err := repo.IncrementSequence(ctx, terminalID)
		if err != nil {
			t.Fatalf("third increment failed: %v", err)
		}
		if seq3 != 3 {
			t.Errorf("expected third sequence 3, got %d", seq3)
		}

		fetched, err := repo.GetByTerminalID(ctx, terminalID)
		if err != nil {
			t.Fatalf("failed to get series: %v", err)
		}
		if fetched.NextSequence != 4 {
			t.Errorf("expected next_sequence 4 after three increments, got %d", fetched.NextSequence)
		}
	})

	t.Run("IncrementSequence errors for non-existent terminal", func(t *testing.T) {
		_, _, err := repo.IncrementSequence(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected error for non-existent terminal, got nil")
		}
	})
}
