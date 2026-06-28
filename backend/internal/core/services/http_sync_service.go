package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

var ErrRemoteSyncSessionUnavailable = errors.New("sync: remote session is unavailable")

const (
	remoteAccessTokenRuntimeFlagKey  = "remote_access_token"
	remoteRefreshTokenRuntimeFlagKey = "remote_refresh_token"
)

type RemoteUserProfile struct {
	ID              string
	Email           string
	FirstName       string
	LastName        string
	BirthDate       time.Time
	AvatarLocalPath string
	AvatarURL       string
}

type HTTPRemoteSyncService struct {
	local        *sql.DB
	remoteAPIURL string
	logger       ports.Logger
	client       *http.Client
	now          func() time.Time
	eventHub     *SyncEventHub
	mutex        sync.Mutex
	sessionMutex sync.RWMutex
	accessToken  string
	refreshToken string
}

func NewHTTPRemoteSyncService(local *sql.DB, remoteAPIURL string, logger ports.Logger) *HTTPRemoteSyncService {
	return &HTTPRemoteSyncService{
		local:        local,
		remoteAPIURL: strings.TrimRight(strings.TrimSpace(remoteAPIURL), "/"),
		logger:       logger,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		now: time.Now,
	}
}

func (service *HTTPRemoteSyncService) SetEventHub(eventHub *SyncEventHub) {
	service.eventHub = eventHub
}

func (service *HTTPRemoteSyncService) EventHub() *SyncEventHub {
	return service.eventHub
}

func (service *HTTPRemoteSyncService) SetSession(accessToken, refreshToken string) {
	service.sessionMutex.Lock()
	defer service.sessionMutex.Unlock()

	service.accessToken = strings.TrimSpace(accessToken)
	service.refreshToken = strings.TrimSpace(refreshToken)
}

func (service *HTTPRemoteSyncService) ClearSession() {
	service.sessionMutex.Lock()
	defer service.sessionMutex.Unlock()

	service.accessToken = ""
	service.refreshToken = ""

	if service.local != nil {
		if err := service.clearPersistedRemoteSession(context.Background()); err != nil {
			service.logger.Warn("[SYNC][AUTH] No se pudo limpiar la sesion remota persistida", "error", err)
		}
	}
}

func (service *HTTPRemoteSyncService) LoginRemoteSession(ctx context.Context, email, password string) error {
	_, err := service.AuthenticateRemoteSession(ctx, email, password)
	return err
}

func (service *HTTPRemoteSyncService) AuthenticateRemoteSession(ctx context.Context, email, password string) (ports.TokenPair, error) {
	service.logger.Info("[SYNC][AUTH] Iniciando login remoto", "email", strings.TrimSpace(email), "endpoint", "/users/login")
	payload := map[string]string{
		"email":    strings.TrimSpace(email),
		"password": password,
	}

	responseBody, statusCode, err := service.doJSONRequest(ctx, http.MethodPost, "/users/login", payload, false, true)
	if err != nil {
		service.logger.Warn("[SYNC][AUTH] Login remoto falló", "email", strings.TrimSpace(email), "statusCode", statusCode, "error", err)
		if statusCode == http.StatusUnauthorized {
			return ports.TokenPair{}, ErrInvalidCredentials
		}
		return ports.TokenPair{}, ErrRemoteAuthUnavailable
	}

	var response tokenPairResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		service.logger.Error("[SYNC][AUTH] No se pudo decodificar la respuesta de login remoto", "email", strings.TrimSpace(email), "error", err)
		return ports.TokenPair{}, fmt.Errorf("sync: failed to decode remote login response: %w", err)
	}
	if strings.TrimSpace(response.AccessToken) == "" || strings.TrimSpace(response.RefreshToken) == "" {
		service.logger.Error("[SYNC][AUTH] Login remoto respondió sin token completo", "email", strings.TrimSpace(email))
		return ports.TokenPair{}, fmt.Errorf("sync: remote login did not return a token pair")
	}

	service.SetSession(response.AccessToken, response.RefreshToken)
	if err := service.persistRemoteSession(ctx, response.AccessToken, response.RefreshToken); err != nil {
		service.logger.Error("[SYNC][AUTH] No se pudo persistir la sesión remota", "email", strings.TrimSpace(email), "error", err)
		return ports.TokenPair{}, fmt.Errorf("sync: failed to persist remote session: %w", err)
	}
	service.logger.Info("[SYNC][AUTH] Login remoto exitoso; sesión remota almacenada", "email", strings.TrimSpace(email))
	return ports.TokenPair{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
	}, nil
}

