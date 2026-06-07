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

type fakePostgresDatabase struct {
	execError        error
	rowToReturn      pgx.Row
	receivedSQL      string
	receivedArgument []interface{}
}

func (database *fakePostgresDatabase) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	database.receivedSQL = sql
	database.receivedArgument = arguments
	return pgconn.CommandTag{}, database.execError
}

func (database *fakePostgresDatabase) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	database.receivedSQL = sql
	database.receivedArgument = arguments
	return database.rowToReturn
}

type fakePostgresRow struct {
	values []interface{}
	err    error
}

func (row fakePostgresRow) Scan(destinations ...interface{}) error {
	if row.err != nil {
		return row.err
	}

	for index, value := range row.values {
		switch destination := destinations[index].(type) {
		case *string:
			*destination = value.(string)
		case *time.Time:
			*destination = value.(time.Time)
		}
	}

	return nil
}

type fakeRepositoryLogger struct {
	errorMessages []string
	warnMessages  []string
}

func (logger *fakeRepositoryLogger) Info(msg string, keysAndValues ...interface{}) {}

func (logger *fakeRepositoryLogger) Warn(msg string, keysAndValues ...interface{}) {
	logger.warnMessages = append(logger.warnMessages, msg)
}

func (logger *fakeRepositoryLogger) Error(msg string, keysAndValues ...interface{}) {
	logger.errorMessages = append(logger.errorMessages, msg)
}

func TestPostgresUserRepository_SaveValidUser_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresDatabase{}
	repository := &PostgresUserRepository{database: database, logger: &fakeRepositoryLogger{}}
	user := createRepositoryTestUser(t)

	// Act
	err := repository.Save(context.Background(), user)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != saveUserQuery {
		t.Errorf("expected save query to be used")
	}
	if len(database.receivedArgument) != 6 {
		t.Errorf("expected six arguments, got %d", len(database.receivedArgument))
	}
}

func TestPostgresUserRepository_SaveNilUser_ReturnsErrRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresUserRepository{database: &fakePostgresDatabase{}, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), nil)

	// Assert
	if !errors.Is(err, ports.ErrRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrRepositoryUnavailable, err)
	}
}

func TestPostgresUserRepository_SaveDuplicateEmail_ReturnsErrUserAlreadyExists(t *testing.T) {
	// Arrange
	database := &fakePostgresDatabase{
		execError: &pgconn.PgError{Code: postgresUniqueViolationCode},
	}
	repository := &PostgresUserRepository{database: database, logger: &fakeRepositoryLogger{}}
	user := createRepositoryTestUser(t)

	// Act
	err := repository.Save(context.Background(), user)

	// Assert
	if !errors.Is(err, ports.ErrUserAlreadyExists) {
		t.Errorf("expected error %v, got %v", ports.ErrUserAlreadyExists, err)
	}
}

func TestPostgresUserRepository_SaveDatabaseError_ReturnsErrRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresDatabase{execError: errors.New("database unavailable")}
	repository := &PostgresUserRepository{database: database, logger: &fakeRepositoryLogger{}}
	user := createRepositoryTestUser(t)

	// Act
	err := repository.Save(context.Background(), user)

	// Assert
	if !errors.Is(err, ports.ErrRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrRepositoryUnavailable, err)
	}
}

func TestPostgresUserRepository_GetByIDExistingUser_ReturnsUser(t *testing.T) {
	// Arrange
	birthDate := time.Now().AddDate(-25, 0, 0)
	database := &fakePostgresDatabase{
		rowToReturn: fakePostgresRow{
			values: []interface{}{"user-123", "test@domain.com", "hashedPassword", "John", "Doe", birthDate},
		},
	}
	repository := &PostgresUserRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	user, err := repository.GetByID(context.Background(), "user-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if user.ID() != "user-123" {
		t.Errorf("expected user ID user-123, got %s", user.ID())
	}
	if database.receivedSQL != getUserByIDQuery {
		t.Errorf("expected get by id query to be used")
	}
}

func TestPostgresUserRepository_GetByEmailMissingUser_ReturnsErrUserNotFound(t *testing.T) {
	// Arrange
	database := &fakePostgresDatabase{rowToReturn: fakePostgresRow{err: pgx.ErrNoRows}}
	repository := &PostgresUserRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByEmail(context.Background(), "test@domain.com")

	// Assert
	if !errors.Is(err, ports.ErrUserNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrUserNotFound, err)
	}
}

func TestPostgresUserRepository_GetByEmailExistingUser_ReturnsUser(t *testing.T) {
	// Arrange
	birthDate := time.Now().AddDate(-25, 0, 0)
	database := &fakePostgresDatabase{
		rowToReturn: fakePostgresRow{
			values: []interface{}{"user-123", "test@domain.com", "hashedPassword", "John", "Doe", birthDate},
		},
	}
	repository := &PostgresUserRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	user, err := repository.GetByEmail(context.Background(), "test@domain.com")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if user.Email() != "test@domain.com" {
		t.Errorf("expected email test@domain.com, got %s", user.Email())
	}
	if database.receivedSQL != getUserByEmailQuery {
		t.Errorf("expected get by email query to be used")
	}
}

func TestPostgresUserRepository_GetByIDDatabaseError_ReturnsErrRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresDatabase{rowToReturn: fakePostgresRow{err: errors.New("database failure")}}
	repository := &PostgresUserRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByID(context.Background(), "user-123")

	// Assert
	if !errors.Is(err, ports.ErrRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrRepositoryUnavailable, err)
	}
}

func TestPostgresUserRepository_GetByEmailCorruptedUser_ReturnsErrRepositoryUnavailable(t *testing.T) {
	// Arrange
	birthDate := time.Now().AddDate(-25, 0, 0)
	database := &fakePostgresDatabase{
		rowToReturn: fakePostgresRow{
			values: []interface{}{"user-123", "invalid-email", "hashedPassword", "John", "Doe", birthDate},
		},
	}
	repository := &PostgresUserRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByEmail(context.Background(), "test@domain.com")

	// Assert
	if !errors.Is(err, ports.ErrRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrRepositoryUnavailable, err)
	}
}

func createRepositoryTestUser(t *testing.T) *domain.User {
	t.Helper()

	profile, err := domain.NewUserProfile("John", "Doe", time.Now().AddDate(-25, 0, 0))
	if err != nil {
		t.Fatalf("expected profile to be valid, got: %v", err)
	}

	user, err := domain.NewUser("user-123", "test@domain.com", "hashedPassword", profile)
	if err != nil {
		t.Fatalf("expected user to be valid, got: %v", err)
	}

	return user
}
