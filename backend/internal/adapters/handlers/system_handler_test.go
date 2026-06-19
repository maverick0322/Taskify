package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/services"
)

type mockSystemTokenValidator struct {
	userIDToReturn string
	errToReturn    error
}

func (validator *mockSystemTokenValidator) ValidateToken(token string) (string, error) {
	if validator.errToReturn != nil {
		return "", validator.errToReturn
	}
	if token == "" {
		return "", errors.New("empty token")
	}
	if validator.userIDToReturn == "" {
		return "user-123", nil
	}
	return validator.userIDToReturn, nil
}

type syncEventRecorder struct {
	header http.Header
	body   strings.Builder
	status int
	mutex  sync.Mutex
}

func newSyncEventRecorder() *syncEventRecorder {
	return &syncEventRecorder{header: make(http.Header)}
}

func (recorder *syncEventRecorder) Header() http.Header {
	return recorder.header
}

func (recorder *syncEventRecorder) Write(bytes []byte) (int, error) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	return recorder.body.Write(bytes)
}

func (recorder *syncEventRecorder) WriteHeader(statusCode int) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.status = statusCode
}

func (recorder *syncEventRecorder) Flush() {}

func (recorder *syncEventRecorder) BodyString() string {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	return recorder.body.String()
}

func (recorder *syncEventRecorder) StatusCode() int {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	if recorder.status == 0 {
		return http.StatusOK
	}
	return recorder.status
}

func TestSystemHandler_SyncEventsRejectsMissingToken(t *testing.T) {
	eventHub := services.NewSyncEventHub()
	syncService := services.NewSyncService(nil, nil, services.SyncDialectSQLite, &mockHandlerLogger{})
	syncService.SetEventHub(eventHub)
	handler := NewSystemHandler(nil, syncService, &mockSystemTokenValidator{}, &mockHandlerLogger{})

	request := httptest.NewRequest(http.MethodGet, "/sync/events", nil)
	response := httptest.NewRecorder()

	handler.SyncEvents(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
}

func TestSystemHandler_SyncEventsStreamsPublishedEvents(t *testing.T) {
	eventHub := services.NewSyncEventHub()
	syncService := services.NewSyncService(nil, nil, services.SyncDialectSQLite, &mockHandlerLogger{})
	syncService.SetEventHub(eventHub)
	handler := NewSystemHandler(nil, syncService, &mockSystemTokenValidator{}, &mockHandlerLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	request := httptest.NewRequest(http.MethodGet, "/sync/events?token=valid-token", nil).WithContext(ctx)
	response := newSyncEventRecorder()

	done := make(chan struct{})
	go func() {
		handler.SyncEvents(response, request)
		close(done)
	}()

	waitForBody(t, response, ": connected")
	eventHub.Publish(services.SyncUpdatedEvent)
	waitForBody(t, response, "event: sync_updated")
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected sync events handler to stop after context cancellation")
	}

	if response.StatusCode() != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream content type, got %s", contentType)
	}
	if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-cache" {
		t.Fatalf("expected no-cache, got %s", cacheControl)
	}
}

func waitForBody(t *testing.T, response *syncEventRecorder, expected string) {
	t.Helper()

	deadline := time.After(time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("expected response body to contain %q, got %q", expected, response.BodyString())
		case <-ticker.C:
			if strings.Contains(response.BodyString(), expected) {
				return
			}
		}
	}
}
