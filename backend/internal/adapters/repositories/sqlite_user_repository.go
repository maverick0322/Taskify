package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	sqliteSaveUserQuery = `
		INSERT INTO users (id, email, password_hash, first_name, last_name, birth_date)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	sqliteGetUserByIDQuery = `
		SELECT id, email, password_hash, first_name, last_name, birth_date
		FROM users
		WHERE id = ?
	`

	sqliteGetUserByEmailQuery = `
		SELECT id, email, password_hash, first_name, last_name, birth_date
		FROM users
		WHERE email = ?
	`
)

type SQLiteUserRepository struct {
	database *sql.DB
	logger   ports.Logger
}

func NewSQLiteUserRepository(database *sql.DB, logger ports.Logger) ports.UserRepository {
	return &SQLiteUserRepository{database: database, logger: logger}
}

func (repository *SQLiteUserRepository) Save(ctx context.Context, user *domain.User) error {
	if user == nil {
		repository.logger.Error("cannot save nil user")
		return ports.ErrRepositoryUnavailable
	}

	profile := user.Profile()
	_, err := repository.database.ExecContext(
		ctx,
		sqliteSaveUserQuery,
		user.ID(),
		user.Email(),
		user.PasswordHash(),
		profile.FirstName(),
		profile.LastName(),
		timeValue(profile.BirthDate()),
	)
	if err == nil {
		return nil
	}
	if isSQLiteConstraintViolation(err) {
		repository.logger.Warn("cannot save user because it already exists", "userID", user.ID())
		return ports.ErrUserAlreadyExists
	}

	repository.logger.Error("failed to save user", "userID", user.ID(), "error", err)
	return ports.ErrRepositoryUnavailable
}

func (repository *SQLiteUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := repository.scanUser(repository.database.QueryRowContext(ctx, sqliteGetUserByIDQuery, id))
	if err == nil {
		return user, nil
	}

	return repository.mapReadError(err, "failed to retrieve user by id", "userID", id)
}

func (repository *SQLiteUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := repository.scanUser(repository.database.QueryRowContext(ctx, sqliteGetUserByEmailQuery, email))
	if err == nil {
		return user, nil
	}

	return repository.mapReadError(err, "failed to retrieve user by email")
}

func (repository *SQLiteUserRepository) scanUser(row interface {
	Scan(dest ...interface{}) error
}) (*domain.User, error) {
	var storedUser storedUser
	if err := row.Scan(
		&storedUser.id,
		&storedUser.email,
		&storedUser.passwordHash,
		&storedUser.firstName,
		&storedUser.lastName,
		&storedUser.birthDate,
	); err != nil {
		return nil, err
	}

	return buildDomainUser(storedUser)
}

func (repository *SQLiteUserRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.User, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrUserNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrRepositoryUnavailable
}
