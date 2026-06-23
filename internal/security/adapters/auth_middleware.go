package adapters

import (
	"context"
	"net/http"
	"strings"

	"ferrowin/internal/security/domain"
)

// contextKey is used for storing claims in the request context.
type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey contextKey = "user_id"
	// UsernameKey is the context key for the authenticated user's username.
	UsernameKey contextKey = "username"
)

// AuthMiddleware provides JWT-based HTTP authentication middleware.
type AuthMiddleware struct {
	jwtCfg *domain.JWTConfig
}

// NewAuthMiddleware creates a new AuthMiddleware with the given JWT configuration.
func NewAuthMiddleware(jwtCfg *domain.JWTConfig) *AuthMiddleware {
	return &AuthMiddleware{jwtCfg: jwtCfg}
}

// Middleware returns an HTTP handler that validates the Bearer token from the
// Authorization header and injects user_id and username into the request context.
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		claims, err := m.jwtCfg.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClaims retrieves the authenticated user's claims from the request context.
// Returns empty strings if the request is not authenticated.
func GetClaims(r *http.Request) (userID, username string) {
	if uid, ok := r.Context().Value(UserIDKey).(string); ok {
		userID = uid
	}
	if un, ok := r.Context().Value(UsernameKey).(string); ok {
		username = un
	}
	return
}
