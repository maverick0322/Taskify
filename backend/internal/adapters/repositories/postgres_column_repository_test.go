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

type fakePostgresColumnDatabase struct {
	execError         error
	queryError        error
	rowToReturn       pgx.Row
	rowsToReturn      pgx.Rows
	receivedSQL       string
	receivedArguments []interface{}
}

func (database *fakePostgresColumnDatabase) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return pgconn.CommandTag{}, database.execError
}

func (database *fakePostgresColumnDatabase) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return database.rowsToReturn, database.queryError
}

func (database *fakePostgresColumnDatabase) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return database.rowToReturn
}

type fakePostgresColumnTransaction struct {
	execError         error
	commitError       error
	rollbackCalled    bool
	commitCalled      bool
	executedQueries   []string
	receivedArguments [][]interface{}
}

func (transaction *fakePostgresColumnTransaction) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	transaction.executedQueries = append(transaction.executedQueries, sql)
	transaction.receivedArguments = append(transaction.receivedArguments, arguments)
	return pgconn.CommandTag{}, transaction.execError
}

func (transaction *fakePostgresColumnTransaction) Commit(ctx context.Context) error {
	transaction.commitCalled = true
	return transaction.commitError
}

func (transaction *fakePostgresColumnTransaction) Rollback(ctx context.Context) error {
	transaction.rollbackCalled = true
	return nil
}

type fakePostgresColumnRow struct {
	values []interface{}
	err    error
}

func (row fakePostgresColumnRow) Scan(destinations ...interface{}) error {
	if row.err != nil {
		return row.err
	}

	assignColumnScanValues(destinations, row.values)
	return nil
}

type fakePostgresColumnRows struct {
	rows        []fakePostgresColumnRow
	currentRow  int
	errToReturn error
	closed      bool
}

func (rows *fakePostgresColumnRows) Close() {
	rows.closed = true
}

func (rows *fakePostgresColumnRows) Err() error {
	return rows.errToReturn
}

func (rows *fakePostgresColumnRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (rows *fakePostgresColumnRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (rows *fakePostgresColumnRows) Next() bool {
	return rows.currentRow < len(rows.rows)
}

func (rows *fakePostgresColumnRows) Scan(destinations ...interface{}) error {
	row := rows.rows[rows.currentRow]
	rows.currentRow++
	return row.Scan(destinations...)
}

func (rows *fakePostgresColumnRows) Values() ([]interface{}, error) {
	return nil, nil
}

func (rows *fakePostgresColumnRows) RawValues() [][]byte {
	return nil
}

func (rows *fakePostgresColumnRows) Conn() *pgx.Conn {
	return nil
}

func TestNewPostgresColumnRepository_NilPool_ReturnsRepository(t *testing.T) {
	// Arrange
	logger := &fakeRepositoryLogger{}

	// Act
	repository := NewPostgresColumnRepository(nil, logger)

	// Assert
	if repository == nil {
		t.Fatal("expected repository, got nil")
	}
}

func TestPostgresColumnRepository_SaveValidColumn_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresColumnDatabase{}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), createRepositoryColumn(t, "column-123", 0))

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != saveColumnQuery {
		t.Errorf("expected save column query to be used")
	}
	if len(database.receivedArguments) != 7 {
		t.Errorf("expected seven arguments, got %d", len(database.receivedArguments))
	}
}

func TestPostgresColumnRepository_SaveNilColumn_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresColumnRepository{database: &fakePostgresColumnDatabase{}, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), nil)

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func TestPostgresColumnRepository_SaveDatabaseFailure_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresColumnDatabase{execError: errors.New("database failure")}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), createRepositoryColumn(t, "column-123", 0))

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func TestPostgresColumnRepository_GetByIDExistingColumn_ReturnsColumn(t *testing.T) {
	// Arrange
	database := &fakePostgresColumnDatabase{rowToReturn: fakePostgresColumnRow{values: validStoredColumnValues("column-123", 0)}}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	column, err := repository.GetByID(context.Background(), "column-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if column.ID() != "column-123" {
		t.Errorf("expected column ID column-123, got %s", column.ID())
	}
	if database.receivedSQL != getColumnByIDQuery {
		t.Errorf("expected get column by id query to be used")
	}
}

