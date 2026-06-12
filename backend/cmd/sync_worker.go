package main

import (
	"context"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

const syncWorkerInterval = 30 * time.Second

func startSyncWorker(ctx context.Context, syncService *services.SyncService, logger ports.Logger) {
	runSafeSyncCycle(ctx, syncService, logger)

	ticker := time.NewTicker(syncWorkerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runSafeSyncCycle(ctx, syncService, logger)
		}
	}
}

func runSafeSyncCycle(ctx context.Context, syncService *services.SyncService, logger ports.Logger) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("background sync recovered from panic", "panic", recovered)
		}
	}()

	if err := syncService.SyncOnce(ctx); err != nil {
		logger.Warn("background sync skipped after recoverable error", "error", err)
	}
}
