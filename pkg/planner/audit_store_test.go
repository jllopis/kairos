package planner

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestMemoryAuditStore(t *testing.T) {
	store := NewMemoryAuditStore()
	event := AuditEvent{
		GraphID:   "graph-1",
		RunID:     "run-1",
		NodeID:    "node-1",
		NodeType:  "noop",
		Status:    "completed",
		Output:    map[string]any{"ok": true},
		StartedAt: time.Now().UTC(),
	}
	if err := store.Record(context.Background(), event); err != nil {
		t.Fatalf("record: %v", err)
	}
	events, err := store.List(context.Background(), AuditFilter{GraphID: "graph-1"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].NodeID != "node-1" {
		t.Fatalf("unexpected node id: %s", events[0].NodeID)
	}
}

func TestSQLiteAuditStore(t *testing.T) {
	db, err := sql.Open("sqlite", "file:planner_audit_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	store, err := NewSQLiteAuditStore(db)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	event := AuditEvent{
		GraphID:   "graph-1",
		RunID:     "run-1",
		NodeID:    "node-1",
		NodeType:  "noop",
		Status:    "completed",
		Output:    map[string]any{"ok": true},
		StartedAt: time.Now().UTC(),
	}
	if err := store.Record(context.Background(), event); err != nil {
		t.Fatalf("record: %v", err)
	}
	events, err := store.List(context.Background(), AuditFilter{GraphID: "graph-1", Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].RunID != "run-1" {
		t.Fatalf("unexpected run id: %s", events[0].RunID)
	}
	if events[0].NodeType != "noop" {
		t.Fatalf("unexpected node type: %s", events[0].NodeType)
	}
}
