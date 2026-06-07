package auth

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewBcryptHasher_ValidCost_ReturnsHasher(t *testing.T) {
	// Arrange
	validCost := bcrypt.MinCost

	// Act
	hasher, err := NewBcryptHasher(validCost)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if hasher == nil {
		t.Fatal("expected hasher, got nil")
	}
}

func TestNewBcryptHasher_InvalidCost_ReturnsErrInvalidBcryptCost(t *testing.T) {
	// Arrange
	invalidCost := bcrypt.MinCost - 1

	// Act
	hasher, err := NewBcryptHasher(invalidCost)

	// Assert
	if hasher != nil {
		t.Fatal("expected nil hasher")
	}
	if !errors.Is(err, ErrInvalidBcryptCost) {
		t.Errorf("expected error %v, got %v", ErrInvalidBcryptCost, err)
	}
}

func TestBcryptHasher_HashValidPassword_ReturnsHash(t *testing.T) {
	// Arrange
	hasher, _ := NewBcryptHasher(bcrypt.MinCost)
	plainPassword := "securePassword123"

	// Act
	hashedPassword, err := hasher.Hash(plainPassword)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if hashedPassword == "" {
		t.Fatal("expected hashed password, got empty string")
	}
	if hashedPassword == plainPassword {
		t.Fatal("expected hashed password to differ from plain password")
	}
}

func TestBcryptHasher_CompareMatchingPassword_ReturnsNil(t *testing.T) {
	// Arrange
	hasher, _ := NewBcryptHasher(bcrypt.MinCost)
	plainPassword := "securePassword123"
	hashedPassword, _ := hasher.Hash(plainPassword)

	// Act
	err := hasher.Compare(plainPassword, hashedPassword)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestBcryptHasher_CompareWrongPassword_ReturnsErrPasswordCompareFailed(t *testing.T) {
	// Arrange
	hasher, _ := NewBcryptHasher(bcrypt.MinCost)
	hashedPassword, _ := hasher.Hash("securePassword123")

	// Act
	err := hasher.Compare("wrongPassword123", hashedPassword)

	// Assert
	if !errors.Is(err, ErrPasswordCompareFailed) {
		t.Errorf("expected error %v, got %v", ErrPasswordCompareFailed, err)
	}
}
