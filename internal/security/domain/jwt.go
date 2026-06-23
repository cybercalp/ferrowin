package domain

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig holds the configuration for JWT token generation and validation.
type JWTConfig struct {
	Secret string
	Expiry time.Duration // Token lifetime (e.g., 24h)
}

// Claims represents the custom JWT claims for the application.
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	RoleIDs  []string `json:"role_ids"`
	jwt.RegisteredClaims
}

// NewJWTConfig creates a new JWTConfig with the given secret and expiry.
func NewJWTConfig(secret string, expiry time.Duration) *JWTConfig {
	return &JWTConfig{
		Secret: secret,
		Expiry: expiry,
	}
}

// GenerateToken creates a signed JWT token for the given user.
func (c *JWTConfig) GenerateToken(userID, username string, roleIDs []string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RoleIDs:  roleIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(c.Expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "ferrowin",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.Secret))
}

// ValidateToken parses and validates a JWT token string. It returns the claims
// if the token is valid, or an error if the token is invalid or expired.
func (c *JWTConfig) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(c.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
