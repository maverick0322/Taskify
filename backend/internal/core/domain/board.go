package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	minBoardNameLength  = 3
	minColumnNameLength = 3
	defaultColumnColor  = "slate"
)

var (
	ErrInvalidBoardID        = errors.New("domain: board ID cannot be empty")
	ErrInvalidBoardUserID    = errors.New("domain: board user ID cannot be empty")
	ErrInvalidBoardName      = errors.New("domain: board name does not meet minimum length")
	ErrInvalidBoardCreatedAt = errors.New("domain: board created at cannot be zero")
	ErrInvalidBoardUpdatedAt = errors.New("domain: board updated at cannot be zero")

	ErrInvalidColumnID        = errors.New("domain: column ID cannot be empty")
	ErrInvalidColumnBoardID   = errors.New("domain: column board ID cannot be empty")
	ErrInvalidColumnName      = errors.New("domain: column name does not meet minimum length")
	ErrInvalidColumnColor     = errors.New("domain: column color cannot be empty")
	ErrInvalidColumnPosition  = errors.New("domain: column position cannot be negative")
	ErrInvalidColumnCreatedAt = errors.New("domain: column created at cannot be zero")
	ErrInvalidColumnUpdatedAt = errors.New("domain: column updated at cannot be zero")
)

// Board is the aggregate root for Kanban board ownership and identity.
type Board struct {
	id        string
	userID    string
	name      string
	createdAt time.Time
	updatedAt time.Time
}

// NewBoard centralizes invariants so invalid board state cannot enter the domain.
func NewBoard(id, userID, name string) (*Board, error) {
	boardFields, err := validateBoardFields(id, userID, name)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	return &Board{
		id:        boardFields.id,
		userID:    boardFields.userID,
		name:      boardFields.name,
		createdAt: currentTime,
		updatedAt: currentTime,
	}, nil
}

// RehydrateBoard restores persisted state without exposing mutation-oriented setters.
func RehydrateBoard(id, userID, name string, createdAt, updatedAt time.Time) (*Board, error) {
	boardFields, err := validateBoardFields(id, userID, name)
	if err != nil {
		return nil, err
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidBoardCreatedAt
	}
	if updatedAt.IsZero() {
		return nil, ErrInvalidBoardUpdatedAt
	}

	return &Board{
		id:        boardFields.id,
		userID:    boardFields.userID,
		name:      boardFields.name,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}, nil
}

func (board *Board) UpdateName(newName string) error {
	trimmedName, err := validateBoardName(newName)
	if err != nil {
		return err
	}

	board.name = trimmedName
	board.touch()
	return nil
}

func (board *Board) ID() string {
	return board.id
}

func (board *Board) UserID() string {
	return board.userID
}

func (board *Board) Name() string {
	return board.name
}

func (board *Board) CreatedAt() time.Time {
	return board.createdAt
}

func (board *Board) UpdatedAt() time.Time {
	return board.updatedAt
}

func (board *Board) touch() {
	board.updatedAt = time.Now()
}

// Column is a board-scoped entity ordered by visual position.
type Column struct {
	id        string
	boardID   string
	name      string
	color     string
	position  int
	createdAt time.Time
	updatedAt time.Time
}

// NewColumn centralizes invariants so invalid column state cannot enter the domain.
func NewColumn(id, boardID, name string, columnOptions ...interface{}) (*Column, error) {
	color, position, err := parseColumnOptions(columnOptions...)
	if err != nil {
		return nil, err
	}
	columnFields, err := validateColumnFields(id, boardID, name, color, position)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	return &Column{
		id:        columnFields.id,
		boardID:   columnFields.boardID,
		name:      columnFields.name,
		color:     columnFields.color,
		position:  position,
		createdAt: currentTime,
		updatedAt: currentTime,
	}, nil
}

// RehydrateColumn restores persisted state without exposing mutation-oriented setters.
func RehydrateColumn(id, boardID, name string, columnOptions ...interface{}) (*Column, error) {
	color, position, createdAt, updatedAt, err := parseRehydrateColumnOptions(columnOptions...)
	if err != nil {
		return nil, err
	}
	columnFields, err := validateColumnFields(id, boardID, name, color, position)
	if err != nil {
		return nil, err
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidColumnCreatedAt
	}
	if updatedAt.IsZero() {
		return nil, ErrInvalidColumnUpdatedAt
	}

	return &Column{
		id:        columnFields.id,
		boardID:   columnFields.boardID,
		name:      columnFields.name,
		color:     columnFields.color,
		position:  position,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}, nil
}

func (column *Column) UpdateName(newName string) error {
	trimmedName, err := validateColumnName(newName)
	if err != nil {
		return err
	}

	column.name = trimmedName
	column.touch()
	return nil
}

func (column *Column) Update(newName, newColor string) error {
	trimmedName, err := validateColumnName(newName)
	if err != nil {
		return err
	}
	trimmedColor, err := validateColumnColor(newColor)
	if err != nil {
		return err
	}

	column.name = trimmedName
	column.color = trimmedColor
	column.touch()
	return nil
}

