// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jllopis/kairos/pkg/core"
	kerrors "github.com/jllopis/kairos/pkg/errors"
	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/pkg/planner"
	"github.com/jllopis/kairos/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	plannerNodeTool     = "tool"
	plannerNodeAgent    = "agent"
	plannerNodeLLM      = "llm"
	plannerNodeNoop     = "noop"
	plannerNodeDecision = "decision"
)

func (a *Agent) runPlanner(ctx context.Context, input any) (any, error) {
	if a.plannerGraph == nil {
		return a.runEmergent(ctx, input)
	}

	ctx, runID := core.EnsureRunID(ctx)
	ctx, span := a.tracer.Start(ctx, "Agent.Run")
	defer span.End()
	traceID, spanID := traceIDs(span)
	log := slog.Default()

	inputStr, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("agent currently only supports string input")
	}

	span.SetAttributes(telemetry.AgentAttributes(a.id, a.role, a.model, runID, 0, a.maxIterations)...)
	span.SetAttributes(telemetry.PlannerAttributes(a.plannerGraph.ID, runID)...)

	if err := a.checkGuardrailsInput(ctx, log, runID, traceID, spanID, inputStr); err != nil {
		agentErrorCounter.Add(ctx, 1)
		if task, ok := core.TaskFromContext(ctx); ok && task != nil {
			task.Fail(err.Error())
		}
		return nil, err
	}

	if task, ok := core.TaskFromContext(ctx); ok && task != nil {
		if task.Goal == "" {
			task.Goal = inputStr
		}
		if task.AssignedTo == "" {
			task.AssignedTo = a.id
		}
		task.Start()
		span.SetAttributes(telemetry.TaskAttributes(task.ID, task.Goal, string(task.Status))...)
	}

	initAgentMetrics()
	agentRunCounter.Add(ctx, 1)
	start := time.Now()

	sessionID, hasSession := core.SessionID(ctx)
	if a.conversationMemory != nil && !hasSession {
		ctx, sessionID = core.EnsureSessionID(ctx)
		hasSession = true
	}
	span.SetAttributes(telemetry.SessionAttributes(sessionID, a.conversationMemory != nil, 0, "")...)

	log.Info("agent.run.start",
		slog.String("agent_id", a.id),
		slog.String("run_id", runID),
		slog.String("trace_id", traceID),
		slog.String("span_id", spanID),
		slog.String("session_id", sessionID),
		slog.String("planner", "explicit"),
	)
	a.emitEvent(ctx, core.EventAgentTaskStarted, map[string]any{
		"run_id":     runID,
		"session_id": sessionID,
		"planner":    "explicit",
	})

	if a.conversationMemory != nil && hasSession {
		if err := a.storeConversationMessage(ctx, sessionID, llm.RoleUser, inputStr, ""); err != nil {
			log.Warn("agent.conversation.store_error",
				slog.String("agent_id", a.id),
				slog.String("session_id", sessionID),
				slog.String("error", err.Error()),
			)
		}
	}

	toolset := a.resolveTools(ctx, log, runID)
	localCount, mcpCount, skillCount := a.countToolsBySource(toolset)
	span.SetAttributes(telemetry.ToolsetAttributes(len(toolset), localCount, mcpCount, skillCount, toolNames(toolset))...)

	mem := a.resolveMemory(ctx)
	memoryRetrieved := 0
	state := planner.NewState()
	state.Last = inputStr
	state.Outputs["input"] = inputStr
	if mem != nil {
		if memoryContext := a.loadMemoryContext(ctx, mem, inputStr); memoryContext != "" {
			state.Outputs["memory"] = memoryContext
			memoryRetrieved = 1
		}
	}
	span.SetAttributes(telemetry.MemoryAttributes(mem != nil, fmt.Sprintf("%T", mem), memoryRetrieved, false)...)

	exec := planner.NewExecutor(a.buildPlannerHandlers(toolset, log, runID, traceID, spanID))
	exec.RunID = runID
	exec.AuditStore = a.plannerAuditStore
	exec.AuditHook = a.plannerAuditHook

	result, err := exec.Execute(ctx, a.plannerGraph, state)
	if err != nil {
		agentErrorCounter.Add(ctx, 1)
		var ke *kerrors.KairosError
		if !stderrors.As(err, &ke) {
			ke = WrapPlannerError(err, a.plannerGraph.ID)
		}
		if em := GetErrorMetrics(); em != nil {
			em.RecordError(ctx, ke, "agent-planner")
		}
		log.Error("agent.planner.error",
			slog.String("agent_id", a.id),
			slog.String("run_id", runID),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.String("error", err.Error()),
			slog.String("error_code", string(kerrors.CodeInternal)),
		)
		a.emitEvent(ctx, core.EventAgentError, map[string]any{
			"run_id":  runID,
			"stage":   "planner",
			"planner": a.plannerGraph.ID,
			"error":   err.Error(),
		})
		if task, ok := core.TaskFromContext(ctx); ok && task != nil {
			task.Fail(err.Error())
		}
		return nil, ke
	}

	output := result.Last
	outputStr := strings.TrimSpace(fmt.Sprint(output))
	outputStr = a.applyGuardrailsOutput(ctx, log, runID, traceID, spanID, outputStr)
	a.storeMemory(ctx, mem, inputStr, outputStr)
	if a.conversationMemory != nil && hasSession {
		if err := a.storeConversationMessage(ctx, sessionID, llm.RoleAssistant, outputStr, ""); err != nil {
			log.Warn("agent.conversation.store_error",
				slog.String("agent_id", a.id),
				slog.String("session_id", sessionID),
				slog.String("error", err.Error()),
			)
		}
	}
	agentRunLatencyMs.Record(ctx, time.Since(start).Seconds()*1000)
	log.Info("agent.run.complete",
		slog.String("agent_id", a.id),
		slog.String("run_id", runID),
		slog.String("trace_id", traceID),
		slog.String("span_id", spanID),
		slog.String("planner", "explicit"),
	)
	a.emitEvent(ctx, core.EventAgentTaskCompleted, map[string]any{
		"run_id":  runID,
		"planner": "explicit",
		"result":  outputStr,
	})
	if task, ok := core.TaskFromContext(ctx); ok && task != nil {
		task.Complete(outputStr)
	}
	return output, nil
}