func (service *HTTPRemoteSyncService) RegisterRemoteUser(ctx context.Context, email, password, firstName, lastName, birthDate string) error {
	payload := map[string]string{
		"email":     strings.TrimSpace(email),
		"password":  password,
		"firstName": strings.TrimSpace(firstName),
		"lastName":  strings.TrimSpace(lastName),
		"birthDate": strings.TrimSpace(birthDate),
	}

	_, statusCode, err := service.doJSONRequest(ctx, http.MethodPost, "/users/register", payload, false, true)
	if err != nil {
		switch statusCode {
		case http.StatusConflict:
			return ErrUserAlreadyExists
		case http.StatusBadRequest:
			return ErrInvalidRemoteUserData
		default:
			return ErrRemoteAuthUnavailable
		}
	}

	return nil
}

func (service *HTTPRemoteSyncService) GetRemoteProfile(ctx context.Context) (*RemoteUserProfile, error) {
	service.logger.Info("[SYNC][AUTH] Solicitando perfil remoto autenticado", "endpoint", "/users/me")
	responseBody, err := service.doJSON(ctx, http.MethodGet, "/users/me", nil, true)
	if err != nil {
		service.logger.Warn("[SYNC][AUTH] No se pudo obtener el perfil remoto", "error", err)
		if errors.Is(err, ErrRemoteSyncSessionUnavailable) {
			return nil, ErrRemoteAuthUnavailable
		}
		return nil, ErrRemoteAuthUnavailable
	}

	var response remoteUserProfileResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		service.logger.Error("[SYNC][AUTH] No se pudo decodificar el perfil remoto", "error", err)
		return nil, fmt.Errorf("sync: failed to decode remote profile response: %w", err)
	}

	var birthDate time.Time
	birthDateValue := strings.TrimSpace(response.BirthDate)
	if birthDateValue != "" {
		parsedBirthDate, err := time.Parse("2006-01-02", birthDateValue)
		if err == nil {
			birthDate = parsedBirthDate.UTC()
		} else {
			service.logger.Warn("[SYNC][AUTH] Perfil remoto con birthDate inválido; se aplicará fallback local", "birthDate", birthDateValue, "error", err)
		}
	}

	missingFields := make([]string, 0, 3)
	if strings.TrimSpace(response.FirstName) == "" {
		missingFields = append(missingFields, "firstName")
	}
	if strings.TrimSpace(response.LastName) == "" {
		missingFields = append(missingFields, "lastName")
	}
	if birthDate.IsZero() {
		missingFields = append(missingFields, "birthDate")
	}
	if len(missingFields) > 0 {
		service.logger.Warn("[SYNC][AUTH] Perfil remoto incompleto; la hidratación local aplicará fallbacks", "email", strings.TrimSpace(response.Email), "missingFields", strings.Join(missingFields, ","))
	}

	service.logger.Info("[SYNC][AUTH] Perfil remoto obtenido correctamente", "email", strings.TrimSpace(response.Email), "userID", strings.TrimSpace(response.ID))

	return &RemoteUserProfile{
		ID:              strings.TrimSpace(response.ID),
		Email:           strings.TrimSpace(response.Email),
		FirstName:       strings.TrimSpace(response.FirstName),
		LastName:        strings.TrimSpace(response.LastName),
		BirthDate:       birthDate,
		AvatarLocalPath: strings.TrimSpace(response.AvatarLocalPath),
		AvatarURL:       strings.TrimSpace(response.AvatarURL),
	}, nil
}

func (service *HTTPRemoteSyncService) RestoreRemoteSession(accessToken, refreshToken string) {
	service.SetSession(accessToken, refreshToken)
	if err := service.persistRemoteSession(context.Background(), accessToken, refreshToken); err != nil {
		service.logger.Warn("[SYNC][AUTH] No se pudo persistir la sesión remota restaurada", "error", err)
	}
}

