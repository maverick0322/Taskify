package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	postgresUniqueViolationCode = "23505"

	saveUserQuery = `
		INSERT INTO users (id, email, password_hash, first_name, last_name, birth_date)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	getUserByIDQuery = `
		SELECT id, email, password_hash, first_name, last_name, birth_date
		FROM users
		WHERE id = $1
	`

	getUserByEmailQuery = `
		SELECT id, email, password_hash, first_name, last_name, birth_date
		FROM users
		WHERE email = $1
	`
)

type postgresDatabase interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

// PostgresUserRepository implements the user persistence port using PostgreSQL.
type PostgresUserRepository struct {
	database postgresDatabase
	logger   ports.Logger
}

// NewPostgresUserRepository receives the concrete pool at the edge while keeping internals testable.
func NewPostgresUserRepository(pool *pgxpool.Pool, logger ports.Logger) ports.UserRepository {
	return &PostgresUserRepository{
		database: pool,
		logger:   logger,
	}
}

func (repository *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
	if user == nil {
		repository.logger.Error("cannot save nil user")
		return ports.ErrRepositoryUnavailable
	}

	profile := user.Profile()
	_, err := repository.database.Exec(
		ctx,
		saveUserQuery,
		user.ID(),
		user.Email(),
		user.PasswordHash(),
		profile.FirstName(),
		profile.LastName(),
		profile.BirthDate(),
	)
	if err == nil {
		return nil
	}

	if isUniqueViolation(err) {
		repository.logger.Warn("cannot save user because it already exists", "userID", user.ID())
		return ports.ErrUserAlreadyExists
	}

	repository.logger.Error("failed to save user", "userID", user.ID(), "error", err)
	return ports.ErrRepositoryUnavailable
}

func (repository *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := repository.scanUser(repository.database.QueryRow(ctx, getUserByIDQuery, id))
	if err == nil {
		return user, nil
	}

	return repository.mapReadError(err, "failed to retrieve user by id", "userID", id)
}

func (repository *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := repository.scanUser(repository.database.QueryRow(ctx, getUserByEmailQuery, email))
	if err == nil {
		return user, nil
	}

	return repository.mapReadError(err, "failed to retrieve user by email")
}

func (repository *PostgresUserRepository) scanUser(row pgx.Row) (*domain.User, error) {
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

func (repository *PostgresUserRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.User, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrUserNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrRepositoryUnavailable
}

type storedUser struct {
	id           string
	email        string
	passwordHash string
	firstName    string
	lastName     string
	birthDate    time.Time
}

func buildDomainUser(storedUser storedUser) (*domain.User, error) {
	profile, err := domain.NewUserProfile(storedUser.firstName, storedUser.lastName, storedUser.birthDate)
	if err != nil {
		return nil, err
	}

	user, err := domain.NewUser(storedUser.id, storedUser.email, storedUser.passwordHash, profile)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return false
	}

	return postgresError.Code == postgresUniqueViolationCode
}
