package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type mockUserRepository struct {
	userToReturn     *domain.User
	userByIDToReturn *domain.User
	getByIDError     error
	getByEmailError  error
	saveError        error
	savedUser        *domain.User
}

func (repository *mockUserRepository) Save(ctx context.Context, user *domain.User) error {
	repository.savedUser = user
	return repository.saveError
}

func (repository *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if repository.userByIDToReturn != nil || repository.getByIDError != nil {
		return repository.userByIDToReturn, repository.getByIDError
	}

	return repository.userToReturn, nil
}

func (repository *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return repository.userToReturn, repository.getByEmailError
}

type mockSessionRepository struct {
	sessionToReturn *domain.RefreshToken
	getError        error
	saveError       error
	rotateError     error
	savedSession    *domain.RefreshToken
	rotatedSession  *domain.RefreshToken
	revokedTokenID  string
}

func (repository *mockSessionRepository) Save(ctx context.Context, refreshToken *domain.RefreshToken) error {
	repository.savedSession = refreshToken
	return repository.saveError
}

func (repository *mockSessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	return repository.sessionToReturn, repository.getError
}

func (repository *mockSessionRepository) Revoke(ctx context.Context, id string) error {
	repository.revokedTokenID = id
	return nil
}

func (repository *mockSessionRepository) Rotate(ctx context.Context, revokedTokenID string, newRefreshToken *domain.RefreshToken) error {
	repository.revokedTokenID = revokedTokenID
	repository.rotatedSession = newRefreshToken
	return repository.rotateError
}

type mockHasher struct {
	hashToReturn string
	errToReturn  error
}

func (hasher *mockHasher) Hash(plain string) (string, error) {
	return hasher.hashToReturn, hasher.errToReturn
}

func (hasher *mockHasher) Compare(plain, hash string) error {
	return hasher.errToReturn
}

type mockTokenGen struct {
	tokenPair   ports.TokenPair
	errToReturn error
	subject     ports.TokenSubject
}

func (generator *mockTokenGen) GenerateTokenPair(subject ports.TokenSubject) (ports.TokenPair, error) {
	generator.subject = subject
	return generator.tokenPair, generator.errToReturn
}

type mockIDGen struct {
	id string
}

func (generator *mockIDGen) Generate() string {
	return generator.id
}

type mockLogger struct{}

func (logger *mockLogger) Info(msg string, keys ...interface{})  {}
func (logger *mockLogger) Warn(msg string, keys ...interface{})  {}
func (logger *mockLogger) Error(msg string, keys ...interface{}) {}

func TestRegister_NewUser_ReturnsUserAndSaves(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{}
	mockHash := &mockHasher{hashToReturn: "hashedPassword"}
	svc := NewUserService(mockRepo, &mockSessionRepository{}, mockHash, &mockTokenGen{}, &mockIDGen{id: "uuid-123"}, &mockLogger{})
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
	if mockRepo.savedUser == nil || mockRepo.savedUser.ID() != "uuid-123" {
		t.Errorf("expected user to be saved with generated ID")
	}
}

