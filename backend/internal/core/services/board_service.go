package services

import (
	"context"
	"errors"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

// boardService keeps Kanban application rules separate from transport and persistence details.
type boardService struct {
	boardRepository  ports.BoardRepository
	columnRepository ports.ColumnRepository
	idGenerator      ports.IDGenerator
	logger           ports.Logger
}

func NewBoardService(
	boardRepository ports.BoardRepository,
	columnRepository ports.ColumnRepository,
	idGenerator ports.IDGenerator,
	logger ports.Logger,
) ports.BoardUseCase {
	return &boardService{
		boardRepository:  boardRepository,
		columnRepository: columnRepository,
		idGenerator:      idGenerator,
		logger:           logger,
	}
}

func (service *boardService) CreateBoard(ctx context.Context, userID, name string) (*domain.Board, error) {
	boardID := service.idGenerator.Generate()
	board, err := domain.NewBoard(boardID, userID, name)
	if err != nil {
		return nil, err
	}

	if err := service.boardRepository.Save(ctx, board); err != nil {
		service.logger.Error("failed to save board", "userID", userID, "boardID", boardID, "error", err)
		return nil, ErrInternalProcessing
	}

	return board, nil
}

func (service *boardService) GetBoard(ctx context.Context, userID, boardID string) (*domain.Board, error) {
	return service.getAuthorizedBoard(ctx, userID, boardID)
}

func (service *boardService) GetUserBoards(ctx context.Context, userID string) ([]*domain.Board, error) {
	boards, err := service.boardRepository.GetByUserID(ctx, userID)
	if err != nil {
		service.logger.Error("failed to retrieve user boards", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	return boards, nil
}

func (service *boardService) UpdateBoardName(ctx context.Context, userID, boardID, name string) error {
	board, err := service.getAuthorizedBoard(ctx, userID, boardID)
	if err != nil {
		return err
	}

	if err := board.UpdateName(name); err != nil {
		return err
	}

	return service.persistBoardUpdate(ctx, board, "failed to update board name")
}

func (service *boardService) DeleteBoard(ctx context.Context, userID, boardID string) error {
	board, err := service.getAuthorizedBoard(ctx, userID, boardID)
	if err != nil {
		return err
	}

	if err := service.boardRepository.Delete(ctx, board.ID()); err != nil {
		service.logger.Error("failed to delete board", "userID", userID, "boardID", board.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func (service *boardService) CreateColumn(ctx context.Context, userID, boardID, name string, options ...interface{}) (*domain.Column, error) {
	board, err := service.getAuthorizedBoard(ctx, userID, boardID)
	if err != nil {
		return nil, err
	}
	color, position, err := parseColumnCreationOptions(options...)
	if err != nil {
		return nil, err
	}

	columnID := service.idGenerator.Generate()
	column, err := domain.NewColumn(columnID, board.ID(), name, color, position)
	if err != nil {
		return nil, err
	}

	if err := service.columnRepository.Save(ctx, column); err != nil {
		service.logger.Error("failed to save column", "boardID", board.ID(), "columnID", columnID, "error", err)
		return nil, ErrInternalProcessing
	}

	return column, nil
}

func (service *boardService) GetBoardColumns(ctx context.Context, userID, boardID string) ([]*domain.Column, error) {
	board, err := service.getAuthorizedBoard(ctx, userID, boardID)
	if err != nil {
		return nil, err
	}

	columns, err := service.columnRepository.GetByBoardID(ctx, board.ID())
	if err != nil {
		service.logger.Error("failed to retrieve board columns", "boardID", board.ID(), "error", err)
		return nil, ErrInternalProcessing
	}

	return columns, nil
}

func (service *boardService) UpdateColumnName(ctx context.Context, userID, columnID, name string) error {
	column, err := service.getAuthorizedColumn(ctx, userID, columnID)
	if err != nil {
		return err
	}

	if err := column.UpdateName(name); err != nil {
		return err
	}

	return service.persistColumnUpdate(ctx, column, "failed to update column name")
}

func (service *boardService) UpdateColumn(ctx context.Context, userID, columnID, name, color string) error {
	column, err := service.getAuthorizedColumn(ctx, userID, columnID)
	if err != nil {
		return err
	}

	if err := column.Update(name, color); err != nil {
		return err
	}

	return service.persistColumnUpdate(ctx, column, "failed to update column")
}

func (service *boardService) MoveColumn(ctx context.Context, userID, columnID string, position int) error {
	column, err := service.getAuthorizedColumn(ctx, userID, columnID)
	if err != nil {
		return err
	}

	oldPosition := column.Position()
	newPosition := position
	if oldPosition == newPosition {
		return nil
	}

	if newPosition < 0 {
		return column.ChangePosition(newPosition)
	}

	columns, err := service.columnRepository.GetByBoardID(ctx, column.BoardID())
	if err != nil {
		service.logger.Error("failed to retrieve columns for move", "boardID", column.BoardID(), "columnID", column.ID(), "error", err)
		return ErrInternalProcessing
	}

	modifiedColumns := make([]*domain.Column, 0, len(columns))
	for _, currentColumn := range columns {
		if currentColumn.ID() == column.ID() {
			continue
		}

		currentPosition := currentColumn.Position()
		if shouldShiftColumnRight(oldPosition, newPosition, currentPosition) {
			if err := currentColumn.ChangePosition(currentPosition - 1); err != nil {
				return err
			}
			modifiedColumns = append(modifiedColumns, currentColumn)
			continue
		}

		if shouldShiftColumnLeft(oldPosition, newPosition, currentPosition) {
			if err := currentColumn.ChangePosition(currentPosition + 1); err != nil {
				return err
			}
			modifiedColumns = append(modifiedColumns, currentColumn)
		}
	}

	if err := column.ChangePosition(newPosition); err != nil {
		return err
	}
	modifiedColumns = append(modifiedColumns, column)

	if err := service.columnRepository.UpdatePositions(ctx, modifiedColumns); err != nil {
		service.logger.Error("failed to update column positions", "boardID", column.BoardID(), "columnID", column.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func (service *boardService) DeleteColumn(ctx context.Context, userID, columnID string) error {
	column, err := service.getAuthorizedColumn(ctx, userID, columnID)
	if err != nil {
		return err
	}

	if err := service.columnRepository.Delete(ctx, column.ID()); err != nil {
		service.logger.Error("failed to delete column", "boardID", column.BoardID(), "columnID", column.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func (service *boardService) getAuthorizedBoard(ctx context.Context, userID, boardID string) (*domain.Board, error) {
	board, err := service.boardRepository.GetByID(ctx, boardID)
	if errors.Is(err, ports.ErrBoardNotFound) {
		return nil, ports.ErrBoardNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve board", "boardID", boardID, "error", err)
		return nil, ErrInternalProcessing
	}
	if board == nil {
		return nil, ports.ErrBoardNotFound
	}
	if board.UserID() != userID {
		service.logger.Warn("unauthorized board access attempt", "boardID", boardID)
		return nil, ports.ErrBoardNotFound
	}

	return board, nil
}

func (service *boardService) getAuthorizedColumn(ctx context.Context, userID, columnID string) (*domain.Column, error) {
	column, err := service.columnRepository.GetByID(ctx, columnID)
	if errors.Is(err, ports.ErrColumnNotFound) {
		return nil, ports.ErrColumnNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve column", "columnID", columnID, "error", err)
		return nil, ErrInternalProcessing
	}
	if column == nil {
		return nil, ports.ErrColumnNotFound
	}

	board, err := service.boardRepository.GetByID(ctx, column.BoardID())
	if errors.Is(err, ports.ErrBoardNotFound) {
		service.logger.Warn("column parent board missing during authorization", "columnID", columnID, "boardID", column.BoardID())
		return nil, ports.ErrColumnNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve column board", "columnID", columnID, "boardID", column.BoardID(), "error", err)
		return nil, ErrInternalProcessing
	}
	if board == nil {
		service.logger.Warn("column parent board missing during authorization", "columnID", columnID, "boardID", column.BoardID())
		return nil, ports.ErrColumnNotFound
	}
	if board.UserID() != userID {
		service.logger.Warn("unauthorized column access attempt", "columnID", columnID, "boardID", column.BoardID())
		return nil, ports.ErrColumnNotFound
	}

	return column, nil
}

func (service *boardService) persistBoardUpdate(ctx context.Context, board *domain.Board, message string) error {
	if err := service.boardRepository.Update(ctx, board); err != nil {
		service.logger.Error(message, "userID", board.UserID(), "boardID", board.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func (service *boardService) persistColumnUpdate(ctx context.Context, column *domain.Column, message string) error {
	if err := service.columnRepository.Update(ctx, column); err != nil {
		service.logger.Error(message, "boardID", column.BoardID(), "columnID", column.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func shouldShiftColumnRight(oldPosition, newPosition, currentPosition int) bool {
	return newPosition > oldPosition && oldPosition < currentPosition && currentPosition <= newPosition
}

func shouldShiftColumnLeft(oldPosition, newPosition, currentPosition int) bool {
	return newPosition < oldPosition && newPosition <= currentPosition && currentPosition < oldPosition
}

func parseColumnCreationOptions(options ...interface{}) (string, int, error) {
	switch len(options) {
	case 1:
		position, ok := options[0].(int)
		if !ok {
			return "", 0, domain.ErrInvalidColumnPosition
		}
		return "slate", position, nil
	case 2:
		color, ok := options[0].(string)
		if !ok {
			return "", 0, domain.ErrInvalidColumnColor
		}
		position, ok := options[1].(int)
		if !ok {
			return "", 0, domain.ErrInvalidColumnPosition
		}
		return color, position, nil
	default:
		return "", 0, domain.ErrInvalidColumnPosition
	}
}
