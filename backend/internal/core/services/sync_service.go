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
	eventHub      *SyncEventHub
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

func (service *SyncService) SetEventHub(eventHub *SyncEventHub) {
	service.eventHub = eventHub
}

func (service *SyncService) EventHub() *SyncEventHub {
	return service.eventHub
}

func (service *SyncService) SyncOnce(ctx context.Context) error {
	return service.syncOnce(ctx, false)
}

func (service *SyncService) ForceFullPull(ctx context.Context) error {
	return service.syncOnce(ctx, true)
}

func (service *SyncService) NeedsBootstrapPull(ctx context.Context) (bool, error) {
	if service == nil || service.local == nil {
		return false, errors.New("sync: local database is required")
	}
	_, err := service.syncState(ctx, remotePullSyncStateKey)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	return false, err
}

func (service *SyncService) syncOnce(ctx context.Context, fullPull bool) error {
	if service == nil || service.local == nil || service.remote == nil {
		return errors.New("sync: databases are required")
	}
	service.mutex.Lock()
	defer service.mutex.Unlock()

	cycleSyncAt := service.now().UTC()
	service.logger.Info("[SYNC] Ciclo iniciado", "fullPull", fullPull)

	pendingOutbox, pushedOutbox, err := service.pushPendingOutbox(ctx)
	if err != nil {
		service.logger.Warn("[SYNC] Ciclo falló en push", "pending", pendingOutbox, "pushed", pushedOutbox, "error", err)
		return err
	}

	lastSyncAt := time.Unix(0, 0).UTC()
	if !fullPull {
		lastSyncAt, err = service.lastRemotePullSyncAt(ctx)
		if err != nil {
			service.logger.Warn("[SYNC] Ciclo falló leyendo watermark", "error", err)
			return err
		}
	}

	userIDMap, err := service.remoteToLocalUserIDMap(ctx)
	if err != nil {
		service.logger.Warn("[SYNC] Ciclo falló resolviendo usuarios por email", "error", err)
		return err
	}

	totalPulledRows := 0
	for _, table := range syncTableSpecs() {
		pulledRows, err := service.pullTable(ctx, table, lastSyncAt, cycleSyncAt, userIDMap)
		if err != nil {
			service.logger.Warn("[SYNC] Ciclo falló en pull", "table", table.name, "from", lastSyncAt, "to", cycleSyncAt, "error", err)
			return fmt.Errorf("sync pull %s: %w", table.name, err)
		}
		totalPulledRows += pulledRows
		service.logger.Info("[SYNC] Pull tabla completado", "table", table.name, "rows", pulledRows)
	}

	if err := service.saveSyncState(ctx, remotePullSyncStateKey, cycleSyncAt); err != nil {
		service.logger.Warn("[SYNC] Ciclo falló guardando watermark", "watermark", cycleSyncAt, "error", err)
		return err
	}

	service.logger.Info(
		"[SYNC] Ciclo completado",
		"fullPull", fullPull,
		"outboxPending", pendingOutbox,
		"outboxPushed", pushedOutbox,
		"pulledRows", totalPulledRows,
		"watermarkFrom", lastSyncAt,
		"watermarkTo", cycleSyncAt,
	)
	if totalPulledRows > 0 && service.eventHub != nil {
		service.eventHub.Publish(SyncUpdatedEvent)
		service.logger.Info("[SYNC] Evento SSE publicado", "event", SyncUpdatedEvent, "pulledRows", totalPulledRows)
	}
	return nil
}