func TestPostgresColumnRepository_GetByIDMissingColumn_ReturnsErrColumnNotFound(t *testing.T) {
	// Arrange
	database := &fakePostgresColumnDatabase{rowToReturn: fakePostgresColumnRow{err: pgx.ErrNoRows}}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByID(context.Background(), "column-123")

	// Assert
	if !errors.Is(err, ports.ErrColumnNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnNotFound, err)
	}
}

func TestPostgresColumnRepository_GetByIDCorruptedColumn_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresColumnDatabase{rowToReturn: fakePostgresColumnRow{values: corruptedStoredColumnValues()}}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByID(context.Background(), "column-123")

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func TestPostgresColumnRepository_GetByBoardIDExistingColumns_ReturnsColumns(t *testing.T) {
	// Arrange
	rows := &fakePostgresColumnRows{
		rows: []fakePostgresColumnRow{
			{values: validStoredColumnValues("column-123", 0)},
			{values: validStoredColumnValues("column-456", 1)},
		},
	}
	database := &fakePostgresColumnDatabase{rowsToReturn: rows}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	columns, err := repository.GetByBoardID(context.Background(), "board-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(columns) != 2 {
		t.Fatalf("expected two columns, got %d", len(columns))
	}
	if !rows.closed {
		t.Fatal("expected rows to be closed")
	}
}

func TestPostgresColumnRepository_GetByBoardIDQueryFailure_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresColumnDatabase{queryError: errors.New("query failure")}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByBoardID(context.Background(), "board-123")

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func TestPostgresColumnRepository_GetByBoardIDScanFailure_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	rows := &fakePostgresColumnRows{rows: []fakePostgresColumnRow{{err: errors.New("scan failure")}}}
	database := &fakePostgresColumnDatabase{rowsToReturn: rows}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByBoardID(context.Background(), "board-123")

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func TestPostgresColumnRepository_GetByBoardIDRowsError_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	rows := &fakePostgresColumnRows{errToReturn: errors.New("rows failure")}
	database := &fakePostgresColumnDatabase{rowsToReturn: rows}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByBoardID(context.Background(), "board-123")

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func TestPostgresColumnRepository_UpdateValidColumn_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresColumnDatabase{}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Update(context.Background(), createRepositoryColumn(t, "column-123", 0))

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != updateColumnQuery {
		t.Errorf("expected update column query to be used")
	}
}

