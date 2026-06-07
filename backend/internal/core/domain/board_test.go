package domain

import (
	"errors"
	"testing"
	"time"
)

const (
	validBoardID     = "board-123"
	validBoardUserID = "user-123"
	validBoardName   = "Product Roadmap"

	validColumnID       = "column-123"
	validColumnBoardID  = "board-123"
	validColumnName     = "Backlog"
	validColumnPosition = 0
)

func TestNewBoard_ValidFields_ReturnsBoard(t *testing.T) {
	// Arrange

	// Act
	board, err := NewBoard(validBoardID, validBoardUserID, validBoardName)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if board == nil {
		t.Fatal("expected board, got nil")
	}
	if board.ID() != validBoardID {
		t.Errorf("expected board ID %s, got %s", validBoardID, board.ID())
	}
	if board.UserID() != validBoardUserID {
		t.Errorf("expected board user ID %s, got %s", validBoardUserID, board.UserID())
	}
	if board.Name() != validBoardName {
		t.Errorf("expected board name %s, got %s", validBoardName, board.Name())
	}
	if board.CreatedAt().IsZero() {
		t.Fatal("expected created at to be set")
	}
	if !board.UpdatedAt().Equal(board.CreatedAt()) {
		t.Errorf("expected updated at %v, got %v", board.CreatedAt(), board.UpdatedAt())
	}
}

func TestNewBoard_ValidFields_TrimsFields(t *testing.T) {
	// Arrange
	idWithSpaces := "  board-123  "
	userIDWithSpaces := "  user-123  "
	nameWithSpaces := "  Product Roadmap  "

	// Act
	board, err := NewBoard(idWithSpaces, userIDWithSpaces, nameWithSpaces)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if board.ID() != validBoardID {
		t.Errorf("expected board ID %s, got %s", validBoardID, board.ID())
	}
	if board.UserID() != validBoardUserID {
		t.Errorf("expected board user ID %s, got %s", validBoardUserID, board.UserID())
	}
	if board.Name() != validBoardName {
		t.Errorf("expected board name %s, got %s", validBoardName, board.Name())
	}
}

func TestNewBoard_EmptyID_ReturnsErrInvalidBoardID(t *testing.T) {
	// Arrange
	emptyID := ""

	// Act
	_, err := NewBoard(emptyID, validBoardUserID, validBoardName)

	// Assert
	if !errors.Is(err, ErrInvalidBoardID) {
		t.Errorf("expected error %v, got %v", ErrInvalidBoardID, err)
	}
}

func TestNewBoard_EmptyUserID_ReturnsErrInvalidBoardUserID(t *testing.T) {
	// Arrange
	emptyUserID := ""

	// Act
	_, err := NewBoard(validBoardID, emptyUserID, validBoardName)

	// Assert
	if !errors.Is(err, ErrInvalidBoardUserID) {
		t.Errorf("expected error %v, got %v", ErrInvalidBoardUserID, err)
	}
}

func TestNewBoard_ShortNameAfterTrim_ReturnsErrInvalidBoardName(t *testing.T) {
	// Arrange
	shortName := "  Go  "

	// Act
	_, err := NewBoard(validBoardID, validBoardUserID, shortName)

	// Assert
	if !errors.Is(err, ErrInvalidBoardName) {
		t.Errorf("expected error %v, got %v", ErrInvalidBoardName, err)
	}
}

func TestRehydrateBoard_ValidFields_ReturnsBoardWithPersistedDates(t *testing.T) {
	// Arrange
	createdAt := time.Now().Add(-2 * time.Hour)
	updatedAt := time.Now().Add(-time.Hour)

	// Act
	board, err := RehydrateBoard(validBoardID, validBoardUserID, validBoardName, createdAt, updatedAt)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !board.CreatedAt().Equal(createdAt) {
		t.Errorf("expected created at %v, got %v", createdAt, board.CreatedAt())
	}
	if !board.UpdatedAt().Equal(updatedAt) {
		t.Errorf("expected updated at %v, got %v", updatedAt, board.UpdatedAt())
	}
}

func TestRehydrateBoard_ZeroCreatedAt_ReturnsErrInvalidBoardCreatedAt(t *testing.T) {
	// Arrange
	zeroCreatedAt := time.Time{}
	updatedAt := time.Now().Add(-time.Hour)

	// Act
	_, err := RehydrateBoard(validBoardID, validBoardUserID, validBoardName, zeroCreatedAt, updatedAt)

	// Assert
	if !errors.Is(err, ErrInvalidBoardCreatedAt) {
		t.Errorf("expected error %v, got %v", ErrInvalidBoardCreatedAt, err)
	}
}

func TestRehydrateBoard_ZeroUpdatedAt_ReturnsErrInvalidBoardUpdatedAt(t *testing.T) {
	// Arrange
	createdAt := time.Now().Add(-2 * time.Hour)
	zeroUpdatedAt := time.Time{}

	// Act
	_, err := RehydrateBoard(validBoardID, validBoardUserID, validBoardName, createdAt, zeroUpdatedAt)

	// Assert
	if !errors.Is(err, ErrInvalidBoardUpdatedAt) {
		t.Errorf("expected error %v, got %v", ErrInvalidBoardUpdatedAt, err)
	}
}

