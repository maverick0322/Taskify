package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	sqliteSaveUserQuery = `
		INSERT INTO users (id, email, password_hash, first_name, last_name, birth_date, avatar_local_path, avatar_url, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	sqliteGetUserByIDQuery = `
		SELECT id, email, password_hash, first_name, last_name, birth_date, avatar_local_path, avatar_url
		FROM users
		WHERE id = ? AND deleted_at IS NULL
	`

	sqliteGetUserByEmailQuery = `
		SELECT id, email, password_hash, first_name, last_name, birth_date, avatar_local_path, avatar_url
		FROM users
		WHERE email = ? AND deleted_at IS NULL
	`

	sqliteUpdateUserAvatarLocalPathQuery = `
		UPDATE users
		SET avatar_local_path = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	sqliteUpdateUserProfileNameQuery = `
		UPDATE users
		SET first_name = ?, last_name = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	sqliteUpdateUserAvatarURLQuery = `
		UPDATE users
		SET avatar_url = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
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
		nullableString(nonEmptyStringPtr(user.AvatarLocalPath())),
		nullableString(nonEmptyStringPtr(user.AvatarURL())),
		timeValue(time.Now()),
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

func (repository *SQLiteUserRepository) UpdateProfileName(ctx context.Context, userID, firstName, lastName string) error {
	result, err := repository.database.ExecContext(
		ctx,
		sqliteUpdateUserProfileNameQuery,
		firstName,
		lastName,
		timeValue(time.Now()),
		userID,
	)
	if err != nil {
		repository.logger.Error("failed to update user profile name", "userID", userID, "error", err)
		return ports.ErrRepositoryUnavailable
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ports.ErrRepositoryUnavailable
	}
	if rowsAffected == 0 {
		return ports.ErrUserNotFound
	}
	return nil
}

func (repository *SQLiteUserRepository) UpdateAvatarLocalPath(ctx context.Context, userID, avatarLocalPath string) error {
	now := time.Now()
	result, err := repository.database.ExecContext(ctx, sqliteUpdateUserAvatarLocalPathQuery, avatarLocalPath, timeValue(now), userID)
	if err != nil {
		repository.logger.Error("failed to update user avatar local path", "userID", userID, "error", err)
		return ports.ErrRepositoryUnavailable
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ports.ErrRepositoryUnavailable
	}
	if rowsAffected == 0 {
		return ports.ErrUserNotFound
	}
	if err := repository.enqueueAvatarStorageJob(ctx, userID, avatarLocalPath, now); err != nil {
		return err
	}
	return nil
}

func (repository *SQLiteUserRepository) UpdateAvatarURL(ctx context.Context, userID, avatarURL string) error {
	result, err := repository.database.ExecContext(ctx, sqliteUpdateUserAvatarURLQuery, avatarURL, timeValue(time.Now()), userID)
	if err != nil {
		repository.logger.Error("failed to update user avatar url", "userID", userID, "error", err)
		return ports.ErrRepositoryUnavailable
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ports.ErrRepositoryUnavailable
	}
	if rowsAffected == 0 {
		return ports.ErrUserNotFound
	}
	return nil
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
		&storedUser.avatarLocalPath,
		&storedUser.avatarURL,
	); err != nil {
		return nil, err
	}

	return buildDomainUser(storedUser)
}

func (repository *SQLiteUserRepository) enqueueAvatarStorageJob(ctx context.Context, userID, avatarLocalPath string, now time.Time) error {
	_, err := repository.database.ExecContext(ctx, `
		INSERT INTO storage_sync_jobs (id, user_id, entity_type, entity_id, local_path, bucket, object_key, status, attempts, created_at, updated_at)
		VALUES (?, ?, 'user_avatar', ?, ?, 'avatars', ?, 'pending', 0, ?, ?)
		ON CONFLICT(entity_type, entity_id) DO UPDATE SET
			local_path = excluded.local_path,
			object_key = excluded.object_key,
			status = 'pending',
			updated_at = excluded.updated_at
	`, userID+"-avatar", userID, userID, avatarLocalPath, "users/"+userID+"/avatar", timeValue(now), timeValue(now))
	if err != nil {
		repository.logger.Error("failed to enqueue avatar storage sync job", "userID", userID, "error", err)
		return ports.ErrRepositoryUnavailable
	}
	return nil
}

func nonEmptyStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (repository *SQLiteUserRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.User, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrUserNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrRepositoryUnavailable
}
