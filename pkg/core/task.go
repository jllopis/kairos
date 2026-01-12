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