func TestBoard_UpdateNameValidName_UpdatesNameAndUpdatedAt(t *testing.T) {
	// Arrange
	board := createValidBoard(t)
	previousUpdatedAt := board.UpdatedAt()
	newName := "Delivery Plan"
	waitForBoardTimestampChange()

	// Act
	err := board.UpdateName(newName)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if board.Name() != newName {
		t.Errorf("expected board name %s, got %s", newName, board.Name())
	}
	if !board.UpdatedAt().After(previousUpdatedAt) {
		t.Errorf("expected updated at to be after %v, got %v", previousUpdatedAt, board.UpdatedAt())
	}
}

func TestBoard_UpdateNameValidName_TrimsName(t *testing.T) {
	// Arrange
	board := createValidBoard(t)
	nameWithSpaces := "  Delivery Plan  "

	// Act
	err := board.UpdateName(nameWithSpaces)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if board.Name() != "Delivery Plan" {
		t.Errorf("expected board name %s, got %s", "Delivery Plan", board.Name())
	}
}

func TestBoard_UpdateNameShortName_ReturnsErrInvalidBoardName(t *testing.T) {
	// Arrange
	board := createValidBoard(t)
	shortName := "No"

	// Act
	err := board.UpdateName(shortName)

	// Assert
	if !errors.Is(err, ErrInvalidBoardName) {
		t.Errorf("expected error %v, got %v", ErrInvalidBoardName, err)
	}
}

func TestNewColumn_ValidFields_ReturnsColumn(t *testing.T) {
	// Arrange

	// Act
	column, err := NewColumn(validColumnID, validColumnBoardID, validColumnName, validColumnPosition)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if column == nil {
		t.Fatal("expected column, got nil")
	}
	if column.ID() != validColumnID {
		t.Errorf("expected column ID %s, got %s", validColumnID, column.ID())
	}
	if column.BoardID() != validColumnBoardID {
		t.Errorf("expected column board ID %s, got %s", validColumnBoardID, column.BoardID())
	}
	if column.Name() != validColumnName {
		t.Errorf("expected column name %s, got %s", validColumnName, column.Name())
	}
	if column.Position() != validColumnPosition {
		t.Errorf("expected column position %d, got %d", validColumnPosition, column.Position())
	}
	if column.CreatedAt().IsZero() {
		t.Fatal("expected created at to be set")
	}
	if !column.UpdatedAt().Equal(column.CreatedAt()) {
		t.Errorf("expected updated at %v, got %v", column.CreatedAt(), column.UpdatedAt())
	}
}

func TestNewColumn_ValidFields_TrimsFields(t *testing.T) {
	// Arrange
	idWithSpaces := "  column-123  "
	boardIDWithSpaces := "  board-123  "
	nameWithSpaces := "  Backlog  "

	// Act
	column, err := NewColumn(idWithSpaces, boardIDWithSpaces, nameWithSpaces, validColumnPosition)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if column.ID() != validColumnID {
		t.Errorf("expected column ID %s, got %s", validColumnID, column.ID())
	}
	if column.BoardID() != validColumnBoardID {
		t.Errorf("expected column board ID %s, got %s", validColumnBoardID, column.BoardID())
	}
	if column.Name() != validColumnName {
		t.Errorf("expected column name %s, got %s", validColumnName, column.Name())
	}
}

func TestNewColumn_EmptyID_ReturnsErrInvalidColumnID(t *testing.T) {
	// Arrange
	emptyID := ""

	// Act
	_, err := NewColumn(emptyID, validColumnBoardID, validColumnName, validColumnPosition)

	// Assert
	if !errors.Is(err, ErrInvalidColumnID) {
		t.Errorf("expected error %v, got %v", ErrInvalidColumnID, err)
	}
}

func TestNewColumn_EmptyBoardID_ReturnsErrInvalidColumnBoardID(t *testing.T) {
	// Arrange
	emptyBoardID := ""

	// Act
	_, err := NewColumn(validColumnID, emptyBoardID, validColumnName, validColumnPosition)

	// Assert
	if !errors.Is(err, ErrInvalidColumnBoardID) {
		t.Errorf("expected error %v, got %v", ErrInvalidColumnBoardID, err)
	}
}

func TestNewColumn_ShortNameAfterTrim_ReturnsErrInvalidColumnName(t *testing.T) {
	// Arrange
	shortName := "  Go  "

	// Act
	_, err := NewColumn(validColumnID, validColumnBoardID, shortName, validColumnPosition)

	// Assert
	if !errors.Is(err, ErrInvalidColumnName) {
		t.Errorf("expected error %v, got %v", ErrInvalidColumnName, err)
	}
}

func TestNewColumn_NegativePosition_ReturnsErrInvalidColumnPosition(t *testing.T) {
	// Arrange
	negativePosition := -1

	// Act
	_, err := NewColumn(validColumnID, validColumnBoardID, validColumnName, negativePosition)

	// Assert
	if !errors.Is(err, ErrInvalidColumnPosition) {
		t.Errorf("expected error %v, got %v", ErrInvalidColumnPosition, err)
	}
}

