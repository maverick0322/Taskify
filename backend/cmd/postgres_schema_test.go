package main

import (
	"strings"
	"testing"
)

func TestPostgresSyncSchemaIncludesRealtimeNotifyTriggers(t *testing.T) {
	expectedFragments := []string{
		"CREATE OR REPLACE FUNCTION notify_taskify_sync()",
		"pg_notify(",
		"taskify_sync_events",
		"CREATE TRIGGER %I AFTER INSERT OR UPDATE OR DELETE",
		"'boards'",
		"'tasks'",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(postgresSyncSchema, fragment) {
			t.Fatalf("expected postgres sync schema to contain %q", fragment)
		}
	}
}
