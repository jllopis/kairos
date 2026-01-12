package planner

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// AuditStore persists planner audit events.
type AuditStore interface {
	Record(ctx context.Context, event AuditEvent) error
	List(ctx context.Context, filter AuditFilter) ([]AuditEvent, error)
}

// AuditFilter limits audit event queries.
type AuditFilter struct {
	GraphID string
	NodeID  string
	Status  string
	Limit   int
}

// MemoryAuditStore keeps audit events in memory.
type MemoryAuditStore struct {
	mu     sync.Mutex
	events []AuditEvent
}

// NewMemoryAuditStore returns an in-memory audit store.
func NewMemoryAuditStore() *MemoryAuditStore {
	return &MemoryAuditStore{}
}

// Record appends an audit event.
func (s *MemoryAuditStore) Record(_ context.Context, event AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

// List returns filtered audit events.
func (s *MemoryAuditStore) List(_ context.Context, filter AuditFilter) ([]AuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AuditEvent, 0, len(s.events))
	for _, ev := range s.events {
		if filter.GraphID != "" && ev.GraphID != filter.GraphID {
			continue
		}
		if filter.NodeID != "" && ev.NodeID != filter.NodeID {
			continue
		}
		if filter.Status != "" && ev.Status != filter.Status {
			continue
		}
		out = append(out, ev)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

// encodeAuditOutput marshals the output payload into JSON.
func encodeAuditOutput(output any) ([]byte, error) {
	if output == nil {
		return []byte("null"), nil
	}
	return json.Marshal(output)
}

// decodeAuditOutput parses JSON output payload.
func decodeAuditOutput(raw []byte) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// normalizeAuditTime ensures timestamps are in UTC.
func normalizeAuditTime(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return value.UTC()
}
