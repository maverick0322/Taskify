package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"

	handlerMiddleware "github.com/maverick0322/taskify/backend/internal/adapters/handlers/middleware"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

type mockSystemTokenValidator struct {
	userIDToReturn string
	errToReturn    error
}

func (validator *mockSystemTokenValidator) ValidateToken(token string) (string, error) {
	if validator.errToReturn != nil {
		return "", validator.errToReturn
	}
	if token == "" {
		return "", errors.New("empty token")
	}
	if validator.userIDToReturn == "" {
		return "user-123", nil
	}
	return validator.userIDToReturn, nil
}

type syncEventRecorder struct {
	header http.Header
	body   strings.Builder
	status int
	mutex  sync.Mutex
}

type mockRemoteSessionService struct {
	tokenPair        ports.TokenPair
	loginErr         error
	restoredAccess   string
	restoredRefresh  string
	restorePersisted bool
	restoreErr       error
	cleared          bool
}

type mockLocalSyncService struct {
	needsBootstrap    bool
	needsBootstrapErr error
	syncErr           error
	forceErr          error
	syncCalls         int
	forceCalls        int
	eventHub          *services.SyncEventHub
}

func (service *mockLocalSyncService) SyncOnce(ctx context.Context) error {
	service.syncCalls++
	return service.syncErr
}

func (service *mockLocalSyncService) ForceFullPull(ctx context.Context) error {
	service.forceCalls++
	return service.forceErr
}

func (service *mockLocalSyncService) NeedsBootstrapPull(ctx context.Context) (bool, error) {
	return service.needsBootstrap, service.needsBootstrapErr
}

func (service *mockLocalSyncService) EventHub() *services.SyncEventHub {
	return service.eventHub
}

func (service *mockRemoteSessionService) LoginRemoteSession(ctx context.Context, email, password string) error {
	_, err := service.AuthenticateRemoteSession(ctx, email, password)
	return err
}

func (service *mockRemoteSessionService) AuthenticateRemoteSession(ctx context.Context, email, password string) (ports.TokenPair, error) {
	return service.tokenPair, service.loginErr
}

func (service *mockRemoteSessionService) RestoreRemoteSession(accessToken, refreshToken string) {
	service.restoredAccess = accessToken
	service.restoredRefresh = refreshToken
}

func (service *mockRemoteSessionService) RestorePersistedRemoteSession(ctx context.Context) (bool, error) {
	return service.restorePersisted, service.restoreErr
}

func (service *mockRemoteSessionService) ClearSession() {
	service.cleared = true
}

func newSyncEventRecorder() *syncEventRecorder {
	return &syncEventRecorder{header: make(http.Header)}
}

func (recorder *syncEventRecorder) Header() http.Header {
	return recorder.header
}

func (recorder *syncEventRecorder) Write(bytes []byte) (int, error) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	return recorder.body.Write(bytes)
}

func (recorder *syncEventRecorder) WriteHeader(statusCode int) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.status = statusCode
}

func (recorder *syncEventRecorder) Flush() {}

func (recorder *syncEventRecorder) BodyString() string {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	return recorder.body.String()
}

func (recorder *syncEventRecorder) StatusCode() int {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	if recorder.status == 0 {
		return http.StatusOK
	}
	return recorder.status
}