func (service *HTTPRemoteSyncService) RestorePersistedRemoteSession(ctx context.Context) (bool, error) {
	accessToken, refreshToken, ok, err := service.loadPersistedRemoteSession(ctx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	service.SetSession(accessToken, refreshToken)
	service.logger.Info("[SYNC][AUTH] Sesión remota cargada desde almacenamiento local")
	return true, nil
}

func (service *HTTPRemoteSyncService) CurrentRemoteSession() (ports.TokenPair, bool) {
	accessToken, refreshToken, ok := service.sessionSnapshot()
	if !ok {
		return ports.TokenPair{}, false
	}
	return ports.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, true
}

func (service *HTTPRemoteSyncService) HasRemoteSession() bool {
	_, _, ok := service.sessionSnapshot()
	return ok
}

func (service *HTTPRemoteSyncService) SyncOnce(ctx context.Context) error {
	return service.syncOnce(ctx, false)
}

func (service *HTTPRemoteSyncService) ForceFullPull(ctx context.Context) error {
	return service.syncOnce(ctx, true)
}

func (service *HTTPRemoteSyncService) NeedsBootstrapPull(ctx context.Context) (bool, error) {
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

func (service *HTTPRemoteSyncService) syncOnce(ctx context.Context, fullPull bool) error {
	if service == nil || service.local == nil {
		return errors.New("sync: local database is required")
	}
	if strings.TrimSpace(service.remoteAPIURL) == "" {
		return errors.New("sync: remote api url is required")
	}

	service.mutex.Lock()
	defer service.mutex.Unlock()

	lastSyncAt := time.Unix(0, 0).UTC()
	var err error
	if !fullPull {
		lastSyncAt, err = service.lastRemotePullSyncAt(ctx)
		if err != nil {
			return err
		}
	}

	service.logger.Info("[SYNC] Ciclo HTTP iniciado", "fullPull", fullPull, "cursor", lastSyncAt)

	pendingOutbox, pushedOutbox, err := service.pushPendingOutbox(ctx)
	if err != nil {
		service.logger.Warn("[SYNC] Push HTTP falló", "pending", pendingOutbox, "pushed", pushedOutbox, "error", err)
		return err
	}

	changes, cursor, err := service.pullRemoteChanges(ctx, lastSyncAt)
	if err != nil {
		service.logger.Warn("[SYNC] Pull HTTP falló", "cursor", lastSyncAt, "error", err)
		return err
	}

	pulledRows, err := service.applyPulledChanges(ctx, changes)
	if err != nil {
		return err
	}

	if err := service.saveSyncState(ctx, remotePullSyncStateKey, cursor); err != nil {
		return err
	}

	service.logger.Info(
		"[SYNC] Ciclo HTTP completado",
		"fullPull", fullPull,
		"outboxPending", pendingOutbox,
		"outboxPushed", pushedOutbox,
		"pulledRows", pulledRows,
		"cursor", cursor,
	)
	if pulledRows > 0 && service.eventHub != nil {
		service.eventHub.Publish(SyncUpdatedEvent)
	}
	return nil
}

func (service *HTTPRemoteSyncService) pushPendingOutbox(ctx context.Context) (int, int, error) {
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
	defer rows.Close()

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
	if len(entries) == 0 {
		return 0, 0, nil
	}

	changes := make([]RemoteSyncChange, 0, len(entries))
	for _, entry := range entries {
		table, ok := syncTableSpecByName(entry.tableName)
		if !ok {
			return len(entries), 0, fmt.Errorf("unknown sync table %q", entry.tableName)
		}
		values, err := service.localRowValues(ctx, table, entry.entityID)
		if err != nil {
			return len(entries), 0, err
		}
		changes = append(changes, RemoteSyncChange{
			Table:  entry.tableName,
			Values: values,
		})
	}

	responseBody, err := service.doJSON(ctx, http.MethodPost, "/sync/push", syncPushHTTPPayload{Changes: changes}, true)
	if err != nil {
		for _, entry := range entries {
			_ = service.markOutboxEntryFailed(ctx, entry.id, err)
		}
		return len(entries), 0, err
	}

	var response syncPushHTTPResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return len(entries), 0, fmt.Errorf("sync: failed to decode push response: %w", err)
	}
	if response.Applied < 0 || response.Applied > len(entries) {
		return len(entries), 0, fmt.Errorf("sync: invalid applied count %d", response.Applied)
	}

	for index := 0; index < response.Applied; index++ {
		if _, err := service.local.ExecContext(ctx, "DELETE FROM sync_outbox WHERE id = ?", entries[index].id); err != nil {
			return len(entries), index, fmt.Errorf("sync: failed to clear outbox entry: %w", err)
		}
	}
	if response.Applied != len(entries) {
		return len(entries), response.Applied, fmt.Errorf("sync: partial push applied %d of %d", response.Applied, len(entries))
	}

	return len(entries), response.Applied, nil
}

func (service *HTTPRemoteSyncService) pullRemoteChanges(ctx context.Context, cursor time.Time) ([]RemoteSyncChange, time.Time, error) {
	query := url.Values{}
	query.Set("cursor", cursor.UTC().Format(time.RFC3339Nano))

	responseBody, err := service.doJSON(ctx, http.MethodGet, "/sync/pull?"+query.Encode(), nil, true)
	if err != nil {
		return nil, time.Time{}, err
	}

	var response syncPullHTTPResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, time.Time{}, fmt.Errorf("sync: failed to decode pull response: %w", err)
	}

	parsedCursor, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(response.Cursor))
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("sync: invalid pull cursor: %w", err)
	}

	return response.Changes, parsedCursor.UTC(), nil
}

