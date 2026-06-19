package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

const (
	taskifySyncEventsChannel   = "taskify_sync_events"
	remoteChangeReconnectDelay = 5 * time.Second
)

func startRemoteChangeListener(ctx context.Context, remoteDatabaseURL string, signalBus *services.SyncSignalBus, logger ports.Logger) {
	if remoteDatabaseURL == "" || signalBus == nil {
		return
	}

	for {
		if ctx.Err() != nil {
			return
		}

		if err := runRemoteChangeListener(ctx, remoteDatabaseURL, signalBus, logger); err != nil && ctx.Err() == nil {
			logger.Warn("[SYNC] Listener remoto desconectado; reintentando", "error", err, "retryIn", remoteChangeReconnectDelay.String())
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(remoteChangeReconnectDelay):
		}
	}
}

func runRemoteChangeListener(ctx context.Context, remoteDatabaseURL string, signalBus *services.SyncSignalBus, logger ports.Logger) error {
	connection, err := pgx.Connect(ctx, remoteDatabaseURL)
	if err != nil {
		return err
	}
	defer connection.Close(context.Background())

	if _, err := connection.Exec(ctx, "LISTEN "+taskifySyncEventsChannel); err != nil {
		return err
	}
	logger.Info("[SYNC] Listener remoto iniciado", "channel", taskifySyncEventsChannel)

	for {
		notification, err := connection.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		if notification == nil {
			continue
		}
		logger.Info("[SYNC] Cambio remoto recibido", "channel", notification.Channel)
		signalBus.Notify(services.SyncSignalRemoteChange)
	}
}
