package services

import (
	"context"
	"errors"
	"testing"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	validBoardServiceUserID      = "user-123"
	validBoardServiceOtherUserID = "user-456"
	validBoardServiceBoardID     = "board-123"
	validBoardServiceColumnID    = "column-123"
	validBoardServiceName        = "Product Roadmap"
	validColumnServiceName       = "Backlog"
	validColumnServicePosition   = 0
)

type mockBoardRepository struct {
	boardToReturn     *domain.Board
	boardsToReturn    []*domain.Board
	saveError         error
	getByIDError      error
	getByUserIDError  error
	updateError       error
	deleteError       error
	savedBoard        *domain.Board
	updatedBoard      *domain.Board
	deletedBoardID    string
	requestedBoardID  string
	requestedUserID   string
	getByIDCallCount  int
	getByUserIDCalled bool
}

func (repository *mockBoardRepository) Save(ctx context.Context, board *domain.Board) error {
	repository.savedBoard = board
	return repository.saveError
}

func (repository *mockBoardRepository) GetByID(ctx context.Context, id string) (*domain.Board, error) {
	repository.requestedBoardID = id
	repository.getByIDCallCount++
	return repository.boardToReturn, repository.getByIDError
}

func (repository *mockBoardRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Board, error) {
	repository.requestedUserID = userID
	repository.getByUserIDCalled = true
	return repository.boardsToReturn, repository.getByUserIDError
}

func (repository *mockBoardRepository) Update(ctx context.Context, board *domain.Board) error {
	repository.updatedBoard = board
	return repository.updateError
}

func (repository *mockBoardRepository) Delete(ctx context.Context, id string) error {
	repository.deletedBoardID = id
	return repository.deleteError
}

type mockColumnRepository struct {
	columnToReturn         *domain.Column
	columnsToReturn        []*domain.Column
	saveError              error
	getByIDError           error
	getByBoardIDError      error
	updateError            error
	updatePositionsError   error
	deleteError            error
	savedColumn            *domain.Column
	updatedColumn          *domain.Column
	updatedPositionColumns []*domain.Column
	deletedColumnID        string
	requestedColumnID      string
	requestedBoardID       string
	getByIDCallCount       int
	getByBoardIDCalled     bool
	saveCalled             bool
	updateCalled           bool
	updatePositionsCalled  bool
	deleteCalled           bool
}

func (repository *mockColumnRepository) Save(ctx context.Context, column *domain.Column) error {
	repository.savedColumn = column
	repository.saveCalled = true
	return repository.saveError
}

func (repository *mockColumnRepository) GetByID(ctx context.Context, id string) (*domain.Column, error) {
	repository.requestedColumnID = id
	repository.getByIDCallCount++
	return repository.columnToReturn, repository.getByIDError
}

func (repository *mockColumnRepository) GetByBoardID(ctx context.Context, boardID string) ([]*domain.Column, error) {
	repository.requestedBoardID = boardID
	repository.getByBoardIDCalled = true
	return repository.columnsToReturn, repository.getByBoardIDError
}

func (repository *mockColumnRepository) Update(ctx context.Context, column *domain.Column) error {
	repository.updatedColumn = column
	repository.updateCalled = true
	return repository.updateError
}

func (repository *mockColumnRepository) UpdatePositions(ctx context.Context, columns []*domain.Column) error {
	repository.updatedPositionColumns = columns
	repository.updatePositionsCalled = true
	return repository.updatePositionsError
}

func (repository *mockColumnRepository) Delete(ctx context.Context, id string) error {
	repository.deletedColumnID = id
	repository.deleteCalled = true
	return repository.deleteError
}

type mockBoardIDGenerator struct {
	ids []string
}

func (generator *mockBoardIDGenerator) Generate() string {
	if len(generator.ids) == 0 {
		return ""
	}

	id := generator.ids[0]
	generator.ids = generator.ids[1:]
	return id
}

type mockBoardLogger struct {
	warnMessages  []string
	errorMessages []string
}

func (logger *mockBoardLogger) Info(msg string, keysAndValues ...interface{}) {}

func (logger *mockBoardLogger) Warn(msg string, keysAndValues ...interface{}) {
	logger.warnMessages = append(logger.warnMessages, msg)
}

func (logger *mockBoardLogger) Error(msg string, keysAndValues ...interface{}) {
	logger.errorMessages = append(logger.errorMessages, msg)
}

