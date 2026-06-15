package repositories

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlitelib "modernc.org/sqlite/lib"
)

func isSQLiteConstraintViolation(err error) bool {
	var sqliteError *sqlite.Error
	if !errors.As(err, &sqliteError) {
		return false
	}

	code := sqliteError.Code()
	return code == sqlitelib.SQLITE_CONSTRAINT ||
		code == sqlitelib.SQLITE_CONSTRAINT_PRIMARYKEY ||
		code == sqlitelib.SQLITE_CONSTRAINT_UNIQUE
}

func nullableString(value *string) interface{} {
	if value == nil {
		return nil
	}

	return *value
}

func nullableInt(value *int) interface{} {
	if value == nil {
		return nil
	}

	return *value
}

func nullableInt64(value *int64) interface{} {
	if value == nil {
		return nil
	}

	return *value
}

func nullableTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}

	return value.UTC().Format(time.RFC3339Nano)
}

func nullableTimePtr(value *time.Time) interface{} {
	if value == nil || value.IsZero() {
		return nil
	}

	return value.UTC().Format(time.RFC3339Nano)
}

func timeValue(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func scanNullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	normalized := strings.TrimSpace(value.String)
	if normalized == "" {
		return nil
	}

	return &normalized
}

func scanNullableInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}

	normalized := int(value.Int64)
	return &normalized
}

func scanNullableInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}

	normalized := value.Int64
	return &normalized
}

func scanNullableTime(value sql.NullTime) time.Time {
	if !value.Valid {
		return time.Time{}
	}

	return value.Time
}

func scanNullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	normalized := value.Time.UTC()
	return &normalized
}
