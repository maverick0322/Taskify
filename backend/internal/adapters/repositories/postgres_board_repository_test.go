package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type fakePostgresBoardDatabase struct {
	execError         error
	queryError        error
	rowToReturn       pgx.Row
	rowsToReturn      pgx.Rows
	receivedSQL       string
	receivedArguments []interface{}
}

func (database *fakePostgresBoardDatabase) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return pgconn.CommandTag{}, database.execError
}

func (database *fakePostgresBoardDatabase) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return database.rowsToReturn, database.queryError
}

func (database *fakePostgresBoardDatabase) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return database.rowToReturn
}

type fakePostgresBoardRow struct {
	values []interface{}
	err    error
}

func (row fakePostgresBoardRow) Scan(destinations ...interface{}) error {
	if row.err != nil {
		return row.err
	}

	assignBoardScanValues(destinations, row.values)
	return nil
}

type fakePostgresBoardRows struct {
	rows        []fakePostgresBoardRow
	currentRow  int
	errToReturn error
	closed      bool
}

func (rows *fakePostgresBoardRows) Close() {
	rows.closed = true
}

func (rows *fakePostgresBoardRows) Err() error {
	return rows.errToReturn
}

func (rows *fakePostgresBoardRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (rows *fakePostgresBoardRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (rows *fakePostgresBoardRows) Next() bool {
	return rows.currentRow < len(rows.rows)
}

func (rows *fakePostgresBoardRows) Scan(destinations ...interface{}) error {
	row := rows.rows[rows.currentRow]
	rows.currentRow++
	return row.Scan(destinations...)
}

func (rows *fakePostgresBoardRows) Values() ([]interface{}, error) {
	return nil, nil
}

func (rows *fakePostgresBoardRows) RawValues() [][]byte {
	return nil
}

func (rows *fakePostgresBoardRows) Conn() *pgx.Conn {
	return nil
}

func TestNewPostgresBoardRepository_NilPool_ReturnsRepository(t *testing.T) {
	// Arrange
	logger := &fakeRepositoryLogger{}

	// Act
	repository := NewPostgresBoardRepository(nil, logger)

	// Assert
	if repository == nil {
		t.Fatal("expected repository, got nil")
	}
}

func TestPostgresBoardRepository_SaveValidBoard_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresBoardDatabase{}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), createRepositoryBoard(t))

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != saveBoardQuery {
		t.Errorf("expected save board query to be used")
	}
	if len(database.receivedArguments) != 5 {
		t.Errorf("expected five arguments, got %d", len(database.receivedArguments))
	}
}

func TestPostgresBoardRepository_SaveNilBoard_ReturnsErrBoardRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresBoardRepository{database: &fakePostgresBoardDatabase{}, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), nil)

	// Assert
	if !errors.Is(err, ports.ErrBoardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardRepositoryUnavailable, err)
	}
}

func TestPostgresBoardRepository_SaveDatabaseFailure_ReturnsErrBoardRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresBoardDatabase{execError: errors.New("database failure")}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), createRepositoryBoard(t))

	// Assert
	if !errors.Is(err, ports.ErrBoardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardRepositoryUnavailable, err)
	}
}

func TestPostgresBoardRepository_GetByIDExistingBoard_ReturnsBoard(t *testing.T) {
	// Arrange
	database := &fakePostgresBoardDatabase{rowToReturn: fakePostgresBoardRow{values: validStoredBoardValues()}}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	board, err := repository.GetByID(context.Background(), "board-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if board.ID() != "board-123" {
		t.Errorf("expected board ID board-123, got %s", board.ID())
	}
	if database.receivedSQL != getBoardByIDQuery {
		t.Errorf("expected get board by id query to be used")
	}
}

func TestPostgresBoardRepository_GetByIDMissingBoard_ReturnsErrBoardNotFound(t *testing.T) {
	// Arrange
	database := &fakePostgresBoardDatabase{rowToReturn: fakePostgresBoardRow{err: pgx.ErrNoRows}}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByID(context.Background(), "board-123")

	// Assert
	if !errors.Is(err, ports.ErrBoardNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardNotFound, err)
	}
}

func TestPostgresBoardRepository_GetByIDCorruptedBoard_ReturnsErrBoardRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresBoardDatabase{rowToReturn: fakePostgresBoardRow{values: corruptedStoredBoardValues()}}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByID(context.Background(), "board-123")

	// Assert
	if !errors.Is(err, ports.ErrBoardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardRepositoryUnavailable, err)
	}
}

