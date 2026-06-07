package ports

import (
	"context"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
)

// UserRepository defines the outbound port (Secondary Port) for User persistence.
// It follows the Interface Segregation Principle (ISP) by only declaring methods
// necessary for user data management, completely abstracting the underlying database.
type UserRepository interface {
	Save(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// UserUseCase defines the inbound port (Primary Port) for User-related business operations.
// This is the contract that external adapters (like HTTP Handlers or CLI tools) will use
// to interact with the core logic.
type UserUseCase interface {
	Register(ctx context.Context, email, plainPassword, firstName, lastName string, birthDate time.Time) (*domain.User, error)
	Authenticate(ctx context.Context, email, plainPassword string) (string, error)
	UpdateProfile(ctx context.Context, userID, firstName, lastName string, birthDate time.Time) error
}

var (
	ErrUserNotFound          = errors.New("repository: user not found")
	ErrUserAlreadyExists     = errors.New("repository: user already exists")
	ErrRepositoryUnavailable = errors.New("repository: persistence layer is unavailable or corrupted")
)