func (service *HTTPRemoteSyncService) applyPulledChanges(ctx context.Context, changes []RemoteSyncChange) (int, error) {
	tx, err := service.local.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "PRAGMA defer_foreign_keys = ON"); err != nil {
		return 0, err
	}

	if err := setOutboxSuppression(ctx, tx, true); err != nil {
		return 0, err
	}

	localUserIDByEmail, err := service.localUserIDByEmail(ctx)
	if err != nil {
		return 0, err
	}
	remoteToLocalUserID := make(map[string]string)
	pulledRows := 0

	orderedChanges, err := orderedRemoteSyncChanges(changes)
	if err != nil {
		return pulledRows, err
	}

	for _, change := range orderedChanges {
		table, ok := syncTableSpecByName(change.Table)
		if !ok {
			return pulledRows, fmt.Errorf("sync: unknown table %q in remote payload", change.Table)
		}
		if len(change.Values) != len(table.columns) {
			return pulledRows, fmt.Errorf("sync: invalid column count for %s", change.Table)
		}

		values := cloneRemoteValues(change.Values)
		if table.name == "users" {
			rewritePulledUserRow(values, localUserIDByEmail, remoteToLocalUserID)
		}
		rewritePulledUserIdentity(table, values, remoteToLocalUserID)

		result, err := tx.ExecContext(ctx, lwwUpsertSQL(table, SyncDialectSQLite), values...)
		if err != nil {
			return pulledRows, err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected > 0 {
			pulledRows++
		}
	}

	if err := setOutboxSuppression(ctx, tx, false); err != nil {
		return pulledRows, err
	}
	if err := tx.Commit(); err != nil {
		return pulledRows, err
	}
	return pulledRows, nil
}

func orderedRemoteSyncChanges(changes []RemoteSyncChange) ([]RemoteSyncChange, error) {
	changesByTable := make(map[string][]RemoteSyncChange)
	for _, change := range changes {
		if _, ok := syncTableSpecByName(change.Table); !ok {
			return nil, fmt.Errorf("sync: unknown table %q in remote payload", change.Table)
		}
		changesByTable[change.Table] = append(changesByTable[change.Table], change)
	}

	ordered := make([]RemoteSyncChange, 0, len(changes))
	for _, table := range syncTableSpecs() {
		ordered = append(ordered, changesByTable[table.name]...)
	}

	return ordered, nil
}

func rewritePulledUserRow(values []interface{}, localUserIDByEmail map[string]string, remoteToLocalUserID map[string]string) {
	if len(values) < 2 {
		return
	}
	remoteID := syncStringValue(values[0])
	email := strings.ToLower(strings.TrimSpace(syncStringValue(values[1])))
	localID, ok := localUserIDByEmail[email]
	if !ok || localID == "" || remoteID == "" || localID == remoteID {
		return
	}
	remoteToLocalUserID[remoteID] = localID
	values[0] = localID
}

func cloneRemoteValues(values []interface{}) []interface{} {
	cloned := make([]interface{}, len(values))
	copy(cloned, values)
	return cloned
}

