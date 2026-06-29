package services

import (
	"testing"
	"time"
)

func TestUserRealtimeHubPublishDeliversToMatchingUserOnly(t *testing.T) {
	hub := NewUserRealtimeHub()
	userEvents, unsubscribeUser := hub.Subscribe("user-1")
	defer unsubscribeUser()
	otherEvents, unsubscribeOther := hub.Subscribe("user-2")
	defer unsubscribeOther()

	hub.Publish("user-1", RealtimeEvent{
		Type:   RealtimeSyncUpdateEvent,
		UserID: "user-1",
		Source: RealtimeSourceSyncPush,
	})

	select {
	case eventName := <-userEvents:
		if eventName.Type != RealtimeSyncUpdateEvent {
			t.Fatalf("expected event %s, got %s", RealtimeSyncUpdateEvent, eventName.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("expected realtime event for matching user")
	}

	select {
	case eventName := <-otherEvents:
		t.Fatalf("expected no event for foreign user, got %+v", eventName)
	case <-time.After(25 * time.Millisecond):
	}
}

func TestUserRealtimeHubUnsubscribeStopsEvents(t *testing.T) {
	hub := NewUserRealtimeHub()
	events, unsubscribe := hub.Subscribe("user-1")
	unsubscribe()

	hub.Publish("user-1", RealtimeEvent{
		Type:   RealtimeSyncUpdateEvent,
		UserID: "user-1",
		Source: RealtimeSourceHTTPMutation,
	})

	select {
	case event, ok := <-events:
		if ok {
			t.Fatalf("expected closed channel after unsubscribe, got %+v", event)
		}
	case <-time.After(25 * time.Millisecond):
		t.Fatal("expected realtime channel to close after unsubscribe")
	}
}