func (service *SyncService) pushPendingOutbox(ctx context.Context) (int, int, error) {
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
		return 0, 0, fmt.Errorf("sync: failed to read outbox: %w", err)
	}

	entries := make([]syncOutboxEntry, 0)
	for rows.Next() {
		var entry syncOutboxEntry
		if err := rows.Scan(&entry.id, &entry.tableName, &entry.entityID); err != nil {
			return len(entries), 0, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return len(entries), 0, err
	}
	if err := rows.Close(); err != nil {
		return len(entries), 0, err
	}

	pushed := 0
	for _, entry := range entries {
		if err := service.pushOutboxEntry(ctx, entry); err != nil {
			if markErr := service.markOutboxEntryFailed(ctx, entry.id, err); markErr != nil {
				return len(entries), pushed, fmt.Errorf("sync push %s.%s failed and could not update outbox: %v: %w", entry.tableName, entry.entityID, err, markErr)
			}
			return len(entries), pushed, fmt.Errorf("sync push %s.%s: %w", entry.tableName, entry.entityID, err)
		}
		pushed++
	}

	return len(entries), pushed, nil
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

func (service *SyncService) pullTable(ctx context.Context, table syncTableSpec, from, to time.Time, userIDMap map[string]string) (int, error) {
	rows, err := service.remote.QueryContext(ctx, incrementalSelectSQL(table, service.remoteDialect), dialectTimeValue(service.remoteDialect, from), dialectTimeValue(service.remoteDialect, to))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	tx, err := service.local.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if err := setOutboxSuppression(ctx, tx, true); err != nil {
		return 0, err
	}

	upsertSQL := lwwUpsertSQL(table, SyncDialectSQLite)
	pulledRows := 0
	for rows.Next() {
		values, err := scanSyncRow(rows, len(table.columns))
		if err != nil {
			return pulledRows, err
		}
		rewritePulledUserIdentity(table, values, userIDMap)
		result, err := tx.ExecContext(ctx, upsertSQL, values...)
		if err != nil {
			return pulledRows, err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected > 0 {
			pulledRows++
		}
	}
	if err := rows.Err(); err != nil {
		return pulledRows, err
	}

	if err := setOutboxSuppression(ctx, tx, false); err != nil {
		return pulledRows, err
	}
	if err := tx.Commit(); err != nil {
		return pulledRows, err
	}

	return pulledRows, nil
}

func (service *SyncService) remoteToLocalUserIDMap(ctx context.Context) (map[string]string, error) {
	localRows, err := service.local.QueryContext(ctx, "SELECT id, email FROM users WHERE deleted_at IS NULL")
	if err != nil {
		return nil, fmt.Errorf("sync: failed to read local users for identity reconciliation: %w", err)
	}
	defer localRows.Close()

	localIDByEmail := make(map[string]string)
	emails := make([]string, 0)
	for localRows.Next() {
		var id string
		var email string
		if err := localRows.Scan(&id, &email); err != nil {
			return nil, err
		}
		normalizedEmail := strings.ToLower(strings.TrimSpace(email))
		if normalizedEmail == "" {
			continue
		}
		localIDByEmail[normalizedEmail] = id
		emails = append(emails, normalizedEmail)
	}
	if err := localRows.Err(); err != nil {
		return nil, err
	}
	if len(emails) == 0 {
		return map[string]string{}, nil
	}

	placeholders := make([]string, 0, len(emails))
	args := make([]interface{}, 0, len(emails))
	for index, email := range emails {
		placeholders = append(placeholders, placeholder(service.remoteDialect, index+1))
		args = append(args, email)
	}

	query := fmt.Sprintf(
		"SELECT id, email FROM users WHERE lower(email) IN (%s) AND deleted_at IS NULL",
		strings.Join(placeholders, ", "),
	)
	remoteRows, err := service.remote.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sync: failed to read remote users for identity reconciliation: %w", err)
	}
	defer remoteRows.Close()

	remoteToLocal := make(map[string]string)
	for remoteRows.Next() {
		var remoteID string
		var email string
		if err := remoteRows.Scan(&remoteID, &email); err != nil {
			return nil, err
		}
		localID, ok := localIDByEmail[strings.ToLower(strings.TrimSpace(email))]
		if !ok || localID == "" || remoteID == "" {
			continue
		}
		if remoteID != localID {
			remoteToLocal[remoteID] = localID
		}
	}
	if err := remoteRows.Err(); err != nil {
		return nil, err
	}
	if len(remoteToLocal) > 0 {
		service.logger.Warn("[SYNC] UUID remoto/local distinto para usuario; se mapeará por email durante pull", "mappedUsers", len(remoteToLocal))
	}

	return remoteToLocal, nil
}

func rewritePulledUserIdentity(table syncTableSpec, values []interface{}, userIDMap map[string]string) {
	if len(userIDMap) == 0 {
		return
	}
	if table.name == "users" && len(values) > 0 {
		if localID, ok := userIDMap[syncStringValue(values[0])]; ok {
			values[0] = localID
		}
	}
	for index, column := range table.columns {
		if column != "user_id" {
			continue
		}
		if localID, ok := userIDMap[syncStringValue(values[index])]; ok {
			values[index] = localID
		}
	}
}

func syncStringValue(value interface{}) string {
	switch typedValue := value.(type) {
	case string:
		return typedValue
	case []byte:
		return string(typedValue)
	default:
		return fmt.Sprint(typedValue)
	}
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
		{name: "credit_cards", columns: []string{"id", "user_id", "name", "bank", "last4", "cutoff_day", "payment_day", "limit_cents", "color", "network", "created_at", "updated_at", "deleted_at"}},
		{name: "financial_accounts", columns: []string{"id", "user_id", "type", "name", "institution", "last4", "opening_balance_cents", "current_balance_cents", "credit_limit_cents", "cutoff_day", "payment_day", "color", "network", "created_at", "updated_at", "deleted_at"}},
		{name: "transactions", columns: []string{"id", "user_id", "credit_card_id", "payment_account_id", "destination_account_id", "type", "concept", "category", "amount_cents", "date", "status", "msi", "installment_number", "installment_count", "is_historical", "recurrence", "recurrence_limit", "last_paid_at", "created_at", "updated_at", "deleted_at"}},
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
