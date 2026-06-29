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

const remoteSyncPullBatchLimit = 500

type RemoteSyncChange struct {
	Table  string        `json:"table"`
	Values []interface{} `json:"values"`
}

type RemoteSyncPullResult struct {
	Changes []RemoteSyncChange
	Cursor  time.Time
}

type RemoteSyncService struct {
	database *sql.DB
	dialect  SyncDialect
	logger   ports.Logger
	now      func() time.Time
	realtime *UserRealtimeHub
}

func NewRemoteSyncService(database *sql.DB, dialect SyncDialect, logger ports.Logger) *RemoteSyncService {
	return &RemoteSyncService{
		database: database,
		dialect:  dialect,
		logger:   logger,
		now:      time.Now,
	}
}

func (service *RemoteSyncService) SetRealtimeHub(hub *UserRealtimeHub) {
	service.realtime = hub
}

func (service *RemoteSyncService) PullChanges(ctx context.Context, userID string, cursor time.Time) (RemoteSyncPullResult, error) {
	if service == nil || service.database == nil {
		return RemoteSyncPullResult{}, errors.New("sync: database is required")
	}
	if strings.TrimSpace(userID) == "" {
		return RemoteSyncPullResult{}, errors.New("sync: user id is required")
	}

	syncUntil := service.now().UTC()
	changes := make([]RemoteSyncChange, 0)

	for _, table := range syncTableSpecs() {
		query, args, err := remoteScopedIncrementalSelect(table, service.dialect, userID, cursor.UTC(), syncUntil)
		if err != nil {
			return RemoteSyncPullResult{}, err
		}

		rows, err := service.database.QueryContext(ctx, query, args...)
		if err != nil {
			return RemoteSyncPullResult{}, fmt.Errorf("sync pull %s: %w", table.name, err)
		}

		for rows.Next() {
			values, err := scanSyncRow(rows, len(table.columns))
			if err != nil {
				rows.Close()
				return RemoteSyncPullResult{}, err
			}
			changes = append(changes, RemoteSyncChange{Table: table.name, Values: values})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return RemoteSyncPullResult{}, err
		}
		if err := rows.Close(); err != nil {
			return RemoteSyncPullResult{}, err
		}
	}

	return RemoteSyncPullResult{
		Changes: changes,
		Cursor:  syncUntil,
	}, nil
}

func (service *RemoteSyncService) PushChanges(ctx context.Context, userID string, changes []RemoteSyncChange) (int, error) {
	if service == nil || service.database == nil {
		return 0, errors.New("sync: database is required")
	}
	if strings.TrimSpace(userID) == "" {
		return 0, errors.New("sync: user id is required")
	}
	if len(changes) == 0 {
		return 0, nil
	}
	for index, change := range changes {
		if _, ok := syncTableSpecByName(change.Table); !ok {
			return 0, fmt.Errorf("sync push %s: unknown table %q", describeRemoteSyncChange(index, change), change.Table)
		}
	}

	orderedChanges, err := orderedRemoteSyncChanges(changes)
	if err != nil {
		return 0, err
	}

	tx, err := service.database.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	applied := 0
	for index, change := range orderedChanges {
		table, ok := syncTableSpecByName(change.Table)
		if !ok {
			return applied, fmt.Errorf("sync push %s: unknown table %q", describeRemoteSyncChange(index, change), change.Table)
		}
		changeLabel := describeRemoteSyncChange(index, change)
		if len(change.Values) != len(table.columns) {
			return applied, fmt.Errorf("sync push %s: invalid value count for table %s", changeLabel, change.Table)
		}

		normalizedValues := make([]interface{}, 0, len(change.Values))
		for index, rawValue := range change.Values {
			value, err := normalizeRemoteSyncValue(table.columns[index], rawValue)
			if err != nil {
				return applied, fmt.Errorf("sync push %s: invalid value for %s.%s: %w", changeLabel, change.Table, table.columns[index], err)
			}
			normalizedValues = append(normalizedValues, value)
		}

		if err := ensureRemoteSyncOwnership(ctx, tx, service.dialect, table, userID, normalizedValues); err != nil {
			return applied, fmt.Errorf("sync push %s: %w", changeLabel, err)
		}

		if _, err := tx.ExecContext(ctx, lwwUpsertSQL(table, service.dialect), normalizedValues...); err != nil {
			return applied, fmt.Errorf("sync push %s: %w", changeLabel, err)
		}
		applied++
	}

	if err := tx.Commit(); err != nil {
		return applied, err
	}
	if service.realtime != nil {
		service.realtime.Publish(userID, RealtimeEvent{
			Type:   RealtimeSyncUpdateEvent,
			UserID: userID,
			Source: RealtimeSourceSyncPush,
		})
	}

	return applied, nil
}

func describeRemoteSyncChange(index int, change RemoteSyncChange) string {
	entityID := ""
	if len(change.Values) > 0 {
		entityID = strings.TrimSpace(syncStringValue(change.Values[0]))
	}
	if entityID != "" {
		return fmt.Sprintf("change[%d] %s(%s)", index, change.Table, entityID)
	}
	return fmt.Sprintf("change[%d] %s", index, change.Table)
}

