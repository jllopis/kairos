package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/protobuf/proto"
)

// ApprovalStatus captures the lifecycle of a human approval.
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

// ApprovalRecord stores approval state for a pending task execution.
type ApprovalRecord struct {
	ID        string         `json:"id"`
	TaskID    string         `json:"task_id"`
	ContextID string         `json:"context_id"`
	Status    ApprovalStatus `json:"status"`
	Reason    string         `json:"reason,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	ExpiresAt time.Time      `json:"expires_at,omitempty"`
	Message   *a2av1.Message `json:"-"`
}

// ApprovalFilter limits approval queries.
type ApprovalFilter struct {
	TaskID         string
	ContextID      string
	Status         ApprovalStatus
	Limit          int
	ExpiringBefore time.Time
}

// ApprovalStore persists approval records.
type ApprovalStore interface {
	Create(ctx context.Context, record ApprovalRecord) (*ApprovalRecord, error)
	Get(ctx context.Context, id string) (*ApprovalRecord, error)
	List(ctx context.Context, filter ApprovalFilter) ([]*ApprovalRecord, error)
	UpdateStatus(ctx context.Context, id string, status ApprovalStatus, reason string) (*ApprovalRecord, error)
}

// ApprovalHandler exposes approval operations for HTTP/JSON bindings.
type ApprovalHandler interface {
	GetApproval(ctx context.Context, id string) (*ApprovalRecord, error)
	ListApprovals(ctx context.Context, filter ApprovalFilter) ([]*ApprovalRecord, error)
	Approve(ctx context.Context, id, reason string) (*a2av1.Task, error)
	Reject(ctx context.Context, id, reason string) (*a2av1.Task, error)
}

// MemoryApprovalStore keeps approvals in memory.
type MemoryApprovalStore struct {
	mu        sync.RWMutex
	approvals map[string]*ApprovalRecord
}

// NewMemoryApprovalStore creates an in-memory approval store.
func NewMemoryApprovalStore() *MemoryApprovalStore {
	return &MemoryApprovalStore{approvals: make(map[string]*ApprovalRecord)}
}

// Create inserts a new approval record.
func (s *MemoryApprovalStore) Create(_ context.Context, record ApprovalRecord) (*ApprovalRecord, error) {
	if record.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.Status == "" {
		record.Status = ApprovalStatusPending
	}
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	if record.ExpiresAt.IsZero() {
		record.ExpiresAt = time.Time{}
	}
	copied := cloneApproval(&record)
	s.mu.Lock()
	s.approvals[record.ID] = copied
	s.mu.Unlock()
	return cloneApproval(copied), nil
}

// Get returns an approval record by id.
func (s *MemoryApprovalStore) Get(_ context.Context, id string) (*ApprovalRecord, error) {
	s.mu.RLock()
	record, ok := s.approvals[id]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("approval %q not found", id)
	}
	return cloneApproval(record), nil
}

// List returns approvals matching the filter.
func (s *MemoryApprovalStore) List(_ context.Context, filter ApprovalFilter) ([]*ApprovalRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ApprovalRecord, 0)
	for _, record := range s.approvals {
		if filter.TaskID != "" && record.TaskID != filter.TaskID {
			continue
		}
		if filter.ContextID != "" && record.ContextID != filter.ContextID {
			continue
		}
		if filter.Status != "" && record.Status != filter.Status {
			continue
		}
		if !filter.ExpiringBefore.IsZero() {
			if record.ExpiresAt.IsZero() || record.ExpiresAt.After(filter.ExpiringBefore) {
				continue
			}
		}
		out = append(out, cloneApproval(record))
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

// UpdateStatus updates the approval status.
func (s *MemoryApprovalStore) UpdateStatus(_ context.Context, id string, status ApprovalStatus, reason string) (*ApprovalRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.approvals[id]
	if !ok {
		return nil, fmt.Errorf("approval %q not found", id)
	}
	record.Status = status
	record.Reason = reason
	record.UpdatedAt = time.Now().UTC()
	return cloneApproval(record), nil
}

func cloneApproval(record *ApprovalRecord) *ApprovalRecord {
	if record == nil {
		return nil
	}
	out := *record
	if record.Message != nil {
		out.Message = proto.Clone(record.Message).(*a2av1.Message)
	}
	return &out
}
