// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package memory provides memory backends for agents.
package memory

import (
	"context"
	"strconv"
	"time"
)

// ConversationMessage represents a single message in a conversation history.
type ConversationMessage struct {
	ID         string            `json:"id"`
	SessionID  string            `json:"session_id"`
	Role       string            `json:"role"` // system, user, assistant, tool
	Content    string            `json:"content"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// ConversationMemory stores and retrieves conversation history for multi-turn interactions.
// Unlike semantic Memory (vector-based retrieval), this maintains ordered message sequences.
type ConversationMemory interface {
	// AppendMessage adds a message to the conversation.
	AppendMessage(ctx context.Context, sessionID string, msg ConversationMessage) error

	// GetMessages retrieves all messages for a session, ordered by creation time.
	GetMessages(ctx context.Context, sessionID string) ([]ConversationMessage, error)

	// GetRecentMessages retrieves the last N messages for a session.
	GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]ConversationMessage, error)

	// Clear removes all messages for a session.
	Clear(ctx context.Context, sessionID string) error

	// DeleteOldMessages removes messages older than the given duration.
	DeleteOldMessages(ctx context.Context, sessionID string, olderThan time.Duration) error
}

// TruncationStrategy defines how to manage conversation length.
type TruncationStrategy interface {
	// Truncate applies the strategy to reduce messages while preserving context.
	// Returns the truncated message list.
	Truncate(ctx context.Context, messages []ConversationMessage) ([]ConversationMessage, error)
}

// WindowStrategy keeps only the last N messages.
type WindowStrategy struct {
	MaxMessages int
	// KeepSystemMessages preserves system messages regardless of window.
	KeepSystemMessages bool
}

// Truncate implements TruncationStrategy.
func (w *WindowStrategy) Truncate(_ context.Context, messages []ConversationMessage) ([]ConversationMessage, error) {
	if len(messages) <= w.MaxMessages {
		return messages, nil
	}

	if !w.KeepSystemMessages {
		// Simple: keep last N messages
		return messages[len(messages)-w.MaxMessages:], nil
	}

	// Extract system messages and non-system messages
	var systemMsgs []ConversationMessage
	var otherMsgs []ConversationMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			systemMsgs = append(systemMsgs, msg)
		} else {
			otherMsgs = append(otherMsgs, msg)
		}
	}

	// Calculate how many non-system messages we can keep
	available := w.MaxMessages - len(systemMsgs)
	if available < 0 {
		available = 0
	}

	if len(otherMsgs) > available {
		otherMsgs = otherMsgs[len(otherMsgs)-available:]
	}

	// Reconstruct: system messages first, then recent messages
	result := make([]ConversationMessage, 0, len(systemMsgs)+len(otherMsgs))
	result = append(result, systemMsgs...)
	result = append(result, otherMsgs...)

	return result, nil
}

// TokenStrategy keeps messages that fit within a token budget.
type TokenStrategy struct {
	MaxTokens int
	// TokenCounter estimates tokens for a message. If nil, uses len(content)/4 approximation.
	TokenCounter func(msg ConversationMessage) int
	// KeepSystemMessages preserves system messages regardless of budget.
	KeepSystemMessages bool
}

// Truncate implements TruncationStrategy.
func (t *TokenStrategy) Truncate(_ context.Context, messages []ConversationMessage) ([]ConversationMessage, error) {
	counter := t.TokenCounter
	if counter == nil {
		counter = func(msg ConversationMessage) int {
			return len(msg.Content) / 4 // Rough approximation
		}
	}

	// Count total tokens
	totalTokens := 0
	for _, msg := range messages {
		totalTokens += counter(msg)
	}

	if totalTokens <= t.MaxTokens {
		return messages, nil
	}

	// Separate system and other messages
	var systemMsgs []ConversationMessage
	var otherMsgs []ConversationMessage
	systemTokens := 0

	for _, msg := range messages {
		if msg.Role == "system" && t.KeepSystemMessages {
			systemMsgs = append(systemMsgs, msg)
			systemTokens += counter(msg)
		} else {
			otherMsgs = append(otherMsgs, msg)
		}
	}

	// Available budget for non-system messages
	budget := t.MaxTokens - systemTokens
	if budget < 0 {
		budget = 0
	}

	// Keep messages from the end until budget exhausted
	var kept []ConversationMessage
	currentTokens := 0

	for i := len(otherMsgs) - 1; i >= 0; i-- {
		msgTokens := counter(otherMsgs[i])
		if currentTokens+msgTokens > budget {
			break
		}
		kept = append([]ConversationMessage{otherMsgs[i]}, kept...)
		currentTokens += msgTokens
	}

	// Reconstruct
	result := make([]ConversationMessage, 0, len(systemMsgs)+len(kept))
	result = append(result, systemMsgs...)
	result = append(result, kept...)

	return result, nil
}

// SummarizationStrategy summarizes old messages to reduce length.
type SummarizationStrategy struct {
	// MaxMessages triggers summarization when exceeded.
	MaxMessages int
	// SummarizeCount is how many old messages to summarize at once.
	SummarizeCount int
	// Summarizer generates a summary from messages. Required.
	Summarizer func(ctx context.Context, messages []ConversationMessage) (string, error)
	// KeepSystemMessages preserves system messages from summarization.
	KeepSystemMessages bool
}

// Truncate implements TruncationStrategy.
func (s *SummarizationStrategy) Truncate(ctx context.Context, messages []ConversationMessage) ([]ConversationMessage, error) {
	if len(messages) <= s.MaxMessages || s.Summarizer == nil {
		return messages, nil
	}

	// Separate system messages if needed
	var systemMsgs []ConversationMessage
	var otherMsgs []ConversationMessage

	if s.KeepSystemMessages {
		for _, msg := range messages {
			if msg.Role == "system" {
				systemMsgs = append(systemMsgs, msg)
			} else {
				otherMsgs = append(otherMsgs, msg)
			}
		}
	} else {
		otherMsgs = messages
	}

	if len(otherMsgs) <= s.MaxMessages {
		result := make([]ConversationMessage, 0, len(systemMsgs)+len(otherMsgs))
		result = append(result, systemMsgs...)
		result = append(result, otherMsgs...)
		return result, nil
	}

	// Determine how many messages to summarize
	toSummarize := s.SummarizeCount
	if toSummarize > len(otherMsgs)-s.MaxMessages {
		toSummarize = len(otherMsgs) - s.MaxMessages + 1 // +1 for the summary message
	}
	if toSummarize < 2 {
		toSummarize = 2
	}

	// Get messages to summarize (oldest ones)
	summarizeThese := otherMsgs[:toSummarize]
	keepThese := otherMsgs[toSummarize:]

	// Generate summary
	summary, err := s.Summarizer(ctx, summarizeThese)
	if err != nil {
		return messages, err // Return original on error
	}

	// Create summary message
	summaryMsg := ConversationMessage{
		Role:      "system",
		Content:   "[Previous conversation summary]\n" + summary,
		CreatedAt: summarizeThese[0].CreatedAt,
		Metadata:  map[string]string{"type": "summary", "summarized_count": strconv.Itoa(toSummarize)},
	}

	// Reconstruct
	result := make([]ConversationMessage, 0, len(systemMsgs)+1+len(keepThese))
	result = append(result, systemMsgs...)
	result = append(result, summaryMsg)
	result = append(result, keepThese...)

	return result, nil
}

// ConversationConfig configures conversation memory behavior.
type ConversationConfig struct {
	// TruncationStrategy to apply when loading messages. Optional.
	TruncationStrategy TruncationStrategy
	// DefaultSessionTTL is how long to keep inactive sessions. Zero means forever.
	DefaultSessionTTL time.Duration
}

// NewWindowStrategy creates a window-based truncation strategy.
func NewWindowStrategy(maxMessages int, keepSystem bool) *WindowStrategy {
	return &WindowStrategy{
		MaxMessages:        maxMessages,
		KeepSystemMessages: keepSystem,
	}
}

// NewTokenStrategy creates a token-based truncation strategy.
func NewTokenStrategy(maxTokens int, keepSystem bool) *TokenStrategy {
	return &TokenStrategy{
		MaxTokens:          maxTokens,
		KeepSystemMessages: keepSystem,
	}
}

// NewSummarizationStrategy creates a summarization-based truncation strategy.
func NewSummarizationStrategy(maxMessages, summarizeCount int, summarizer func(ctx context.Context, messages []ConversationMessage) (string, error)) *SummarizationStrategy {
	return &SummarizationStrategy{
		MaxMessages:        maxMessages,
		SummarizeCount:     summarizeCount,
		Summarizer:         summarizer,
		KeepSystemMessages: true,
	}
}
