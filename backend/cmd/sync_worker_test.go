package main

import (
	"context"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

type mockBackgroundSyncService struct {
	hasRemoteSession bool
	needsBootstrap   bool
	syncCalls        int
	forceCalls       int
}

func (service *mockBackgroundSyncService) SyncOnce(ctx context.Context) error {
	service.syncCalls++
	return nil
}

func (service *mockBackgroundSyncService) ForceFullPull(ctx context.Context) error {
	service.forceCalls++
	return nil
}

func (service *mockBackgroundSyncService) NeedsBootstrapPull(ctx context.Context) (bool, error) {
	return service.needsBootstrap, nil
}

func (service *mockBackgroundSyncService) HasRemoteSession() bool {
	return service.hasRemoteSession
}

type syncWorkerTestLogger struct{}

func (logger *syncWorkerTestLogger) Info(msg string, keys ...interface{})  {}
func (logger *syncWorkerTestLogger) Warn(msg string, keys ...interface{})  {}
func (logger *syncWorkerTestLogger) Error(msg string, keys ...interface{}) {}

var _ ports.Logger = (*syncWorkerTestLogger)(nil)

func TestRunSafeSyncCycle_WithoutRemoteSession_SkipsSync(t *testing.T) {
	service := &mockBackgroundSyncService{hasRemoteSession: false}

	runSafeSyncCycle(context.Background(), service, &syncWorkerTestLogger{}, true)
	runSafeSyncCycle(context.Background(), service, &syncWorkerTestLogger{}, false)

	if service.forceCalls != 0 || service.syncCalls != 0 {
		t.Fatalf("expected no sync calls without remote session, got force=%d sync=%d", service.forceCalls, service.syncCalls)
	}
}

func TestRunSafeSyncCycle_WithRemoteSessionAndBootstrap_RunsForceFullPull(t *testing.T) {
	service := &mockBackgroundSyncService{hasRemoteSession: true}

	runSafeSyncCycle(context.Background(), service, &syncWorkerTestLogger{}, true)

	if service.forceCalls != 1 || service.syncCalls != 0 {
		t.Fatalf("expected one force full pull, got force=%d sync=%d", service.forceCalls, service.syncCalls)
	}
}

func TestRunSafeSyncCycle_WithRemoteSessionAndIncremental_RunsSyncOnce(t *testing.T) {
	service := &mockBackgroundSyncService{hasRemoteSession: true}

	runSafeSyncCycle(context.Background(), service, &syncWorkerTestLogger{}, false)

	if service.forceCalls != 0 || service.syncCalls != 1 {
		t.Fatalf("expected one incremental sync, got force=%d sync=%d", service.forceCalls, service.syncCalls)
	}
}

func TestStartSyncWorker_SignalWithoutRemoteSession_SkipsWakeUpCycle(t *testing.T) {
	service := &mockBackgroundSyncService{hasRemoteSession: false}
	signalBus := services.NewSyncSignalBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startSyncWorker(ctx, service, signalBus, &syncWorkerTestLogger{})
	time.Sleep(20 * time.Millisecond)
	signalBus.Notify(services.SyncSignalLocalMutation)
	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)

	if service.forceCalls != 0 || service.syncCalls != 0 {
		t.Fatalf("expected no sync calls while remote session is absent, got force=%d sync=%d", service.forceCalls, service.syncCalls)
	}
}
