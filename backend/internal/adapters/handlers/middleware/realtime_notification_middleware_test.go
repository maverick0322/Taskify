package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/services"
)

func TestNotifyRealtimeOnSuccessfulMutationPublishesForSuccessfulMutation(t *testing.T) {
	hub := services.NewUserRealtimeHub()
	events, unsubscribe := hub.Subscribe("user-1")
	defer unsubscribe()

	handler := NotifyRealtimeOnSuccessfulMutation(hub)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusCreated)
	}))

	request := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	request = request.WithContext(ContextWithUserID(request.Context(), "user-1"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	select {
	case event := <-events:
		if event.Type != services.RealtimeSyncUpdateEvent || event.Source != services.RealtimeSourceHTTPMutation {
			t.Fatalf("unexpected realtime event %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected realtime event after successful mutation")
	}
}

func TestNotifyRealtimeOnSuccessfulMutationIgnoresFailedMutation(t *testing.T) {
	hub := services.NewUserRealtimeHub()
	events, unsubscribe := hub.Subscribe("user-1")
	defer unsubscribe()

	handler := NotifyRealtimeOnSuccessfulMutation(hub)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusBadRequest)
	}))

	request := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	request = request.WithContext(ContextWithUserID(request.Context(), "user-1"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	select {
	case event := <-events:
		t.Fatalf("expected no realtime event, got %+v", event)
	case <-time.After(25 * time.Millisecond):
	}
}

func TestNotifyRealtimeOnSuccessfulMutationIgnoresSystemRoutes(t *testing.T) {
	hub := services.NewUserRealtimeHub()
	events, unsubscribe := hub.Subscribe("user-1")
	defer unsubscribe()

	handler := NotifyRealtimeOnSuccessfulMutation(hub)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodPost, "/sync/push", nil)
	request = request.WithContext(ContextWithUserID(request.Context(), "user-1"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	select {
	case event := <-events:
		t.Fatalf("expected no realtime event, got %+v", event)
	case <-time.After(25 * time.Millisecond):
	}
}
