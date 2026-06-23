package adapters

import (
	"context"
	"database/sql"
	"fmt"

	"ferrowin/internal/billing/domain"

	"github.com/google/uuid"
)

// SQLTerminalRepository implements ports.TerminalRepository.
type SQLTerminalRepository struct {
	db       *sql.DB
	isSQLite bool
}

// NewSQLTerminalRepository creates a new SQLTerminalRepository.
func NewSQLTerminalRepository(db *sql.DB, isSQLite bool) *SQLTerminalRepository {
	return &SQLTerminalRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

// GetByID retrieves a terminal by its ID.
func (r *SQLTerminalRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Terminal, error) {
	query := "SELECT id, name, is_active FROM terminals WHERE id = $1"
	if r.isSQLite {
		query = "SELECT id, name, is_active FROM terminals WHERE id = ?"
	}
	var t domain.Terminal
	var idStr string
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(&idStr, &t.Name, &t.IsActive)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	t.ID = parsedID
	return &t, nil
}

// Save inserts or updates a terminal.
func (r *SQLTerminalRepository) Save(ctx context.Context, t *domain.Terminal) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO terminals (id, name, is_active) VALUES (?, ?, ?) 
                 ON CONFLICT(id) DO UPDATE SET name = excluded.name, is_active = excluded.is_active`
	} else {
		query = `INSERT INTO terminals (id, name, is_active) VALUES ($1, $2, $3) 
                 ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active`
	}
	_, err := r.db.ExecContext(ctx, query, t.ID.String(), t.Name, t.IsActive)
	return err
}

// SQLInvoicingSeriesRepository implements ports.InvoicingSeriesRepository.
type SQLInvoicingSeriesRepository struct {
	db       *sql.DB
	isSQLite bool
}

// NewSQLInvoicingSeriesRepository creates a new SQLInvoicingSeriesRepository.
func NewSQLInvoicingSeriesRepository(db *sql.DB, isSQLite bool) *SQLInvoicingSeriesRepository {
	return &SQLInvoicingSeriesRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

// GetByID retrieves an invoicing series by its ID.
func (r *SQLInvoicingSeriesRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.InvoicingSeries, error) {
	query := "SELECT id, terminal_id, prefix, next_sequence FROM invoicing_series WHERE id = $1"
	if r.isSQLite {
		query = "SELECT id, terminal_id, prefix, next_sequence FROM invoicing_series WHERE id = ?"
	}
	var s domain.InvoicingSeries
	var idStr, terminalIDStr string
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(&idStr, &terminalIDStr, &s.Prefix, &s.NextSequence)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	parsedTerminalID, err := uuid.Parse(terminalIDStr)
	if err != nil {
		return nil, err
	}
	s.ID = parsedID
	s.TerminalID = parsedTerminalID
	return &s, nil
}

// GetByTerminalID retrieves the invoicing series for a terminal.
func (r *SQLInvoicingSeriesRepository) GetByTerminalID(ctx context.Context, terminalID uuid.UUID) (*domain.InvoicingSeries, error) {
	query := "SELECT id, terminal_id, prefix, next_sequence FROM invoicing_series WHERE terminal_id = $1"
	if r.isSQLite {
		query = "SELECT id, terminal_id, prefix, next_sequence FROM invoicing_series WHERE terminal_id = ?"
	}
	var s domain.InvoicingSeries
	var idStr, terminalIDStr string
	err := r.db.QueryRowContext(ctx, query, terminalID.String()).Scan(&idStr, &terminalIDStr, &s.Prefix, &s.NextSequence)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	parsedTerminalID, err := uuid.Parse(terminalIDStr)
	if err != nil {
		return nil, err
	}
	s.ID = parsedID
	s.TerminalID = parsedTerminalID
	return &s, nil
}

// Save inserts or updates an invoicing series.
func (r *SQLInvoicingSeriesRepository) Save(ctx context.Context, s *domain.InvoicingSeries) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO invoicing_series (id, terminal_id, prefix, next_sequence) VALUES (?, ?, ?, ?) 
                 ON CONFLICT(id) DO UPDATE SET terminal_id = excluded.terminal_id, prefix = excluded.prefix, next_sequence = excluded.next_sequence`
	} else {
		query = `INSERT INTO invoicing_series (id, terminal_id, prefix, next_sequence) VALUES ($1, $2, $3, $4) 
                 ON CONFLICT(id) DO UPDATE SET terminal_id = EXCLUDED.terminal_id, prefix = EXCLUDED.prefix, next_sequence = EXCLUDED.next_sequence`
	}
	_, err := r.db.ExecContext(ctx, query, s.ID.String(), s.TerminalID.String(), s.Prefix, s.NextSequence)
	return err
}

// IncrementSequence locks the invoicing series row for the given terminal_id,
// increments the sequence, updates it in the database, and returns the prefix and the new sequence number.
func (r *SQLInvoicingSeriesRepository) IncrementSequence(ctx context.Context, terminalID uuid.UUID) (string, int, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback()

	if r.isSQLite {
		// In SQLite, BEGIN IMMEDIATE is needed to lock the database immediately
		_, _ = tx.ExecContext(ctx, "BEGIN IMMEDIATE")
	}

	var selectQuery string
	if r.isSQLite {
		selectQuery = "SELECT prefix, next_sequence FROM invoicing_series WHERE terminal_id = ?"
	} else {
		selectQuery = "SELECT prefix, next_sequence FROM invoicing_series WHERE terminal_id = $1 FOR UPDATE"
	}

	var prefix string
	var nextSeq int
	err = tx.QueryRowContext(ctx, selectQuery, terminalID.String()).Scan(&prefix, &nextSeq)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, fmt.Errorf("no invoicing series found for terminal %s", terminalID)
		}
		return "", 0, err
	}

	assignedSeq := nextSeq
	updatedNextSeq := nextSeq + 1

	var updateQuery string
	if r.isSQLite {
		updateQuery = "UPDATE invoicing_series SET next_sequence = ? WHERE terminal_id = ?"
	} else {
		updateQuery = "UPDATE invoicing_series SET next_sequence = $1 WHERE terminal_id = $2"
	}

	_, err = tx.ExecContext(ctx, updateQuery, updatedNextSeq, terminalID.String())
	if err != nil {
		return "", 0, err
	}

	if err := tx.Commit(); err != nil {
		return "", 0, err
	}

	return prefix, assignedSeq, nil
}