func (a *Agent) buildPlannerHandlers(toolset []core.Tool, log *slog.Logger, runID, traceID, spanID string) map[string]planner.Handler {
	handlers := map[string]planner.Handler{
		plannerNodeTool:     a.plannerToolHandler(toolset, log, runID, traceID, spanID),
		plannerNodeAgent:    a.plannerAgentHandler(log, runID, traceID, spanID),
		plannerNodeLLM:      a.plannerLLMHandler(log, runID, traceID, spanID),
		plannerNodeNoop:     func(_ context.Context, _ planner.Node, state *planner.State) (any, error) { return state.Last, nil },
		plannerNodeDecision: func(_ context.Context, _ planner.Node, state *planner.State) (any, error) { return state.Last, nil },
	}

	for _, tool := range toolset {
		if _, ok := handlers[tool.Name()]; ok {
			continue
		}
		handlers[tool.Name()] = a.plannerNamedToolHandler(tool, log, runID, traceID, spanID)
	}

	alias := map[string]string{
		"init":          plannerNodeNoop,
		"validation":    plannerNodeDecision,
		"llm_call":      plannerNodeLLM,
		"format_output": plannerNodeNoop,
		"error_handler": plannerNodeNoop,
		"terminal":      plannerNodeNoop,
	}
	for name, target := range alias {
		if _, ok := handlers[name]; ok {
			continue
		}
		if handler, ok := handlers[target]; ok {
			handlers[name] = handler
		}
	}

	if len(a.plannerHandlers) > 0 {
		for name, handler := range a.plannerHandlers {
			if handler != nil {
				handlers[name] = handler
			}
		}
	}

	return handlers
}

func (a *Agent) plannerToolHandler(toolset []core.Tool, log *slog.Logger, runID, traceID, spanID string) planner.Handler {
	tools := make(map[string]core.Tool, len(toolset))
	for _, tool := range toolset {
		tools[tool.Name()] = tool
	}
	return func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
		toolName := resolvePlannerToolName(node)
		if toolName == "" {
			return nil, fmt.Errorf("planner node %q missing tool name", node.ID)
		}
		tool := tools[toolName]
		if tool == nil {
			return nil, fmt.Errorf("planner tool %q not found", toolName)
		}
		input := resolvePlannerInput(node, state)
		return a.callPlannerTool(ctx, log, toolName, tool, input, node.ID, runID, traceID, spanID)
	}
}

func (a *Agent) plannerNamedToolHandler(tool core.Tool, log *slog.Logger, runID, traceID, spanID string) planner.Handler {
	return func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
		input := resolvePlannerInput(node, state)
		return a.callPlannerTool(ctx, log, tool.Name(), tool, input, node.ID, runID, traceID, spanID)
	}
}

func (a *Agent) plannerAgentHandler(log *slog.Logger, runID, traceID, spanID string) planner.Handler {
	return func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
		input := resolvePlannerInput(node, state)
		inputStr := strings.TrimSpace(fmt.Sprint(input))
		if inputStr == "" {
			return "", nil
		}
		log.Info("planner.agent.invoke",
			slog.String("agent_id", a.id),
			slog.String("run_id", runID),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.String("node_id", node.ID),
		)
		return a.runEmergent(ctx, inputStr)
	}
}

