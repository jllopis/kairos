// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package telemetry provides OpenTelemetry integration with rich attributes
// for agent observability.
package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
)

// Semantic conventions for Kairos agent telemetry.
// These follow OpenTelemetry naming conventions where applicable.
const (
	// Agent attributes
	AttrAgentID        = "kairos.agent.id"
	AttrAgentRole      = "kairos.agent.role"
	AttrAgentModel     = "kairos.agent.model"
	AttrAgentRunID     = "kairos.agent.run_id"
	AttrAgentIteration = "kairos.agent.iteration"
	AttrAgentMaxIter   = "kairos.agent.max_iterations"

	// Session/Conversation attributes
	AttrSessionID            = "kairos.session.id"
	AttrConversationEnabled  = "kairos.conversation.enabled"
	AttrConversationMsgCount = "kairos.conversation.message_count"
	AttrConversationStrategy = "kairos.conversation.truncation_strategy"

	// Memory attributes
	AttrMemoryEnabled   = "kairos.memory.enabled"
	AttrMemoryType      = "kairos.memory.type"
	AttrMemoryRetrieved = "kairos.memory.retrieved_count"
	AttrMemoryStored    = "kairos.memory.stored"

	// Tool attributes
	AttrToolName       = "kairos.tool.name"
	AttrToolCallID     = "kairos.tool.call_id"
	AttrToolArgs       = "kairos.tool.arguments"
	AttrToolResult     = "kairos.tool.result"
	AttrToolDurationMs = "kairos.tool.duration_ms"
	AttrToolSuccess    = "kairos.tool.success"
	AttrToolSource     = "kairos.tool.source" // "local", "mcp", "skill"

	// Tool set attributes
	AttrToolsCount      = "kairos.tools.count"
	AttrToolsNames      = "kairos.tools.names"
	AttrToolsMCPCount   = "kairos.tools.mcp_count"
	AttrToolsLocalCount = "kairos.tools.local_count"
	AttrToolsSkillCount = "kairos.tools.skill_count"

	// LLM attributes (extending standard gen_ai conventions)
	AttrLLMModel        = "gen_ai.request.model"
	AttrLLMProvider     = "gen_ai.system"
	AttrLLMMessages     = "gen_ai.request.messages"
	AttrLLMTokensInput  = "gen_ai.usage.input_tokens"
	AttrLLMTokensOutput = "gen_ai.usage.output_tokens"
	AttrLLMTokensTotal  = "gen_ai.usage.total_tokens"
	AttrLLMDurationMs   = "gen_ai.duration_ms"
	AttrLLMToolCalls    = "gen_ai.tool_calls"
	AttrLLMFinishReason = "gen_ai.finish_reason"

	// Skill attributes
	AttrSkillName     = "kairos.skill.name"
	AttrSkillAction   = "kairos.skill.action"
	AttrSkillResource = "kairos.skill.resource"

	// Governance attributes
	AttrPolicyEvaluated = "kairos.policy.evaluated"
	AttrPolicyAllowed   = "kairos.policy.allowed"
	AttrPolicyReason    = "kairos.policy.reason"

	// Task attributes
	AttrTaskID     = "kairos.task.id"
	AttrTaskGoal   = "kairos.task.goal"
	AttrTaskStatus = "kairos.task.status"

	// Planner attributes
	AttrPlannerID         = "kairos.planner.id"
	AttrPlannerRunID      = "kairos.planner.run_id"
	AttrPlannerNodeID     = "kairos.planner.node.id"
	AttrPlannerNodeType   = "kairos.planner.node.type"
	AttrPlannerNodeStatus = "kairos.planner.node.status"
	AttrPlannerNodeInput  = "kairos.planner.node.input"
	AttrPlannerNodeOutput = "kairos.planner.node.output"

	// Guardrails attributes
	AttrGuardrailsInputBlocked     = "kairos.guardrails.input.blocked"
	AttrGuardrailsInputID          = "kairos.guardrails.input.id"
	AttrGuardrailsInputConfidence  = "kairos.guardrails.input.confidence"
	AttrGuardrailsOutputModified   = "kairos.guardrails.output.modified"
	AttrGuardrailsOutputRedactions = "kairos.guardrails.output.redactions"

	// Event attributes
	AttrEventType    = "kairos.event.type"
	AttrEventPayload = "kairos.event.payload"
)

