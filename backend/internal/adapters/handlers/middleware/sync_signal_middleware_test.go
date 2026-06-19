package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maverick0322/taskify/backend/internal/core/services"
)

func TestNotifySyncOnSuccessfulMutationSignalsForSuccessfulMutation(t *testing.T) {
	bus := services.NewSyncSignalBus()
	handler := NotifySyncOnSuccessfulMutation(bus)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusCreated)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/tasks", nil))

	select {
	case reason := <-bus.Signals():
		if reason != services.SyncSignalLocalMutation {
			t.Fatalf("expected local mutation signal, got %s", reason)
		}
	default:
		t.Fatal("expected sync signal")
	}
}

func TestNotifySyncOnSuccessfulMutationIgnoresFailedMutation(t *testing.T) {
	bus := services.NewSyncSignalBus()
	handler := NotifySyncOnSuccessfulMutation(bus)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusBadRequest)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/tasks", nil))

	assertNoSyncSignal(t, bus)
}

func TestNotifySyncOnSuccessfulMutationIgnoresSystemRoutes(t *testing.T) {
	bus := services.NewSyncSignalBus()
	handler := NotifySyncOnSuccessfulMutation(bus)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/sync/force", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/system/sqlite/checkpoint", nil))

	assertNoSyncSignal(t, bus)
}

func assertNoSyncSignal(t *testing.T, bus *services.SyncSignalBus) {
	t.Helper()

	select {
	case reason := <-bus.Signals():
		t.Fatalf("expected no sync signal, got %s", reason)
	default:
	}
}
