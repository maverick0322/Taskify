package services

type SyncSignalReason string

const (
	SyncSignalLocalMutation SyncSignalReason = "local_mutation"
	SyncSignalRemoteChange  SyncSignalReason = "remote_change"
	SyncSignalFallbackTick  SyncSignalReason = "fallback_tick"
)

type SyncSignalBus struct {
	signals chan SyncSignalReason
}

func NewSyncSignalBus() *SyncSignalBus {
	return &SyncSignalBus{
		signals: make(chan SyncSignalReason, 1),
	}
}

func (bus *SyncSignalBus) Notify(reason SyncSignalReason) {
	if bus == nil {
		return
	}
	select {
	case bus.signals <- reason:
	default:
	}
}

func (bus *SyncSignalBus) Signals() <-chan SyncSignalReason {
	if bus == nil {
		return nil
	}
	return bus.signals
}
