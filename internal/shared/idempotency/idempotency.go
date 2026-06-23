package idempotency

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Tracker handles idempotency key registration and verification.
type Tracker struct {
	db       *sql.DB
	isSQLite bool
}

// NewTracker creates a new Tracker.
func NewTracker(db *sql.DB, isSQLite bool) *Tracker {
	return &Tracker{
		db:       db,
		isSQLite: isSQLite,
	}
}

// InitSchema creates the idempotency_keys table if it does not exist.
func (t *Tracker) InitSchema(ctx context.Context) error {
	var query string
	if t.isSQLite {
		query = `CREATE TABLE IF NOT EXISTS idempotency_keys (
			key_val TEXT PRIMARY KEY,
			response_body TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	} else {
		query = `CREATE TABLE IF NOT EXISTS idempotency_keys (
			key_val VARCHAR(255) PRIMARY KEY,
			response_body TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`
	}
	_, err := t.db.ExecContext(ctx, query)
	return err
}

// IsValidKey checks if the key is a valid UUID.
func (t *Tracker) IsValidKey(key string) bool {
	_, err := uuid.Parse(key)
	return err == nil
}

// GetResponse retrieves the saved response body for a key if it exists.
// If the key is not found, it returns false, "", nil.
func (t *Tracker) GetResponse(ctx context.Context, key string) (bool, string, error) {
	var query string
	if t.isSQLite {
		query = `SELECT response_body FROM idempotency_keys WHERE key_val = ?`
	} else {
		query = `SELECT response_body FROM idempotency_keys WHERE key_val = $1`
	}

	var body sql.NullString
	err := t.db.QueryRowContext(ctx, query, key).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, body.String, nil
}

// SaveResponse saves the response body for a key.
func (t *Tracker) SaveResponse(ctx context.Context, key string, responseBody string) error {
	var query string
	if t.isSQLite {
		query = `UPDATE idempotency_keys SET response_body = ? WHERE key_val = ?`
	} else {
		query = `UPDATE idempotency_keys SET response_body = $1 WHERE key_val = $2`
	}
	_, err := t.db.ExecContext(ctx, query, responseBody, key)
	return err
}

// ReserveKey inserts the key with an empty response body.
// Returns an error if the key already exists (violating primary key constraint).
func (t *Tracker) ReserveKey(ctx context.Context, key string) error {
	var query string
	if t.isSQLite {
		query = `INSERT INTO idempotency_keys (key_val, response_body, created_at) VALUES (?, '', ?)`
	} else {
		query = `INSERT INTO idempotency_keys (key_val, response_body, created_at) VALUES ($1, '', $2)`
	}
	_, err := t.db.ExecContext(ctx, query, key, time.Now().UTC())
	return err
}
