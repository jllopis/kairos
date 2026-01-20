// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryConversation implements ConversationMemory with in-memory storage.
// Suitable for development, testing, and single-instance deployments.
// Data is lost on restart.
type InMemoryConversation struct {
	mu       sync.RWMutex
	sessions map[string][]ConversationMessage
	config   ConversationConfig
}

// NewInMemoryConversation creates a new in-memory conversation store.
func NewInMemoryConversation(config ConversationConfig) *InMemoryConversation {
	return &InMemoryConversation{
		sessions: make(map[string][]ConversationMessage),
		config:   config,
	}
}

// AppendMessage adds a message to the conversation.
func (m *InMemoryConversation) AppendMessage(_ context.Context, sessionID string, msg ConversationMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.SessionID == "" {
		msg.SessionID = sessionID
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	m.sessions[sessionID] = append(m.sessions[sessionID], msg)
	return nil
}

// GetMessages retrieves all messages for a session.
func (m *InMemoryConversation) GetMessages(ctx context.Context, sessionID string) ([]ConversationMessage, error) {
	m.mu.RLock()
	messages := make([]ConversationMessage, len(m.sessions[sessionID]))
	copy(messages, m.sessions[sessionID])
	m.mu.RUnlock()

	// Apply truncation strategy if configured
	if m.config.TruncationStrategy != nil && len(messages) > 0 {
		return m.config.TruncationStrategy.Truncate(ctx, messages)
	}

	return messages, nil
}

// GetRecentMessages retrieves the last N messages for a session.
func (m *InMemoryConversation) GetRecentMessages(_ context.Context, sessionID string, limit int) ([]ConversationMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	all := m.sessions[sessionID]
	if len(all) <= limit {
		result := make([]ConversationMessage, len(all))
		copy(result, all)
		return result, nil
	}

	result := make([]ConversationMessage, limit)
	copy(result, all[len(all)-limit:])
	return result, nil
}

// Clear removes all messages for a session.
func (m *InMemoryConversation) Clear(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, sessionID)
	return nil
}

// DeleteOldMessages removes messages older than the given duration.
func (m *InMemoryConversation) DeleteOldMessages(_ context.Context, sessionID string, olderThan time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	messages, ok := m.sessions[sessionID]
	if !ok {
		return nil
	}

	cutoff := time.Now().Add(-olderThan)
	var kept []ConversationMessage
	for _, msg := range messages {
		if msg.CreatedAt.After(cutoff) {
			kept = append(kept, msg)
		}
	}

	m.sessions[sessionID] = kept
	return nil
}

// ListSessions returns all active session IDs.
func (m *InMemoryConversation) ListSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// MessageCount returns the number of messages in a session.
func (m *InMemoryConversation) MessageCount(sessionID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions[sessionID])
}
