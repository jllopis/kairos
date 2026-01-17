// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FileConversation implements ConversationMemory with file-based storage.
// Each session is stored as a separate JSON file.
// Suitable for simple persistence without external dependencies.
type FileConversation struct {
	mu      sync.RWMutex
	baseDir string
	config  ConversationConfig
}

// NewFileConversation creates a new file-based conversation store.
func NewFileConversation(baseDir string, config ConversationConfig) (*FileConversation, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create conversation directory: %w", err)
	}

	return &FileConversation{
		baseDir: baseDir,
		config:  config,
	}, nil
}

func (f *FileConversation) sessionFile(sessionID string) string {
	// Sanitize sessionID to prevent path traversal
	safe := filepath.Base(sessionID)
	return filepath.Join(f.baseDir, safe+".json")
}

// AppendMessage adds a message to the conversation.
func (f *FileConversation) AppendMessage(ctx context.Context, sessionID string, msg ConversationMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.SessionID == "" {
		msg.SessionID = sessionID
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	// Load existing messages
	messages, err := f.loadMessages(sessionID)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load messages: %w", err)
	}

	// Append new message
	messages = append(messages, msg)

	// Save
	return f.saveMessages(sessionID, messages)
}

// GetMessages retrieves all messages for a session.
func (f *FileConversation) GetMessages(ctx context.Context, sessionID string) ([]ConversationMessage, error) {
	f.mu.RLock()
	messages, err := f.loadMessages(sessionID)
	f.mu.RUnlock()

	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Apply truncation strategy if configured
	if f.config.TruncationStrategy != nil && len(messages) > 0 {
		return f.config.TruncationStrategy.Truncate(ctx, messages)
	}

	return messages, nil
}

// GetRecentMessages retrieves the last N messages for a session.
func (f *FileConversation) GetRecentMessages(_ context.Context, sessionID string, limit int) ([]ConversationMessage, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	messages, err := f.loadMessages(sessionID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if len(messages) <= limit {
		return messages, nil
	}

	return messages[len(messages)-limit:], nil
}

// Clear removes all messages for a session.
func (f *FileConversation) Clear(_ context.Context, sessionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	path := f.sessionFile(sessionID)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// DeleteOldMessages removes messages older than the given duration.
func (f *FileConversation) DeleteOldMessages(_ context.Context, sessionID string, olderThan time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	messages, err := f.loadMessages(sessionID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().Add(-olderThan)
	var kept []ConversationMessage
	for _, msg := range messages {
		if msg.CreatedAt.After(cutoff) {
			kept = append(kept, msg)
		}
	}

	if len(kept) == 0 {
		return os.Remove(f.sessionFile(sessionID))
	}

	return f.saveMessages(sessionID, kept)
}

func (f *FileConversation) loadMessages(sessionID string) ([]ConversationMessage, error) {
	path := f.sessionFile(sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var messages []ConversationMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse conversation file: %w", err)
	}

	return messages, nil
}

func (f *FileConversation) saveMessages(sessionID string, messages []ConversationMessage) error {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	path := f.sessionFile(sessionID)
	return os.WriteFile(path, data, 0644)
}

// ListSessions returns all session IDs with stored conversations.
func (f *FileConversation) ListSessions() ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	entries, err := os.ReadDir(f.baseDir)
	if err != nil {
		return nil, err
	}

	var sessions []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".json" {
			sessions = append(sessions, name[:len(name)-5])
		}
	}

	sort.Strings(sessions)
	return sessions, nil
}
