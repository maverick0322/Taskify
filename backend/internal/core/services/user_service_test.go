package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

// --- Manual Mocks ---

type mockUserRepository struct {
	userToReturn    *domain.User
	getByEmailError error
	saveError       error
	savedUser       *domain.User
}

func (m *mockUserRepository) Save(ctx context.Context, user *domain.User) error {
	m.savedUser = user
	return m.saveError
}
func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.userToReturn, m.getByEmailError
}

type mockHasher struct {
	hashToReturn string
	errToReturn  error
}

func (m *mockHasher) Hash(plain string) (string, error) { return m.hashToReturn, m.errToReturn }
func (m *mockHasher) Compare(plain, hash string) error  { return m.errToReturn }

type mockTokenGen struct {
	token       string
	errToReturn error
}

func (m *mockTokenGen) GenerateToken(userID string) (string, error) {
	return m.token, m.errToReturn
}

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
	existingUser, _ := domain.NewUser("old-uuid", "test@domain.com", "hashedPassword", existingProfile)

	mockRepo := &mockUserRepository{userToReturn: existingUser} // Simulating email exists
	svc := NewUserService(mockRepo, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, err := svc.Register(context.Background(), "test@domain.com", "pass123", "John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Assert
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Errorf("expected error %v, got %v", ErrUserAlreadyExists, err)
	}
}

func TestRegister_UserNotFound_ContinuesAndSaves(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{getByEmailError: ports.ErrUserNotFound}
	mockHash := &mockHasher{hashToReturn: "hashedPassword"}
	svc := NewUserService(mockRepo, mockHash, &mockTokenGen{}, &mockIDGen{id: "uuid-123"}, &mockLogger{})
	validBirthDate := time.Now().AddDate(-20, 0, 0)

	// Act
	user, err := svc.Register(context.Background(), "test@domain.com", "securePass123", "John", "Doe", validBirthDate)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if mockRepo.savedUser == nil {
		t.Fatal("expected user to be saved")
	}
}

func TestRegister_RepositoryUnavailable_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{getByEmailError: ports.ErrRepositoryUnavailable}
	svc := NewUserService(mockRepo, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, err := svc.Register(context.Background(), "test@domain.com", "securePass123", "John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestRegister_HasherFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{getByEmailError: ports.ErrUserNotFound}
	mockHash := &mockHasher{errToReturn: errors.New("hash failure")}
	svc := NewUserService(mockRepo, mockHash, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, err := svc.Register(context.Background(), "test@domain.com", "securePass123", "John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestRegister_InvalidProfile_ReturnsDomainError(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{getByEmailError: ports.ErrUserNotFound}
	mockHash := &mockHasher{hashToReturn: "hashedPassword"}
	svc := NewUserService(mockRepo, mockHash, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, err := svc.Register(context.Background(), "test@domain.com", "securePass123", "J", "Doe", time.Now().AddDate(-20, 0, 0))

	// Assert
	if !errors.Is(err, domain.ErrInvalidName) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidName, err)
	}
}

func TestRegister_InvalidGeneratedID_ReturnsDomainError(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{getByEmailError: ports.ErrUserNotFound}
	mockHash := &mockHasher{hashToReturn: "hashedPassword"}
	svc := NewUserService(mockRepo, mockHash, &mockTokenGen{}, &mockIDGen{id: ""}, &mockLogger{})

	// Act
	_, err := svc.Register(context.Background(), "test@domain.com", "securePass123", "John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Assert
	if !errors.Is(err, domain.ErrEmptyID) {
		t.Errorf("expected error %v, got %v", domain.ErrEmptyID, err)
	}
}

func TestRegister_SaveFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{
		getByEmailError: ports.ErrUserNotFound,
		saveError:       ports.ErrRepositoryUnavailable,
	}
	mockHash := &mockHasher{hashToReturn: "hashedPassword"}
	svc := NewUserService(mockRepo, mockHash, &mockTokenGen{}, &mockIDGen{id: "uuid-123"}, &mockLogger{})

	// Act
	_, err := svc.Register(context.Background(), "test@domain.com", "securePass123", "John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestAuthenticate_ValidCredentials_ReturnsToken(t *testing.T) {
	// Arrange
	profile, _ := domain.NewUserProfile("Jane", "Doe", time.Now().AddDate(-25, 0, 0))
	existingUser, _ := domain.NewUser("uuid-123", "test@domain.com", "hashedPassword", profile)
	mockRepo := &mockUserRepository{userToReturn: existingUser}
	svc := NewUserService(mockRepo, &mockHasher{}, &mockTokenGen{token: "token-123"}, &mockIDGen{}, &mockLogger{})

	// Act
	token, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if token != "token-123" {
		t.Errorf("expected token token-123, got %s", token)
	}
}

func TestAuthenticate_NilUser_ReturnsErrInvalidCredentials(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{}
	svc := NewUserService(mockRepo, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected error %v, got %v", ErrInvalidCredentials, err)
	}
}

func TestAuthenticate_InvalidPassword_ReturnsErrInvalidCredentials(t *testing.T) {
	// Arrange
	profile, _ := domain.NewUserProfile("Jane", "Doe", time.Now().AddDate(-25, 0, 0))
	existingUser, _ := domain.NewUser("uuid-123", "test@domain.com", "hashedPassword", profile)
	mockRepo := &mockUserRepository{userToReturn: existingUser}
	mockHash := &mockHasher{errToReturn: errors.New("password mismatch")}
	svc := NewUserService(mockRepo, mockHash, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, err := svc.Authenticate(context.Background(), "test@domain.com", "wrongPassword")

	// Assert
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected error %v, got %v", ErrInvalidCredentials, err)
	}
}

func TestAuthenticate_TokenFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	profile, _ := domain.NewUserProfile("Jane", "Doe", time.Now().AddDate(-25, 0, 0))
	existingUser, _ := domain.NewUser("uuid-123", "test@domain.com", "hashedPassword", profile)
	mockRepo := &mockUserRepository{userToReturn: existingUser}
	mockToken := &mockTokenGen{errToReturn: errors.New("token failure")}
	svc := NewUserService(mockRepo, &mockHasher{}, mockToken, &mockIDGen{}, &mockLogger{})

	// Act
	_, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestAuthenticate_UserNotFound_ReturnsErrInvalidCredentials(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{getByEmailError: ports.ErrUserNotFound}
	svc := NewUserService(mockRepo, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected error %v, got %v", ErrInvalidCredentials, err)
	}
}

func TestAuthenticate_RepositoryUnavailable_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{getByEmailError: ports.ErrRepositoryUnavailable}
	svc := NewUserService(mockRepo, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}
