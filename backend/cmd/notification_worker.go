package main

import (
	"context"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const notificationWorkerInterval = 15 * time.Minute

func startNotificationWorker(ctx context.Context, notificationUseCase ports.NotificationUseCase, logger ports.Logger) {
	runSafeNotificationCycle(ctx, notificationUseCase, logger)

	ticker := time.NewTicker(notificationWorkerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runSafeNotificationCycle(ctx, notificationUseCase, logger)
		}
	}
}

func runSafeNotificationCycle(ctx context.Context, notificationUseCase ports.NotificationUseCase, logger ports.Logger) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("notification worker recovered from panic", "panic", recovered)
		}
	}()

	if err := notificationUseCase.CheckDueNotifications(ctx, time.Now()); err != nil {
		logger.Warn("notification worker skipped after recoverable error", "error", err)
	}
}
