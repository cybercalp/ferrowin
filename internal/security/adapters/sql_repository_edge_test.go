package adapters_test

import (
	"context"
	"testing"

	"ferrowin/internal/security/adapters"
	"ferrowin/internal/security/domain"

	"github.com/google/uuid"
)

func TestSQLUserRepository_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := adapters.NewSQLUserRepository(db, true)

	t.Run("non-existent UUID returns nil, no error", func(t *testing.T) {
		nonExistentID := uuid.New()
		user, err := userRepo.GetByID(ctx, nonExistentID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user != nil {
			t.Fatal("expected nil for non-existent user, got a user object")
		}
	})
}

func TestSQLUserRepository_GetByUsername_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := adapters.NewSQLUserRepository(db, true)

	t.Run("non-existent username returns nil, no error", func(t *testing.T) {
		user, err := userRepo.GetByUsername(ctx, "nonexistent-user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user != nil {
			t.Fatal("expected nil for non-existent username, got a user object")
		}
	})
}

func TestSQLUserRepository_UpdateUser(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := adapters.NewSQLUserRepository(db, true)

	t.Run("save same ID with different data updates the record", func(t *testing.T) {
		userID := uuid.New()

		// First save
		original := &domain.User{
			ID:           userID,
			Username:     "original_user",
			PasswordHash: "hash_original",
		}
		err := userRepo.Save(ctx, original)
		if err != nil {
			t.Fatalf("failed to save original user: %v", err)
		}

		// Second save with same ID, different data (upsert)
		updated := &domain.User{
			ID:           userID,
			Username:     "updated_user",
			PasswordHash: "hash_updated",
		}
		err = userRepo.Save(ctx, updated)
		if err != nil {
			t.Fatalf("failed to update user: %v", err)
		}

		// Verify fetched user has the updated values
		fetched, err := userRepo.GetByID(ctx, userID)
		if err != nil {
			t.Fatalf("failed to get user after update: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected user to exist after upsert")
		}
		if fetched.Username != "updated_user" {
			t.Errorf("expected username 'updated_user', got %q", fetched.Username)
		}
		if fetched.PasswordHash != "hash_updated" {
			t.Errorf("expected password_hash 'hash_updated', got %q", fetched.PasswordHash)
		}
	})
}

func TestSQLUserRepository_GetByUsername_Found(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := adapters.NewSQLUserRepository(db, true)

	t.Run("find user by username after save", func(t *testing.T) {
		userID := uuid.New()
		user := &domain.User{
			ID:           userID,
			Username:     "find_by_username",
			PasswordHash: "some_hash",
		}
		err := userRepo.Save(ctx, user)
		if err != nil {
			t.Fatalf("failed to save user: %v", err)
		}

		fetched, err := userRepo.GetByUsername(ctx, "find_by_username")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected user to be found by username")
		}
		if fetched.ID != userID {
			t.Errorf("expected user ID %s, got %s", userID, fetched.ID)
		}
		if fetched.Username != "find_by_username" {
			t.Errorf("expected username %q, got %q", "find_by_username", fetched.Username)
		}
		if fetched.PasswordHash != "some_hash" {
			t.Errorf("expected password_hash %q, got %q", "some_hash", fetched.PasswordHash)
		}
	})
}

func TestSQLUserRepository_Save_DuplicateUsername(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := adapters.NewSQLUserRepository(db, true)

	t.Run("saving user with duplicate username is allowed (different IDs)", func(t *testing.T) {
		// The schema has username UNIQUE, so this should fail
		user1 := &domain.User{
			ID:           uuid.New(),
			Username:     "duplicate_user",
			PasswordHash: "hash1",
		}
		err := userRepo.Save(ctx, user1)
		if err != nil {
			t.Fatalf("failed to save first user: %v", err)
		}

		user2 := &domain.User{
			ID:           uuid.New(),
			Username:     "duplicate_user",
			PasswordHash: "hash2",
		}
		err = userRepo.Save(ctx, user2)
		if err == nil {
			t.Fatal("expected error when saving user with duplicate username, got nil")
		}
	})
}

func TestSQLRoleRepository_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	roleRepo := adapters.NewSQLRoleRepository(db, true)

	t.Run("non-existent role UUID returns nil", func(t *testing.T) {
		role, err := roleRepo.GetByID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if role != nil {
			t.Fatal("expected nil for non-existent role")
		}
	})
}

func TestSQLGroupRepository_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	groupRepo := adapters.NewSQLGroupRepository(db, true)

	t.Run("non-existent group UUID returns nil", func(t *testing.T) {
		group, err := groupRepo.GetByID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if group != nil {
			t.Fatal("expected nil for non-existent group")
		}
	})
}

func TestSQLRoleSetRepository_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	roleSetRepo := adapters.NewSQLRoleSetRepository(db, true)

	t.Run("non-existent role set UUID returns nil", func(t *testing.T) {
		roleSet, err := roleSetRepo.GetByID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if roleSet != nil {
			t.Fatal("expected nil for non-existent role set")
		}
	})
}
