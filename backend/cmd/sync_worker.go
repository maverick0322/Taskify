package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

const syncWorkerInterval = time.Minute

func startSyncWorker(ctx context.Context, syncService *services.SyncService, logger ports.Logger) {
	logger.Info("[SYNC] Worker iniciado; intervalo=1m")
	runSafeSyncCycle(ctx, syncService, logger, shouldRunBootstrapPull(ctx, syncService, logger))

	ticker := time.NewTicker(syncWorkerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runSafeSyncCycle(ctx, syncService, logger, false)
		}
	}
}

func shouldRunBootstrapPull(ctx context.Context, syncService *services.SyncService, logger ports.Logger) bool {
	if strings.EqualFold(os.Getenv("SYNC_BOOTSTRAP_PULL"), "true") {
		logger.Info("[SYNC] Bootstrap pull activado por SYNC_BOOTSTRAP_PULL=true")
		return true
	}
	needsBootstrap, err := syncService.NeedsBootstrapPull(ctx)
	if err != nil {
		logger.Warn("[SYNC] No se pudo leer watermark inicial; se intentará sync normal", "error", err)
		return false
	}
	if needsBootstrap {
		logger.Info("[SYNC] Bootstrap pull activado: no existe watermark remote_pull")
	}
	return needsBootstrap
}

func runSafeSyncCycle(ctx context.Context, syncService *services.SyncService, logger ports.Logger, fullPull bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("background sync recovered from panic", "panic", recovered)
		}
	}()

	var err error
	if fullPull {
		err = syncService.ForceFullPull(ctx)
	} else {
		err = syncService.SyncOnce(ctx)
	}
	if err != nil {
		logger.Warn("[SYNC] Background sync falló", "fullPull", fullPull, "error", err)
	}
}
