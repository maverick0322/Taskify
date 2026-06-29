package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/maverick0322/taskify/backend/internal/adapters/handlers/middleware"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

const (
	syncEventsHeartbeatInterval = 25 * time.Second
	realtimePingInterval        = 25 * time.Second
	realtimeWriteTimeout        = 10 * time.Second
	realtimeReadLimit           = 1024
	realtimePongWait            = 60 * time.Second
)

var realtimeUpgrader = websocket.Upgrader{
	CheckOrigin: func(request *http.Request) bool {
		return true
	},
}

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
	RestorePersistedRemoteSession(ctx context.Context) (bool, error)
	ClearSession()
}

type SystemHandler struct {
	sqliteDatabase *sql.DB
	syncService    localSyncService
	remoteSync     *services.RemoteSyncService
	sessionSync    remoteSessionService
	tokenValidator ports.TokenValidator
	logger         ports.Logger
	realtimeHub    *services.UserRealtimeHub
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
	router.Get("/realtime/ws", handler.RealtimeWS)
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

func (handler *SystemHandler) SetRealtimeHub(hub *services.UserRealtimeHub) {
	handler.realtimeHub = hub
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

func (handler *SystemHandler) RealtimeWS(response http.ResponseWriter, request *http.Request) {
	if handler.realtimeHub == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "cloud sync is not configured"})
		return
	}
	if handler.tokenValidator == nil {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	token := strings.TrimSpace(request.URL.Query().Get("token"))
	if token == "" {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	userID, err := handler.tokenValidator.ValidateToken(token)
	if err != nil {
		handler.logger.Warn("realtime websocket request rejected because access token is invalid")
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	connection, err := realtimeUpgrader.Upgrade(response, request, nil)
	if err != nil {
		handler.logger.Warn("realtime websocket upgrade failed", "userID", userID, "error", err)
		return
	}
	defer connection.Close()

	connection.SetReadLimit(realtimeReadLimit)
	_ = connection.SetReadDeadline(time.Now().Add(realtimePongWait))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(realtimePongWait))
	})

	events, unsubscribe := handler.realtimeHub.Subscribe(userID)
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := connection.ReadMessage(); err != nil {
				return
			}
		}
	}()

	pingTicker := time.NewTicker(realtimePingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-request.Context().Done():
			return
		case <-done:
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			_ = connection.SetWriteDeadline(time.Now().Add(realtimeWriteTimeout))
			if err := connection.WriteJSON(event); err != nil {
				handler.logger.Warn("realtime websocket write failed", "userID", userID, "error", err)
				return
			}
		case <-pingTicker.C:
			_ = connection.SetWriteDeadline(time.Now().Add(realtimeWriteTimeout))
			if err := connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
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
	handler.logger.Info("[SYNC][SESSION] Solicitud de login remoto recibida", "email", strings.TrimSpace(payload.Email))

	_, err := handler.sessionSync.AuthenticateRemoteSession(request.Context(), payload.Email, payload.Password)
	if err != nil {
		handler.logger.Warn("[SYNC][SESSION] Login remoto del sidecar falló", "email", strings.TrimSpace(payload.Email), "error", err)
		writeJSON(response, http.StatusBadGateway, errorResponse{Error: "remote sync login failed"})
		return
	}
	handler.logger.Info("[SYNC][SESSION] Login remoto del sidecar exitoso; iniciando sync inicial", "email", strings.TrimSpace(payload.Email))

	if err := handler.runInitialDesktopSync(request.Context()); err != nil {
		handler.logger.Error("[SYNC][SESSION] Sync inicial falló después del login remoto", "email", strings.TrimSpace(payload.Email), "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "initial sync failed"})
		return
	}
	handler.logger.Info("[SYNC][SESSION] Login remoto completo con sync inicial aplicado", "email", strings.TrimSpace(payload.Email))

	writeJSON(response, http.StatusOK, map[string]interface{}{
		"connected":            true,
		"initialSyncCompleted": true,
	})
}

func (handler *SystemHandler) RestoreRemoteSyncSession(response http.ResponseWriter, request *http.Request) {
	if handler.sessionSync == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "cloud sync is not configured"})
		return
	}

	var payload restoreRemoteSessionRequest
	if request.Body != nil {
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
			return
		}
	}
	if strings.TrimSpace(payload.AccessToken) != "" && strings.TrimSpace(payload.RefreshToken) != "" {
		handler.logger.Info("[SYNC][SESSION] Restaurando sesión remota desde payload legado")
		handler.sessionSync.RestoreRemoteSession(payload.AccessToken, payload.RefreshToken)
	} else {
		handler.logger.Info("[SYNC][SESSION] Restaurando sesión remota persistida en el sidecar")
		restored, err := handler.sessionSync.RestorePersistedRemoteSession(request.Context())
		if err != nil {
			handler.logger.Error("[SYNC][SESSION] No se pudo restaurar la sesión remota persistida", "error", err)
			writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "initial sync failed"})
			return
		}
		if !restored {
			writeJSON(response, http.StatusOK, map[string]interface{}{
				"restored":             false,
				"initialSyncCompleted": false,
			})
			return
		}
	}

	if err := handler.runInitialDesktopSync(request.Context()); err != nil {
		handler.logger.Error("[SYNC][SESSION] Sync inicial falló después de restaurar la sesión remota", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "initial sync failed"})
		return
	}
	handler.logger.Info("[SYNC][SESSION] Sesión remota restaurada con sync inicial aplicado")

	writeJSON(response, http.StatusOK, map[string]interface{}{
		"restored":             true,
		"initialSyncCompleted": true,
	})
}

func (handler *SystemHandler) runInitialDesktopSync(ctx context.Context) error {
	if handler.syncService == nil {
		handler.logger.Info("[SYNC][SESSION] Sync inicial omitido porque syncService es nil")
		return nil
	}

	needsBootstrap, err := handler.syncService.NeedsBootstrapPull(ctx)
	if err != nil {
		handler.logger.Error("[SYNC][SESSION] No se pudo decidir el modo de sync inicial", "error", err)
		return err
	}
	if needsBootstrap {
		handler.logger.Info("[SYNC][SESSION] Ejecutando ForceFullPull inicial", "reason", "missing_remote_pull_watermark")
		if err := handler.syncService.ForceFullPull(ctx); err != nil {
			handler.logger.Error("[SYNC][SESSION] ForceFullPull inicial falló", "error", err)
			return err
		}
		handler.logger.Info("[SYNC][SESSION] ForceFullPull inicial completado")
		return nil
	}
	handler.logger.Info("[SYNC][SESSION] Ejecutando SyncOnce inicial", "reason", "remote_pull_watermark_present")
	if err := handler.syncService.SyncOnce(ctx); err != nil {
		handler.logger.Error("[SYNC][SESSION] SyncOnce inicial falló", "error", err)
		return err
	}
	handler.logger.Info("[SYNC][SESSION] SyncOnce inicial completado")
	return nil
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
		handler.logger.Error("sync push failed", "userID", userID, "changesCount", len(payload.Changes), "applied", applied, "error", err)
		writeJSON(response, http.StatusBadRequest, syncPushErrorResponse{
			Error:   "sync push rejected",
			Detail:  err.Error(),
			Applied: applied,
		})
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

type syncPushErrorResponse struct {
	Error   string `json:"error"`
	Detail  string `json:"detail"`
	Applied int    `json:"applied"`
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
