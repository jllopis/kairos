package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TaskFilter defines filtering options for listing tasks.
type TaskFilter struct {
	ContextID        string
	Status           a2av1.TaskState
	PageSize         int32
	HistoryLength    int32
	IncludeArtifacts bool
	LastUpdatedAfter time.Time
}

// TaskStore provides access to A2A task records.
type TaskStore interface {
	CreateTask(ctx context.Context, message *a2av1.Message) (*a2av1.Task, error)
	AppendHistory(ctx context.Context, taskID string, message *a2av1.Message) error
	UpdateStatus(ctx context.Context, taskID string, status *a2av1.TaskStatus) error
	AddArtifacts(ctx context.Context, taskID string, artifacts []*a2av1.Artifact) error
	GetTask(ctx context.Context, taskID string, historyLength int32, includeArtifacts bool) (*a2av1.Task, error)
	ListTasks(ctx context.Context, filter TaskFilter) ([]*a2av1.Task, int, error)
	CancelTask(ctx context.Context, taskID string) (*a2av1.Task, error)
}

// MemoryTaskStore keeps tasks in memory for the MVP.
type MemoryTaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*taskRecord
}

type taskRecord struct {
	task      *a2av1.Task
	updatedAt time.Time
}

// NewMemoryTaskStore creates a new in-memory task store.
func NewMemoryTaskStore() *MemoryTaskStore {
	return &MemoryTaskStore{
		tasks: make(map[string]*taskRecord),
	}
}

// CreateTask stores a new task and returns it.
func (s *MemoryTaskStore) CreateTask(ctx context.Context, message *a2av1.Message) (*a2av1.Task, error) {
	if message == nil {
		return nil, fmt.Errorf("message is nil")
	}

	taskID := uuid.NewString()
	contextID := message.ContextId
	if contextID == "" {
		contextID = uuid.NewString()
	}

	message = cloneMessage(message)
	message.TaskId = taskID
	message.ContextId = contextID

	status := newStatus(a2av1.TaskState_TASK_STATE_SUBMITTED, message)
	task := &a2av1.Task{
		Id:        taskID,
		ContextId: contextID,
		Status:    status,
		History:   []*a2av1.Message{message},
	}

	now := time.Now().UTC()
	s.mu.Lock()
	s.tasks[taskID] = &taskRecord{task: task, updatedAt: now}
	s.mu.Unlock()

	return cloneTask(task), nil
}

// AppendHistory adds a message to the task history.
func (s *MemoryTaskStore) AppendHistory(ctx context.Context, taskID string, message *a2av1.Message) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	record.task.History = append(record.task.History, cloneMessage(message))
	record.updatedAt = time.Now().UTC()
	return nil
}

// UpdateStatus updates the task status.
func (s *MemoryTaskStore) UpdateStatus(ctx context.Context, taskID string, status *a2av1.TaskStatus) error {
	if status == nil {
		return fmt.Errorf("status is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	record.task.Status = status
	record.updatedAt = time.Now().UTC()
	return nil
}

// AddArtifacts appends artifacts to the task.
func (s *MemoryTaskStore) AddArtifacts(ctx context.Context, taskID string, artifacts []*a2av1.Artifact) error {
	if len(artifacts) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}
		record.task.Artifacts = append(record.task.Artifacts, artifact)
	}
	record.updatedAt = time.Now().UTC()
	return nil
}

// GetTask returns a task with optional history/artifact filtering.
func (s *MemoryTaskStore) GetTask(ctx context.Context, taskID string, historyLength int32, includeArtifacts bool) (*a2av1.Task, error) {
	s.mu.RLock()
	record, ok := s.tasks[taskID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("task %q not found", taskID)
	}
	return filterTask(record.task, historyLength, includeArtifacts), nil
}

// ListTasks lists tasks with filtering and simple pagination (page size only).
func (s *MemoryTaskStore) ListTasks(ctx context.Context, filter TaskFilter) ([]*a2av1.Task, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*a2av1.Task
	for _, record := range s.tasks {
		if filter.ContextID != "" && record.task.ContextId != filter.ContextID {
			continue
		}
		if filter.Status != a2av1.TaskState_TASK_STATE_UNSPECIFIED && record.task.GetStatus().GetState() != filter.Status {
			continue
		}
		if !filter.LastUpdatedAfter.IsZero() && record.updatedAt.Before(filter.LastUpdatedAfter) {
			continue
		}
		out = append(out, filterTask(record.task, filter.HistoryLength, filter.IncludeArtifacts))
	}

	total := len(out)
	pageSize := int(filter.PageSize)
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize < total {
		out = out[:pageSize]
	}
	return out, total, nil
}

// CancelTask marks a task as cancelled and returns it.
func (s *MemoryTaskStore) CancelTask(ctx context.Context, taskID string) (*a2av1.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task %q not found", taskID)
	}
	status := newStatus(a2av1.TaskState_TASK_STATE_CANCELLED, record.task.GetStatus().GetMessage())
	record.task.Status = status
	record.updatedAt = time.Now().UTC()
	return cloneTask(record.task), nil
}

func newStatus(state a2av1.TaskState, message *a2av1.Message) *a2av1.TaskStatus {
	return &a2av1.TaskStatus{
		State:     state,
		Message:   message,
		Timestamp: timestamppb.Now(),
	}
}

func filterTask(task *a2av1.Task, historyLength int32, includeArtifacts bool) *a2av1.Task {
	cloned := cloneTask(task)
	if !includeArtifacts {
		cloned.Artifacts = nil
	}
	if historyLength > 0 && int(historyLength) < len(cloned.History) {
		cloned.History = cloned.History[len(cloned.History)-int(historyLength):]
	}
	return cloned
}

func cloneTask(task *a2av1.Task) *a2av1.Task {
	if task == nil {
		return nil
	}
	return proto.Clone(task).(*a2av1.Task)
}

func cloneMessage(message *a2av1.Message) *a2av1.Message {
	if message == nil {
		return nil
	}
	return proto.Clone(message).(*a2av1.Message)
}
