package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	syncStateKey           = "bidirectional_sync"
	remotePullSyncStateKey = "remote_pull"
	syncOutboxBatchSize    = 100
)

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
	mutex         sync.Mutex
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
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if err := service.pushPendingOutbox(ctx); err != nil {
		return err
	}

	lastSyncAt, err := service.lastRemotePullSyncAt(ctx)
	if err != nil {
		return err
	}
	cycleSyncAt := service.now().UTC()

	for _, table := range syncTableSpecs() {
		if err := service.pullTable(ctx, table, lastSyncAt, cycleSyncAt); err != nil {
			return fmt.Errorf("sync pull %s: %w", table.name, err)
		}
	}

	if err := service.saveSyncState(ctx, remotePullSyncStateKey, cycleSyncAt); err != nil {
		return err
	}

	return nil
}

func (service *SyncService) pushPendingOutbox(ctx context.Context) error {
	rows, err := service.local.QueryContext(
		ctx,
		`SELECT id, table_name, entity_id
		 FROM sync_outbox
		 WHERE status = 'pending'
		   AND (next_attempt_at IS NULL OR next_attempt_at <= ?)
		 ORDER BY updated_at ASC
		 LIMIT ?`,
		timeValue(service.now().UTC()),
		syncOutboxBatchSize,
	)
	if err != nil {
		return fmt.Errorf("sync: failed to read outbox: %w", err)
	}

	entries := make([]syncOutboxEntry, 0)
	for rows.Next() {
		var entry syncOutboxEntry
		if err := rows.Scan(&entry.id, &entry.tableName, &entry.entityID); err != nil {
			return err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, entry := range entries {
		if err := service.pushOutboxEntry(ctx, entry); err != nil {
			if markErr := service.markOutboxEntryFailed(ctx, entry.id, err); markErr != nil {
				return fmt.Errorf("sync push %s.%s failed and could not update outbox: %v: %w", entry.tableName, entry.entityID, err, markErr)
			}
			return fmt.Errorf("sync push %s.%s: %w", entry.tableName, entry.entityID, err)
		}
	}

	return nil
}

func (service *SyncService) pushOutboxEntry(ctx context.Context, entry syncOutboxEntry) error {
	table, ok := syncTableSpecByName(entry.tableName)
	if !ok {
		return fmt.Errorf("unknown sync table %q", entry.tableName)
	}

	values, err := service.localRowValues(ctx, table, entry.entityID)
	if err != nil {
		return err
	}

	if _, err := service.remote.ExecContext(ctx, lwwUpsertSQL(table, service.remoteDialect), values...); err != nil {
		return err
	}

	if _, err := service.local.ExecContext(ctx, "DELETE FROM sync_outbox WHERE id = ?", entry.id); err != nil {
		return fmt.Errorf("sync: failed to clear outbox entry: %w", err)
	}

	return nil
}

func (service *SyncService) localRowValues(ctx context.Context, table syncTableSpec, entityID string) ([]interface{}, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", strings.Join(table.columns, ", "), table.name)
	row := service.local.QueryRowContext(ctx, query, entityID)
	values, err := scanSyncRow(row, len(table.columns))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sync: local row %s.%s not found", table.name, entityID)
	}
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (service *SyncService) markOutboxEntryFailed(ctx context.Context, entryID string, failure error) error {
	now := service.now().UTC()
	nextAttemptAt := now.Add(time.Minute)
	_, err := service.local.ExecContext(
		ctx,
		`UPDATE sync_outbox
		 SET status = 'pending',
		     attempts = attempts + 1,
		     last_error = ?,
		     next_attempt_at = ?,
		     updated_at = ?
		 WHERE id = ?`,
		failure.Error(),
		timeValue(nextAttemptAt),
		timeValue(now),
		entryID,
	)
	return err
}

func (service *SyncService) pullTable(ctx context.Context, table syncTableSpec, from, to time.Time) error {
	rows, err := service.remote.QueryContext(ctx, incrementalSelectSQL(table, service.remoteDialect), dialectTimeValue(service.remoteDialect, from), dialectTimeValue(service.remoteDialect, to))
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := service.local.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setOutboxSuppression(ctx, tx, true); err != nil {
		return err
	}

	upsertSQL := lwwUpsertSQL(table, SyncDialectSQLite)
	for rows.Next() {
		values, err := scanSyncRow(rows, len(table.columns))
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, upsertSQL, values...); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if err := setOutboxSuppression(ctx, tx, false); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func setOutboxSuppression(ctx context.Context, tx *sql.Tx, enabled bool) error {
	value := "0"
	if enabled {
		value = "1"
	}
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO sync_runtime_flags (key, value)
		 VALUES ('suppress_outbox', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		value,
	)
	return err
}

func (service *SyncService) lastRemotePullSyncAt(ctx context.Context) (time.Time, error) {
	lastSyncAt, err := service.syncState(ctx, remotePullSyncStateKey)
	if err == nil {
		return lastSyncAt, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, err
	}

	lastSyncAt, err = service.syncState(ctx, syncStateKey)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Unix(0, 0).UTC(), nil
	}
	return lastSyncAt, err
}

func (service *SyncService) syncState(ctx context.Context, key string) (time.Time, error) {
	var lastSyncAt time.Time
	err := service.local.QueryRowContext(ctx, "SELECT last_successful_sync_at FROM sync_state WHERE key = ?", key).Scan(&lastSyncAt)
	if err != nil {
		return time.Time{}, err
	}

	return lastSyncAt.UTC(), nil
}

func (service *SyncService) saveSyncState(ctx context.Context, key string, syncedAt time.Time) error {
	_, err := service.local.ExecContext(
		ctx,
		`INSERT INTO sync_state (key, last_successful_sync_at, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
			last_successful_sync_at = excluded.last_successful_sync_at,
			updated_at = excluded.updated_at`,
		key,
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

type syncOutboxEntry struct {
	id        string
	tableName string
	entityID  string
}

func syncTableSpecs() []syncTableSpec {
	return []syncTableSpec{
		{name: "users", columns: []string{"id", "email", "password_hash", "first_name", "last_name", "birth_date", "avatar_local_path", "avatar_url", "created_at", "updated_at", "deleted_at"}},
		{name: "boards", columns: []string{"id", "user_id", "name", "created_at", "updated_at", "deleted_at"}},
		{name: "columns", columns: []string{"id", "board_id", "name", "color", "position", "created_at", "updated_at", "deleted_at"}},
		{name: "tasks", columns: []string{"id", "user_id", "board_id", "column_id", "title", "description", "status", "priority", "due_date", "created_at", "updated_at", "deleted_at"}},
		{name: "financial_accounts", columns: []string{"id", "user_id", "type", "name", "institution", "last4", "opening_balance_cents", "current_balance_cents", "credit_limit_cents", "cutoff_day", "payment_day", "color", "created_at", "updated_at", "deleted_at"}},
		{name: "transactions", columns: []string{"id", "user_id", "credit_card_id", "payment_account_id", "destination_account_id", "type", "concept", "category", "amount_cents", "date", "status", "msi", "installment_number", "installment_count", "recurrence", "recurrence_limit", "last_paid_at", "created_at", "updated_at", "deleted_at"}},
		{name: "ledger_entries", columns: []string{"id", "user_id", "account_id", "transaction_id", "amount_cents", "entry_type", "created_at", "updated_at", "deleted_at"}},
		{name: "credit_card_statements", columns: []string{"id", "user_id", "credit_account_id", "cycle_start", "cycle_end", "payment_due_date", "statement_amount_cents", "paid_amount_cents", "status", "created_at", "updated_at", "deleted_at"}},
		{name: "account_payable_payments", columns: []string{"id", "account_payable_id", "user_id", "due_date", "paid_at", "amount_cents", "concept", "category", "created_transaction_id", "created_at", "updated_at", "deleted_at"}},
		{name: "notifications", columns: []string{"id", "user_id", "title", "message", "is_read", "created_at", "updated_at", "deleted_at"}},
	}
}

func syncTableSpecByName(name string) (syncTableSpec, bool) {
	for _, table := range syncTableSpecs() {
		if table.name == name {
			return table, true
		}
	}
	return syncTableSpec{}, false
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

type syncScanner interface {
	Scan(dest ...interface{}) error
}

func scanSyncRow(rows syncScanner, count int) ([]interface{}, error) {
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