func TestCreateBoard_ValidData_ReturnsBoardAndSaves(t *testing.T) {
	// Arrange
	boardRepository := &mockBoardRepository{}
	service := NewBoardService(boardRepository, &mockColumnRepository{}, &mockBoardIDGenerator{ids: []string{validBoardServiceBoardID}}, &mockBoardLogger{})

	// Act
	board, err := service.CreateBoard(context.Background(), validBoardServiceUserID, validBoardServiceName)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if board.ID() != validBoardServiceBoardID {
		t.Errorf("expected board ID %s, got %s", validBoardServiceBoardID, board.ID())
	}
	if board.UserID() != validBoardServiceUserID {
		t.Errorf("expected user ID %s, got %s", validBoardServiceUserID, board.UserID())
	}
	if boardRepository.savedBoard == nil {
		t.Fatal("expected board to be saved")
	}
}

func TestCreateBoard_InvalidName_ReturnsDomainError(t *testing.T) {
	// Arrange
	service := NewBoardService(&mockBoardRepository{}, &mockColumnRepository{}, &mockBoardIDGenerator{ids: []string{validBoardServiceBoardID}}, &mockBoardLogger{})

	// Act
	_, err := service.CreateBoard(context.Background(), validBoardServiceUserID, "No")

	// Assert
	if !errors.Is(err, domain.ErrInvalidBoardName) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidBoardName, err)
	}
}

func TestCreateBoard_SaveFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	boardRepository := &mockBoardRepository{saveError: ports.ErrBoardRepositoryUnavailable}
	service := NewBoardService(boardRepository, &mockColumnRepository{}, &mockBoardIDGenerator{ids: []string{validBoardServiceBoardID}}, &mockBoardLogger{})

	// Act
	_, err := service.CreateBoard(context.Background(), validBoardServiceUserID, validBoardServiceName)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestGetBoard_OwnedBoard_ReturnsBoard(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	retrievedBoard, err := service.GetBoard(context.Background(), validBoardServiceUserID, validBoardServiceBoardID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if retrievedBoard.ID() != validBoardServiceBoardID {
		t.Errorf("expected board ID %s, got %s", validBoardServiceBoardID, retrievedBoard.ID())
	}
}

func TestGetBoard_MissingBoard_ReturnsErrBoardNotFound(t *testing.T) {
	// Arrange
	service := NewBoardService(&mockBoardRepository{getByIDError: ports.ErrBoardNotFound}, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	_, err := service.GetBoard(context.Background(), validBoardServiceUserID, validBoardServiceBoardID)

	// Assert
	if !errors.Is(err, ports.ErrBoardNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardNotFound, err)
	}
}

func TestGetBoard_NilBoard_ReturnsErrBoardNotFound(t *testing.T) {
	// Arrange
	service := NewBoardService(&mockBoardRepository{}, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	_, err := service.GetBoard(context.Background(), validBoardServiceUserID, validBoardServiceBoardID)

	// Assert
	if !errors.Is(err, ports.ErrBoardNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardNotFound, err)
	}
}

func TestGetBoard_RepositoryUnavailable_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	service := NewBoardService(&mockBoardRepository{getByIDError: ports.ErrBoardRepositoryUnavailable}, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	_, err := service.GetBoard(context.Background(), validBoardServiceUserID, validBoardServiceBoardID)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestGetBoard_BoardOwnedByAnotherUser_ReturnsErrBoardNotFoundAndWarns(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceOtherUserID)
	logger := &mockBoardLogger{}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, &mockColumnRepository{}, &mockBoardIDGenerator{}, logger)

	// Act
	_, err := service.GetBoard(context.Background(), validBoardServiceUserID, validBoardServiceBoardID)

	// Assert
	if !errors.Is(err, ports.ErrBoardNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardNotFound, err)
	}
	if len(logger.warnMessages) != 1 {
		t.Fatalf("expected one warning log, got %d", len(logger.warnMessages))
	}
}

