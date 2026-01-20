// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryConversation_AppendAndGet(t *testing.T) {
	mem := NewInMemoryConversation(ConversationConfig{})

	ctx := context.Background()
	sessionID := "test-session"

	// Append messages
	err := mem.AppendMessage(ctx, sessionID, ConversationMessage{
		Role:    "user",
		Content: "Hello",
	})
	if err != nil {
		t.Fatalf("AppendMessage failed: %v", err)
	}

	err = mem.AppendMessage(ctx, sessionID, ConversationMessage{
		Role:    "assistant",
		Content: "Hi there!",
	})
	if err != nil {
		t.Fatalf("AppendMessage failed: %v", err)
	}

	// Get messages
	messages, err := mem.GetMessages(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	if messages[0].Role != "user" || messages[0].Content != "Hello" {
		t.Errorf("unexpected first message: %+v", messages[0])
	}

	if messages[1].Role != "assistant" || messages[1].Content != "Hi there!" {
		t.Errorf("unexpected second message: %+v", messages[1])
	}
}

func TestInMemoryConversation_GetRecentMessages(t *testing.T) {
	mem := NewInMemoryConversation(ConversationConfig{})

	ctx := context.Background()
	sessionID := "test-session"

	// Append 5 messages
	for i := 0; i < 5; i++ {
		err := mem.AppendMessage(ctx, sessionID, ConversationMessage{
			Role:    "user",
			Content: string(rune('A' + i)),
		})
		if err != nil {
			t.Fatalf("AppendMessage failed: %v", err)
		}
	}

	// Get last 3
	messages, err := mem.GetRecentMessages(ctx, sessionID, 3)
	if err != nil {
		t.Fatalf("GetRecentMessages failed: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// Should be C, D, E
	if messages[0].Content != "C" || messages[1].Content != "D" || messages[2].Content != "E" {
		t.Errorf("unexpected messages: %+v", messages)
	}
}

func TestInMemoryConversation_Clear(t *testing.T) {
	mem := NewInMemoryConversation(ConversationConfig{})

	ctx := context.Background()
	sessionID := "test-session"

	mem.AppendMessage(ctx, sessionID, ConversationMessage{Role: "user", Content: "test"})

	if mem.MessageCount(sessionID) != 1 {
		t.Fatal("expected 1 message")
	}

	err := mem.Clear(ctx, sessionID)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if mem.MessageCount(sessionID) != 0 {
		t.Fatal("expected 0 messages after clear")
	}
}

func TestWindowStrategy(t *testing.T) {
	strategy := NewWindowStrategy(3, false)

	messages := []ConversationMessage{
		{Role: "user", Content: "1"},
		{Role: "assistant", Content: "2"},
		{Role: "user", Content: "3"},
		{Role: "assistant", Content: "4"},
		{Role: "user", Content: "5"},
	}

	result, err := strategy.Truncate(context.Background(), messages)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}

	// Should be last 3: 3, 4, 5
	if result[0].Content != "3" || result[1].Content != "4" || result[2].Content != "5" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestWindowStrategy_KeepSystem(t *testing.T) {
	strategy := NewWindowStrategy(3, true)

	messages := []ConversationMessage{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "1"},
		{Role: "assistant", Content: "2"},
		{Role: "user", Content: "3"},
		{Role: "assistant", Content: "4"},
	}

	result, err := strategy.Truncate(context.Background(), messages)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}

	// Should be: system, 3, 4
	if result[0].Role != "system" {
		t.Error("first message should be system")
	}
	if result[1].Content != "3" || result[2].Content != "4" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestTokenStrategy(t *testing.T) {
	strategy := NewTokenStrategy(20, false) // Smaller budget to force truncation
	strategy.TokenCounter = func(msg ConversationMessage) int {
		return len(msg.Content)
	}

	messages := []ConversationMessage{
		{Role: "user", Content: "This is a long message"},       // 22 chars
		{Role: "assistant", Content: "Short"},                   // 5 chars
		{Role: "user", Content: "Also short"},                   // 10 chars
	}

	result, err := strategy.Truncate(context.Background(), messages)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Budget is 20, last two are 5+10=15, fits. First is 22, doesn't fit alone.
	// So we keep last two messages.
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d: %+v", len(result), result)
	}

	if result[0].Content != "Short" || result[1].Content != "Also short" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestInMemoryConversation_WithTruncation(t *testing.T) {
	strategy := NewWindowStrategy(2, false)
	mem := NewInMemoryConversation(ConversationConfig{
		TruncationStrategy: strategy,
	})

	ctx := context.Background()
	sessionID := "test-session"

	// Append 4 messages
	for i := 0; i < 4; i++ {
		mem.AppendMessage(ctx, sessionID, ConversationMessage{
			Role:    "user",
			Content: string(rune('A' + i)),
		})
	}

	// GetMessages should apply truncation
	messages, err := mem.GetMessages(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	// Should return only last 2 due to window strategy
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages after truncation, got %d", len(messages))
	}

	if messages[0].Content != "C" || messages[1].Content != "D" {
		t.Errorf("unexpected truncated messages: %+v", messages)
	}
}

func TestInMemoryConversation_DeleteOldMessages(t *testing.T) {
	mem := NewInMemoryConversation(ConversationConfig{})

	ctx := context.Background()
	sessionID := "test-session"

	// Add old message
	oldMsg := ConversationMessage{
		Role:      "user",
		Content:   "old",
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	mem.AppendMessage(ctx, sessionID, oldMsg)

	// Add recent message
	newMsg := ConversationMessage{
		Role:    "user",
		Content: "new",
	}
	mem.AppendMessage(ctx, sessionID, newMsg)

	// Delete messages older than 1 hour
	err := mem.DeleteOldMessages(ctx, sessionID, time.Hour)
	if err != nil {
		t.Fatalf("DeleteOldMessages failed: %v", err)
	}

	messages, _ := mem.GetMessages(ctx, sessionID)
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "new" {
		t.Errorf("wrong message kept: %+v", messages[0])
	}
}

func TestInMemoryConversation_MultipleSessions(t *testing.T) {
	mem := NewInMemoryConversation(ConversationConfig{})

	ctx := context.Background()

	mem.AppendMessage(ctx, "session-1", ConversationMessage{Role: "user", Content: "s1-msg"})
	mem.AppendMessage(ctx, "session-2", ConversationMessage{Role: "user", Content: "s2-msg"})

	sessions := mem.ListSessions()
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	s1msgs, _ := mem.GetMessages(ctx, "session-1")
	s2msgs, _ := mem.GetMessages(ctx, "session-2")

	if len(s1msgs) != 1 || s1msgs[0].Content != "s1-msg" {
		t.Errorf("unexpected session-1 messages: %+v", s1msgs)
	}

	if len(s2msgs) != 1 || s2msgs[0].Content != "s2-msg" {
		t.Errorf("unexpected session-2 messages: %+v", s2msgs)
	}
}
