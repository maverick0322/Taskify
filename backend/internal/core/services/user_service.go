package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

// Service-level errors for predictable flow control in the HTTP handlers.
var (
	ErrUserAlreadyExists     = errors.New("service: user with this email already exists")
	ErrInvalidCredentials    = errors.New("service: invalid email or password")
	ErrInternalProcessing    = errors.New("service: an internal error occurred while processing the request")
	ErrInvalidRefreshToken   = errors.New("service: invalid refresh token")
	ErrSessionRevoked        = errors.New("service: refresh session has been revoked")
	ErrRefreshSessionExpired = errors.New("service: refresh session has expired")
)

// userService implements ports.UserUseCase.
// Unexported struct ensures it can only be created via the constructor (Factory Pattern).
type userService struct {
	userRepo    ports.UserRepository
	sessionRepo ports.SessionRepository
	hasher      ports.PasswordHasher
	tokenGen    ports.TokenGenerator
	idGen       ports.IDGenerator
	logger      ports.Logger
}

// NewUserService creates a new instance injecting all required dependencies.
func NewUserService(
	repo ports.UserRepository,
	sessionRepo ports.SessionRepository,
	hasher ports.PasswordHasher,
	tokenGen ports.TokenGenerator,
	idGen ports.IDGenerator,
	logger ports.Logger,
) ports.UserUseCase {
	return &userService{
		userRepo:    repo,
		sessionRepo: sessionRepo,
		hasher:      hasher,
		tokenGen:    tokenGen,
		idGen:       idGen,
		logger:      logger,
	}
}

func (s *userService) Register(ctx context.Context, email, plainPassword, firstName, lastName string, birthDate time.Time) (*domain.User, error) {
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		// Log as Warn since it's a client error, not a system failure. Do not log the email to protect PII.
		s.logger.Warn("registration attempt with existing email")
		return nil, ErrUserAlreadyExists
	}
	if err != nil && !errors.Is(err, ports.ErrUserNotFound) {
		s.logger.Error("failed to verify existing user during registration", "error", err)
		return nil, ErrInternalProcessing
	}

	hashedPassword, err := s.hasher.Hash(plainPassword)
	if err != nil {
		s.logger.Error("failed to hash password during registration", "error", err)
		return nil, ErrInternalProcessing
	}

	profile, err := domain.NewUserProfile(firstName, lastName, birthDate)
	if err != nil {
		return nil, err // Return domain error directly (e.g., ErrUnderageUser)
	}

	userID := s.idGen.Generate()
	newUser, err := domain.NewUser(userID, email, hashedPassword, profile)
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.Save(ctx, newUser); err != nil {
		s.logger.Error("failed to save new user to database", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	s.logger.Info("user registered successfully", "userID", userID)
	return newUser, nil
}

func (s *userService) Authenticate(ctx context.Context, email, plainPassword string) (string, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if errors.Is(err, ports.ErrUserNotFound) {
		s.logger.Warn("authentication failed: user not found")
		// We return a generic invalid credentials error to prevent email enumeration attacks (Security).
		return "", "", ErrInvalidCredentials
	}
	if err != nil {
		s.logger.Error("failed to retrieve user during authentication", "error", err)
		return "", "", ErrInternalProcessing
	}
	if user == nil {
		s.logger.Warn("authentication failed: user not found")
		return "", "", ErrInvalidCredentials
	}

	if err := s.hasher.Compare(plainPassword, user.PasswordHash()); err != nil {
		s.logger.Warn("authentication failed: incorrect password", "userID", user.ID())
		return "", "", ErrInvalidCredentials
	}

	tokenPair, err := s.tokenGen.GenerateTokenPair(tokenSubjectFromUser(user))
	if err != nil {
		s.logger.Error("failed to generate token pair", "userID", user.ID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	refreshSession, err := s.buildRefreshSession(user.ID(), tokenPair)
	if err != nil {
		s.logger.Error("failed to build refresh session", "userID", user.ID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	if err := s.sessionRepo.Save(ctx, refreshSession); err != nil {
		s.logger.Error("failed to save refresh session", "userID", user.ID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	return tokenPair.AccessToken, tokenPair.RefreshToken, nil
}

func (s *userService) RefreshSession(ctx context.Context, refreshToken string) (string, string, error) {
	if strings.TrimSpace(refreshToken) == "" {
		s.logger.Warn("refresh session rejected because token is empty")
		return "", "", ErrInvalidRefreshToken
	}

	refreshTokenHash := hashRefreshToken(refreshToken)
	currentSession, err := s.sessionRepo.GetByTokenHash(ctx, refreshTokenHash)
	if errors.Is(err, ports.ErrSessionNotFound) {
		s.logger.Warn("refresh session rejected because token was not found")
		return "", "", ErrInvalidRefreshToken
	}
	if err != nil {
		s.logger.Error("failed to retrieve refresh session", "error", err)
		return "", "", ErrInternalProcessing
	}
	if currentSession == nil {
		s.logger.Warn("refresh session rejected because token was not found")
		return "", "", ErrInvalidRefreshToken
	}
	if currentSession.IsRevoked() {
		s.logger.Warn("refresh session rejected because token is revoked", "sessionID", currentSession.ID())
		return "", "", ErrSessionRevoked
	}
	if currentSession.IsExpired(time.Now()) {
		s.logger.Warn("refresh session rejected because token is expired", "sessionID", currentSession.ID())
		return "", "", ErrRefreshSessionExpired
	}

	user, err := s.userRepo.GetByID(ctx, currentSession.UserID())
	if errors.Is(err, ports.ErrUserNotFound) || user == nil {
		s.logger.Warn("refresh session rejected because user was not found", "userID", currentSession.UserID())
		return "", "", ErrInvalidRefreshToken
	}
	if err != nil {
		s.logger.Error("failed to retrieve user during refresh", "userID", currentSession.UserID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	tokenPair, err := s.tokenGen.GenerateTokenPair(tokenSubjectFromUser(user))
	if err != nil {
		s.logger.Error("failed to generate refreshed token pair", "userID", currentSession.UserID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	newRefreshSession, err := s.buildRefreshSession(currentSession.UserID(), tokenPair)
	if err != nil {
		s.logger.Error("failed to build refreshed session", "userID", currentSession.UserID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	if err := s.sessionRepo.Rotate(ctx, currentSession.ID(), newRefreshSession); err != nil {
		s.logger.Error("failed to rotate refresh session", "sessionID", currentSession.ID(), "error", err)
		return "", "", ErrInternalProcessing
	}

	return tokenPair.AccessToken, tokenPair.RefreshToken, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID, firstName, lastName string, birthDate time.Time) error {
	// Implementation pending for the next iteration, but interface is satisfied.
	return errors.New("not implemented")
}

func (s *userService) buildRefreshSession(userID string, tokenPair ports.TokenPair) (*domain.RefreshToken, error) {
	sessionID := s.idGen.Generate()
	refreshTokenHash := hashRefreshToken(tokenPair.RefreshToken)
	return domain.NewRefreshToken(sessionID, userID, refreshTokenHash, tokenPair.RefreshTokenExpiresAt, false)
}

func hashRefreshToken(refreshToken string) string {
	refreshTokenHash := sha256.Sum256([]byte(refreshToken))
	return hex.EncodeToString(refreshTokenHash[:])
}

func tokenSubjectFromUser(user *domain.User) ports.TokenSubject {
	profile := user.Profile()
	return ports.TokenSubject{
		UserID:    user.ID(),
		Email:     user.Email(),
		FirstName: profile.FirstName(),
		LastName:  profile.LastName(),
	}
}
