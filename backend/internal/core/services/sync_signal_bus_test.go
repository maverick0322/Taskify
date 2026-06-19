package services

import "testing"

func TestSyncSignalBusNotifyCoalescesWithoutBlocking(t *testing.T) {
	bus := NewSyncSignalBus()

	bus.Notify(SyncSignalLocalMutation)
	bus.Notify(SyncSignalRemoteChange)
	bus.Notify(SyncSignalFallbackTick)

	select {
	case reason := <-bus.Signals():
		if reason != SyncSignalLocalMutation {
			t.Fatalf("expected first signal %s, got %s", SyncSignalLocalMutation, reason)
		}
	default:
		t.Fatal("expected one coalesced signal")
	}

	select {
	case reason := <-bus.Signals():
		t.Fatalf("expected additional signals to be coalesced, got %s", reason)
	default:
	}
}
