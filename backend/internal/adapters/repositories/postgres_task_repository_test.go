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

type fakePostgresTaskDatabase struct {
	execError         error
	queryError        error
	rowToReturn       pgx.Row
	rowsToReturn      pgx.Rows
	receivedSQL       string
	receivedArguments []interface{}
}

func (database *fakePostgresTaskDatabase) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return pgconn.CommandTag{}, database.execError
}

func (database *fakePostgresTaskDatabase) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return database.rowsToReturn, database.queryError
}

func (database *fakePostgresTaskDatabase) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	database.receivedSQL = sql
	database.receivedArguments = arguments
	return database.rowToReturn
}

type fakePostgresTaskRow struct {
	values []interface{}
	err    error
}

func (row fakePostgresTaskRow) Scan(destinations ...interface{}) error {
	if row.err != nil {
		return row.err
	}

	assignTaskScanValues(destinations, row.values)
	return nil
}

type fakePostgresTaskRows struct {
	rows        []fakePostgresTaskRow
	currentRow  int
	errToReturn error
	closed      bool
}

func (rows *fakePostgresTaskRows) Close() {
	rows.closed = true
}

func (rows *fakePostgresTaskRows) Err() error {
	return rows.errToReturn
}

func (rows *fakePostgresTaskRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (rows *fakePostgresTaskRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (rows *fakePostgresTaskRows) Next() bool {
	return rows.currentRow < len(rows.rows)
}

func (rows *fakePostgresTaskRows) Scan(destinations ...interface{}) error {
	row := rows.rows[rows.currentRow]
	rows.currentRow++
	return row.Scan(destinations...)
}

func (rows *fakePostgresTaskRows) Values() ([]interface{}, error) {
	return nil, nil
}

func (rows *fakePostgresTaskRows) RawValues() [][]byte {
	return nil
}

func (rows *fakePostgresTaskRows) Conn() *pgx.Conn {
	return nil
}

func TestPostgresTaskRepository_SaveValidTask_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}
	task := createRepositoryTask(t, time.Now().Add(24*time.Hour))

	// Act
	err := repository.Save(context.Background(), task)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != saveTaskQuery {
		t.Errorf("expected save task query to be used")
	}
	if len(database.receivedArguments) != 10 {
		t.Errorf("expected ten arguments, got %d", len(database.receivedArguments))
	}
}

func TestPostgresTaskRepository_SaveZeroDueDate_StoresNilDueDate(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}
	task := createRepositoryTask(t, time.Time{})

	// Act
	err := repository.Save(context.Background(), task)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedArguments[7] != nil {
		t.Errorf("expected nil due date argument, got %v", database.receivedArguments[7])
	}
}

func TestNewPostgresTaskRepository_NilPool_ReturnsRepository(t *testing.T) {
	// Arrange
	logger := &fakeRepositoryLogger{}

	// Act
	repository := NewPostgresTaskRepository(nil, logger)

	// Assert
	if repository == nil {
		t.Fatal("expected repository, got nil")
	}
}

func TestPostgresTaskRepository_SaveNilTask_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresTaskRepository{database: &fakePostgresTaskDatabase{}, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), nil)

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_SaveDuplicateTask_ReturnsErrTaskAlreadyExists(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{execError: &pgconn.PgError{Code: postgresUniqueViolationCode}}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), createRepositoryTask(t, time.Time{}))

	// Assert
	if !errors.Is(err, ports.ErrTaskAlreadyExists) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskAlreadyExists, err)
	}
}

func TestPostgresTaskRepository_SaveDatabaseFailure_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{execError: errors.New("database failure")}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), createRepositoryTask(t, time.Time{}))

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_GetByIDExistingTask_ReturnsTask(t *testing.T) {
	// Arrange
	dueDate := time.Now().Add(24 * time.Hour)
	database := &fakePostgresTaskDatabase{rowToReturn: fakePostgresTaskRow{values: validStoredTaskValues(dueDate)}}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	task, err := repository.GetByID(context.Background(), "task-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if task.ID() != "task-123" {
		t.Errorf("expected task ID task-123, got %s", task.ID())
	}
	if !task.DueDate().Equal(dueDate) {
		t.Errorf("expected due date %v, got %v", dueDate, task.DueDate())
	}
}

func TestPostgresTaskRepository_GetByIDMissingTask_ReturnsErrTaskNotFound(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{rowToReturn: fakePostgresTaskRow{err: pgx.ErrNoRows}}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByID(context.Background(), "task-123")

	// Assert
	if !errors.Is(err, ports.ErrTaskNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskNotFound, err)
	}
}

func TestPostgresTaskRepository_GetByIDCorruptedTask_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{rowToReturn: fakePostgresTaskRow{values: corruptedStoredTaskValues()}}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByID(context.Background(), "task-123")

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_GetByUserIDExistingTasks_ReturnsTasks(t *testing.T) {
	// Arrange
	rows := &fakePostgresTaskRows{
		rows: []fakePostgresTaskRow{
			{values: validStoredTaskValues(time.Time{})},
			{values: validStoredTaskValues(time.Now().Add(24 * time.Hour))},
		},
	}
	database := &fakePostgresTaskDatabase{rowsToReturn: rows}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	tasks, err := repository.GetByUserID(context.Background(), "user-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected two tasks, got %d", len(tasks))
	}
	if !tasks[0].DueDate().IsZero() {
		t.Errorf("expected first task due date to be zero")
	}
	if database.receivedSQL != getTasksByUserIDQuery {
		t.Errorf("expected global user task query to be used")
	}
	if len(database.receivedArguments) != 1 {
		t.Fatalf("expected one query argument, got %d", len(database.receivedArguments))
	}
	if database.receivedArguments[0] != "user-123" {
		t.Errorf("expected user ID argument user-123, got %v", database.receivedArguments[0])
	}
	if !rows.closed {
		t.Fatal("expected rows to be closed")
	}
}

