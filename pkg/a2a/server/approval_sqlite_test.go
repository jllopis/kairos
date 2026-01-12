package server

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"

	_ "modernc.org/sqlite"
)

func TestSQLiteApprovalStore_CRUD(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	store, err := NewSQLiteApprovalStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	message := &a2av1.Message{
		MessageId: uuid.NewString(),
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	}
	record, err := store.Create(context.Background(), ApprovalRecord{
		TaskID:    "task-1",
		ContextID: "ctx-1",
		ExpiresAt: time.Now().UTC().Add(-time.Minute),
		Message:   message,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if record.ID == "" {
		t.Fatalf("expected id")
	}
	found, err := store.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if found.TaskID != "task-1" {
		t.Fatalf("unexpected task id: %s", found.TaskID)
	}
	if _, err := store.UpdateStatus(context.Background(), record.ID, ApprovalStatusApproved, "ok"); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, err := store.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.Status != ApprovalStatusApproved {
		t.Fatalf("expected approved")
	}
	list, err := store.List(context.Background(), ApprovalFilter{Status: ApprovalStatusApproved})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected list results")
	}
	expiring, err := store.List(context.Background(), ApprovalFilter{
		Status:         ApprovalStatusApproved,
		ExpiringBefore: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("list expiring: %v", err)
	}
	if len(expiring) == 0 {
		t.Fatalf("expected expiring results")
	}
}
