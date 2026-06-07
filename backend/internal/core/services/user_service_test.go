package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
)

// --- Manual Mocks ---

type mockUserRepository struct {
	userToReturn *domain.User
	errToReturn  error
	savedUser    *domain.User
}

func (m *mockUserRepository) Save(ctx context.Context, user *domain.User) error {
	m.savedUser = user
	return m.errToReturn
}
func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.userToReturn, m.errToReturn
}

type mockHasher struct {
	hashToReturn string
	errToReturn  error
}

func (m *mockHasher) Hash(plain string) (string, error) { return m.hashToReturn, m.errToReturn }
func (m *mockHasher) Compare(plain, hash string) error  { return m.errToReturn }

type mockTokenGen struct{ token string }

func (m *mockTokenGen) GenerateToken(userID string) (string, error) { return m.token, nil }

type mockIDGen struct{ id string }

func (m *mockIDGen) Generate() string { return m.id }

type mockLogger struct{}

func (m *mockLogger) Info(msg string, keys ...interface{})  {}
func (m *mockLogger) Warn(msg string, keys ...interface{})  {}
func (m *mockLogger) Error(msg string, keys ...interface{}) {}

// --- Tests using AAA Pattern ---

func TestRegister_NewUser_ReturnsUserAndSaves(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{}
	mockHash := &mockHasher{hashToReturn: "hashedPassword"}
	svc := NewUserService(mockRepo, mockHash, &mockTokenGen{}, &mockIDGen{id: "uuid-123"}, &mockLogger{})

	ctx := context.Background()
	validBirthDate := time.Now().AddDate(-20, 0, 0)

	// Act
	user, err := svc.Register(ctx, "test@domain.com", "securePass123", "John", "Doe", validBirthDate)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if mockRepo.savedUser == nil || mockRepo.savedUser.ID() != "uuid-123" {
		t.Errorf("expected user to be saved with generated ID")
	}
}

func TestRegister_ExistingEmail_ReturnsErrUserAlreadyExists(t *testing.T) {
	// Arrange
	existingProfile, _ := domain.NewUserProfile("Jane", "Doe", time.Now().AddDate(-25, 0, 0))
	existingUser, _ := domain.NewUser("old-uuid", "test@domain.com", "hash", existingProfile)

	mockRepo := &mockUserRepository{userToReturn: existingUser} // Simulating email exists
	svc := NewUserService(mockRepo, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, err := svc.Register(context.Background(), "test@domain.com", "pass123", "John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Assert
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Errorf("expected error %v, got %v", ErrUserAlreadyExists, err)
	}
}
