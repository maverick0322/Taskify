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

type fakePostgresSessionDatabase struct {
	execError       error
	rowToReturn     pgx.Row
	receivedSQL     string
	executedQueries []string
}

func (database *fakePostgresSessionDatabase) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	database.receivedSQL = sql
	database.executedQueries = append(database.executedQueries, sql)
	return pgconn.CommandTag{}, database.execError
}

func (database *fakePostgresSessionDatabase) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	database.receivedSQL = sql
	return database.rowToReturn
}

type fakePostgresSessionTransaction struct {
	execError       error
	execErrors      []error
	commitError     error
	rollbackCalled  bool
	commitCalled    bool
	executedQueries []string
}

func (transaction *fakePostgresSessionTransaction) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	transaction.executedQueries = append(transaction.executedQueries, sql)
	if len(transaction.execErrors) > 0 {
		err := transaction.execErrors[0]
		transaction.execErrors = transaction.execErrors[1:]
		return pgconn.CommandTag{}, err
	}
	return pgconn.CommandTag{}, transaction.execError
}

func (transaction *fakePostgresSessionTransaction) Commit(ctx context.Context) error {
	transaction.commitCalled = true
	return transaction.commitError
}

func (transaction *fakePostgresSessionTransaction) Rollback(ctx context.Context) error {
	transaction.rollbackCalled = true
	return nil
}

type fakePostgresSessionRow struct {
	values []interface{}
	err    error
}

func (row fakePostgresSessionRow) Scan(destinations ...interface{}) error {
	if row.err != nil {
		return row.err
	}

	for index, value := range row.values {
		switch destination := destinations[index].(type) {
		case *string:
			*destination = value.(string)
		case *time.Time:
			*destination = value.(time.Time)
		case *bool:
			*destination = value.(bool)
		}
	}

	return nil
}

func TestPostgresSessionRepository_SaveValidSession_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresSessionDatabase{}
	repository := &PostgresSessionRepository{database: database, logger: &fakeRepositoryLogger{}}
	refreshToken := createRepositoryRefreshToken(t)

	// Act
	err := repository.Save(context.Background(), refreshToken)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != saveRefreshTokenQuery {
		t.Errorf("expected save refresh token query to be used")
	}
}

func TestNewPostgresSessionRepository_NilPool_ReturnsRepository(t *testing.T) {
	// Arrange
	logger := &fakeRepositoryLogger{}

	// Act
	repository := NewPostgresSessionRepository(nil, logger)

	// Assert
	if repository == nil {
		t.Fatal("expected repository, got nil")
	}
}

func TestPostgresSessionRepository_SaveNilSession_ReturnsErrSessionRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresSessionRepository{database: &fakePostgresSessionDatabase{}, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), nil)

	// Assert
	if !errors.Is(err, ports.ErrSessionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionRepositoryUnavailable, err)
	}
}

func TestPostgresSessionRepository_SaveDuplicateSession_ReturnsErrSessionAlreadyExists(t *testing.T) {
	// Arrange
	database := &fakePostgresSessionDatabase{execError: &pgconn.PgError{Code: postgresUniqueViolationCode}}
	repository := &PostgresSessionRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), createRepositoryRefreshToken(t))

	// Assert
	if !errors.Is(err, ports.ErrSessionAlreadyExists) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionAlreadyExists, err)
	}
}

func TestPostgresSessionRepository_SaveDatabaseError_ReturnsErrSessionRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresSessionDatabase{execError: errors.New("database failure")}
	repository := &PostgresSessionRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Save(context.Background(), createRepositoryRefreshToken(t))

	// Assert
	if !errors.Is(err, ports.ErrSessionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionRepositoryUnavailable, err)
	}
}

func TestPostgresSessionRepository_GetByTokenHashExistingSession_ReturnsSession(t *testing.T) {
	// Arrange
	expiresAt := time.Now().Add(time.Hour)
	database := &fakePostgresSessionDatabase{
		rowToReturn: fakePostgresSessionRow{
			values: []interface{}{"session-123", "user-123", "token-hash", expiresAt, false},
		},
	}
	repository := &PostgresSessionRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	refreshToken, err := repository.GetByTokenHash(context.Background(), "token-hash")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if refreshToken.ID() != "session-123" {
		t.Errorf("expected session ID session-123, got %s", refreshToken.ID())
	}
	if database.receivedSQL != getRefreshTokenByHashQuery {
		t.Errorf("expected get refresh token query to be used")
	}
}

func TestPostgresSessionRepository_GetByTokenHashMissingSession_ReturnsErrSessionNotFound(t *testing.T) {
	// Arrange
	database := &fakePostgresSessionDatabase{rowToReturn: fakePostgresSessionRow{err: pgx.ErrNoRows}}
	repository := &PostgresSessionRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByTokenHash(context.Background(), "token-hash")

	// Assert
	if !errors.Is(err, ports.ErrSessionNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionNotFound, err)
	}
}

func TestPostgresSessionRepository_GetByTokenHashCorruptedSession_ReturnsErrSessionRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresSessionDatabase{
		rowToReturn: fakePostgresSessionRow{
			values: []interface{}{"", "user-123", "token-hash", time.Now().Add(time.Hour), false},
		},
	}
	repository := &PostgresSessionRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	_, err := repository.GetByTokenHash(context.Background(), "token-hash")

	// Assert
	if !errors.Is(err, ports.ErrSessionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionRepositoryUnavailable, err)
	}
}

