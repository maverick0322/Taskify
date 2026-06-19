package services

import (
	"testing"
	"time"
)

func TestSyncEventHubPublishDeliversToSubscribers(t *testing.T) {
	hub := NewSyncEventHub()
	events, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	hub.Publish(SyncUpdatedEvent)

	select {
	case eventName := <-events:
		if eventName != SyncUpdatedEvent {
			t.Fatalf("expected event %s, got %s", SyncUpdatedEvent, eventName)
		}
	case <-time.After(time.Second):
		t.Fatal("expected sync event")
	}
}

func TestSyncEventHubUnsubscribeStopsEvents(t *testing.T) {
	hub := NewSyncEventHub()
	events, unsubscribe := hub.Subscribe()
	unsubscribe()

	hub.Publish(SyncUpdatedEvent)

	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("expected closed event channel")
		}
	case <-time.After(time.Second):
		t.Fatal("expected closed channel after unsubscribe")
	}
}