func TestGetUserBoards_RepositorySuccess_ReturnsBoards(t *testing.T) {
	// Arrange
	boards := []*domain.Board{createBoardServiceBoard(t, validBoardServiceUserID)}
	boardRepository := &mockBoardRepository{boardsToReturn: boards}
	service := NewBoardService(boardRepository, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	retrievedBoards, err := service.GetUserBoards(context.Background(), validBoardServiceUserID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(retrievedBoards) != 1 {
		t.Fatalf("expected one board, got %d", len(retrievedBoards))
	}
	if boardRepository.requestedUserID != validBoardServiceUserID {
		t.Errorf("expected requested user ID %s, got %s", validBoardServiceUserID, boardRepository.requestedUserID)
	}
}

func TestGetUserBoards_RepositoryFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	service := NewBoardService(&mockBoardRepository{getByUserIDError: ports.ErrBoardRepositoryUnavailable}, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	_, err := service.GetUserBoards(context.Background(), validBoardServiceUserID)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestUpdateBoardName_OwnedBoard_UpdatesAndPersists(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	boardRepository := &mockBoardRepository{boardToReturn: board}
	service := NewBoardService(boardRepository, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.UpdateBoardName(context.Background(), validBoardServiceUserID, validBoardServiceBoardID, "Delivery Plan")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if boardRepository.updatedBoard == nil {
		t.Fatal("expected board to be updated")
	}
	if boardRepository.updatedBoard.Name() != "Delivery Plan" {
		t.Errorf("expected board name Delivery Plan, got %s", boardRepository.updatedBoard.Name())
	}
}

func TestUpdateBoardName_InvalidName_ReturnsDomainError(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.UpdateBoardName(context.Background(), validBoardServiceUserID, validBoardServiceBoardID, "No")

	// Assert
	if !errors.Is(err, domain.ErrInvalidBoardName) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidBoardName, err)
	}
}

func TestUpdateBoardName_UpdateFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	service := NewBoardService(&mockBoardRepository{boardToReturn: board, updateError: ports.ErrBoardRepositoryUnavailable}, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.UpdateBoardName(context.Background(), validBoardServiceUserID, validBoardServiceBoardID, "Delivery Plan")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestDeleteBoard_OwnedBoard_DeletesBoard(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	boardRepository := &mockBoardRepository{boardToReturn: board}
	service := NewBoardService(boardRepository, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.DeleteBoard(context.Background(), validBoardServiceUserID, validBoardServiceBoardID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if boardRepository.deletedBoardID != validBoardServiceBoardID {
		t.Errorf("expected deleted board ID %s, got %s", validBoardServiceBoardID, boardRepository.deletedBoardID)
	}
}

func TestDeleteBoard_DeleteFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	service := NewBoardService(&mockBoardRepository{boardToReturn: board, deleteError: ports.ErrBoardRepositoryUnavailable}, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.DeleteBoard(context.Background(), validBoardServiceUserID, validBoardServiceBoardID)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestCreateColumn_AuthorizedBoard_ReturnsColumnAndSaves(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	columnRepository := &mockColumnRepository{}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{ids: []string{validBoardServiceColumnID}}, &mockBoardLogger{})

	// Act
	column, err := service.CreateColumn(context.Background(), validBoardServiceUserID, validBoardServiceBoardID, validColumnServiceName, validColumnServicePosition)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if column.ID() != validBoardServiceColumnID {
		t.Errorf("expected column ID %s, got %s", validBoardServiceColumnID, column.ID())
	}
	if columnRepository.savedColumn == nil {
		t.Fatal("expected column to be saved")
	}
}

func TestCreateColumn_UnauthorizedBoard_ReturnsErrBoardNotFound(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceOtherUserID)
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	_, err := service.CreateColumn(context.Background(), validBoardServiceUserID, validBoardServiceBoardID, validColumnServiceName, validColumnServicePosition)

	// Assert
	if !errors.Is(err, ports.ErrBoardNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardNotFound, err)
	}
}

func TestCreateColumn_InvalidName_ReturnsDomainError(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, &mockColumnRepository{}, &mockBoardIDGenerator{ids: []string{validBoardServiceColumnID}}, &mockBoardLogger{})

	// Act
	_, err := service.CreateColumn(context.Background(), validBoardServiceUserID, validBoardServiceBoardID, "No", validColumnServicePosition)

	// Assert
	if !errors.Is(err, domain.ErrInvalidColumnName) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidColumnName, err)
	}
}

func TestCreateColumn_NegativePosition_ReturnsDomainError(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, &mockColumnRepository{}, &mockBoardIDGenerator{ids: []string{validBoardServiceColumnID}}, &mockBoardLogger{})

	// Act
	_, err := service.CreateColumn(context.Background(), validBoardServiceUserID, validBoardServiceBoardID, validColumnServiceName, -1)

	// Assert
	if !errors.Is(err, domain.ErrInvalidColumnPosition) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidColumnPosition, err)
	}
}

