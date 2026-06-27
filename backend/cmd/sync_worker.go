package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

const syncWorkerFallbackInterval = 5 * time.Minute

type backgroundSyncService interface {
	SyncOnce(ctx context.Context) error
	ForceFullPull(ctx context.Context) error
	NeedsBootstrapPull(ctx context.Context) (bool, error)
}

func startSyncWorker(ctx context.Context, syncService backgroundSyncService, signalBus *services.SyncSignalBus, logger ports.Logger) {
	logger.Info("[SYNC] Worker iniciado; fallback=5m")
	runSafeSyncCycle(ctx, syncService, logger, shouldRunBootstrapPull(ctx, syncService, logger))

	ticker := time.NewTicker(syncWorkerFallbackInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case reason := <-signalBus.Signals():
			logger.Info("[SYNC] Worker despertado por señal", "reason", string(reason))
			runSafeSyncCycle(ctx, syncService, logger, false)
		case <-ticker.C:
			logger.Info("[SYNC] Worker despertado por fallback ticker", "reason", string(services.SyncSignalFallbackTick))
			runSafeSyncCycle(ctx, syncService, logger, false)
		}
	}
}

func shouldRunBootstrapPull(ctx context.Context, syncService backgroundSyncService, logger ports.Logger) bool {
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

func runSafeSyncCycle(ctx context.Context, syncService backgroundSyncService, logger ports.Logger, fullPull bool) {
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
