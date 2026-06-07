package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	minNameLength     = 2
	minPasswordLength = 8
	legalAgeYears     = 18
)

// Exported sentinel errors allow upper layers (e.g., Application/Services)
// to perform type-safe error checking using errors.Is() without relying on string matching,
// ensuring predictable control flow and defensive programming.
var (
	ErrInvalidName     = errors.New("domain: first or last name does not meet minimum length")
	ErrInvalidEmail    = errors.New("domain: invalid email format")
	ErrInvalidPassword = errors.New("domain: password does not meet security requirements")
	ErrUnderageUser    = errors.New("domain: user does not meet the minimum age requirement")
	ErrEmptyID         = errors.New("domain: user ID cannot be empty")
)

// UserProfile acts as a Value Object. Fields are unexported (private) to enforce immutability
// after creation, preventing accidental state corruption across the application.
type UserProfile struct {
	firstName string
	lastName  string
	birthDate time.Time
}

// NewUserProfile acts as a factory, enforcing business invariants at creation time.
// This guarantees that an invalid state can never exist within the domain layer.
func NewUserProfile(firstName, lastName string, birthDate time.Time) (UserProfile, error) {
	if len(strings.TrimSpace(firstName)) < minNameLength || len(strings.TrimSpace(lastName)) < minNameLength {
		return UserProfile{}, ErrInvalidName
	}

	minimumRequiredAgeDate := time.Now().AddDate(-legalAgeYears, 0, 0)
	if birthDate.After(minimumRequiredAgeDate) {
		return UserProfile{}, ErrUnderageUser
	}

	return UserProfile{
		firstName: strings.TrimSpace(firstName),
		lastName:  strings.TrimSpace(lastName),
		birthDate: birthDate,
	}, nil
}

// User is the Aggregate Root. State is strictly encapsulated to adhere to the
// Open/Closed Principle (OCP); state transitions must happen through controlled behaviors,
// not direct assignment.
type User struct {
	id           string
	email        string
	passwordHash string
	profile      UserProfile
}

// NewUser initializes a User entity ensuring its validity upon creation.
func NewUser(id, email, passwordHash string, profile UserProfile) (*User, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrEmptyID
	}

	if !strings.Contains(email, "@") {
		return nil, ErrInvalidEmail
	}

	if len(passwordHash) < minPasswordLength {
		return nil, ErrInvalidPassword
	}

	return &User{
		id:           id,
		email:        email,
		passwordHash: passwordHash,
		profile:      profile,
	}, nil
}

// Getters expose state without allowing direct modification, maintaining encapsulation.
func (user *User) ID() string {
	return user.id
}

func (user *User) Email() string {
	return user.email
}

func (user *User) Profile() UserProfile {
	return user.profile
}

// PasswordHash safely exposes the hashed password for authentication comparisons
// without allowing direct modification of the entity's state.
func (user *User) PasswordHash() string {
	return user.passwordHash
}

// Getters for UserProfile to allow persistence adapters to read the state
// without breaking encapsulation and immutability.

func (p UserProfile) FirstName() string {
	return p.firstName
}

func (p UserProfile) LastName() string {
	return p.lastName
}

func (p UserProfile) BirthDate() time.Time {
	return p.birthDate
}
