package main

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

const remoteRealtimeReconnectDelay = 3 * time.Second

type remoteRealtimeSessionProvider interface {
	EnsureRemoteSession(ctx context.Context) (ports.TokenPair, error)
}

func startRemoteRealtimeClient(ctx context.Context, remoteAPIURL string, sessionProvider remoteRealtimeSessionProvider, signalBus *services.SyncSignalBus, logger ports.Logger) {
	if strings.TrimSpace(remoteAPIURL) == "" || sessionProvider == nil || signalBus == nil {
		return
	}

	logger.Info("[SYNC][REALTIME] Cliente realtime remoto iniciado")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		tokenPair, err := sessionProvider.EnsureRemoteSession(ctx)
		if err != nil || strings.TrimSpace(tokenPair.AccessToken) == "" {
			if !waitForRemoteRealtimeReconnect(ctx) {
				return
			}
			continue
		}

		websocketURL, err := remoteRealtimeWebSocketURL(remoteAPIURL, tokenPair.AccessToken)
		if err != nil {
			logger.Warn("[SYNC][REALTIME] URL websocket remota invalida", "error", err)
			if !waitForRemoteRealtimeReconnect(ctx) {
				return
			}
			continue
		}

		if err := consumeRemoteRealtimeConnection(ctx, websocketURL, signalBus, logger); err != nil {
			logger.Warn("[SYNC][REALTIME] Conexion realtime remota cerrada", "error", err)
		}
		if !waitForRemoteRealtimeReconnect(ctx) {
			return
		}
	}
}

func consumeRemoteRealtimeConnection(ctx context.Context, websocketURL string, signalBus *services.SyncSignalBus, logger ports.Logger) error {
	connection, _, err := websocket.DefaultDialer.DialContext(ctx, websocketURL, nil)
	if err != nil {
		return err
	}
	defer connection.Close()

	logger.Info("[SYNC][REALTIME] Conexion realtime remota establecida", "url", websocketURL)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, payload, err := connection.ReadMessage()
		if err != nil {
			return err
		}

		var event services.RealtimeEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			logger.Warn("[SYNC][REALTIME] Evento realtime remoto invalido", "error", err)
			continue
		}
		if event.Type != services.RealtimeSyncUpdateEvent {
			continue
		}

		logger.Info("[SYNC][REALTIME] Cambio remoto detectado; se despertara el worker", "userID", event.UserID, "source", event.Source)
		signalBus.Notify(services.SyncSignalRemoteChange)
	}
}

func remoteRealtimeWebSocketURL(remoteAPIURL, accessToken string) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(remoteAPIURL))
	if err != nil {
		return "", err
	}
	switch parsedURL.Scheme {
	case "http":
		parsedURL.Scheme = "ws"
	case "https":
		parsedURL.Scheme = "wss"
	}
	parsedURL.Path = "/realtime/ws"
	parsedURL.RawQuery = ""
	query := parsedURL.Query()
	query.Set("token", strings.TrimSpace(accessToken))
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func waitForRemoteRealtimeReconnect(ctx context.Context) bool {
	timer := time.NewTimer(remoteRealtimeReconnectDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
