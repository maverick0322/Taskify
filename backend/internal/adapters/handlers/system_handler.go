package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

const syncEventsHeartbeatInterval = 25 * time.Second

type SystemHandler struct {
	sqliteDatabase *sql.DB
	syncService    *services.SyncService
	tokenValidator ports.TokenValidator
	logger         ports.Logger
}

func NewSystemHandler(sqliteDatabase *sql.DB, syncService *services.SyncService, tokenValidator ports.TokenValidator, logger ports.Logger) *SystemHandler {
	return &SystemHandler{
		sqliteDatabase: sqliteDatabase,
		syncService:    syncService,
		tokenValidator: tokenValidator,
		logger:         logger,
	}
}

func (handler *SystemHandler) RegisterEventRoutes(router chi.Router) {
	router.Get("/sync/events", handler.SyncEvents)
}

func (handler *SystemHandler) RegisterRoutes(router chi.Router) {
	router.Post("/sync/force", handler.ForceSync)
	router.Post("/system/sqlite/checkpoint", handler.CheckpointSQLite)
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
