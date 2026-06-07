package domain

import (
	"errors"
	"testing"
	"time"
)

// Tests strictly follow the AAA (Arrange, Act, Assert) pattern to isolate setup from
// execution and validation, maintaining a cyclomatic complexity of 1 per test case.

func TestNewUserProfile_ValidFields_ReturnsValidInstance(t *testing.T) {
	// Arrange
	firstName := "John"
	lastName := "Doe"
	validBirthDate := time.Now().AddDate(-25, 0, 0)

	// Act
	profile, err := NewUserProfile(firstName, lastName, validBirthDate)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if profile.firstName != firstName {
		t.Errorf("expected firstName %s, got %s", firstName, profile.firstName)
	}
}

func TestNewUserProfile_InvalidName_ReturnsError(t *testing.T) {
	// Arrange
	invalidName := "A"
	lastName := "Doe"
	validBirthDate := time.Now().AddDate(-20, 0, 0)

	// Act
	_, err := NewUserProfile(invalidName, lastName, validBirthDate)

	// Assert
	if !errors.Is(err, ErrInvalidName) {
		t.Errorf("expected error %v, got %v", ErrInvalidName, err)
	}
}

func TestNewUserProfile_UnderageUser_ReturnsError(t *testing.T) {
	// Arrange
	firstName := "John"
	lastName := "Doe"
	underageBirthDate := time.Now().AddDate(-17, 0, 0)

	// Act
	_, err := NewUserProfile(firstName, lastName, underageBirthDate)

	// Assert
	if !errors.Is(err, ErrUnderageUser) {
		t.Errorf("expected error %v, got %v", ErrUnderageUser, err)
	}
}

func TestNewUser_ValidData_ReturnsValidUser(t *testing.T) {
	// Arrange
	id := "uuid-1234"
	email := "test@domain.com"
	securePasswordHash := "secureHash1234!"
	profile, _ := NewUserProfile("John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Act
	user, err := NewUser(id, email, securePasswordHash, profile)

	// Assert
	if err != nil {
		t.Fatalf("did not expect error, got: %v", err)
	}
	if user.ID() != id {
		t.Errorf("expected id %s, got %s", id, user.ID())
	}
}

// --- Nuevas pruebas para alcanzar >90% de Coverage ---

func TestNewUser_EmptyID_ReturnsError(t *testing.T) {
	// Arrange
	emptyID := ""
	email := "test@domain.com"
	securePasswordHash := "secureHash1234!"
	profile, _ := NewUserProfile("John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Act
	_, err := NewUser(emptyID, email, securePasswordHash, profile)

	// Assert
	if !errors.Is(err, ErrEmptyID) {
		t.Errorf("expected error %v, got %v", ErrEmptyID, err)
	}
}

func TestNewUser_InvalidEmail_ReturnsError(t *testing.T) {
	// Arrange
	id := "uuid-1234"
	invalidEmail := "testdomain.com" // Missing '@'
	securePasswordHash := "secureHash1234!"
	profile, _ := NewUserProfile("John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Act
	_, err := NewUser(id, invalidEmail, securePasswordHash, profile)

	// Assert
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected error %v, got %v", ErrInvalidEmail, err)
	}
}

func TestNewUser_InvalidPassword_ReturnsError(t *testing.T) {
	// Arrange
	id := "uuid-1234"
	email := "test@domain.com"
	shortPasswordHash := "short" // Length < 8
	profile, _ := NewUserProfile("John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Act
	_, err := NewUser(id, email, shortPasswordHash, profile)

	// Assert
	if !errors.Is(err, ErrInvalidPassword) {
		t.Errorf("expected error %v, got %v", ErrInvalidPassword, err)
	}
}

func TestUser_Getters_ReturnExpectedValues(t *testing.T) {
	// Arrange
	id := "uuid-1234"
	email := "test@domain.com"
	securePasswordHash := "secureHash1234!"
	profile, _ := NewUserProfile("John", "Doe", time.Now().AddDate(-20, 0, 0))

	// We safely ignore the error here because we already tested that valid data creates a user
	user, _ := NewUser(id, email, securePasswordHash, profile)

	// Act
	retrievedID := user.ID()
	retrievedEmail := user.Email()
	retrievedProfile := user.Profile()

	// Assert
	if retrievedID != id {
		t.Errorf("expected id %s, got %s", id, retrievedID)
	}
	if retrievedEmail != email {
		t.Errorf("expected email %s, got %s", email, retrievedEmail)
	}
	if retrievedProfile.firstName != profile.firstName {
		t.Errorf("expected profile firstName %s, got %s", profile.firstName, retrievedProfile.firstName)
	}
}
