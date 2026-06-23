package domain

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a system user in the domain model.
type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	Groups       []Group
}

// VerifyPassword checks if the given password matches the stored bcrypt hash.
// Returns false (without error) on any bcrypt error, including empty or malformed passwords.
func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}
