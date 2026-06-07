package auth

import (
	"errors"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidBcryptCost     = errors.New("auth: invalid bcrypt cost")
	ErrPasswordHashFailed    = errors.New("auth: password hash failed")
	ErrPasswordCompareFailed = errors.New("auth: password comparison failed")
)

// BcryptHasher keeps password hashing behind the core password port.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher receives the cost so each environment can tune security and latency.
func NewBcryptHasher(cost int) (ports.PasswordHasher, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, ErrInvalidBcryptCost
	}

	return &BcryptHasher{cost: cost}, nil
}

func (hasher *BcryptHasher) Hash(plainPassword string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), hasher.cost)
	if err != nil {
		return "", ErrPasswordHashFailed
	}

	return string(hashedPassword), nil
}

func (hasher *BcryptHasher) Compare(plainPassword, hashedPassword string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword)); err != nil {
		return ErrPasswordCompareFailed
	}

	return nil
}