func TestPostgresBoardRepository_GetByUserIDExistingBoards_ReturnsBoards(t *testing.T) {
	// Arrange
	rows := &fakePostgresBoardRows{
		rows: []fakePostgresBoardRow{
			{values: validStoredBoardValues()},
			{values: validStoredBoardValues()},
		},
	}
	database := &fakePostgresBoardDatabase{rowsToReturn: rows}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	boards, err := repository.GetByUserID(context.Background(), "user-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(boards) != 2 {
		t.Fatalf("expected two boards, got %d", len(boards))
	}
	if !rows.closed {
		t.Fatal("expected rows to be closed")
	}
}

func TestPostgresBoardRepository_GetByUserIDQueryFailure_ReturnsErrBoardRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresBoardDatabase{queryError: errors.New("query failure")}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByUserID(context.Background(), "user-123")

	// Assert
	if !errors.Is(err, ports.ErrBoardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardRepositoryUnavailable, err)
	}
}

func TestPostgresBoardRepository_GetByUserIDScanFailure_ReturnsErrBoardRepositoryUnavailable(t *testing.T) {
	// Arrange
	rows := &fakePostgresBoardRows{rows: []fakePostgresBoardRow{{err: errors.New("scan failure")}}}
	database := &fakePostgresBoardDatabase{rowsToReturn: rows}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByUserID(context.Background(), "user-123")

	// Assert
	if !errors.Is(err, ports.ErrBoardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardRepositoryUnavailable, err)
	}
}

func TestPostgresBoardRepository_GetByUserIDRowsError_ReturnsErrBoardRepositoryUnavailable(t *testing.T) {
	// Arrange
	rows := &fakePostgresBoardRows{errToReturn: errors.New("rows failure")}
	database := &fakePostgresBoardDatabase{rowsToReturn: rows}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByUserID(context.Background(), "user-123")

	// Assert
	if !errors.Is(err, ports.ErrBoardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardRepositoryUnavailable, err)
	}
}

func TestPostgresBoardRepository_UpdateValidBoard_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresBoardDatabase{}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Update(context.Background(), createRepositoryBoard(t))

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != updateBoardQuery {
		t.Errorf("expected update board query to be used")
	}
}

func TestPostgresBoardRepository_UpdateNilBoard_ReturnsErrBoardRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresBoardRepository{database: &fakePostgresBoardDatabase{}, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Update(context.Background(), nil)

	// Assert
	if !errors.Is(err, ports.ErrBoardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardRepositoryUnavailable, err)
	}
}

func TestPostgresBoardRepository_UpdateDatabaseFailure_ReturnsErrBoardRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresBoardDatabase{execError: errors.New("database failure")}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Update(context.Background(), createRepositoryBoard(t))

	// Assert
	if !errors.Is(err, ports.ErrBoardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardRepositoryUnavailable, err)
	}
}

func TestPostgresBoardRepository_DeleteValidBoard_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresBoardDatabase{}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Delete(context.Background(), "board-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != deleteBoardQuery {
		t.Errorf("expected delete board query to be used")
	}
}

func TestPostgresBoardRepository_DeleteDatabaseFailure_ReturnsErrBoardRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresBoardDatabase{execError: errors.New("database failure")}
	repository := &PostgresBoardRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Delete(context.Background(), "board-123")

	// Assert
	if !errors.Is(err, ports.ErrBoardRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardRepositoryUnavailable, err)
	}
}

func assignBoardScanValues(destinations []interface{}, values []interface{}) {
	for index, value := range values {
		switch destination := destinations[index].(type) {
		case *string:
			*destination = value.(string)
		case *time.Time:
			*destination = value.(time.Time)
		}
	}
}

func validStoredBoardValues() []interface{} {
	return []interface{}{
		"board-123",
		"user-123",
		"Product Roadmap",
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-time.Hour),
	}
}

func corruptedStoredBoardValues() []interface{} {
	values := validStoredBoardValues()
	values[2] = "No"
	return values
}

func createRepositoryBoard(t *testing.T) *domain.Board {
	t.Helper()

	board, err := domain.NewBoard("board-123", "user-123", "Product Roadmap")
	if err != nil {
		t.Fatalf("expected board to be valid, got: %v", err)
	}

	return board
}
