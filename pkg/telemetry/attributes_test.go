// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestAgentAttributes(t *testing.T) {
	attrs := AgentAttributes("test-agent", "assistant", "gpt-4", "run-123", 2, 10)

	expected := map[string]any{
		AttrAgentID:        "test-agent",
		AttrAgentRunID:     "run-123",
		AttrAgentRole:      "assistant",
		AttrAgentModel:     "gpt-4",
		AttrAgentIteration: 2,
		AttrAgentMaxIter:   10,
	}

	assertAttributes(t, attrs, expected)
}

func TestSessionAttributes(t *testing.T) {
	attrs := SessionAttributes("session-123", true, 5, "window")

	expected := map[string]any{
		AttrConversationEnabled:  true,
		AttrSessionID:            "session-123",
		AttrConversationMsgCount: 5,
		AttrConversationStrategy: "window",
	}

	assertAttributes(t, attrs, expected)
}

func TestMemoryAttributes(t *testing.T) {
	attrs := MemoryAttributes(true, "vector", 3, true)

	expected := map[string]any{
		AttrMemoryEnabled:   true,
		AttrMemoryType:      "vector",
		AttrMemoryRetrieved: 3,
		AttrMemoryStored:    true,
	}

	assertAttributes(t, attrs, expected)
}

func TestToolCallAttributes(t *testing.T) {
	attrs := ToolCallAttributes("search", "call-1", "mcp", 150.5, true)

	expected := map[string]any{
		AttrToolName:       "search",
		AttrToolCallID:     "call-1",
		AttrToolSource:     "mcp",
		AttrToolDurationMs: 150.5,
		AttrToolSuccess:    true,
	}

	assertAttributes(t, attrs, expected)
}

func TestToolCallArgsResult_Truncation(t *testing.T) {
	longArgs := string(make([]byte, 600)) // 600 chars
	longResult := string(make([]byte, 700))

	attrs := ToolCallArgsResult(longArgs, longResult, 500)

	for _, attr := range attrs {
		val := attr.Value.AsString()
		if len(val) > 504 { // 500 + "..."
			t.Errorf("attribute %s not truncated: len=%d", attr.Key, len(val))
		}
	}
}

func TestToolsetAttributes(t *testing.T) {
	attrs := ToolsetAttributes(5, 2, 2, 1, []string{"search", "calc", "read"})

	expected := map[string]any{
		AttrToolsCount:      5,
		AttrToolsLocalCount: 2,
		AttrToolsMCPCount:   2,
		AttrToolsSkillCount: 1,
	}

	assertAttributes(t, attrs, expected)

	// Check names slice
	for _, attr := range attrs {
		if string(attr.Key) == AttrToolsNames {
			names := attr.Value.AsStringSlice()
			if len(names) != 3 {
				t.Errorf("expected 3 tool names, got %d", len(names))
			}
		}
	}
}

func TestLLMAttributes(t *testing.T) {
	attrs := LLMAttributes("gpt-4", "openai", 5, 2)

	expected := map[string]any{
		AttrLLMModel:     "gpt-4",
		AttrLLMProvider:  "openai",
		AttrLLMMessages:  5,
		AttrLLMToolCalls: 2,
	}

	assertAttributes(t, attrs, expected)
}

func TestLLMUsageAttributes(t *testing.T) {
	attrs := LLMUsageAttributes(100, 50, 1500.0, "stop")

	expected := map[string]any{
		AttrLLMTokensInput:  100,
		AttrLLMTokensOutput: 50,
		AttrLLMTokensTotal:  150,
		AttrLLMDurationMs:   1500.0,
		AttrLLMFinishReason: "stop",
	}

	assertAttributes(t, attrs, expected)
}

func TestSkillAttributes(t *testing.T) {
	attrs := SkillAttributes("web-search", "activate", "")

	expected := map[string]any{
		AttrSkillName:   "web-search",
		AttrSkillAction: "activate",
	}

	assertAttributes(t, attrs, expected)
}

func TestPolicyAttributes(t *testing.T) {
	attrs := PolicyAttributes(true, false, "access denied")

	expected := map[string]any{
		AttrPolicyEvaluated: true,
		AttrPolicyAllowed:   false,
		AttrPolicyReason:    "access denied",
	}

	assertAttributes(t, attrs, expected)
}

func TestTaskAttributes(t *testing.T) {
	attrs := TaskAttributes("task-123", "Summarize document", "running")

	expected := map[string]any{
		AttrTaskID:     "task-123",
		AttrTaskGoal:   "Summarize document",
		AttrTaskStatus: "running",
	}

	assertAttributes(t, attrs, expected)
}

func TestTaskAttributes_GoalTruncation(t *testing.T) {
	longGoal := string(make([]byte, 300))
	attrs := TaskAttributes("task-123", longGoal, "running")

	for _, attr := range attrs {
		if string(attr.Key) == AttrTaskGoal {
			val := attr.Value.AsString()
			if len(val) > 204 { // 200 + "..."
				t.Errorf("goal not truncated: len=%d", len(val))
			}
		}
	}
}

// assertAttributes checks that expected key-value pairs exist in attrs
func assertAttributes(t *testing.T, attrs []attribute.KeyValue, expected map[string]any) {
	t.Helper()

	found := make(map[string]attribute.KeyValue)
	for _, attr := range attrs {
		found[string(attr.Key)] = attr
	}

	for key, expectedVal := range expected {
		attr, ok := found[key]
		if !ok {
			t.Errorf("missing attribute %s", key)
			continue
		}

		var actualVal any
		switch attr.Value.Type() {
		case attribute.STRING:
			actualVal = attr.Value.AsString()
		case attribute.INT64:
			actualVal = int(attr.Value.AsInt64())
		case attribute.FLOAT64:
			actualVal = attr.Value.AsFloat64()
		case attribute.BOOL:
			actualVal = attr.Value.AsBool()
		}

		if actualVal != expectedVal {
			t.Errorf("attribute %s: got %v, want %v", key, actualVal, expectedVal)
		}
	}
}
