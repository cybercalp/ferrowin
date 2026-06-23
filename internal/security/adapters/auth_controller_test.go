package adapters_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ferrowin/internal/security/adapters"
	"ferrowin/internal/security/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type mockAuthService struct {
	users map[string]*domain.User // keyed by username
}

func (m *mockAuthService) Login(ctx context.Context, username, password string) (*domain.LoginResponse, error) {
	if username == "" || password == "" {
		return nil, domain.ErrInvalidCredentials
	}
	user, ok := m.users[username]
	if !ok {
		return nil, domain.ErrInvalidCredentials
	}
	if !user.VerifyPassword(password) {
		return nil, domain.ErrInvalidCredentials
	}

	jwtCfg := domain.NewJWTConfig("test-secret-for-controller-test", 1*time.Hour)
	token, err := jwtCfg.GenerateToken(user.ID.String(), user.Username, nil)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return &domain.LoginResponse{
		Token: token,
		User: &domain.UserResponse{
			ID:       user.ID,
			Username: user.Username,
		},
	}, nil
}

func hashPw(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hash)
}

func setupTestUser(t *testing.T) (string, string) {
	t.Helper()
	correctPassword := "testpass123"
	hash := hashPw(t, correctPassword)
	_ = hash
	return "testuser", correctPassword
}

func newTestJWTConfig() *domain.JWTConfig {
	return domain.NewJWTConfig("test-secret-for-controller-test", 1*time.Hour)
}

// ---------------------------------------------------------------------------
// Login endpoint tests
// ---------------------------------------------------------------------------

func TestAuthController_Login_Success(t *testing.T) {
	password := "correct-password"
	hash := hashPw(t, password)
	userID := uuid.New()

	mockSvc := &mockAuthService{
		users: map[string]*domain.User{
			"admin": {
				ID:           userID,
				Username:     "admin",
				PasswordHash: hash,
			},
		},
	}
	controller := adapters.NewAuthController(mockSvc)

	body := `{"username":"admin","password":"correct-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
		User  struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.User.ID == "" {
		t.Error("expected non-empty user id")
	}
	if resp.User.Username != "admin" {
		t.Errorf("expected username 'admin', got %q", resp.User.Username)
	}
}

func TestAuthController_Login_BadCredentials(t *testing.T) {
	password := "correct-password"
	hash := hashPw(t, password)

	mockSvc := &mockAuthService{
		users: map[string]*domain.User{
			"admin": {
				ID:           uuid.New(),
				Username:     "admin",
				PasswordHash: hash,
			},
		},
	}
	controller := adapters.NewAuthController(mockSvc)

	tests := []struct {
		name string
		body string
	}{
		{"wrong password", `{"username":"admin","password":"wrong"}`},
		{"unknown user", `{"username":"nobody","password":"correct-password"}`},
		{"empty username", `{"username":"","password":"correct-password"}`},
		{"empty password", `{"username":"admin","password":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			controller.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
			}

			var errResp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}
			if errResp["error"] != "Invalid credentials" {
				t.Errorf("expected error message 'Invalid credentials', got %q", errResp["error"])
			}
		})
	}
}

func TestAuthController_Login_MethodNotAllowed(t *testing.T) {
	mockSvc := &mockAuthService{}
	controller := adapters.NewAuthController(mockSvc)

	// GET instead of POST
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	w := httptest.NewRecorder()

	controller.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestAuthController_Login_InvalidJSON(t *testing.T) {
	mockSvc := &mockAuthService{}
	controller := adapters.NewAuthController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Auth middleware tests
// ---------------------------------------------------------------------------

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtCfg := newTestJWTConfig()
	middleware := adapters.NewAuthMiddleware(jwtCfg)

	token, err := jwtCfg.GenerateToken("user-456", "operator", []string{"role-1"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, username := adapters.GetClaims(r)
		if userID != "user-456" {
			t.Errorf("expected userID 'user-456', got %q", userID)
		}
		if username != "operator" {
			t.Errorf("expected username 'operator', got %q", username)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	jwtCfg := newTestJWTConfig()
	middleware := adapters.NewAuthMiddleware(jwtCfg)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidBearerFormat(t *testing.T) {
	jwtCfg := newTestJWTConfig()
	middleware := adapters.NewAuthMiddleware(jwtCfg)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	// Use negative expiry so token is already expired
	expiredCfg := domain.NewJWTConfig("test-secret-for-controller-test", -1*time.Hour)
	middleware := adapters.NewAuthMiddleware(expiredCfg)

	token, err := expiredCfg.GenerateToken("user-456", "operator", nil)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for expired token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_EmptyClaimsWhenNotAuthenticated(t *testing.T) {
	userID, username := adapters.GetClaims(httptest.NewRequest(http.MethodGet, "/", nil))
	if userID != "" {
		t.Errorf("expected empty userID, got %q", userID)
	}
	if username != "" {
		t.Errorf("expected empty username, got %q", username)
	}
}