func TestCreateColumn_SaveFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	columnRepository := &mockColumnRepository{saveError: ports.ErrColumnRepositoryUnavailable}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{ids: []string{validBoardServiceColumnID}}, &mockBoardLogger{})

	// Act
	_, err := service.CreateColumn(context.Background(), validBoardServiceUserID, validBoardServiceBoardID, validColumnServiceName, validColumnServicePosition)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestGetBoardColumns_AuthorizedBoard_ReturnsColumns(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	columns := []*domain.Column{createBoardServiceColumn(t)}
	columnRepository := &mockColumnRepository{columnsToReturn: columns}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	retrievedColumns, err := service.GetBoardColumns(context.Background(), validBoardServiceUserID, validBoardServiceBoardID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(retrievedColumns) != 1 {
		t.Fatalf("expected one column, got %d", len(retrievedColumns))
	}
	if columnRepository.requestedBoardID != validBoardServiceBoardID {
		t.Errorf("expected requested board ID %s, got %s", validBoardServiceBoardID, columnRepository.requestedBoardID)
	}
}

func TestGetBoardColumns_RepositoryFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	columnRepository := &mockColumnRepository{getByBoardIDError: ports.ErrColumnRepositoryUnavailable}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	_, err := service.GetBoardColumns(context.Background(), validBoardServiceUserID, validBoardServiceBoardID)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestUpdateColumnName_AuthorizedColumn_UpdatesAndPersists(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	column := createBoardServiceColumn(t)
	columnRepository := &mockColumnRepository{columnToReturn: column}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.UpdateColumnName(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, "In Progress")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if columnRepository.updatedColumn == nil {
		t.Fatal("expected column to be updated")
	}
	if columnRepository.updatedColumn.Name() != "In Progress" {
		t.Errorf("expected column name In Progress, got %s", columnRepository.updatedColumn.Name())
	}
}

func TestUpdateColumnName_MissingColumn_ReturnsErrColumnNotFound(t *testing.T) {
	// Arrange
	service := NewBoardService(&mockBoardRepository{}, &mockColumnRepository{getByIDError: ports.ErrColumnNotFound}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.UpdateColumnName(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, "In Progress")

	// Assert
	if !errors.Is(err, ports.ErrColumnNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnNotFound, err)
	}
}

func TestUpdateColumnName_NilColumn_ReturnsErrColumnNotFound(t *testing.T) {
	// Arrange
	service := NewBoardService(&mockBoardRepository{}, &mockColumnRepository{}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.UpdateColumnName(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, "In Progress")

	// Assert
	if !errors.Is(err, ports.ErrColumnNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnNotFound, err)
	}
}

func TestUpdateColumnName_ColumnRepositoryUnavailable_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	service := NewBoardService(&mockBoardRepository{}, &mockColumnRepository{getByIDError: ports.ErrColumnRepositoryUnavailable}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.UpdateColumnName(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, "In Progress")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestUpdateColumnName_ParentBoardMissing_ReturnsErrColumnNotFoundAndWarns(t *testing.T) {
	// Arrange
	column := createBoardServiceColumn(t)
	logger := &mockBoardLogger{}
	service := NewBoardService(&mockBoardRepository{getByIDError: ports.ErrBoardNotFound}, &mockColumnRepository{columnToReturn: column}, &mockBoardIDGenerator{}, logger)

	// Act
	err := service.UpdateColumnName(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, "In Progress")

	// Assert
	if !errors.Is(err, ports.ErrColumnNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnNotFound, err)
	}
	if len(logger.warnMessages) != 1 {
		t.Fatalf("expected one warning log, got %d", len(logger.warnMessages))
	}
}

func TestUpdateColumnName_NilParentBoard_ReturnsErrColumnNotFoundAndWarns(t *testing.T) {
	// Arrange
	column := createBoardServiceColumn(t)
	logger := &mockBoardLogger{}
	service := NewBoardService(&mockBoardRepository{}, &mockColumnRepository{columnToReturn: column}, &mockBoardIDGenerator{}, logger)

	// Act
	err := service.UpdateColumnName(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, "In Progress")

	// Assert
	if !errors.Is(err, ports.ErrColumnNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnNotFound, err)
	}
	if len(logger.warnMessages) != 1 {
		t.Fatalf("expected one warning log, got %d", len(logger.warnMessages))
	}
}

