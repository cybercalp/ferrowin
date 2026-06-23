package adapters

import (
	"context"
	"encoding/json"
	"net/http"

	"ferrowin/internal/security/domain"
)

// LoginRequest is the JSON body for POST /api/v1/auth/login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is the JSON response for a successful login.
type LoginResponse struct {
	Token string         `json:"token"`
	User  *UserInfoJSON `json:"user"`
}

// UserInfoJSON is the public user info returned in login responses.
type UserInfoJSON struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// authServiceRequired defines the subset of AuthService used by the controller.
type authServiceRequired interface {
	Login(ctx context.Context, username, password string) (*domain.LoginResponse, error)
}

// AuthController handles authentication HTTP endpoints.
type AuthController struct {
	authService authServiceRequired
}

// NewAuthController creates a new AuthController.
func NewAuthController(authService authServiceRequired) *AuthController {
	return &AuthController{authService: authService}
}

// ServeHTTP dispatches requests to the appropriate handler.
func (c *AuthController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	c.handleLogin(w, r)
}

// handleLogin processes POST /api/v1/auth/login.
func (c *AuthController) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		return
	}

	resp, err := c.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Token: resp.Token,
		User: &UserInfoJSON{
			ID:       resp.User.ID.String(),
			Username: resp.User.Username,
		},
	})
}

// writeJSON marshals v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
