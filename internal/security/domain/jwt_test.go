package domain_test

import (
	"testing"
	"time"

	"ferrowin/internal/security/domain"
)

func newTestJWTConfig(expiry time.Duration) *domain.JWTConfig {
	return domain.NewJWTConfig("test-secret-key-for-unit-tests", expiry)
}

func TestJWT_GenerateAndValidate_Roundtrip(t *testing.T) {
	cfg := newTestJWTConfig(1 * time.Hour)

	token, err := cfg.GenerateToken("user-123", "testuser", []string{"role-1", "role-2"})
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	claims, err := cfg.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() failed: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got %q", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Errorf("expected Username 'testuser', got %q", claims.Username)
	}
	if len(claims.RoleIDs) != 2 || claims.RoleIDs[0] != "role-1" || claims.RoleIDs[1] != "role-2" {
		t.Errorf("unexpected RoleIDs: %v", claims.RoleIDs)
	}
}

func TestJWT_TamperedToken_ReturnsError(t *testing.T) {
	cfg := newTestJWTConfig(1 * time.Hour)

	token, err := cfg.GenerateToken("user-123", "testuser", nil)
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	// Tamper with the token
	tampered := token + "tampered"

	_, err = cfg.ValidateToken(tampered)
	if err == nil {
		t.Error("expected error for tampered token, got nil")
	}
}

func TestJWT_ExpiredToken_ReturnsError(t *testing.T) {
	// Create a config with negative expiry so the token is already expired
	cfg := newTestJWTConfig(-1 * time.Hour)

	token, err := cfg.GenerateToken("user-123", "testuser", nil)
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	_, err = cfg.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestJWT_WrongSecret_ReturnsError(t *testing.T) {
	cfg := newTestJWTConfig(1 * time.Hour)

	token, err := cfg.GenerateToken("user-123", "testuser", nil)
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	// Validate with a different secret
	wrongCfg := domain.NewJWTConfig("different-secret", 1*time.Hour)
	_, err = wrongCfg.ValidateToken(token)
	if err == nil {
		t.Error("expected error for wrong secret, got nil")
	}
}

func TestJWT_EmptyToken_ReturnsError(t *testing.T) {
	cfg := newTestJWTConfig(1 * time.Hour)

	_, err := cfg.ValidateToken("")
	if err == nil {
		t.Error("expected error for empty token, got nil")
	}
}

func TestJWT_MalformedToken_ReturnsError(t *testing.T) {
	cfg := newTestJWTConfig(1 * time.Hour)

	_, err := cfg.ValidateToken("not-a-jwt-token")
	if err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}