func remoteScopedIncrementalSelect(table syncTableSpec, dialect SyncDialect, userID string, from, to time.Time) (string, []interface{}, error) {
	switch table.name {
	case "users":
		return fmt.Sprintf(
			"SELECT %s FROM users WHERE id = %s AND updated_at > %s AND updated_at <= %s ORDER BY updated_at ASC LIMIT %d",
			strings.Join(table.columns, ", "),
			placeholder(dialect, 1),
			placeholder(dialect, 2),
			placeholder(dialect, 3),
			remoteSyncPullBatchLimit,
		), []interface{}{userID, dialectTimeValue(dialect, from), dialectTimeValue(dialect, to)}, nil
	case "columns":
		return fmt.Sprintf(
			`SELECT %s
			 FROM columns
			 WHERE board_id IN (
			 	SELECT id FROM boards WHERE user_id = %s AND deleted_at IS NULL
			 )
			   AND updated_at > %s
			   AND updated_at <= %s
			 ORDER BY updated_at ASC
			 LIMIT %d`,
			strings.Join(table.columns, ", "),
			placeholder(dialect, 1),
			placeholder(dialect, 2),
			placeholder(dialect, 3),
			remoteSyncPullBatchLimit,
		), []interface{}{userID, dialectTimeValue(dialect, from), dialectTimeValue(dialect, to)}, nil
	default:
		if hasSyncColumn(table, "user_id") {
			return fmt.Sprintf(
				"SELECT %s FROM %s WHERE user_id = %s AND updated_at > %s AND updated_at <= %s ORDER BY updated_at ASC LIMIT %d",
				strings.Join(table.columns, ", "),
				table.name,
				placeholder(dialect, 1),
				placeholder(dialect, 2),
				placeholder(dialect, 3),
				remoteSyncPullBatchLimit,
			), []interface{}{userID, dialectTimeValue(dialect, from), dialectTimeValue(dialect, to)}, nil
		}
	}

	return "", nil, fmt.Errorf("sync: table %s has no user scope", table.name)
}

func ensureRemoteSyncOwnership(ctx context.Context, executor sqlExecutor, dialect SyncDialect, table syncTableSpec, userID string, values []interface{}) error {
	switch table.name {
	case "users":
		if syncStringValue(values[0]) != userID {
			return errors.New("sync: users row does not belong to authenticated user")
		}
		return nil
	case "columns":
		boardID := syncStringValue(values[columnIndex(table, "board_id")])
		if boardID == "" {
			return errors.New("sync: columns row requires board_id")
		}
		var ownerUserID string
		query := fmt.Sprintf("SELECT user_id FROM boards WHERE id = %s", placeholder(dialect, 1))
		if err := executor.QueryRowContext(ctx, query, boardID).Scan(&ownerUserID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("sync: board not found for columns row")
			}
			return err
		}
		if ownerUserID != userID {
			return errors.New("sync: columns row does not belong to authenticated user")
		}
		return nil
	default:
		if hasSyncColumn(table, "user_id") {
			index := columnIndex(table, "user_id")
			if syncStringValue(values[index]) != userID {
				return fmt.Errorf("sync: %s row does not belong to authenticated user", table.name)
			}
			return nil
		}
	}

	return fmt.Errorf("sync: table %s has no user scope", table.name)
}

func normalizeRemoteSyncValue(column string, rawValue interface{}) (interface{}, error) {
	if rawValue == nil {
		return nil, nil
	}

	switch column {
	case "is_read", "is_historical":
		booleanValue, ok := rawValue.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool, got %T", rawValue)
		}
		return booleanValue, nil
	case "position", "opening_balance_cents", "current_balance_cents", "credit_limit_cents",
		"cutoff_day", "payment_day", "amount_cents", "msi", "installment_number",
		"installment_count", "recurrence_limit", "paid_amount_cents", "statement_amount_cents":
		switch typedValue := rawValue.(type) {
		case float64:
			return int64(typedValue), nil
		case int:
			return int64(typedValue), nil
		case int32:
			return int64(typedValue), nil
		case int64:
			return typedValue, nil
		default:
			return nil, fmt.Errorf("expected number, got %T", rawValue)
		}
	default:
		switch typedValue := rawValue.(type) {
		case string:
			return typedValue, nil
		case float64:
			return fmt.Sprintf("%.0f", typedValue), nil
		case bool:
			return fmt.Sprintf("%t", typedValue), nil
		default:
			return fmt.Sprint(typedValue), nil
		}
	}
}

func hasSyncColumn(table syncTableSpec, target string) bool {
	for _, column := range table.columns {
		if column == target {
			return true
		}
	}
	return false
}

func columnIndex(table syncTableSpec, target string) int {
	for index, column := range table.columns {
		if column == target {
			return index
		}
	}
	return -1
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
