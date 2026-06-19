package services

import "sync"

const SyncUpdatedEvent = "sync_updated"

type SyncEventHub struct {
	mutex       sync.RWMutex
	subscribers map[chan string]struct{}
}

func NewSyncEventHub() *SyncEventHub {
	return &SyncEventHub{
		subscribers: make(map[chan string]struct{}),
	}
}

func (hub *SyncEventHub) Subscribe() (<-chan string, func()) {
	eventChannel := make(chan string, 8)

	hub.mutex.Lock()
	hub.subscribers[eventChannel] = struct{}{}
	hub.mutex.Unlock()

	unsubscribe := func() {
		hub.mutex.Lock()
		if _, ok := hub.subscribers[eventChannel]; ok {
			delete(hub.subscribers, eventChannel)
			close(eventChannel)
		}
		hub.mutex.Unlock()
	}

	return eventChannel, unsubscribe
}

func (hub *SyncEventHub) Publish(eventName string) {
	hub.mutex.RLock()
	defer hub.mutex.RUnlock()

	for subscriber := range hub.subscribers {
		select {
		case subscriber <- eventName:
		default:
		}
	}
}