func TestSystemHandler_SyncEventsRejectsMissingToken(t *testing.T) {
	eventHub := services.NewSyncEventHub()
	syncService := services.NewSyncService(nil, nil, services.SyncDialectSQLite, &mockHandlerLogger{})
	syncService.SetEventHub(eventHub)
	handler := NewSystemHandler(nil, syncService, nil, nil, &mockSystemTokenValidator{}, &mockHandlerLogger{})

	request := httptest.NewRequest(http.MethodGet, "/sync/events", nil)
	response := httptest.NewRecorder()

	handler.SyncEvents(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
}

func TestSystemHandler_SyncEventsStreamsPublishedEvents(t *testing.T) {
	eventHub := services.NewSyncEventHub()
	syncService := services.NewSyncService(nil, nil, services.SyncDialectSQLite, &mockHandlerLogger{})
	syncService.SetEventHub(eventHub)
	handler := NewSystemHandler(nil, syncService, nil, nil, &mockSystemTokenValidator{}, &mockHandlerLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	request := httptest.NewRequest(http.MethodGet, "/sync/events?token=valid-token", nil).WithContext(ctx)
	response := newSyncEventRecorder()

	done := make(chan struct{})
	go func() {
		handler.SyncEvents(response, request)
		close(done)
	}()

	waitForBody(t, response, ": connected")
	eventHub.Publish(services.SyncUpdatedEvent)
	waitForBody(t, response, "event: sync_updated")
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected sync events handler to stop after context cancellation")
	}

	if response.StatusCode() != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream content type, got %s", contentType)
	}
	if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-cache" {
		t.Fatalf("expected no-cache, got %s", cacheControl)
	}
}

