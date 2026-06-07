package ports

import (
	"context"
	"errors"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
)

// SessionRepository defines the outbound port for persisted refresh-token sessions.
type SessionRepository interface {
	Save(ctx context.Context, refreshToken *domain.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	Rotate(ctx context.Context, revokedTokenID string, newRefreshToken *domain.RefreshToken) error
}

var (
	ErrSessionNotFound              = errors.New("repository: session not found")
	ErrSessionAlreadyExists         = errors.New("repository: session already exists")
	ErrSessionRepositoryUnavailable = errors.New("repository: session persistence layer is unavailable or corrupted")
)