func (a *Agent) plannerLLMHandler(log *slog.Logger, runID, traceID, spanID string) planner.Handler {
	return func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
		input := resolvePlannerInput(node, state)
		prompt := strings.TrimSpace(fmt.Sprint(input))
		if prompt == "" {
			return "", nil
		}

		systemPrompt := a.role
		if a.agentsDoc != nil && strings.TrimSpace(a.agentsDoc.Raw) != "" {
			if systemPrompt != "" {
				systemPrompt += "\n\n"
			}
			systemPrompt += "AGENTS.md:\n" + a.agentsDoc.Raw
		}
		messages := make([]llm.Message, 0, 3)
		if systemPrompt != "" {
			messages = append(messages, llm.Message{Role: llm.RoleSystem, Content: systemPrompt})
		}
		if memContext, ok := state.Outputs["memory"].(string); ok && strings.TrimSpace(memContext) != "" {
			messages = append(messages, llm.Message{Role: llm.RoleSystem, Content: memContext})
		}
		messages = append(messages, llm.Message{Role: llm.RoleUser, Content: prompt})

		llmStart := time.Now()
		llmCtx, llmSpan := a.tracer.Start(ctx, "Agent.LLM.Chat", trace.WithAttributes(
			attribute.String("planner.node_id", node.ID),
		))
		llmSpan.SetAttributes(telemetry.LLMAttributes(a.model, "", len(messages), 0)...)
		resp, err := a.llm.Chat(llmCtx, llm.ChatRequest{
			Model:    a.model,
			Messages: messages,
		})
		llmDurationMs := time.Since(llmStart).Seconds() * 1000
		if resp != nil {
			llmSpan.SetAttributes(telemetry.LLMAttributes(a.model, "", len(messages), len(resp.ToolCalls))...)
			llmSpan.SetAttributes(telemetry.LLMUsageAttributes(0, 0, llmDurationMs, "")...)
		}
		llmSpan.End()
		llmLatencyMs.Record(ctx, llmDurationMs)
		if err != nil {
			ke := WrapLLMError(err, a.model)
			if em := GetErrorMetrics(); em != nil {
				em.RecordError(ctx, ke, "agent-llm")
			}
			log.Error("planner.llm.error",
				slog.String("agent_id", a.id),
				slog.String("run_id", runID),
				slog.String("trace_id", traceID),
				slog.String("span_id", spanID),
				slog.String("node_id", node.ID),
				slog.String("error", err.Error()),
				slog.String("error_code", string(kerrors.CodeLLMError)),
			)
			return nil, ke
		}
		return resp.Content, nil
	}
}

func (a *Agent) callPlannerTool(ctx context.Context, log *slog.Logger, toolName string, tool core.Tool, input any, toolCallID, runID, traceID, spanID string) (any, error) {
	decision, ok := a.evaluatePolicy(ctx, log, runID, traceID, spanID, toolName, toolCallID)
	if ok && !decision.IsAllowed() {
		return nil, fmt.Errorf("policy denied: %s", decision.Reason)
	}

	args := input
	if raw, ok := input.(string); ok {
		if parsed := parseToolArguments(raw); parsed != nil {
			args = parsed
		}
	}

	toolStart := time.Now()
	toolCtx, toolSpan := a.tracer.Start(ctx, "Agent.Tool.Call")
	res, err := tool.Call(toolCtx, args)
	toolDurationMs := time.Since(toolStart).Seconds() * 1000
	toolSource := a.getToolSource(tool)
	toolSpan.SetAttributes(telemetry.ToolCallAttributes(toolName, toolCallID, toolSource, toolDurationMs, err == nil)...)
	toolSpan.SetAttributes(telemetry.ToolCallArgsResult(fmt.Sprint(args), fmt.Sprint(res), 500)...)
	toolSpan.End()

	toolLatencyMs.Record(ctx, toolDurationMs, metric.WithAttributes(
		attribute.String("tool.name", toolName),
	))

	if err != nil {
		ke := WrapToolError(err, toolName, toolCallID)
		if em := GetErrorMetrics(); em != nil {
			em.RecordError(ctx, ke, "agent-tool")
		}
		log.Error("planner.tool.error",
			slog.String("agent_id", a.id),
			slog.String("run_id", runID),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.String("tool", toolName),
			slog.String("tool_call_id", toolCallID),
			slog.String("error", err.Error()),
			slog.String("error_code", string(kerrors.CodeToolFailure)),
		)
		return nil, ke
	}

	log.Info("planner.tool.complete",
		slog.String("agent_id", a.id),
		slog.String("run_id", runID),
		slog.String("trace_id", traceID),
		slog.String("span_id", spanID),
		slog.String("tool", toolName),
		slog.String("tool_call_id", toolCallID),
	)
	return res, nil
}

func resolvePlannerToolName(node planner.Node) string {
	if strings.TrimSpace(node.Tool) != "" {
		return strings.TrimSpace(node.Tool)
	}
	if node.Metadata != nil {
		if toolName := strings.TrimSpace(node.Metadata["tool"]); toolName != "" {
			return toolName
		}
	}
	if strings.TrimSpace(node.Type) != plannerNodeTool {
		return strings.TrimSpace(node.Type)
	}
	return ""
}

func resolvePlannerInput(node planner.Node, state *planner.State) any {
	if node.Input != nil {
		return node.Input
	}
	return state.Last
}

func copyPlannerHandlers(src map[string]planner.Handler) map[string]planner.Handler {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]planner.Handler, len(src))
	for name, handler := range src {
		if handler == nil {
			continue
		}
		out[name] = handler
	}
	return out
}
