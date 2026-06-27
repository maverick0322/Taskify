package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maverick0322/taskify/backend/internal/adapters/handlers/middleware"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

const syncEventsHeartbeatInterval = 25 * time.Second

type localSyncService interface {
	SyncOnce(ctx context.Context) error
	ForceFullPull(ctx context.Context) error
	NeedsBootstrapPull(ctx context.Context) (bool, error)
	EventHub() *services.SyncEventHub
}

type remoteSessionService interface {
	LoginRemoteSession(ctx context.Context, email, password string) error
	AuthenticateRemoteSession(ctx context.Context, email, password string) (ports.TokenPair, error)
	RestoreRemoteSession(accessToken, refreshToken string)
	ClearSession()
}

type SystemHandler struct {
	sqliteDatabase *sql.DB
	syncService    localSyncService
	remoteSync     *services.RemoteSyncService
	sessionSync    remoteSessionService
	tokenValidator ports.TokenValidator
	logger         ports.Logger
}

func NewSystemHandler(sqliteDatabase *sql.DB, syncService localSyncService, remoteSync *services.RemoteSyncService, sessionSync remoteSessionService, tokenValidator ports.TokenValidator, logger ports.Logger) *SystemHandler {
	return &SystemHandler{
		sqliteDatabase: sqliteDatabase,
		syncService:    syncService,
		remoteSync:     remoteSync,
		sessionSync:    sessionSync,
		tokenValidator: tokenValidator,
		logger:         logger,
	}
}

func (handler *SystemHandler) RegisterEventRoutes(router chi.Router) {
	router.Get("/sync/events", handler.SyncEvents)
}

func (handler *SystemHandler) RegisterPublicRoutes(router chi.Router) {
	router.Post("/sync/session/login", handler.LoginRemoteSyncSession)
	router.Post("/sync/session/restore", handler.RestoreRemoteSyncSession)
	router.Post("/sync/session/logout", handler.LogoutRemoteSyncSession)
}

func (handler *SystemHandler) RegisterRoutes(router chi.Router) {
	router.Get("/sync/pull", handler.PullSync)
	router.Post("/sync/push", handler.PushSync)
	router.Post("/sync/force", handler.ForceSync)
	router.Post("/system/sqlite/checkpoint", handler.CheckpointSQLite)
	router.Post("/system/sqlite/purge", handler.PurgeSQLite)
}

func (handler *SystemHandler) SyncEvents(response http.ResponseWriter, request *http.Request) {
	if handler.syncService == nil || handler.syncService.EventHub() == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "cloud sync is not configured"})
		return
	}

	token := request.URL.Query().Get("token")
	if token == "" || handler.tokenValidator == nil {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	if _, err := handler.tokenValidator.ValidateToken(token); err != nil {
		handler.logger.Warn("sync events request rejected because access token is invalid")
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	flusher, ok := response.(http.Flusher)
	if !ok {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "streaming unsupported"})
		return
	}

	headers := response.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")
	headers.Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)

	events, unsubscribe := handler.syncService.EventHub().Subscribe()
	defer unsubscribe()

	heartbeat := time.NewTicker(syncEventsHeartbeatInterval)
	defer heartbeat.Stop()

	if _, err := fmt.Fprint(response, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case <-request.Context().Done():
			return
		case eventName, ok := <-events:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(response, "event: %s\ndata: {}\n\n", eventName); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprint(response, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (handler *SystemHandler) ForceSync(response http.ResponseWriter, request *http.Request) {
	if handler.syncService == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "cloud sync is not configured"})
		return
	}

	fullPull := request.URL.Query().Get("full") == "true"
	var err error
	if fullPull {
		err = handler.syncService.ForceFullPull(request.Context())
	} else {
		err = handler.syncService.SyncOnce(request.Context())
	}
	if err != nil {
		handler.logger.Error("manual sync failed", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "sync failed"})
		return
	}

	writeJSON(response, http.StatusOK, map[string]bool{"synced": true, "full": fullPull})
}

func (handler *SystemHandler) LoginRemoteSyncSession(response http.ResponseWriter, request *http.Request) {
	if handler.sessionSync == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "cloud sync is not configured"})
		return
	}

	var payload loginUserRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	tokenPair, err := handler.sessionSync.AuthenticateRemoteSession(request.Context(), payload.Email, payload.Password)
	if err != nil {
		handler.logger.Warn("remote sync session login failed", "error", err)
		writeJSON(response, http.StatusBadGateway, errorResponse{Error: "remote sync login failed"})
		return
	}

	if err := handler.runInitialDesktopSync(request.Context()); err != nil {
		handler.logger.Error("initial sync failed after remote login", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "initial sync failed"})
		return
	}

	writeJSON(response, http.StatusOK, map[string]interface{}{
		"connected":            true,
		"accessToken":          tokenPair.AccessToken,
		"refreshToken":         tokenPair.RefreshToken,
		"initialSyncCompleted": true,
	})
}