func TestPostgresTaskRepository_GetByUserIDQueryFailure_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{queryError: errors.New("query failure")}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByUserID(context.Background(), "user-123")

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_GetByUserIDScanFailure_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	rows := &fakePostgresTaskRows{rows: []fakePostgresTaskRow{{err: errors.New("scan failure")}}}
	database := &fakePostgresTaskDatabase{rowsToReturn: rows}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByUserID(context.Background(), "user-123")

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_GetByUserIDRowsError_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	rows := &fakePostgresTaskRows{errToReturn: errors.New("rows failure")}
	database := &fakePostgresTaskDatabase{rowsToReturn: rows}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByUserID(context.Background(), "user-123")

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_GetByUserIDAndBoardIDExistingTasks_ReturnsTasks(t *testing.T) {
	// Arrange
	rows := &fakePostgresTaskRows{
		rows: []fakePostgresTaskRow{
			{values: validStoredTaskValues(time.Time{})},
		},
	}
	database := &fakePostgresTaskDatabase{rowsToReturn: rows}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	tasks, err := repository.GetByUserIDAndBoardID(context.Background(), "user-123", "board-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(tasks))
	}
	if database.receivedSQL != getTasksByUserIDAndBoardIDQuery {
		t.Errorf("expected board task query to be used")
	}
	if database.receivedArguments[1] != "board-123" {
		t.Errorf("expected board ID argument board-123, got %v", database.receivedArguments[1])
	}
}

func TestPostgresTaskRepository_GetByUserIDAndBoardIDQueryFailure_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{queryError: errors.New("query failure")}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByUserIDAndBoardID(context.Background(), "user-123", "board-123")

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_GetByUserIDAndBoardIDScanFailure_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	rows := &fakePostgresTaskRows{rows: []fakePostgresTaskRow{{err: errors.New("scan failure")}}}
	database := &fakePostgresTaskDatabase{rowsToReturn: rows}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByUserIDAndBoardID(context.Background(), "user-123", "board-123")

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_GetByUserIDAndBoardIDRowsError_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	rows := &fakePostgresTaskRows{errToReturn: errors.New("rows failure")}
	database := &fakePostgresTaskDatabase{rowsToReturn: rows}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByUserIDAndBoardID(context.Background(), "user-123", "board-123")

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_UpdateValidTask_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Update(context.Background(), createRepositoryTask(t, time.Time{}))

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != updateTaskQuery {
		t.Errorf("expected update task query to be used")
	}
}

func TestPostgresTaskRepository_UpdateNilTask_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresTaskRepository{database: &fakePostgresTaskDatabase{}, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Update(context.Background(), nil)

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_UpdateDatabaseFailure_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{execError: errors.New("database failure")}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Update(context.Background(), createRepositoryTask(t, time.Time{}))

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func TestPostgresTaskRepository_DeleteValidTask_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Delete(context.Background(), "task-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != deleteTaskQuery {
		t.Errorf("expected delete task query to be used")
	}
}

func TestPostgresTaskRepository_DeleteDatabaseFailure_ReturnsErrTaskRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresTaskDatabase{execError: errors.New("database failure")}
	repository := &PostgresTaskRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Delete(context.Background(), "task-123")

	// Assert
	if !errors.Is(err, ports.ErrTaskRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskRepositoryUnavailable, err)
	}
}

func assignTaskScanValues(destinations []interface{}, values []interface{}) {
	for index, value := range values {
		switch destination := destinations[index].(type) {
		case *string:
			*destination = value.(string)
		case **time.Time:
			if value == nil {
				*destination = nil
				continue
			}
			dueDate := value.(time.Time)
			*destination = &dueDate
		case *time.Time:
			*destination = value.(time.Time)
		}
	}
}

func validStoredTaskValues(dueDate time.Time) []interface{} {
	createdAt := time.Now().Add(-2 * time.Hour)
	updatedAt := time.Now().Add(-time.Hour)
	var storedDueDate interface{}
	if !dueDate.IsZero() {
		storedDueDate = dueDate
	}

	return []interface{}{
		"task-123",
		"user-123",
		"board-123",
		"Write tests",
		"Cover repository rules",
		string(domain.TaskStatusTodo),
		string(domain.TaskPriorityMedium),
		storedDueDate,
		createdAt,
		updatedAt,
	}
}

func corruptedStoredTaskValues() []interface{} {
	values := validStoredTaskValues(time.Time{})
	values[5] = "blocked"
	return values
}

func createRepositoryTask(t *testing.T, dueDate time.Time) *domain.Task {
	t.Helper()

	task, err := domain.NewTask(
		"task-123",
		"user-123",
		"board-123",
		"Write tests",
		"Cover repository rules",
		domain.TaskStatusTodo,
		domain.TaskPriorityMedium,
		dueDate,
	)
	if err != nil {
		t.Fatalf("expected task to be valid, got: %v", err)
	}

	return task
}
