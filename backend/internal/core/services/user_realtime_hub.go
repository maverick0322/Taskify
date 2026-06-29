package services

import "sync"

const (
	RealtimeSyncUpdateEvent   = "sync_update"
	RealtimeSourceHTTPMutation = "http_mutation"
	RealtimeSourceSyncPush     = "sync_push"
)

type RealtimeEvent struct {
	Type   string `json:"type"`
	UserID string `json:"userId"`
	Source string `json:"source"`
}

type UserRealtimeHub struct {
	mutex       sync.RWMutex
	subscribers map[string]map[chan RealtimeEvent]struct{}
}

func NewUserRealtimeHub() *UserRealtimeHub {
	return &UserRealtimeHub{
		subscribers: make(map[string]map[chan RealtimeEvent]struct{}),
	}
}

func (hub *UserRealtimeHub) Subscribe(userID string) (<-chan RealtimeEvent, func()) {
	eventChannel := make(chan RealtimeEvent, 8)

	hub.mutex.Lock()
	if _, ok := hub.subscribers[userID]; !ok {
		hub.subscribers[userID] = make(map[chan RealtimeEvent]struct{})
	}
	hub.subscribers[userID][eventChannel] = struct{}{}
	hub.mutex.Unlock()

	unsubscribe := func() {
		hub.mutex.Lock()
		if userSubscribers, ok := hub.subscribers[userID]; ok {
			if _, ok := userSubscribers[eventChannel]; ok {
				delete(userSubscribers, eventChannel)
				close(eventChannel)
			}
			if len(userSubscribers) == 0 {
				delete(hub.subscribers, userID)
			}
		}
		hub.mutex.Unlock()
	}

	return eventChannel, unsubscribe
}

func (hub *UserRealtimeHub) Publish(userID string, event RealtimeEvent) {
	if hub == nil || userID == "" {
		return
	}

	hub.mutex.RLock()
	defer hub.mutex.RUnlock()

	for subscriber := range hub.subscribers[userID] {
		select {
		case subscriber <- event:
		default:
		}
	}
}