func (handler *SystemHandler) RestoreRemoteSyncSession(response http.ResponseWriter, request *http.Request) {
	if handler.sessionSync == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "cloud sync is not configured"})
		return
	}

	var payload restoreRemoteSessionRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	if strings.TrimSpace(payload.AccessToken) == "" || strings.TrimSpace(payload.RefreshToken) == "" {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	handler.sessionSync.RestoreRemoteSession(payload.AccessToken, payload.RefreshToken)
	if err := handler.runInitialDesktopSync(request.Context()); err != nil {
		handler.logger.Error("initial sync failed after remote session restore", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "initial sync failed"})
		return
	}

	writeJSON(response, http.StatusOK, map[string]interface{}{
		"restored":             true,
		"initialSyncCompleted": true,
	})
}

func (handler *SystemHandler) runInitialDesktopSync(ctx context.Context) error {
	if handler.syncService == nil {
		return nil
	}

	needsBootstrap, err := handler.syncService.NeedsBootstrapPull(ctx)
	if err != nil {
		return err
	}
	if needsBootstrap {
		return handler.syncService.ForceFullPull(ctx)
	}
	return handler.syncService.SyncOnce(ctx)
}

func (handler *SystemHandler) LogoutRemoteSyncSession(response http.ResponseWriter, request *http.Request) {
	if handler.sessionSync == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "cloud sync is not configured"})
		return
	}

	handler.sessionSync.ClearSession()
	writeJSON(response, http.StatusOK, map[string]bool{"cleared": true})
}

func (handler *SystemHandler) PullSync(response http.ResponseWriter, request *http.Request) {
	if handler.remoteSync == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "cloud sync is not configured"})
		return
	}

	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	cursorValue := strings.TrimSpace(request.URL.Query().Get("cursor"))
	cursor := time.Unix(0, 0).UTC()
	if cursorValue != "" {
		parsedCursor, err := time.Parse(time.RFC3339Nano, cursorValue)
		if err != nil {
			writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid sync cursor"})
			return
		}
		cursor = parsedCursor.UTC()
	}

	result, err := handler.remoteSync.PullChanges(request.Context(), userID, cursor)
	if err != nil {
		handler.logger.Error("sync pull failed", "userID", userID, "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "sync failed"})
		return
	}

	writeJSON(response, http.StatusOK, syncPullResponse{
		Changes: result.Changes,
		Cursor:  result.Cursor.Format(time.RFC3339Nano),
	})
}

func (handler *SystemHandler) PushSync(response http.ResponseWriter, request *http.Request) {
	if handler.remoteSync == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "cloud sync is not configured"})
		return
	}

	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	var payload syncPushRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	applied, err := handler.remoteSync.PushChanges(request.Context(), userID, payload.Changes)
	if err != nil {
		handler.logger.Error("sync push failed", "userID", userID, "error", err)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "sync failed"})
		return
	}

	writeJSON(response, http.StatusOK, syncPushResponse{Applied: applied})
}

func (handler *SystemHandler) CheckpointSQLite(response http.ResponseWriter, request *http.Request) {
	if handler.sqliteDatabase == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "sqlite database is not available"})
		return
	}

	if _, err := handler.sqliteDatabase.ExecContext(request.Context(), "PRAGMA wal_checkpoint(FULL)"); err != nil {
		handler.logger.Error("sqlite checkpoint failed", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "sqlite checkpoint failed"})
		return
	}

	writeJSON(response, http.StatusOK, map[string]bool{"checkpointed": true})
}

func (handler *SystemHandler) PurgeSQLite(response http.ResponseWriter, request *http.Request) {
	if handler.sqliteDatabase == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "sqlite database is not available"})
		return
	}

	if err := purgeSQLiteData(request.Context(), handler.sqliteDatabase); err != nil {
		handler.logger.Error("sqlite purge failed", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "sqlite purge failed"})
		return
	}

	writeJSON(response, http.StatusOK, map[string]bool{"purged": true})
}

type syncPushRequest struct {
	Changes []services.RemoteSyncChange `json:"changes"`
}

type syncPushResponse struct {
	Applied int `json:"applied"`
}

type restoreRemoteSessionRequest struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type syncPullResponse struct {
	Changes []services.RemoteSyncChange `json:"changes"`
	Cursor  string                      `json:"cursor"`
}

func purgeSQLiteData(ctx context.Context, database *sql.DB) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statements := []string{
		`INSERT INTO sync_runtime_flags (key, value)
		 VALUES ('suppress_outbox', '1')
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		"DELETE FROM account_payable_payments",
		"DELETE FROM credit_card_statements",
		"DELETE FROM ledger_entries",
		"DELETE FROM transactions",
		"DELETE FROM notifications",
		"DELETE FROM storage_sync_jobs",
		"DELETE FROM financial_accounts",
		"DELETE FROM credit_cards",
		"DELETE FROM tasks",
		"DELETE FROM columns",
		"DELETE FROM boards",
		"DELETE FROM refresh_tokens",
		"DELETE FROM users",
		"DELETE FROM sync_outbox",
		"DELETE FROM sync_state",
		"UPDATE sync_runtime_flags SET value = '0' WHERE key = 'suppress_outbox'",
	}

	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	_, _ = database.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")

	return nil
}
