package ports

import (
	"context"
	"errors"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
)

// BoardRepository defines the outbound port for Kanban board persistence.
type BoardRepository interface {
	Save(ctx context.Context, board *domain.Board) error
	GetByID(ctx context.Context, id string) (*domain.Board, error)
	GetByUserID(ctx context.Context, userID string) ([]*domain.Board, error)
	Update(ctx context.Context, board *domain.Board) error
	Delete(ctx context.Context, id string) error
}

// ColumnRepository defines the outbound port for Kanban column persistence.
type ColumnRepository interface {
	Save(ctx context.Context, column *domain.Column) error
	GetByID(ctx context.Context, id string) (*domain.Column, error)
	GetByBoardID(ctx context.Context, boardID string) ([]*domain.Column, error)
	Update(ctx context.Context, column *domain.Column) error
	UpdatePositions(ctx context.Context, columns []*domain.Column) error
	Delete(ctx context.Context, id string) error
}

// BoardUseCase defines user-scoped application operations for Kanban boards.
type BoardUseCase interface {
	CreateBoard(ctx context.Context, userID, name string) (*domain.Board, error)
	GetBoard(ctx context.Context, userID, boardID string) (*domain.Board, error)
	GetUserBoards(ctx context.Context, userID string) ([]*domain.Board, error)
	UpdateBoardName(ctx context.Context, userID, boardID, name string) error
	DeleteBoard(ctx context.Context, userID, boardID string) error
	CreateColumn(ctx context.Context, userID, boardID, name string, options ...interface{}) (*domain.Column, error)
	GetBoardColumns(ctx context.Context, userID, boardID string) ([]*domain.Column, error)
	UpdateColumnName(ctx context.Context, userID, columnID, name string) error
	UpdateColumn(ctx context.Context, userID, columnID, name, color string) error
	MoveColumn(ctx context.Context, userID, columnID string, position int) error
	DeleteColumn(ctx context.Context, userID, columnID string) error
}

var (
	ErrBoardNotFound               = errors.New("repository: board not found")
	ErrColumnNotFound              = errors.New("repository: column not found")
	ErrBoardRepositoryUnavailable  = errors.New("repository: board persistence layer is unavailable or corrupted")
	ErrColumnRepositoryUnavailable = errors.New("repository: column persistence layer is unavailable or corrupted")
)