func TestRehydrateColumn_ValidFields_ReturnsColumnWithPersistedDates(t *testing.T) {
	// Arrange
	createdAt := time.Now().Add(-2 * time.Hour)
	updatedAt := time.Now().Add(-time.Hour)

	// Act
	column, err := RehydrateColumn(validColumnID, validColumnBoardID, validColumnName, validColumnPosition, createdAt, updatedAt)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !column.CreatedAt().Equal(createdAt) {
		t.Errorf("expected created at %v, got %v", createdAt, column.CreatedAt())
	}
	if !column.UpdatedAt().Equal(updatedAt) {
		t.Errorf("expected updated at %v, got %v", updatedAt, column.UpdatedAt())
	}
}

func TestRehydrateColumn_ZeroCreatedAt_ReturnsErrInvalidColumnCreatedAt(t *testing.T) {
	// Arrange
	zeroCreatedAt := time.Time{}
	updatedAt := time.Now().Add(-time.Hour)

	// Act
	_, err := RehydrateColumn(validColumnID, validColumnBoardID, validColumnName, validColumnPosition, zeroCreatedAt, updatedAt)

	// Assert
	if !errors.Is(err, ErrInvalidColumnCreatedAt) {
		t.Errorf("expected error %v, got %v", ErrInvalidColumnCreatedAt, err)
	}
}

func TestRehydrateColumn_ZeroUpdatedAt_ReturnsErrInvalidColumnUpdatedAt(t *testing.T) {
	// Arrange
	createdAt := time.Now().Add(-2 * time.Hour)
	zeroUpdatedAt := time.Time{}

	// Act
	_, err := RehydrateColumn(validColumnID, validColumnBoardID, validColumnName, validColumnPosition, createdAt, zeroUpdatedAt)

	// Assert
	if !errors.Is(err, ErrInvalidColumnUpdatedAt) {
		t.Errorf("expected error %v, got %v", ErrInvalidColumnUpdatedAt, err)
	}
}

func TestColumn_UpdateNameValidName_UpdatesNameAndUpdatedAt(t *testing.T) {
	// Arrange
	column := createValidColumn(t)
	previousUpdatedAt := column.UpdatedAt()
	newName := "In Progress"
	waitForBoardTimestampChange()

	// Act
	err := column.UpdateName(newName)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if column.Name() != newName {
		t.Errorf("expected column name %s, got %s", newName, column.Name())
	}
	if !column.UpdatedAt().After(previousUpdatedAt) {
		t.Errorf("expected updated at to be after %v, got %v", previousUpdatedAt, column.UpdatedAt())
	}
}

func TestColumn_UpdateNameValidName_TrimsName(t *testing.T) {
	// Arrange
	column := createValidColumn(t)
	nameWithSpaces := "  In Progress  "

	// Act
	err := column.UpdateName(nameWithSpaces)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if column.Name() != "In Progress" {
		t.Errorf("expected column name %s, got %s", "In Progress", column.Name())
	}
}

func TestColumn_UpdateNameShortName_ReturnsErrInvalidColumnName(t *testing.T) {
	// Arrange
	column := createValidColumn(t)
	shortName := "No"

	// Act
	err := column.UpdateName(shortName)

	// Assert
	if !errors.Is(err, ErrInvalidColumnName) {
		t.Errorf("expected error %v, got %v", ErrInvalidColumnName, err)
	}
}

func TestColumn_ChangePositionValidPosition_UpdatesPositionAndUpdatedAt(t *testing.T) {
	// Arrange
	column := createValidColumn(t)
	previousUpdatedAt := column.UpdatedAt()
	newPosition := 2
	waitForBoardTimestampChange()

	// Act
	err := column.ChangePosition(newPosition)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if column.Position() != newPosition {
		t.Errorf("expected column position %d, got %d", newPosition, column.Position())
	}
	if !column.UpdatedAt().After(previousUpdatedAt) {
		t.Errorf("expected updated at to be after %v, got %v", previousUpdatedAt, column.UpdatedAt())
	}
}

func TestColumn_ChangePositionNegativePosition_ReturnsErrInvalidColumnPosition(t *testing.T) {
	// Arrange
	column := createValidColumn(t)
	negativePosition := -1

	// Act
	err := column.ChangePosition(negativePosition)

	// Assert
	if !errors.Is(err, ErrInvalidColumnPosition) {
		t.Errorf("expected error %v, got %v", ErrInvalidColumnPosition, err)
	}
}

func createValidBoard(t *testing.T) *Board {
	t.Helper()

	board, err := NewBoard(validBoardID, validBoardUserID, validBoardName)
	if err != nil {
		t.Fatalf("expected board to be valid, got: %v", err)
	}

	return board
}

func createValidColumn(t *testing.T) *Column {
	t.Helper()

	column, err := NewColumn(validColumnID, validColumnBoardID, validColumnName, validColumnPosition)
	if err != nil {
		t.Fatalf("expected column to be valid, got: %v", err)
	}

	return column
}

func waitForBoardTimestampChange() {
	time.Sleep(time.Millisecond)
}
