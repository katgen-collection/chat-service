package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher implements simple password hashing & comparison using bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher returns a BcryptHasher with given cost.
// If cost <= 0, bcrypt.DefaultCost is used.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// Hash returns the bcrypt hash of the given password (UTF-8).
// The returned string is safe to store in DB.
func (b *BcryptHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Compare checks whether the provided password matches the hashed value.
// Returns nil if they match, bcrypt.ErrMismatchedHashAndPassword if not.
func (b *BcryptHasher) Compare(hash, password string) error {
	if hash == "" || password == "" {
		return fmt.Errorf("hash and password must be provided")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// HashPassword is a convenience function to hash password using default cost.
func HashPassword(password string) (string, error) {
	hasher := NewBcryptHasher(bcrypt.DefaultCost)
	return hasher.Hash(password)
}

// ComparePassword is a convenience function to compare password using default cost.
func ComparePassword(hash, password string) error {
	hasher := NewBcryptHasher(bcrypt.DefaultCost)
	return hasher.Compare(hash, password)
}
