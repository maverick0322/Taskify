package services

import (
	"context"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

// Service-level errors for predictable flow control in the HTTP handlers.
var (
	ErrUserAlreadyExists  = errors.New("service: user with this email already exists")
	ErrInvalidCredentials = errors.New("service: invalid email or password")
	ErrInternalProcessing = errors.New("service: an internal error occurred while processing the request")
)

// userService implements ports.UserUseCase.
// Unexported struct ensures it can only be created via the constructor (Factory Pattern).
type userService struct {
	userRepo ports.UserRepository
	hasher   ports.PasswordHasher
	tokenGen ports.TokenGenerator
	idGen    ports.IDGenerator
	logger   ports.Logger
}

// NewUserService creates a new instance injecting all required dependencies.
func NewUserService(
	repo ports.UserRepository,
	hasher ports.PasswordHasher,
	tokenGen ports.TokenGenerator,
	idGen ports.IDGenerator,
	logger ports.Logger,
) ports.UserUseCase {
	return &userService{
		userRepo: repo,
		hasher:   hasher,
		tokenGen: tokenGen,
		idGen:    idGen,
		logger:   logger,
	}
}

func (s *userService) Register(ctx context.Context, email, plainPassword, firstName, lastName string, birthDate time.Time) (*domain.User, error) {
	// 1. Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		// Log as Warn since it's a client error, not a system failure. Do not log the email to protect PII.
		s.logger.Warn("registration attempt with existing email")
		return nil, ErrUserAlreadyExists
	}

	// 2. Hash password (Infrastructure concern, abstracted)
	hashedPassword, err := s.hasher.Hash(plainPassword)
	if err != nil {
		s.logger.Error("failed to hash password during registration", "error", err)
		return nil, ErrInternalProcessing
	}

	// 3. Create Domain Objects
	profile, err := domain.NewUserProfile(firstName, lastName, birthDate)
	if err != nil {
		return nil, err // Return domain error directly (e.g., ErrUnderageUser)
	}

	userID := s.idGen.Generate()
	newUser, err := domain.NewUser(userID, email, hashedPassword, profile)
	if err != nil {
		return nil, err
	}

	// 4. Persist
	if err := s.userRepo.Save(ctx, newUser); err != nil {
		s.logger.Error("failed to save new user to database", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	s.logger.Info("user registered successfully", "userID", userID)
	return newUser, nil
}

func (s *userService) Authenticate(ctx context.Context, email, plainPassword string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		s.logger.Warn("authentication failed: user not found")
		// We return a generic invalid credentials error to prevent email enumeration attacks (Security).
		return "", ErrInvalidCredentials
	}

	if err := s.hasher.Compare(plainPassword, user.PasswordHash()); err != nil {
		s.logger.Warn("authentication failed: incorrect password", "userID", user.ID())
		return "", ErrInvalidCredentials
	}

	token, err := s.tokenGen.GenerateToken(user.ID())
	if err != nil {
		s.logger.Error("failed to generate token", "userID", user.ID(), "error", err)
		return "", ErrInternalProcessing
	}

	return token, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID, firstName, lastName string, birthDate time.Time) error {
	// Implementation pending for the next iteration, but interface is satisfied.
	return errors.New("not implemented")
}
