// Package memory provides memory backends and embedding helpers.
package memory

import (
	"context"
	"errors"
	"sync"
)

// ErrNotFound indicates no matching item was found.
var ErrNotFound = errors.New("memory: not found")

// InMemory is a simple in-process memory backend.
type InMemory struct {
	mu   sync.RWMutex
	data []any
}

// NewInMemory creates an empty in-memory store.
func NewInMemory() *InMemory {
	return &InMemory{}
}

// Store appends data to the memory.
func (m *InMemory) Store(_ context.Context, data any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = append(m.data, data)
	return nil
}

// Retrieve returns the most recent match. If query is nil, returns the last item.
// If query is a func(any) bool, it returns the last item that satisfies it.
func (m *InMemory) Retrieve(_ context.Context, query any) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.data) == 0 {
		return nil, ErrNotFound
	}

	if query == nil {
		return m.data[len(m.data)-1], nil
	}

	if match, ok := query.(func(any) bool); ok {
		for i := len(m.data) - 1; i >= 0; i-- {
			if match(m.data[i]) {
				return m.data[i], nil
			}
		}
		return nil, ErrNotFound
	}

	return nil, errors.New("memory: unsupported query type")
}