func TestUpdateColumnName_ParentBoardRepositoryUnavailable_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	column := createBoardServiceColumn(t)
	service := NewBoardService(&mockBoardRepository{getByIDError: ports.ErrBoardRepositoryUnavailable}, &mockColumnRepository{columnToReturn: column}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.UpdateColumnName(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, "In Progress")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestUpdateColumnName_UnauthorizedColumn_ReturnsErrColumnNotFoundAndWarns(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceOtherUserID)
	column := createBoardServiceColumn(t)
	logger := &mockBoardLogger{}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, &mockColumnRepository{columnToReturn: column}, &mockBoardIDGenerator{}, logger)

	// Act
	err := service.UpdateColumnName(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, "In Progress")

	// Assert
	if !errors.Is(err, ports.ErrColumnNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnNotFound, err)
	}
	if len(logger.warnMessages) != 1 {
		t.Fatalf("expected one warning log, got %d", len(logger.warnMessages))
	}
}

func TestUpdateColumnName_InvalidName_ReturnsDomainError(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	column := createBoardServiceColumn(t)
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, &mockColumnRepository{columnToReturn: column}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.UpdateColumnName(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, "No")

	// Assert
	if !errors.Is(err, domain.ErrInvalidColumnName) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidColumnName, err)
	}
}

func TestUpdateColumnName_UpdateFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	column := createBoardServiceColumn(t)
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, &mockColumnRepository{columnToReturn: column, updateError: ports.ErrColumnRepositoryUnavailable}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.UpdateColumnName(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, "In Progress")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestMoveColumn_MoveRight_ShiftsAffectedColumnsAndPersistsBatch(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	targetColumn := createBoardServiceColumnWithPosition(t, "column-1", 0)
	firstShiftedColumn := createBoardServiceColumnWithPosition(t, "column-2", 1)
	secondShiftedColumn := createBoardServiceColumnWithPosition(t, "column-3", 2)
	unchangedColumn := createBoardServiceColumnWithPosition(t, "column-4", 3)
	columnRepository := &mockColumnRepository{
		columnToReturn:  targetColumn,
		columnsToReturn: []*domain.Column{targetColumn, firstShiftedColumn, secondShiftedColumn, unchangedColumn},
	}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.MoveColumn(context.Background(), validBoardServiceUserID, targetColumn.ID(), 2)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if targetColumn.Position() != 2 {
		t.Errorf("expected target position 2, got %d", targetColumn.Position())
	}
	if firstShiftedColumn.Position() != 0 {
		t.Errorf("expected first shifted column position 0, got %d", firstShiftedColumn.Position())
	}
	if secondShiftedColumn.Position() != 1 {
		t.Errorf("expected second shifted column position 1, got %d", secondShiftedColumn.Position())
	}
	if unchangedColumn.Position() != 3 {
		t.Errorf("expected unchanged column position 3, got %d", unchangedColumn.Position())
	}
	assertUpdatedPositionColumnIDs(t, columnRepository.updatedPositionColumns, []string{"column-2", "column-3", "column-1"})
}

func TestMoveColumn_MoveLeft_ShiftsAffectedColumnsAndPersistsBatch(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	firstShiftedColumn := createBoardServiceColumnWithPosition(t, "column-1", 0)
	secondShiftedColumn := createBoardServiceColumnWithPosition(t, "column-2", 1)
	targetColumn := createBoardServiceColumnWithPosition(t, "column-3", 2)
	unchangedColumn := createBoardServiceColumnWithPosition(t, "column-4", 3)
	columnRepository := &mockColumnRepository{
		columnToReturn:  targetColumn,
		columnsToReturn: []*domain.Column{firstShiftedColumn, secondShiftedColumn, targetColumn, unchangedColumn},
	}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.MoveColumn(context.Background(), validBoardServiceUserID, targetColumn.ID(), 0)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if targetColumn.Position() != 0 {
		t.Errorf("expected target position 0, got %d", targetColumn.Position())
	}
	if firstShiftedColumn.Position() != 1 {
		t.Errorf("expected first shifted column position 1, got %d", firstShiftedColumn.Position())
	}
	if secondShiftedColumn.Position() != 2 {
		t.Errorf("expected second shifted column position 2, got %d", secondShiftedColumn.Position())
	}
	if unchangedColumn.Position() != 3 {
		t.Errorf("expected unchanged column position 3, got %d", unchangedColumn.Position())
	}
	assertUpdatedPositionColumnIDs(t, columnRepository.updatedPositionColumns, []string{"column-1", "column-2", "column-3"})
}

