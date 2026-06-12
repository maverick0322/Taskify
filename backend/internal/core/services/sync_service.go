package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const syncStateKey = "bidirectional_sync"

type SyncDialect string

const (
	SyncDialectSQLite   SyncDialect = "sqlite"
	SyncDialectPostgres SyncDialect = "postgres"
)

type SyncService struct {
	local         *sql.DB
	remote        *sql.DB
	remoteDialect SyncDialect
	logger        ports.Logger
	now           func() time.Time
}

func NewSyncService(local, remote *sql.DB, remoteDialect SyncDialect, logger ports.Logger) *SyncService {
	return &SyncService{
		local:         local,
		remote:        remote,
		remoteDialect: remoteDialect,
		logger:        logger,
		now:           time.Now,
	}
}

func (service *SyncService) SyncOnce(ctx context.Context) error {
	if service == nil || service.local == nil || service.remote == nil {
		return errors.New("sync: databases are required")
	}

	lastSyncAt, err := service.lastSuccessfulSyncAt(ctx)
	if err != nil {
		return err
	}
	cycleSyncAt := service.now().UTC()

	for _, table := range syncTableSpecs() {
		if err := service.syncTable(ctx, table, service.local, service.remote, SyncDialectSQLite, service.remoteDialect, lastSyncAt, cycleSyncAt); err != nil {
			return fmt.Errorf("sync push %s: %w", table.name, err)
		}
	}

	for _, table := range syncTableSpecs() {
		if err := service.syncTable(ctx, table, service.remote, service.local, service.remoteDialect, SyncDialectSQLite, lastSyncAt, cycleSyncAt); err != nil {
			return fmt.Errorf("sync pull %s: %w", table.name, err)
		}
	}

	if err := service.saveLastSuccessfulSyncAt(ctx, cycleSyncAt); err != nil {
		return err
	}

	return nil
}

func (service *SyncService) syncTable(ctx context.Context, table syncTableSpec, source, destination *sql.DB, sourceDialect, destinationDialect SyncDialect, from, to time.Time) error {
	rows, err := source.QueryContext(ctx, incrementalSelectSQL(table, sourceDialect), dialectTimeValue(sourceDialect, from), dialectTimeValue(sourceDialect, to))
	if err != nil {
		return err
	}
	defer rows.Close()

	upsertSQL := lwwUpsertSQL(table, destinationDialect)
	for rows.Next() {
		values, err := scanSyncRow(rows, len(table.columns))
		if err != nil {
			return err
		}
		if _, err := destination.ExecContext(ctx, upsertSQL, values...); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func (service *SyncService) lastSuccessfulSyncAt(ctx context.Context) (time.Time, error) {
	var lastSyncAt time.Time
	err := service.local.QueryRowContext(ctx, "SELECT last_successful_sync_at FROM sync_state WHERE key = ?", syncStateKey).Scan(&lastSyncAt)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Unix(0, 0).UTC(), nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("sync: failed to read local sync state: %w", err)
	}

	return lastSyncAt.UTC(), nil
}

func (service *SyncService) saveLastSuccessfulSyncAt(ctx context.Context, syncedAt time.Time) error {
	_, err := service.local.ExecContext(
		ctx,
		`INSERT INTO sync_state (key, last_successful_sync_at, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
			last_successful_sync_at = excluded.last_successful_sync_at,
			updated_at = excluded.updated_at`,
		syncStateKey,
		timeValue(syncedAt),
		timeValue(syncedAt),
	)
	if err != nil {
		return fmt.Errorf("sync: failed to save local sync state: %w", err)
	}

	return nil
}

type syncTableSpec struct {
	name    string
	columns []string
}

func syncTableSpecs() []syncTableSpec {
	return []syncTableSpec{
		{name: "users", columns: []string{"id", "email", "password_hash", "first_name", "last_name", "birth_date", "created_at", "updated_at", "deleted_at"}},
		{name: "boards", columns: []string{"id", "user_id", "name", "created_at", "updated_at", "deleted_at"}},
		{name: "columns", columns: []string{"id", "board_id", "name", "color", "position", "created_at", "updated_at", "deleted_at"}},
		{name: "tasks", columns: []string{"id", "user_id", "board_id", "column_id", "title", "description", "status", "priority", "due_date", "created_at", "updated_at", "deleted_at"}},
		{name: "credit_cards", columns: []string{"id", "user_id", "name", "bank", "last4", "cutoff_day", "payment_day", "limit_cents", "color", "created_at", "updated_at", "deleted_at"}},
		{name: "transactions", columns: []string{"id", "user_id", "credit_card_id", "type", "concept", "category", "amount_cents", "date", "status", "msi", "created_at", "updated_at", "deleted_at"}},
	}
}

func incrementalSelectSQL(table syncTableSpec, dialect SyncDialect) string {
	return fmt.Sprintf(
		"SELECT %s FROM %s WHERE updated_at > %s AND updated_at <= %s ORDER BY updated_at ASC",
		strings.Join(table.columns, ", "),
		table.name,
		placeholder(dialect, 1),
		placeholder(dialect, 2),
	)
}

func lwwUpsertSQL(table syncTableSpec, dialect SyncDialect) string {
	placeholders := make([]string, 0, len(table.columns))
	assignments := make([]string, 0, len(table.columns)-1)
	for index, column := range table.columns {
		placeholders = append(placeholders, placeholder(dialect, index+1))
		if column != "id" {
			assignments = append(assignments, fmt.Sprintf("%s = excluded.%s", column, column))
		}
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(id) DO UPDATE SET %s WHERE excluded.updated_at > %s.updated_at",
		table.name,
		strings.Join(table.columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(assignments, ", "),
		table.name,
	)
}

func placeholder(dialect SyncDialect, position int) string {
	if dialect == SyncDialectPostgres {
		return fmt.Sprintf("$%d", position)
	}

	return "?"
}

func dialectTimeValue(dialect SyncDialect, value time.Time) interface{} {
	if dialect == SyncDialectPostgres {
		return value.UTC()
	}

	return timeValue(value)
}

func scanSyncRow(rows *sql.Rows, count int) ([]interface{}, error) {
	values := make([]interface{}, count)
	destinations := make([]interface{}, count)
	for index := range values {
		destinations[index] = &values[index]
	}

	if err := rows.Scan(destinations...); err != nil {
		return nil, err
	}

	for index, value := range values {
		if bytesValue, ok := value.([]byte); ok {
			values[index] = string(bytesValue)
		}
	}

	return values, nil
}

func timeValue(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
