package handlers

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

type SystemHandler struct {
	sqliteDatabase *sql.DB
	syncService    *services.SyncService
	logger         ports.Logger
}

func NewSystemHandler(sqliteDatabase *sql.DB, syncService *services.SyncService, logger ports.Logger) *SystemHandler {
	return &SystemHandler{
		sqliteDatabase: sqliteDatabase,
		syncService:    syncService,
		logger:         logger,
	}
}

func (handler *SystemHandler) RegisterRoutes(router chi.Router) {
	router.Post("/sync/force", handler.ForceSync)
	router.Post("/system/sqlite/checkpoint", handler.CheckpointSQLite)
}

func (handler *SystemHandler) ForceSync(response http.ResponseWriter, request *http.Request) {
	if handler.syncService == nil {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Error: "cloud sync is not configured"})
		return
	}

	if err := handler.syncService.SyncOnce(request.Context()); err != nil {
		handler.logger.Error("manual sync failed", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "sync failed"})
		return
	}

	writeJSON(response, http.StatusOK, map[string]bool{"synced": true})
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