func TestPostgresSessionRepository_RevokeValidSession_ReturnsNil(t *testing.T) {
	// Arrange
	database := &fakePostgresSessionDatabase{}
	repository := &PostgresSessionRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Revoke(context.Background(), "session-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if database.receivedSQL != revokeRefreshTokenQuery {
		t.Errorf("expected revoke refresh token query to be used")
	}
}

func TestPostgresSessionRepository_RevokeDatabaseError_ReturnsErrSessionRepositoryUnavailable(t *testing.T) {
	// Arrange
	database := &fakePostgresSessionDatabase{execError: errors.New("database failure")}
	repository := &PostgresSessionRepository{database: database, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Revoke(context.Background(), "session-123")

	// Assert
	if !errors.Is(err, ports.ErrSessionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionRepositoryUnavailable, err)
	}
}

func TestPostgresSessionRepository_RotateNilSession_ReturnsErrSessionRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresSessionRepository{database: &fakePostgresSessionDatabase{}, logger: &fakeRepositoryLogger{}}

	// Act
	err := repository.Rotate(context.Background(), "session-123", nil)

	// Assert
	if !errors.Is(err, ports.ErrSessionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionRepositoryUnavailable, err)
	}
}

func TestPostgresSessionRepository_RotateValidSession_CommitsTransaction(t *testing.T) {
	// Arrange
	transaction := &fakePostgresSessionTransaction{}
	repository := &PostgresSessionRepository{
		database: &fakePostgresSessionDatabase{},
		beginTransaction: func(ctx context.Context) (postgresSessionTransaction, error) {
			return transaction, nil
		},
		logger: &fakeRepositoryLogger{},
	}

	// Act
	err := repository.Rotate(context.Background(), "session-123", createRepositoryRefreshToken(t))

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
}

func TestPostgresSessionRepository_RotateRevokeFailure_ReturnsErrSessionRepositoryUnavailable(t *testing.T) {
	// Arrange
	transaction := &fakePostgresSessionTransaction{execError: errors.New("revoke failure")}
	repository := &PostgresSessionRepository{
		database: &fakePostgresSessionDatabase{},
		beginTransaction: func(ctx context.Context) (postgresSessionTransaction, error) {
			return transaction, nil
		},
		logger: &fakeRepositoryLogger{},
	}

	// Act
	err := repository.Rotate(context.Background(), "session-123", createRepositoryRefreshToken(t))

	// Assert
	if !errors.Is(err, ports.ErrSessionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionRepositoryUnavailable, err)
	}
}

func TestPostgresSessionRepository_RotateSaveDuplicate_ReturnsErrSessionAlreadyExists(t *testing.T) {
	// Arrange
	transaction := &fakePostgresSessionTransaction{execErrors: []error{nil, &pgconn.PgError{Code: postgresUniqueViolationCode}}}
	repository := &PostgresSessionRepository{
		database: &fakePostgresSessionDatabase{},
		beginTransaction: func(ctx context.Context) (postgresSessionTransaction, error) {
			return transaction, nil
		},
		logger: &fakeRepositoryLogger{},
	}

	// Act
	err := repository.Rotate(context.Background(), "session-123", createRepositoryRefreshToken(t))

	// Assert
	if !errors.Is(err, ports.ErrSessionAlreadyExists) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionAlreadyExists, err)
	}
}

func TestPostgresSessionRepository_RotateBeginFailure_ReturnsErrSessionRepositoryUnavailable(t *testing.T) {
	// Arrange
	repository := &PostgresSessionRepository{
		database: &fakePostgresSessionDatabase{},
		beginTransaction: func(ctx context.Context) (postgresSessionTransaction, error) {
			return nil, errors.New("begin failure")
		},
		logger: &fakeRepositoryLogger{},
	}

	// Act
	err := repository.Rotate(context.Background(), "session-123", createRepositoryRefreshToken(t))

	// Assert
	if !errors.Is(err, ports.ErrSessionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionRepositoryUnavailable, err)
	}
}

func TestPostgresSessionRepository_RotateCommitFailure_ReturnsErrSessionRepositoryUnavailable(t *testing.T) {
	// Arrange
	transaction := &fakePostgresSessionTransaction{commitError: errors.New("commit failure")}
	repository := &PostgresSessionRepository{
		database: &fakePostgresSessionDatabase{},
		beginTransaction: func(ctx context.Context) (postgresSessionTransaction, error) {
			return transaction, nil
		},
		logger: &fakeRepositoryLogger{},
	}

	// Act
	err := repository.Rotate(context.Background(), "session-123", createRepositoryRefreshToken(t))

	// Assert
	if !errors.Is(err, ports.ErrSessionRepositoryUnavailable) {
		t.Errorf("expected error %v, got %v", ports.ErrSessionRepositoryUnavailable, err)
	}
}

func createRepositoryRefreshToken(t *testing.T) *domain.RefreshToken {
	t.Helper()

	refreshToken, err := domain.NewRefreshToken("session-123", "user-123", "token-hash", time.Now().Add(time.Hour), false)
	if err != nil {
		t.Fatalf("expected refresh token to be valid, got: %v", err)
	}

	return refreshToken
}