// AgentAttributes returns common attributes for agent spans.
func AgentAttributes(agentID, role, model, runID string, iteration, maxIter int) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String(AttrAgentID, agentID),
		attribute.String(AttrAgentRunID, runID),
	}
	if role != "" {
		attrs = append(attrs, attribute.String(AttrAgentRole, role))
	}
	if model != "" {
		attrs = append(attrs, attribute.String(AttrAgentModel, model))
	}
	if iteration > 0 {
		attrs = append(attrs, attribute.Int(AttrAgentIteration, iteration))
	}
	if maxIter > 0 {
		attrs = append(attrs, attribute.Int(AttrAgentMaxIter, maxIter))
	}
	return attrs
}

// SessionAttributes returns attributes for session/conversation tracking.
func SessionAttributes(sessionID string, enabled bool, msgCount int, strategy string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.Bool(AttrConversationEnabled, enabled),
	}
	if sessionID != "" {
		attrs = append(attrs, attribute.String(AttrSessionID, sessionID))
	}
	if enabled {
		attrs = append(attrs, attribute.Int(AttrConversationMsgCount, msgCount))
		if strategy != "" {
			attrs = append(attrs, attribute.String(AttrConversationStrategy, strategy))
		}
	}
	return attrs
}

// MemoryAttributes returns attributes for memory operations.
func MemoryAttributes(enabled bool, memType string, retrieved int, stored bool) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.Bool(AttrMemoryEnabled, enabled),
	}
	if enabled && memType != "" {
		attrs = append(attrs, attribute.String(AttrMemoryType, memType))
	}
	if retrieved > 0 {
		attrs = append(attrs, attribute.Int(AttrMemoryRetrieved, retrieved))
	}
	if stored {
		attrs = append(attrs, attribute.Bool(AttrMemoryStored, stored))
	}
	return attrs
}

// ToolCallAttributes returns attributes for a tool call span.
func ToolCallAttributes(name, callID, source string, durationMs float64, success bool) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrToolName, name),
		attribute.String(AttrToolCallID, callID),
		attribute.String(AttrToolSource, source),
		attribute.Float64(AttrToolDurationMs, durationMs),
		attribute.Bool(AttrToolSuccess, success),
	}
}

// ToolCallArgsResult returns attributes with tool arguments and result (truncated for safety).
func ToolCallArgsResult(args, result string, maxLen int) []attribute.KeyValue {
	if maxLen <= 0 {
		maxLen = 500
	}
	attrs := []attribute.KeyValue{}
	if args != "" {
		if len(args) > maxLen {
			args = args[:maxLen] + "..."
		}
		attrs = append(attrs, attribute.String(AttrToolArgs, args))
	}
	if result != "" {
		if len(result) > maxLen {
			result = result[:maxLen] + "..."
		}
		attrs = append(attrs, attribute.String(AttrToolResult, result))
	}
	return attrs
}

// ToolsetAttributes returns attributes describing the available tools.
func ToolsetAttributes(total, local, mcp, skill int, names []string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.Int(AttrToolsCount, total),
		attribute.Int(AttrToolsLocalCount, local),
		attribute.Int(AttrToolsMCPCount, mcp),
		attribute.Int(AttrToolsSkillCount, skill),
	}
	if len(names) > 0 {
		attrs = append(attrs, attribute.StringSlice(AttrToolsNames, names))
	}
	return attrs
}

// LLMAttributes returns attributes for LLM call spans.
func LLMAttributes(model, provider string, msgCount int, toolCallCount int) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String(AttrLLMModel, model),
		attribute.Int(AttrLLMMessages, msgCount),
	}
	if provider != "" {
		attrs = append(attrs, attribute.String(AttrLLMProvider, provider))
	}
	if toolCallCount > 0 {
		attrs = append(attrs, attribute.Int(AttrLLMToolCalls, toolCallCount))
	}
	return attrs
}

// LLMUsageAttributes returns token usage attributes.
func LLMUsageAttributes(inputTokens, outputTokens int, durationMs float64, finishReason string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{}
	if inputTokens > 0 {
		attrs = append(attrs, attribute.Int(AttrLLMTokensInput, inputTokens))
	}
	if outputTokens > 0 {
		attrs = append(attrs, attribute.Int(AttrLLMTokensOutput, outputTokens))
	}
	if inputTokens > 0 || outputTokens > 0 {
		attrs = append(attrs, attribute.Int(AttrLLMTokensTotal, inputTokens+outputTokens))
	}
	if durationMs > 0 {
		attrs = append(attrs, attribute.Float64(AttrLLMDurationMs, durationMs))
	}
	if finishReason != "" {
		attrs = append(attrs, attribute.String(AttrLLMFinishReason, finishReason))
	}
	return attrs
}