func (column *Column) ChangePosition(newPosition int) error {
	if newPosition < 0 {
		return ErrInvalidColumnPosition
	}

	column.position = newPosition
	column.touch()
	return nil
}

func (column *Column) ID() string {
	return column.id
}

func (column *Column) BoardID() string {
	return column.boardID
}

func (column *Column) Name() string {
	return column.name
}

func (column *Column) Color() string {
	return column.color
}

func (column *Column) Position() int {
	return column.position
}

func (column *Column) CreatedAt() time.Time {
	return column.createdAt
}

func (column *Column) UpdatedAt() time.Time {
	return column.updatedAt
}

func (column *Column) touch() {
	column.updatedAt = time.Now()
}

func validateBoardFields(id, userID, name string) (validatedBoardFields, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return validatedBoardFields{}, ErrInvalidBoardID
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return validatedBoardFields{}, ErrInvalidBoardUserID
	}

	trimmedName, err := validateBoardName(name)
	if err != nil {
		return validatedBoardFields{}, err
	}

	return validatedBoardFields{
		id:     trimmedID,
		userID: trimmedUserID,
		name:   trimmedName,
	}, nil
}

func validateColumnFields(id, boardID, name, color string, position int) (validatedColumnFields, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return validatedColumnFields{}, ErrInvalidColumnID
	}

	trimmedBoardID := strings.TrimSpace(boardID)
	if trimmedBoardID == "" {
		return validatedColumnFields{}, ErrInvalidColumnBoardID
	}

	trimmedName, err := validateColumnName(name)
	if err != nil {
		return validatedColumnFields{}, err
	}

	trimmedColor, err := validateColumnColor(color)
	if err != nil {
		return validatedColumnFields{}, err
	}

	if position < 0 {
		return validatedColumnFields{}, ErrInvalidColumnPosition
	}

	return validatedColumnFields{
		id:      trimmedID,
		boardID: trimmedBoardID,
		name:    trimmedName,
		color:   trimmedColor,
	}, nil
}

func validateBoardName(name string) (string, error) {
	trimmedName := strings.TrimSpace(name)
	if len(trimmedName) < minBoardNameLength {
		return "", ErrInvalidBoardName
	}

	return trimmedName, nil
}

func validateColumnName(name string) (string, error) {
	trimmedName := strings.TrimSpace(name)
	if len(trimmedName) < minColumnNameLength {
		return "", ErrInvalidColumnName
	}

	return trimmedName, nil
}

func validateColumnColor(color string) (string, error) {
	trimmedColor := strings.TrimSpace(color)
	if trimmedColor == "" {
		return "", ErrInvalidColumnColor
	}

	return trimmedColor, nil
}

func parseColumnOptions(columnOptions ...interface{}) (string, int, error) {
	switch len(columnOptions) {
	case 1:
		position, ok := columnOptions[0].(int)
		if !ok {
			return "", 0, ErrInvalidColumnPosition
		}
		return defaultColumnColor, position, nil
	case 2:
		color, ok := columnOptions[0].(string)
		if !ok {
			return "", 0, ErrInvalidColumnColor
		}
		position, ok := columnOptions[1].(int)
		if !ok {
			return "", 0, ErrInvalidColumnPosition
		}
		return color, position, nil
	default:
		return "", 0, ErrInvalidColumnPosition
	}
}

func parseRehydrateColumnOptions(columnOptions ...interface{}) (string, int, time.Time, time.Time, error) {
	switch len(columnOptions) {
	case 3:
		position, ok := columnOptions[0].(int)
		if !ok {
			return "", 0, time.Time{}, time.Time{}, ErrInvalidColumnPosition
		}
		createdAt, ok := columnOptions[1].(time.Time)
		if !ok {
			return "", 0, time.Time{}, time.Time{}, ErrInvalidColumnCreatedAt
		}
		updatedAt, ok := columnOptions[2].(time.Time)
		if !ok {
			return "", 0, time.Time{}, time.Time{}, ErrInvalidColumnUpdatedAt
		}
		return defaultColumnColor, position, createdAt, updatedAt, nil
	case 4:
		color, ok := columnOptions[0].(string)
		if !ok {
			return "", 0, time.Time{}, time.Time{}, ErrInvalidColumnColor
		}
		position, ok := columnOptions[1].(int)
		if !ok {
			return "", 0, time.Time{}, time.Time{}, ErrInvalidColumnPosition
		}
		createdAt, ok := columnOptions[2].(time.Time)
		if !ok {
			return "", 0, time.Time{}, time.Time{}, ErrInvalidColumnCreatedAt
		}
		updatedAt, ok := columnOptions[3].(time.Time)
		if !ok {
			return "", 0, time.Time{}, time.Time{}, ErrInvalidColumnUpdatedAt
		}
		return color, position, createdAt, updatedAt, nil
	default:
		return "", 0, time.Time{}, time.Time{}, ErrInvalidColumnPosition
	}
}

type validatedBoardFields struct {
	id     string
	userID string
	name   string
}

type validatedColumnFields struct {
	id      string
	boardID string
	name    string
	color   string
}
