package idempotency_test

import (
	"context"
	"database/sql"
	"testing"

	"ferrowin/internal/shared/idempotency"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestTracker_IdempotencyKey(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker := idempotency.NewTracker(db, true)
	err := tracker.InitSchema(ctx)
	if err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	t.Run("Valid UUID validation", func(t *testing.T) {
		validKey := uuid.New().String()
		invalidKey := "not-a-uuid"

		if !tracker.IsValidKey(validKey) {
			t.Errorf("expected key %s to be valid", validKey)
		}
		if tracker.IsValidKey(invalidKey) {
			t.Errorf("expected key %s to be invalid", invalidKey)
		}
	})

	t.Run("Reserve key and check duplicate rejection", func(t *testing.T) {
		key := uuid.New().String()

		// First reservation should succeed
		err := tracker.ReserveKey(ctx, key)
		if err != nil {
			t.Fatalf("expected reserve to succeed, got %v", err)
		}

		// Second reservation with same key should fail (duplicate key rejection)
		err = tracker.ReserveKey(ctx, key)
		if err == nil {
			t.Errorf("expected duplicate key reservation to fail, but it succeeded")
		}

		// GetResponse should find the key but return empty body (as we reserved it)
		found, body, err := tracker.GetResponse(ctx, key)
		if err != nil {
			t.Fatalf("unexpected error getting response: %v", err)
		}
		if !found {
			t.Errorf("expected key %s to be found", key)
		}
		if body != "" {
			t.Errorf("expected body to be empty, got %s", body)
		}

		// Save response body
		expectedBody := `{"status":"success"}`
		err = tracker.SaveResponse(ctx, key, expectedBody)
		if err != nil {
			t.Fatalf("failed to save response: %v", err)
		}

		// GetResponse should now return the saved body
		found, body, err = tracker.GetResponse(ctx, key)
		if err != nil {
			t.Fatalf("unexpected error getting response: %v", err)
		}
		if !found {
			t.Errorf("expected key to be found")
		}
		if body != expectedBody {
			t.Errorf("expected body %s, got %s", expectedBody, body)
		}
	})
}