func TestMoveColumn_SamePosition_ReturnsNilWithoutLoadingSiblings(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	column := createBoardServiceColumn(t)
	columnRepository := &mockColumnRepository{columnToReturn: column}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.MoveColumn(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, validColumnServicePosition)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if columnRepository.getByBoardIDCalled {
		t.Fatal("expected board columns not to be loaded")
	}
	if columnRepository.updatePositionsCalled {
		t.Fatal("expected positions not to be updated")
	}
}

func TestMoveColumn_NegativePosition_ReturnsDomainError(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	column := createBoardServiceColumn(t)
	columnRepository := &mockColumnRepository{columnToReturn: column}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.MoveColumn(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, -1)

	// Assert
	if !errors.Is(err, domain.ErrInvalidColumnPosition) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidColumnPosition, err)
	}
	if columnRepository.updatePositionsCalled {
		t.Fatal("expected positions not to be updated")
	}
}

func TestMoveColumn_GetBoardColumnsFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	column := createBoardServiceColumn(t)
	columnRepository := &mockColumnRepository{
		columnToReturn:    column,
		getByBoardIDError: ports.ErrColumnRepositoryUnavailable,
	}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.MoveColumn(context.Background(), validBoardServiceUserID, validBoardServiceColumnID, 2)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestMoveColumn_UpdatePositionsFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	targetColumn := createBoardServiceColumnWithPosition(t, "column-1", 0)
	shiftedColumn := createBoardServiceColumnWithPosition(t, "column-2", 1)
	columnRepository := &mockColumnRepository{
		columnToReturn:       targetColumn,
		columnsToReturn:      []*domain.Column{targetColumn, shiftedColumn},
		updatePositionsError: ports.ErrColumnRepositoryUnavailable,
	}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.MoveColumn(context.Background(), validBoardServiceUserID, targetColumn.ID(), 1)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestDeleteColumn_AuthorizedColumn_DeletesColumn(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	column := createBoardServiceColumn(t)
	columnRepository := &mockColumnRepository{columnToReturn: column}
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, columnRepository, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.DeleteColumn(context.Background(), validBoardServiceUserID, validBoardServiceColumnID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if columnRepository.deletedColumnID != validBoardServiceColumnID {
		t.Errorf("expected deleted column ID %s, got %s", validBoardServiceColumnID, columnRepository.deletedColumnID)
	}
}

func TestDeleteColumn_DeleteFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	board := createBoardServiceBoard(t, validBoardServiceUserID)
	column := createBoardServiceColumn(t)
	service := NewBoardService(&mockBoardRepository{boardToReturn: board}, &mockColumnRepository{columnToReturn: column, deleteError: ports.ErrColumnRepositoryUnavailable}, &mockBoardIDGenerator{}, &mockBoardLogger{})

	// Act
	err := service.DeleteColumn(context.Background(), validBoardServiceUserID, validBoardServiceColumnID)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func createBoardServiceBoard(t *testing.T, userID string) *domain.Board {
	t.Helper()

	board, err := domain.NewBoard(validBoardServiceBoardID, userID, validBoardServiceName)
	if err != nil {
		t.Fatalf("expected board to be valid, got: %v", err)
	}

	return board
}

func createBoardServiceColumn(t *testing.T) *domain.Column {
	t.Helper()

	return createBoardServiceColumnWithPosition(t, validBoardServiceColumnID, validColumnServicePosition)
}

func createBoardServiceColumnWithPosition(t *testing.T, columnID string, position int) *domain.Column {
	t.Helper()

	column, err := domain.NewColumn(columnID, validBoardServiceBoardID, validColumnServiceName, position)
	if err != nil {
		t.Fatalf("expected column to be valid, got: %v", err)
	}

	return column
}

func assertUpdatedPositionColumnIDs(t *testing.T, columns []*domain.Column, expectedIDs []string) {
	t.Helper()

	if len(columns) != len(expectedIDs) {
		t.Fatalf("expected %d updated columns, got %d", len(expectedIDs), len(columns))
	}

	for index, expectedID := range expectedIDs {
		if columns[index].ID() != expectedID {
			t.Errorf("expected updated column ID %s at index %d, got %s", expectedID, index, columns[index].ID())
		}
	}
}