func TestRegister_ExistingEmail_ReturnsErrUserAlreadyExists(t *testing.T) {
	// Arrange
	existingProfile, _ := domain.NewUserProfile("Jane", "Doe", time.Now().AddDate(-25, 0, 0))
	existingUser, _ := domain.NewUser("old-uuid", "test@domain.com", "hashedPassword", existingProfile)
	mockRepo := &mockUserRepository{userToReturn: existingUser}
	svc := NewUserService(mockRepo, &mockSessionRepository{}, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

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
	svc := NewUserService(mockRepo, &mockSessionRepository{}, mockHash, &mockTokenGen{}, &mockIDGen{id: "uuid-123"}, &mockLogger{})

	// Act
	user, err := svc.Register(context.Background(), "test@domain.com", "securePass123", "John", "Doe", time.Now().AddDate(-20, 0, 0))

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
	svc := NewUserService(mockRepo, &mockSessionRepository{}, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

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
	svc := NewUserService(mockRepo, &mockSessionRepository{}, mockHash, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

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
	svc := NewUserService(mockRepo, &mockSessionRepository{}, mockHash, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

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
	svc := NewUserService(mockRepo, &mockSessionRepository{}, mockHash, &mockTokenGen{}, &mockIDGen{id: ""}, &mockLogger{})

	// Act
	_, err := svc.Register(context.Background(), "test@domain.com", "securePass123", "John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Assert
	if !errors.Is(err, domain.ErrEmptyID) {
		t.Errorf("expected error %v, got %v", domain.ErrEmptyID, err)
	}
}

func TestRegister_SaveFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{getByEmailError: ports.ErrUserNotFound, saveError: ports.ErrRepositoryUnavailable}
	mockHash := &mockHasher{hashToReturn: "hashedPassword"}
	svc := NewUserService(mockRepo, &mockSessionRepository{}, mockHash, &mockTokenGen{}, &mockIDGen{id: "uuid-123"}, &mockLogger{})

	// Act
	_, err := svc.Register(context.Background(), "test@domain.com", "securePass123", "John", "Doe", time.Now().AddDate(-20, 0, 0))

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestAuthenticate_ValidCredentials_ReturnsTokenPairAndSavesSession(t *testing.T) {
	// Arrange
	existingUser := createServiceTestUser(t)
	mockRepo := &mockUserRepository{userToReturn: existingUser}
	mockSessions := &mockSessionRepository{}
	mockTokens := &mockTokenGen{tokenPair: validServiceTokenPair()}
	svc := NewUserService(mockRepo, mockSessions, &mockHasher{}, mockTokens, &mockIDGen{id: "session-123"}, &mockLogger{})

	// Act
	accessToken, refreshToken, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if accessToken != "access-token" {
		t.Errorf("expected access token access-token, got %s", accessToken)
	}
	if refreshToken != "refresh-token" {
		t.Errorf("expected refresh token refresh-token, got %s", refreshToken)
	}
	if mockSessions.savedSession == nil {
		t.Fatal("expected refresh session to be saved")
	}
	if mockSessions.savedSession.TokenHash() == "refresh-token" {
		t.Fatal("expected stored refresh token value to be hashed")
	}
	if mockTokens.subject.Email != "test@domain.com" {
		t.Errorf("expected token subject email test@domain.com, got %s", mockTokens.subject.Email)
	}
	if mockTokens.subject.FirstName != "Jane" || mockTokens.subject.LastName != "Doe" {
		t.Errorf("expected token subject name Jane Doe, got %s %s", mockTokens.subject.FirstName, mockTokens.subject.LastName)
	}
}

func TestAuthenticate_NilUser_ReturnsErrInvalidCredentials(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{}
	svc := NewUserService(mockRepo, &mockSessionRepository{}, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected error %v, got %v", ErrInvalidCredentials, err)
	}
}

func TestAuthenticate_InvalidPassword_ReturnsErrInvalidCredentials(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{userToReturn: createServiceTestUser(t)}
	mockHash := &mockHasher{errToReturn: errors.New("password mismatch")}
	svc := NewUserService(mockRepo, &mockSessionRepository{}, mockHash, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.Authenticate(context.Background(), "test@domain.com", "wrongPassword")

	// Assert
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected error %v, got %v", ErrInvalidCredentials, err)
	}
}

func TestAuthenticate_TokenFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{userToReturn: createServiceTestUser(t)}
	mockTokens := &mockTokenGen{errToReturn: errors.New("token failure")}
	svc := NewUserService(mockRepo, &mockSessionRepository{}, &mockHasher{}, mockTokens, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestAuthenticate_SaveSessionFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{userToReturn: createServiceTestUser(t)}
	mockSessions := &mockSessionRepository{saveError: ports.ErrSessionRepositoryUnavailable}
	mockTokens := &mockTokenGen{tokenPair: validServiceTokenPair()}
	svc := NewUserService(mockRepo, mockSessions, &mockHasher{}, mockTokens, &mockIDGen{id: "session-123"}, &mockLogger{})

	// Act
	_, _, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestAuthenticate_UserNotFound_ReturnsErrInvalidCredentials(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{getByEmailError: ports.ErrUserNotFound}
	svc := NewUserService(mockRepo, &mockSessionRepository{}, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected error %v, got %v", ErrInvalidCredentials, err)
	}
}

func TestAuthenticate_RepositoryUnavailable_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	mockRepo := &mockUserRepository{getByEmailError: ports.ErrRepositoryUnavailable}
	svc := NewUserService(mockRepo, &mockSessionRepository{}, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.Authenticate(context.Background(), "test@domain.com", "securePass123")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestRefreshSession_ValidToken_RotatesAndReturnsTokenPair(t *testing.T) {
	// Arrange
	currentSession, _ := domain.NewRefreshToken("session-123", "user-123", hashRefreshToken("refresh-token"), time.Now().Add(time.Hour), false)
	mockSessions := &mockSessionRepository{sessionToReturn: currentSession}
	mockTokens := &mockTokenGen{tokenPair: validServiceTokenPair()}
	mockRepo := &mockUserRepository{userByIDToReturn: createServiceTestUser(t)}
	svc := NewUserService(mockRepo, mockSessions, &mockHasher{}, mockTokens, &mockIDGen{id: "session-456"}, &mockLogger{})

	// Act
	accessToken, refreshToken, err := svc.RefreshSession(context.Background(), "refresh-token")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if accessToken != "access-token" {
		t.Errorf("expected access token access-token, got %s", accessToken)
	}
	if refreshToken != "refresh-token" {
		t.Errorf("expected refresh token refresh-token, got %s", refreshToken)
	}
	if mockSessions.revokedTokenID != "session-123" {
		t.Errorf("expected revoked token ID session-123, got %s", mockSessions.revokedTokenID)
	}
	if mockSessions.rotatedSession == nil {
		t.Fatal("expected new session to be persisted through rotation")
	}
	if mockTokens.subject.Email != "test@domain.com" {
		t.Errorf("expected token subject email test@domain.com, got %s", mockTokens.subject.Email)
	}
}

func TestRefreshSession_EmptyToken_ReturnsErrInvalidRefreshToken(t *testing.T) {
	// Arrange
	svc := NewUserService(&mockUserRepository{}, &mockSessionRepository{}, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.RefreshSession(context.Background(), "")

	// Assert
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("expected error %v, got %v", ErrInvalidRefreshToken, err)
	}
}

func TestRefreshSession_MissingSession_ReturnsErrInvalidRefreshToken(t *testing.T) {
	// Arrange
	mockSessions := &mockSessionRepository{getError: ports.ErrSessionNotFound}
	svc := NewUserService(&mockUserRepository{}, mockSessions, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.RefreshSession(context.Background(), "refresh-token")

	// Assert
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("expected error %v, got %v", ErrInvalidRefreshToken, err)
	}
}

func TestRefreshSession_RepositoryUnavailable_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	mockSessions := &mockSessionRepository{getError: ports.ErrSessionRepositoryUnavailable}
	svc := NewUserService(&mockUserRepository{}, mockSessions, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.RefreshSession(context.Background(), "refresh-token")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestRefreshSession_RevokedSession_ReturnsErrSessionRevoked(t *testing.T) {
	// Arrange
	currentSession, _ := domain.NewRefreshToken("session-123", "user-123", hashRefreshToken("refresh-token"), time.Now().Add(time.Hour), true)
	mockSessions := &mockSessionRepository{sessionToReturn: currentSession}
	svc := NewUserService(&mockUserRepository{}, mockSessions, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.RefreshSession(context.Background(), "refresh-token")

	// Assert
	if !errors.Is(err, ErrSessionRevoked) {
		t.Errorf("expected error %v, got %v", ErrSessionRevoked, err)
	}
}

func TestRefreshSession_ExpiredSession_ReturnsErrRefreshSessionExpired(t *testing.T) {
	// Arrange
	currentSession, _ := domain.NewRefreshToken("session-123", "user-123", hashRefreshToken("refresh-token"), time.Now().Add(-time.Hour), false)
	mockSessions := &mockSessionRepository{sessionToReturn: currentSession}
	svc := NewUserService(&mockUserRepository{}, mockSessions, &mockHasher{}, &mockTokenGen{}, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.RefreshSession(context.Background(), "refresh-token")

	// Assert
	if !errors.Is(err, ErrRefreshSessionExpired) {
		t.Errorf("expected error %v, got %v", ErrRefreshSessionExpired, err)
	}
}

func TestRefreshSession_TokenGenerationFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	currentSession, _ := domain.NewRefreshToken("session-123", "user-123", hashRefreshToken("refresh-token"), time.Now().Add(time.Hour), false)
	mockSessions := &mockSessionRepository{sessionToReturn: currentSession}
	mockTokens := &mockTokenGen{errToReturn: errors.New("token failure")}
	svc := NewUserService(&mockUserRepository{userByIDToReturn: createServiceTestUser(t)}, mockSessions, &mockHasher{}, mockTokens, &mockIDGen{}, &mockLogger{})

	// Act
	_, _, err := svc.RefreshSession(context.Background(), "refresh-token")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestRefreshSession_RotateFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	currentSession, _ := domain.NewRefreshToken("session-123", "user-123", hashRefreshToken("refresh-token"), time.Now().Add(time.Hour), false)
	mockSessions := &mockSessionRepository{sessionToReturn: currentSession, rotateError: ports.ErrSessionRepositoryUnavailable}
	mockTokens := &mockTokenGen{tokenPair: validServiceTokenPair()}
	svc := NewUserService(&mockUserRepository{userByIDToReturn: createServiceTestUser(t)}, mockSessions, &mockHasher{}, mockTokens, &mockIDGen{id: "session-456"}, &mockLogger{})

	// Act
	_, _, err := svc.RefreshSession(context.Background(), "refresh-token")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func createServiceTestUser(t *testing.T) *domain.User {
	t.Helper()

	profile, err := domain.NewUserProfile("Jane", "Doe", time.Now().AddDate(-25, 0, 0))
	if err != nil {
		t.Fatalf("expected profile to be valid, got: %v", err)
	}

	user, err := domain.NewUser("uuid-123", "test@domain.com", "hashedPassword", profile)
	if err != nil {
		t.Fatalf("expected user to be valid, got: %v", err)
	}

	return user
}

func validServiceTokenPair() ports.TokenPair {
	return ports.TokenPair{
		AccessToken:           "access-token",
		RefreshToken:          "refresh-token",
		AccessTokenExpiresAt:  time.Now().Add(5 * time.Minute),
		RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour),
	}
}
