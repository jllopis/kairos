package core

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus describes the lifecycle state of a task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusRejected  TaskStatus = "rejected"
)

// Task represents a first-class unit of work in Kairos.
type Task struct {
	ID         string
	Goal       string
	AssignedTo string
	Status     TaskStatus
	Result     any
	Error      string
	CreatedAt  time.Time
	StartedAt  time.Time
	FinishedAt time.Time
	Metadata   map[string]string
}

// NewTask creates a task with a generated ID.
func NewTask(goal, assignedTo string) *Task {
	now := time.Now().UTC()
	return &Task{
		ID:         uuid.NewString(),
		Goal:       goal,
		AssignedTo: assignedTo,
		Status:     TaskStatusPending,
		CreatedAt:  now,
	}
}

// Start marks the task as running and sets the start time.
func (t *Task) Start() {
	if t == nil {
		return
	}
	if t.Status == TaskStatusPending {
		t.Status = TaskStatusRunning
	}
	if t.StartedAt.IsZero() {
		t.StartedAt = time.Now().UTC()
	}
}

// Complete marks the task as completed with a result.
func (t *Task) Complete(result any) {
	if t == nil {
		return
	}
	t.Status = TaskStatusCompleted
	t.Result = result
	t.Error = ""
	if t.StartedAt.IsZero() {
		t.StartedAt = time.Now().UTC()
	}
	t.FinishedAt = time.Now().UTC()
}

// Fail marks the task as failed with an error message.
func (t *Task) Fail(err string) {
	if t == nil {
		return
	}
	t.Status = TaskStatusFailed
	t.Error = err
	if t.StartedAt.IsZero() {
		t.StartedAt = time.Now().UTC()
	}
	t.FinishedAt = time.Now().UTC()
}

// Reject marks the task as rejected with an error message.
func (t *Task) Reject(reason string) {
	if t == nil {
		return
	}
	t.Status = TaskStatusRejected
	t.Error = reason
	if t.StartedAt.IsZero() {
		t.StartedAt = time.Now().UTC()
	}
	t.FinishedAt = time.Now().UTC()
}

// Cancel marks the task as cancelled with an optional reason.
func (t *Task) Cancel(reason string) {
	if t == nil {
		return
	}
	t.Status = TaskStatusCancelled
	t.Error = reason
	if t.StartedAt.IsZero() {
		t.StartedAt = time.Now().UTC()
	}
	t.FinishedAt = time.Now().UTC()
}
