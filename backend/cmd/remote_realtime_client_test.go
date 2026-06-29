package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

type remoteRealtimeTestSessionProvider struct {
	tokenPair ports.TokenPair
	ok        bool
	err       error
}

func (provider *remoteRealtimeTestSessionProvider) EnsureRemoteSession(ctx context.Context) (ports.TokenPair, error) {
	if provider.err != nil {
		return ports.TokenPair{}, provider.err
	}
	if !provider.ok {
		return ports.TokenPair{}, services.ErrRemoteSyncSessionUnavailable
	}
	return provider.tokenPair, nil
}

func TestRemoteRealtimeWebSocketURL(t *testing.T) {
	websocketURL, err := remoteRealtimeWebSocketURL("https://taskify-7n1b.onrender.com", "token-123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if websocketURL != "wss://taskify-7n1b.onrender.com/realtime/ws?token=token-123" {
		t.Fatalf("unexpected websocket url %s", websocketURL)
	}
}

func TestStartRemoteRealtimeClientSignalsWorkerOnSyncUpdate(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(request *http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(response, request, nil)
		if err != nil {
			t.Fatalf("failed to upgrade websocket: %v", err)
		}
		defer connection.Close()

		if err := connection.WriteJSON(services.RealtimeEvent{
			Type:   services.RealtimeSyncUpdateEvent,
			UserID: "user-1",
			Source: services.RealtimeSourceSyncPush,
		}); err != nil {
			t.Fatalf("failed to write realtime event: %v", err)
		}
		time.Sleep(25 * time.Millisecond)
	}))
	defer server.Close()

	signalBus := services.NewSyncSignalBus()
	sessionProvider := &remoteRealtimeTestSessionProvider{
		tokenPair: ports.TokenPair{AccessToken: "remote-access"},
		ok:        true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startRemoteRealtimeClient(ctx, server.URL, sessionProvider, signalBus, &syncWorkerTestLogger{})

	select {
	case reason := <-signalBus.Signals():
		if reason != services.SyncSignalRemoteChange {
			t.Fatalf("expected remote change signal, got %s", reason)
		}
	case <-time.After(time.Second):
		t.Fatal("expected remote realtime signal")
	}
}

func TestStartRemoteRealtimeClientSkipsWhenSessionIsUnavailable(t *testing.T) {
	signalBus := services.NewSyncSignalBus()
	sessionProvider := &remoteRealtimeTestSessionProvider{}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	go startRemoteRealtimeClient(ctx, "https://taskify-7n1b.onrender.com", sessionProvider, signalBus, &syncWorkerTestLogger{})

	select {
	case reason := <-signalBus.Signals():
		t.Fatalf("expected no signal without session, got %s", reason)
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("unexpected context error %v", ctx.Err())
		}
	}
}

func TestRemoteRealtimeWebSocketURLUsesWSForHTTP(t *testing.T) {
	websocketURL, err := remoteRealtimeWebSocketURL("http://localhost:8080", "token-123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !strings.HasPrefix(websocketURL, "ws://localhost:8080/realtime/ws") {
		t.Fatalf("unexpected websocket url %s", websocketURL)
	}
}
