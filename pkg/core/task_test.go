package core

import "testing"

func TestTaskLifecycle(t *testing.T) {
	task := NewTask("goal", "agent-1")
	if task.Status != TaskStatusPending {
		t.Fatalf("expected pending status")
	}
	task.Start()
	if task.Status != TaskStatusRunning {
		t.Fatalf("expected running status")
	}
	task.Complete("done")
	if task.Status != TaskStatusCompleted {
		t.Fatalf("expected completed status")
	}
	if task.Result != "done" {
		t.Fatalf("expected result to be set")
	}
	task.Fail("err")
	if task.Status != TaskStatusFailed {
		t.Fatalf("expected failed status")
	}
	if task.Error == "" {
		t.Fatalf("expected error to be set")
	}
}