func TestSystemHandler_PullSyncRequiresAuthenticatedUser(t *testing.T) {
	service := services.NewRemoteSyncService(nil, services.SyncDialectSQLite, &mockHandlerLogger{})
	handler := NewSystemHandler(nil, nil, service, nil, &mockSystemTokenValidator{}, &mockHandlerLogger{})

	request := httptest.NewRequest(http.MethodGet, "/sync/pull", nil)
	response := httptest.NewRecorder()

	handler.PullSync(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
}

func TestSystemHandler_PullSyncReturnsScopedChanges(t *testing.T) {
	database := openHandlerSyncDatabase(t)
	insertHandlerSyncUser(t, database, "user-1", "user1@example.com")
	insertHandlerSyncBoard(t, database, "board-1", "user-1", "Board 1", time.Date(2025, 6, 26, 12, 0, 0, 0, time.UTC))
	insertHandlerSyncBoard(t, database, "board-2", "user-2", "Board 2", time.Date(2025, 6, 26, 12, 30, 0, 0, time.UTC))

	service := services.NewRemoteSyncService(database, services.SyncDialectSQLite, &mockHandlerLogger{})

	handler := NewSystemHandler(nil, nil, service, nil, &mockSystemTokenValidator{}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodGet, "/sync/pull?cursor=1970-01-01T00:00:00Z", nil)
	request = request.WithContext(handlerMiddleware.ContextWithUserID(request.Context(), "user-1"))
	response := httptest.NewRecorder()

	handler.PullSync(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "\"table\":\"users\"") || !strings.Contains(body, "\"table\":\"boards\"") {
		t.Fatalf("expected pull response with user and board changes, got %s", body)
	}
	if strings.Contains(body, "board-2") {
		t.Fatalf("expected pull response to exclude foreign board, got %s", body)
	}
}

func TestSystemHandler_PushSyncRejectsInvalidBody(t *testing.T) {
	service := services.NewRemoteSyncService(nil, services.SyncDialectSQLite, &mockHandlerLogger{})
	handler := NewSystemHandler(nil, nil, service, nil, &mockSystemTokenValidator{}, &mockHandlerLogger{})

	request := httptest.NewRequest(http.MethodPost, "/sync/push", strings.NewReader("{"))
	request = request.WithContext(handlerMiddleware.ContextWithUserID(request.Context(), "user-1"))
	response := httptest.NewRecorder()

	handler.PushSync(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}

func TestSystemHandler_PushSyncReturnsDetailedFailure(t *testing.T) {
	database := openHandlerSyncDatabase(t)
	insertHandlerSyncUser(t, database, "user-1", "user1@example.com")

	service := services.NewRemoteSyncService(database, services.SyncDialectSQLite, &mockHandlerLogger{})
	handler := NewSystemHandler(nil, nil, service, nil, &mockSystemTokenValidator{}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/sync/push", strings.NewReader(`{"changes":[{"table":"boards","values":["board-1","user-2","Invalid","2025-06-26T12:00:00Z","2025-06-26T12:00:00Z",null]}]}`))
	request = request.WithContext(handlerMiddleware.ContextWithUserID(request.Context(), "user-1"))
	response := httptest.NewRecorder()

	handler.PushSync(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"error":"sync push rejected"`) {
		t.Fatalf("expected sync push rejection body, got %s", body)
	}
	if !strings.Contains(body, `"detail":"sync push change[0] boards(board-1): sync: boards row does not belong to authenticated user"`) {
		t.Fatalf("expected detailed sync push error body, got %s", body)
	}
	if !strings.Contains(body, `"applied":0`) {
		t.Fatalf("expected applied count in error body, got %s", body)
	}
}

func TestSystemHandler_RealtimeWSRejectsInvalidToken(t *testing.T) {
	handler := NewSystemHandler(nil, nil, nil, nil, &mockSystemTokenValidator{errToReturn: errors.New("bad token")}, &mockHandlerLogger{})
	handler.SetRealtimeHub(services.NewUserRealtimeHub())
	router := chi.NewRouter()
	handler.RegisterEventRoutes(router)
	server := httptest.NewServer(router)
	defer server.Close()

	response, err := http.Get(server.URL + "/realtime/ws?token=invalid-token")
	if err != nil {
		t.Fatalf("expected http response, got %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.StatusCode)
	}
}

func TestSystemHandler_RealtimeWSStreamsUserScopedEvents(t *testing.T) {
	realtimeHub := services.NewUserRealtimeHub()
	handler := NewSystemHandler(nil, nil, nil, nil, &mockSystemTokenValidator{userIDToReturn: "user-1"}, &mockHandlerLogger{})
	handler.SetRealtimeHub(realtimeHub)
	router := chi.NewRouter()
	handler.RegisterEventRoutes(router)
	server := httptest.NewServer(router)
	defer server.Close()

	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/realtime/ws?token=valid-token"
	connection, _, err := websocket.DefaultDialer.Dial(websocketURL, nil)
	if err != nil {
		t.Fatalf("expected websocket dial success, got %v", err)
	}
	defer connection.Close()

	realtimeHub.Publish("user-1", services.RealtimeEvent{
		Type:   services.RealtimeSyncUpdateEvent,
		UserID: "user-1",
		Source: services.RealtimeSourceSyncPush,
	})

	var event services.RealtimeEvent
	if err := connection.ReadJSON(&event); err != nil {
		t.Fatalf("expected websocket event, got %v", err)
	}
	if event.Type != services.RealtimeSyncUpdateEvent || event.UserID != "user-1" {
		t.Fatalf("unexpected realtime event %+v", event)
	}
}

func TestSystemHandler_PurgeSQLiteClearsLocalData(t *testing.T) {
	database := openHandlerSyncDatabase(t)
	insertHandlerSyncUser(t, database, "user-1", "user1@example.com")
	insertHandlerSyncBoard(t, database, "board-1", "user-1", "Board 1", time.Date(2025, 6, 26, 12, 0, 0, 0, time.UTC))
	if _, err := database.Exec(`INSERT INTO sync_outbox (id, table_name, entity_id, operation) VALUES ('boards:board-1', 'boards', 'board-1', 'upsert')`); err != nil {
		t.Fatalf("failed to insert sync outbox row: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO sync_state (key, last_successful_sync_at, updated_at) VALUES ('remote_pull', '2025-06-26T12:00:00Z', '2025-06-26T12:00:00Z')`); err != nil {
		t.Fatalf("failed to insert sync state row: %v", err)
	}

	if err := purgeSQLiteData(context.Background(), database); err != nil {
		t.Fatalf("expected purge to succeed, got %v", err)
	}
	assertRowCount(t, database, "users", 0)
	assertRowCount(t, database, "boards", 0)
	assertRowCount(t, database, "sync_outbox", 0)
	assertRowCount(t, database, "sync_state", 0)
}

func TestSystemHandler_LoginRemoteSyncSessionReturnsCompletionOnly(t *testing.T) {
	sessionSync := &mockRemoteSessionService{tokenPair: ports.TokenPair{AccessToken: "remote-access", RefreshToken: "remote-refresh"}}
	syncService := &mockLocalSyncService{}
	handler := NewSystemHandler(nil, syncService, nil, sessionSync, &mockSystemTokenValidator{}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/sync/session/login", strings.NewReader(validLoginJSON()))
	response := httptest.NewRecorder()

	handler.LoginRemoteSyncSession(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, `"accessToken"`) || strings.Contains(body, `"refreshToken"`) {
		t.Fatalf("expected no remote token pair in response, got %s", body)
	}
	if !strings.Contains(body, `"initialSyncCompleted":true`) {
		t.Fatalf("expected initial sync completion in response, got %s", body)
	}
	if syncService.syncCalls != 1 || syncService.forceCalls != 0 {
		t.Fatalf("expected incremental sync once after login, got sync=%d force=%d", syncService.syncCalls, syncService.forceCalls)
	}
}

func TestSystemHandler_RestoreRemoteSyncSessionRestoresTokens(t *testing.T) {
	sessionSync := &mockRemoteSessionService{}
	syncService := &mockLocalSyncService{}
	handler := NewSystemHandler(nil, syncService, nil, sessionSync, &mockSystemTokenValidator{}, &mockHandlerLogger{})
	payload, _ := json.Marshal(map[string]string{
		"accessToken":  "remote-access",
		"refreshToken": "remote-refresh",
	})
	request := httptest.NewRequest(http.MethodPost, "/sync/session/restore", strings.NewReader(string(payload)))
	response := httptest.NewRecorder()

	handler.RestoreRemoteSyncSession(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if sessionSync.restoredAccess != "remote-access" || sessionSync.restoredRefresh != "remote-refresh" {
		t.Fatalf("expected remote session to be restored, got %q %q", sessionSync.restoredAccess, sessionSync.restoredRefresh)
	}
	if syncService.syncCalls != 1 || syncService.forceCalls != 0 {
		t.Fatalf("expected incremental sync once after restore, got sync=%d force=%d", syncService.syncCalls, syncService.forceCalls)
	}
}

func TestSystemHandler_RestoreRemoteSyncSessionLoadsPersistedSessionWhenBodyIsEmpty(t *testing.T) {
	sessionSync := &mockRemoteSessionService{restorePersisted: true}
	syncService := &mockLocalSyncService{}
	handler := NewSystemHandler(nil, syncService, nil, sessionSync, &mockSystemTokenValidator{}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/sync/session/restore", nil)
	response := httptest.NewRecorder()

	handler.RestoreRemoteSyncSession(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if syncService.syncCalls != 1 || syncService.forceCalls != 0 {
		t.Fatalf("expected incremental sync once after persisted restore, got sync=%d force=%d", syncService.syncCalls, syncService.forceCalls)
	}
}

func TestSystemHandler_LoginRemoteSyncSessionUsesForceFullPullWhenBootstrapIsNeeded(t *testing.T) {
	sessionSync := &mockRemoteSessionService{tokenPair: ports.TokenPair{AccessToken: "remote-access", RefreshToken: "remote-refresh"}}
	syncService := &mockLocalSyncService{needsBootstrap: true}
	handler := NewSystemHandler(nil, syncService, nil, sessionSync, &mockSystemTokenValidator{}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/sync/session/login", strings.NewReader(validLoginJSON()))
	response := httptest.NewRecorder()

	handler.LoginRemoteSyncSession(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if syncService.forceCalls != 1 || syncService.syncCalls != 0 {
		t.Fatalf("expected force full pull after login bootstrap, got sync=%d force=%d", syncService.syncCalls, syncService.forceCalls)
	}
}

func TestSystemHandler_RestoreRemoteSyncSessionUsesForceFullPullWhenBootstrapIsNeeded(t *testing.T) {
	sessionSync := &mockRemoteSessionService{}
	syncService := &mockLocalSyncService{needsBootstrap: true}
	handler := NewSystemHandler(nil, syncService, nil, sessionSync, &mockSystemTokenValidator{}, &mockHandlerLogger{})
	payload, _ := json.Marshal(map[string]string{
		"accessToken":  "remote-access",
		"refreshToken": "remote-refresh",
	})
	request := httptest.NewRequest(http.MethodPost, "/sync/session/restore", strings.NewReader(string(payload)))
	response := httptest.NewRecorder()

	handler.RestoreRemoteSyncSession(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if syncService.forceCalls != 1 || syncService.syncCalls != 0 {
		t.Fatalf("expected force full pull after restore bootstrap, got sync=%d force=%d", syncService.syncCalls, syncService.forceCalls)
	}
}

func TestSystemHandler_LoginRemoteSyncSessionReturnsErrorWhenInitialSyncFails(t *testing.T) {
	sessionSync := &mockRemoteSessionService{tokenPair: ports.TokenPair{AccessToken: "remote-access", RefreshToken: "remote-refresh"}}
	syncService := &mockLocalSyncService{forceErr: errors.New("boom"), needsBootstrap: true}
	handler := NewSystemHandler(nil, syncService, nil, sessionSync, &mockSystemTokenValidator{}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/sync/session/login", strings.NewReader(validLoginJSON()))
	response := httptest.NewRecorder()

	handler.LoginRemoteSyncSession(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"initialSyncCompleted":false`) || !strings.Contains(body, `"syncState":"pending"`) {
		t.Fatalf("expected degraded sync response body, got %s", body)
	}
}

func openHandlerSyncDatabase(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite", "file:handler-sync-test?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if _, err := database.Exec(handlerSyncTestSchema); err != nil {
		t.Fatalf("failed to initialize handler sync schema: %v", err)
	}

	return database
}

func insertHandlerSyncUser(t *testing.T, database *sql.DB, id, email string) {
	t.Helper()
	now := time.Date(2025, 6, 26, 11, 0, 0, 0, time.UTC)
	if _, err := database.Exec(
		`INSERT INTO users (id, email, password_hash, first_name, last_name, birth_date, avatar_local_path, avatar_url, created_at, updated_at, deleted_at)
		 VALUES (?, ?, 'hash', 'User', 'One', '1990-01-01T00:00:00Z', '', '', ?, ?, NULL)`,
		id,
		email,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		t.Fatalf("failed to insert handler sync user: %v", err)
	}
}

func insertHandlerSyncBoard(t *testing.T, database *sql.DB, id, userID, name string, updatedAt time.Time) {
	t.Helper()
	if _, err := database.Exec(
		`INSERT INTO boards (id, user_id, name, created_at, updated_at, deleted_at)
		 VALUES (?, ?, ?, ?, ?, NULL)`,
		id,
		userID,
		name,
		updatedAt.Format(time.RFC3339Nano),
		updatedAt.Format(time.RFC3339Nano),
	); err != nil {
		t.Fatalf("failed to insert handler sync board: %v", err)
	}
}

const handlerSyncTestSchema = `
CREATE TABLE users (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	first_name TEXT NOT NULL,
	last_name TEXT NOT NULL,
	birth_date DATETIME NOT NULL,
	avatar_local_path TEXT NOT NULL,
	avatar_url TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE refresh_tokens (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	token_hash TEXT NOT NULL UNIQUE,
	expires_at DATETIME NOT NULL,
	is_revoked INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE boards (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	name TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE columns (
	id TEXT PRIMARY KEY,
	board_id TEXT NOT NULL,
	name TEXT NOT NULL,
	color TEXT NOT NULL,
	position INTEGER NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE tasks (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	board_id TEXT,
	column_id TEXT,
	title TEXT NOT NULL,
	description TEXT NOT NULL,
	status TEXT NOT NULL,
	priority TEXT NOT NULL,
	due_date DATETIME,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE credit_cards (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	name TEXT NOT NULL,
	bank TEXT NOT NULL,
	last4 TEXT NOT NULL,
	cutoff_day INTEGER NOT NULL,
	payment_day INTEGER NOT NULL,
	limit_cents INTEGER NOT NULL,
	color TEXT NOT NULL,
	network TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE financial_accounts (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	type TEXT NOT NULL,
	name TEXT NOT NULL,
	institution TEXT NOT NULL,
	last4 TEXT,
	opening_balance_cents INTEGER NOT NULL,
	current_balance_cents INTEGER NOT NULL,
	credit_limit_cents INTEGER,
	cutoff_day INTEGER,
	payment_day INTEGER,
	color TEXT NOT NULL,
	network TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE transactions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	credit_card_id TEXT,
	payment_account_id TEXT,
	destination_account_id TEXT,
	type TEXT NOT NULL,
	concept TEXT NOT NULL,
	category TEXT NOT NULL,
	amount_cents INTEGER NOT NULL,
	date DATETIME NOT NULL,
	status TEXT NOT NULL,
	msi INTEGER,
	installment_number INTEGER,
	installment_count INTEGER,
	is_historical BOOLEAN NOT NULL DEFAULT 0,
	recurrence TEXT NOT NULL DEFAULT 'once',
	recurrence_limit INTEGER,
	last_paid_at DATETIME,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE ledger_entries (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	account_id TEXT NOT NULL,
	transaction_id TEXT NOT NULL,
	amount_cents INTEGER NOT NULL,
	entry_type TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE credit_card_statements (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	credit_account_id TEXT NOT NULL,
	cycle_start DATETIME NOT NULL,
	cycle_end DATETIME NOT NULL,
	payment_due_date DATETIME NOT NULL,
	statement_amount_cents INTEGER NOT NULL,
	paid_amount_cents INTEGER NOT NULL,
	status TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE credit_card_statement_items (
	id TEXT PRIMARY KEY, user_id TEXT, statement_id TEXT, transaction_id TEXT,
	amount_cents INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
);
CREATE TABLE credit_card_payment_allocations (
	id TEXT PRIMARY KEY, user_id TEXT, statement_id TEXT, payment_transaction_id TEXT,
	amount_cents INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
);
CREATE TABLE account_payable_payments (
	id TEXT PRIMARY KEY,
	account_payable_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	due_date DATETIME NOT NULL,
	paid_at DATETIME NOT NULL,
	amount_cents INTEGER NOT NULL,
	concept TEXT NOT NULL,
	category TEXT NOT NULL,
	created_transaction_id TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE notifications (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	title TEXT NOT NULL,
	message TEXT NOT NULL,
	is_read BOOLEAN NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE storage_sync_jobs (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	entity_type TEXT NOT NULL,
	entity_id TEXT NOT NULL,
	local_path TEXT NOT NULL,
	bucket TEXT NOT NULL,
	object_key TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'pending',
	attempts INTEGER NOT NULL DEFAULT 0,
	last_error TEXT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);
CREATE TABLE sync_runtime_flags (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
INSERT INTO sync_runtime_flags (key, value) VALUES ('suppress_outbox', '0');
CREATE TABLE sync_outbox (
	id TEXT PRIMARY KEY,
	table_name TEXT NOT NULL,
	entity_id TEXT NOT NULL,
	operation TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'pending',
	attempts INTEGER NOT NULL DEFAULT 0,
	last_error TEXT NULL,
	next_attempt_at DATETIME NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE sync_state (
	key TEXT PRIMARY KEY,
	last_successful_sync_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

func waitForBody(t *testing.T, response *syncEventRecorder, expected string) {
	t.Helper()

	deadline := time.After(time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("expected response body to contain %q, got %q", expected, response.BodyString())
		case <-ticker.C:
			if strings.Contains(response.BodyString(), expected) {
				return
			}
		}
	}
}

func assertRowCount(t *testing.T, database *sql.DB, table string, expected int) {
	t.Helper()

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("failed to count %s rows: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %s row count %d, got %d", table, expected, count)
	}
}