// PlannerAttributes returns attributes for planner executions.
func PlannerAttributes(planID, runID string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{}
	if planID != "" {
		attrs = append(attrs, attribute.String(AttrPlannerID, planID))
	}
	if runID != "" {
		attrs = append(attrs, attribute.String(AttrPlannerRunID, runID))
	}
	return attrs
}

// PlannerNodeAttributes returns attributes for planner node spans.
func PlannerNodeAttributes(nodeID, nodeType, status, planID, runID string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{}
	if planID != "" {
		attrs = append(attrs, attribute.String(AttrPlannerID, planID))
	}
	if runID != "" {
		attrs = append(attrs, attribute.String(AttrPlannerRunID, runID))
	}
	if nodeID != "" {
		attrs = append(attrs, attribute.String(AttrPlannerNodeID, nodeID))
	}
	if nodeType != "" {
		attrs = append(attrs, attribute.String(AttrPlannerNodeType, nodeType))
	}
	if status != "" {
		attrs = append(attrs, attribute.String(AttrPlannerNodeStatus, status))
	}
	return attrs
}

// PlannerNodeIO returns truncated input/output attributes for planner nodes.
func PlannerNodeIO(input, output string, maxLen int) []attribute.KeyValue {
	if maxLen <= 0 {
		maxLen = 200
	}
	attrs := []attribute.KeyValue{}
	if input != "" {
		attrs = append(attrs, attribute.String(AttrPlannerNodeInput, truncateAttr(input, maxLen)))
	}
	if output != "" {
		attrs = append(attrs, attribute.String(AttrPlannerNodeOutput, truncateAttr(output, maxLen)))
	}
	return attrs
}

// GuardrailInputAttributes returns attributes for guardrails input checks.
func GuardrailInputAttributes(blocked bool, guardrailID string, confidence float64) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.Bool(AttrGuardrailsInputBlocked, blocked),
	}
	if guardrailID != "" {
		attrs = append(attrs, attribute.String(AttrGuardrailsInputID, guardrailID))
	}
	if confidence > 0 {
		attrs = append(attrs, attribute.Float64(AttrGuardrailsInputConfidence, confidence))
	}
	return attrs
}

// GuardrailOutputAttributes returns attributes for guardrails output filtering.
func GuardrailOutputAttributes(modified bool, redactions int) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.Bool(AttrGuardrailsOutputModified, modified),
	}
	if redactions > 0 {
		attrs = append(attrs, attribute.Int(AttrGuardrailsOutputRedactions, redactions))
	}
	return attrs
}

// SkillAttributes returns attributes for skill activation spans.
func SkillAttributes(name, action, resource string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String(AttrSkillName, name),
	}
	if action != "" {
		attrs = append(attrs, attribute.String(AttrSkillAction, action))
	}
	if resource != "" {
		attrs = append(attrs, attribute.String(AttrSkillResource, resource))
	}
	return attrs
}

// PolicyAttributes returns attributes for policy evaluation.
func PolicyAttributes(evaluated, allowed bool, reason string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.Bool(AttrPolicyEvaluated, evaluated),
	}
	if evaluated {
		attrs = append(attrs, attribute.Bool(AttrPolicyAllowed, allowed))
		if reason != "" {
			attrs = append(attrs, attribute.String(AttrPolicyReason, reason))
		}
	}
	return attrs
}

// TaskAttributes returns attributes for task tracking.
func TaskAttributes(taskID, goal, status string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{}
	if taskID != "" {
		attrs = append(attrs, attribute.String(AttrTaskID, taskID))
	}
	if goal != "" {
		// Truncate long goals
		if len(goal) > 200 {
			goal = goal[:200] + "..."
		}
		attrs = append(attrs, attribute.String(AttrTaskGoal, goal))
	}
	if status != "" {
		attrs = append(attrs, attribute.String(AttrTaskStatus, status))
	}
	return attrs
}

func truncateAttr(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "..."
}