func (service *HTTPRemoteSyncService) localUserIDByEmail(ctx context.Context) (map[string]string, error) {
	rows, err := service.local.QueryContext(ctx, "SELECT id, email FROM users WHERE deleted_at IS NULL")
	if err != nil {
		return nil, fmt.Errorf("sync: failed to read local users: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var id string
		var email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, err
		}
		email = strings.ToLower(strings.TrimSpace(email))
		if email != "" && id != "" {
			result[email] = id
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (service *HTTPRemoteSyncService) localRowValues(ctx context.Context, table syncTableSpec, entityID string) ([]interface{}, error) {
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

func (service *HTTPRemoteSyncService) markOutboxEntryFailed(ctx context.Context, entryID string, failure error) error {
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

func (service *HTTPRemoteSyncService) lastRemotePullSyncAt(ctx context.Context) (time.Time, error) {
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

func (service *HTTPRemoteSyncService) syncState(ctx context.Context, key string) (time.Time, error) {
	var lastSyncAt time.Time
	err := service.local.QueryRowContext(ctx, "SELECT last_successful_sync_at FROM sync_state WHERE key = ?", key).Scan(&lastSyncAt)
	if err != nil {
		return time.Time{}, err
	}

	return lastSyncAt.UTC(), nil
}

func (service *HTTPRemoteSyncService) saveSyncState(ctx context.Context, key string, syncedAt time.Time) error {
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

func (service *HTTPRemoteSyncService) doJSON(ctx context.Context, method, path string, payload interface{}, requiresAuth bool) ([]byte, error) {
	responseBody, statusCode, err := service.doJSONRequest(ctx, method, path, payload, requiresAuth, false)
	if err == nil {
		return responseBody, nil
	}
	if !requiresAuth || statusCode != http.StatusUnauthorized {
		return nil, err
	}

	if refreshErr := service.refreshRemoteTokens(ctx); refreshErr != nil {
		return nil, refreshErr
	}
	responseBody, _, err = service.doJSONRequest(ctx, method, path, payload, true, true)
	return responseBody, err
}

func (service *HTTPRemoteSyncService) doJSONRequest(ctx context.Context, method, path string, payload interface{}, requiresAuth bool, skipRefresh bool) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		jsonBody, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, fmt.Errorf("sync: failed to encode request payload: %w", err)
		}
		body = bytes.NewReader(jsonBody)
	}

	request, err := http.NewRequestWithContext(ctx, method, service.remoteAPIURL+path, body)
	if err != nil {
		return nil, 0, fmt.Errorf("sync: failed to create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if requiresAuth {
		accessToken, _, ok := service.sessionSnapshot()
		if !ok {
			return nil, 0, ErrRemoteSyncSessionUnavailable
		}
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}

	response, err := service.client.Do(request)
	if err != nil {
		service.logger.Warn("[SYNC][HTTP] Solicitud remota falló antes de responder", "method", method, "path", path, "requiresAuth", requiresAuth, "error", err)
		return nil, 0, fmt.Errorf("sync: remote request failed: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, response.StatusCode, fmt.Errorf("sync: failed to read remote response: %w", err)
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return responseBody, response.StatusCode, nil
	}
	if requiresAuth && response.StatusCode == http.StatusUnauthorized && !skipRefresh {
		service.logger.Warn("[SYNC][HTTP] Solicitud remota no autorizada; se intentará refresh", "method", method, "path", path)
		return nil, response.StatusCode, fmt.Errorf("sync: remote request unauthorized")
	}
	service.logger.Warn("[SYNC][HTTP] Solicitud remota respondió con error", "method", method, "path", path, "statusCode", response.StatusCode, "requiresAuth", requiresAuth, "skipRefresh", skipRefresh, "body", strings.TrimSpace(string(responseBody)))
	return nil, response.StatusCode, fmt.Errorf("sync: remote request failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
}

func (service *HTTPRemoteSyncService) refreshRemoteTokens(ctx context.Context) error {
	_, refreshToken, ok := service.sessionSnapshot()
	if !ok {
		service.logger.Warn("[SYNC][AUTH] Refresh remoto omitido porque no hay sesión remota en memoria")
		return ErrRemoteSyncSessionUnavailable
	}

	responseBody, statusCode, err := service.doJSONRequest(
		ctx,
		http.MethodPost,
		"/users/refresh",
		map[string]string{"refreshToken": refreshToken},
		false,
		true,
	)
	if err != nil {
		if statusCode == http.StatusUnauthorized {
			service.logger.Warn("[SYNC][AUTH] Refresh remoto rechazado; se limpiará la sesión remota", "statusCode", statusCode)
			service.ClearSession()
			return ErrRemoteSyncSessionUnavailable
		}
		service.logger.Warn("[SYNC][AUTH] Refresh remoto falló", "statusCode", statusCode, "error", err)
		return err
	}

	var response tokenPairResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		service.logger.Error("[SYNC][AUTH] No se pudo decodificar la respuesta de refresh remoto", "error", err)
		return fmt.Errorf("sync: failed to decode remote refresh response: %w", err)
	}
	if strings.TrimSpace(response.AccessToken) == "" || strings.TrimSpace(response.RefreshToken) == "" {
		service.logger.Error("[SYNC][AUTH] Refresh remoto respondió sin token completo")
		return fmt.Errorf("sync: remote refresh did not return a token pair")
	}

	service.SetSession(response.AccessToken, response.RefreshToken)
	if err := service.persistRemoteSession(ctx, response.AccessToken, response.RefreshToken); err != nil {
		service.logger.Error("[SYNC][AUTH] No se pudo persistir la sesión remota refrescada", "error", err)
		return fmt.Errorf("sync: failed to persist remote refresh session: %w", err)
	}
	service.logger.Info("[SYNC][AUTH] Refresh remoto exitoso; sesión remota actualizada")
	return nil
}

func (service *HTTPRemoteSyncService) persistRemoteSession(ctx context.Context, accessToken, refreshToken string) error {
	if service == nil || service.local == nil {
		return errors.New("sync: local database is required")
	}

	tx, err := service.local.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sync: failed to start remote session persistence: %w", err)
	}
	defer tx.Rollback()

	statements := []struct {
		key   string
		value string
	}{
		{key: remoteAccessTokenRuntimeFlagKey, value: strings.TrimSpace(accessToken)},
		{key: remoteRefreshTokenRuntimeFlagKey, value: strings.TrimSpace(refreshToken)},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO sync_runtime_flags (key, value)
			 VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			statement.key,
			statement.value,
		); err != nil {
			return fmt.Errorf("sync: failed to persist remote session key %s: %w", statement.key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sync: failed to commit remote session persistence: %w", err)
	}
	return nil
}

func (service *HTTPRemoteSyncService) loadPersistedRemoteSession(ctx context.Context) (string, string, bool, error) {
	if service == nil || service.local == nil {
		return "", "", false, errors.New("sync: local database is required")
	}

	rows, err := service.local.QueryContext(
		ctx,
		`SELECT key, value
		 FROM sync_runtime_flags
		 WHERE key IN (?, ?)`,
		remoteAccessTokenRuntimeFlagKey,
		remoteRefreshTokenRuntimeFlagKey,
	)
	if err != nil {
		return "", "", false, fmt.Errorf("sync: failed to load persisted remote session: %w", err)
	}
	defer rows.Close()

	values := map[string]string{}
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return "", "", false, fmt.Errorf("sync: failed to scan persisted remote session: %w", err)
		}
		values[key] = strings.TrimSpace(value)
	}
	if err := rows.Err(); err != nil {
		return "", "", false, fmt.Errorf("sync: failed to iterate persisted remote session rows: %w", err)
	}

	accessToken := values[remoteAccessTokenRuntimeFlagKey]
	refreshToken := values[remoteRefreshTokenRuntimeFlagKey]
	if accessToken == "" || refreshToken == "" {
		return "", "", false, nil
	}
	return accessToken, refreshToken, true, nil
}

func (service *HTTPRemoteSyncService) clearPersistedRemoteSession(ctx context.Context) error {
	if service == nil || service.local == nil {
		return errors.New("sync: local database is required")
	}

	_, err := service.local.ExecContext(
		ctx,
		`DELETE FROM sync_runtime_flags
		 WHERE key IN (?, ?)`,
		remoteAccessTokenRuntimeFlagKey,
		remoteRefreshTokenRuntimeFlagKey,
	)
	if err != nil {
		return fmt.Errorf("sync: failed to clear persisted remote session: %w", err)
	}
	return nil
}

func (service *HTTPRemoteSyncService) sessionSnapshot() (string, string, bool) {
	service.sessionMutex.RLock()
	defer service.sessionMutex.RUnlock()

	if service.accessToken == "" || service.refreshToken == "" {
		return "", "", false
	}
	return service.accessToken, service.refreshToken, true
}

type tokenPairResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type remoteUserProfileResponse struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	BirthDate       string `json:"birthDate"`
	AvatarLocalPath string `json:"avatarLocalPath"`
	AvatarURL       string `json:"avatarUrl"`
}

type syncPushHTTPPayload struct {
	Changes []RemoteSyncChange `json:"changes"`
}

type syncPushHTTPResponse struct {
	Applied int `json:"applied"`
}

type syncPullHTTPResponse struct {
	Changes []RemoteSyncChange `json:"changes"`
	Cursor  string             `json:"cursor"`
}