func TestPostgresColumnRepository_UpdateNilColumn_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresColumnRepository{database: &fakePostgresColumnDatabase{}, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Update(context.Background(), nil)

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func TestPostgresColumnRepository_UpdateDatabaseFailure_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresColumnDatabase{execError: errors.New("database failure")}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Update(context.Background(), createRepositoryColumn(t, "column-123", 0))

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func TestPostgresColumnRepository_UpdatePositionsValidColumns_CommitsTransaction(t *testing.T) {
	// Arrange
	transaction := &fakePostgresColumnTransaction{}
	repository := &PostgresColumnRepository{
		database: &fakePostgresColumnDatabase{},
		beginTransaction: func(ctx context.Context) (postgresColumnTransaction, error) {
			return transaction, nil
		},
		logger: &fakeRepositoryLogger{},
	}
	columns := []*domain.Column{
		createRepositoryColumn(t, "column-123", 0),
		createRepositoryColumn(t, "column-456", 1),
	}

	// Act
	err := repository.UpdatePositions(context.Background(), columns)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !transaction.commitCalled {
		t.Fatal("expected transaction to be committed")
	}
	if len(transaction.executedQueries) != 2 {
		t.Fatalf("expected two executed queries, got %d", len(transaction.executedQueries))
	}
	if transaction.executedQueries[0] != updateColumnPositionQuery {
		t.Errorf("expected update column position query to be used")
	}
}

func TestPostgresColumnRepository_UpdatePositionsBeginFailure_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresColumnRepository{
		database: &fakePostgresColumnDatabase{},
		beginTransaction: func(ctx context.Context) (postgresColumnTransaction, error) {
			return nil, errors.New("begin failure")
		},
		logger: &fakeRepositoryLogger{},
	}

	// Act
	err := repository.UpdatePositions(context.Background(), []*domain.Column{createRepositoryColumn(t, "column-123", 0)})

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func TestPostgresColumnRepository_UpdatePositionsNilColumn_RollsBack(t *testing.T) {
	// Arrange
	transaction := &fakePostgresColumnTransaction{}
	repository := &PostgresColumnRepository{
		database: &fakePostgresColumnDatabase{},
		beginTransaction: func(ctx context.Context) (postgresColumnTransaction, error) {
			return transaction, nil
		},
		logger: &fakeRepositoryLogger{},
	}

	// Act
	err := repository.UpdatePositions(context.Background(), []*domain.Column{nil})

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
	if !transaction.rollbackCalled {
		t.Fatal("expected transaction to be rolled back")
	}
}

func TestPostgresColumnRepository_UpdatePositionsExecFailure_RollsBack(t *testing.T) {
	// Arrange
	transaction := &fakePostgresColumnTransaction{execError: errors.New("exec failure")}
	repository := &PostgresColumnRepository{
		database: &fakePostgresColumnDatabase{},
		beginTransaction: func(ctx context.Context) (postgresColumnTransaction, error) {
			return transaction, nil
		},
		logger: &fakeRepositoryLogger{},
	}

	// Act
	err := repository.UpdatePositions(context.Background(), []*domain.Column{createRepositoryColumn(t, "column-123", 0)})

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
	if !transaction.rollbackCalled {
		t.Fatal("expected transaction to be rolled back")
	}
}

func TestPostgresColumnRepository_UpdatePositionsCommitFailure_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	transaction := &fakePostgresColumnTransaction{commitError: errors.New("commit failure")}
	repository := &PostgresColumnRepository{
		database: &fakePostgresColumnDatabase{},
		beginTransaction: func(ctx context.Context) (postgresColumnTransaction, error) {
			return transaction, nil
		},
		logger: &fakeRepositoryLogger{},
	}

	// Act
	err := repository.UpdatePositions(context.Background(), []*domain.Column{createRepositoryColumn(t, "column-123", 0)})

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func TestPostgresColumnRepository_DeleteValidColumn_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresColumnDatabase{}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Delete(context.Background(), "column-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != deleteColumnQuery {
		t.Errorf("expected delete column query to be used")
	}
}

func TestPostgresColumnRepository_DeleteDatabaseFailure_ReturnsErrColumnRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresColumnDatabase{execError: errors.New("database failure")}
	repository := &PostgresColumnRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Delete(context.Background(), "column-123")

	// Assert
	if !errors.Is(err, ports.ErrColumnRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrColumnRepositoryUnavailable, err)
	}
}

func assignColumnScanValues(destinations []interface{}, values []interface{}) {
	for index, value := range values {
		switch destination := destinations[index].(type) {
		case *string:
			*destination = value.(string)
		case *int:
			*destination = value.(int)
		case *time.Time:
			*destination = value.(time.Time)
		}
	}
}

func validStoredColumnValues(columnID string, position int) []interface{} {
	return []interface{}{
		columnID,
		"board-123",
		"Backlog",
		"slate",
		position,
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-time.Hour),
	}
}

func corruptedStoredColumnValues() []interface{} {
	values := validStoredColumnValues("column-123", 0)
	values[4] = -1
	return values
}

func createRepositoryColumn(t *testing.T, columnID string, position int) *domain.Column {
	t.Helper()

	column, err := domain.NewColumn(columnID, "board-123", "Backlog", position)
	if err != nil {
		t.Fatalf("expected column to be valid, got: %v", err)
	}

	return column
}
