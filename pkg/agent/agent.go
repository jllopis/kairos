// Package agent implements the LLM-driven agent loop and configuration options.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/core"
	kerrors "github.com/jllopis/kairos/pkg/errors"
	"github.com/jllopis/kairos/pkg/governance"
	"github.com/jllopis/kairos/pkg/guardrails"
	"github.com/jllopis/kairos/pkg/llm"
	kmcp "github.com/jllopis/kairos/pkg/mcp"
	"github.com/jllopis/kairos/pkg/memory"
	"github.com/jllopis/kairos/pkg/planner"
	"github.com/jllopis/kairos/pkg/skills"
	"github.com/jllopis/kairos/pkg/telemetry"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Agent is an LLM-driven agent implementation.
type Agent struct {
	id                    string
	role                  string
	roleManifest          core.RoleManifest
	skills                []core.Skill
	skillTools            []*skills.SkillTool // Skills exposed as tools for LLM tool calling
	tools                 []core.Tool
	memory                core.Memory
	conversationMemory    memory.ConversationMemory // Multi-turn conversation history
	llm                   llm.Provider
	tracer                trace.Tracer
	model                 string
	maxIterations         int
	mcpClients            []*kmcp.Client
	disableActionFallback bool
	warnOnActionFallback  bool
	policyEngine          governance.PolicyEngine
	toolFilter            *governance.ToolFilter // Centralized tool filtering
	eventEmitter          core.EventEmitter
	agentsDoc             *governance.AgentInstructions
	plannerGraph          *planner.Graph
	plannerHandlers       map[string]planner.Handler
	plannerIDHandlers     map[string]planner.Handler
	plannerAuditStore     planner.AuditStore
	plannerAuditHook      func(context.Context, planner.AuditEvent)
	approvalHook          governance.ApprovalHook
	guardrails            *guardrails.Guardrails
}

// Option configures an Agent instance.
type Option func(*Agent) error

// New creates a new Agent with a required id, llm provider, and options.
func New(id string, llmProvider llm.Provider, opts ...Option) (*Agent, error) {
	if id == "" {
		return nil, errors.New("agent id is required")
	}
	if llmProvider == nil {
		return nil, errors.New("llm provider is required")
	}

	a := &Agent{
		id:            id,
		llm:           llmProvider,
		tracer:        otel.Tracer("kairos/agent"),
		model:         "default",
		maxIterations: 10, // default
	}

	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, err
		}
	}
	if a.agentsDoc == nil {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		doc, err := governance.LoadAGENTS(cwd)
		if err != nil {
			return nil, err
		}
		a.agentsDoc = doc
	}
	return a, nil
}

// WithRole sets the agent role.
func WithRole(role string) Option {
	return func(a *Agent) error {
		a.role = role
		return nil
	}
}

// WithRoleManifest attaches a semantic role manifest to the agent.
func WithRoleManifest(manifest core.RoleManifest) Option {
	return func(a *Agent) error {
		a.roleManifest = manifest
		return nil
	}
}

// WithSkills assigns semantic skills to the agent.
// Note: For skills to be available as LLM tools, use WithSkillsFromDir instead.
func WithSkills(skills []core.Skill) Option {
	return func(a *Agent) error {
		a.skills = append([]core.Skill(nil), skills...)
		return nil
	}
}

// WithSkillsFromDir loads AgentSkills from a directory containing skill subdirectories.
// Skills are loaded as tools following the AgentSkills specification with progressive disclosure:
// - Metadata (name, description) is exposed to the LLM initially
// - Instructions (Body) are injected when the LLM activates the skill
// - Resources (scripts/, references/, assets/) are loaded on demand
func WithSkillsFromDir(dir string) Option {
	return func(a *Agent) error {
		skillTools, err := skills.LoadToolsFromDir(dir)
		if err != nil {
			return err
		}

		// Store skill tools for LLM tool calling
		a.skillTools = append(a.skillTools, skillTools...)

		// Also populate legacy skills slice for backward compatibility
		for _, st := range skillTools {
			a.skills = append(a.skills, skillFromSpec(st.Spec()))
		}

		return nil
	}
}

// WithToolFilter sets the governance tool filter for access control.
func WithToolFilter(filter *governance.ToolFilter) Option {
	return func(a *Agent) error {
		a.toolFilter = filter
		return nil
	}
}

// WithConversationMemory enables multi-turn conversation history.
// Messages are stored and retrieved using the sessionID from the context.
// Use core.WithSessionID(ctx, "my-session") to set the session ID.
func WithConversationMemory(cm memory.ConversationMemory) Option {
	return func(a *Agent) error {
		a.conversationMemory = cm
		return nil
	}
}

// WithTools assigns executable tools to the agent.
func WithTools(tools []core.Tool) Option {
	return func(a *Agent) error {
		a.tools = append([]core.Tool(nil), tools...)
		return nil
	}
}

// WithMCPClients registers MCP clients for tool discovery and execution.
func WithMCPClients(clients ...*kmcp.Client) Option {
	return func(a *Agent) error {
		for _, client := range clients {
			if client == nil {
				return errors.New("mcp client cannot be nil")
			}
		}
		a.mcpClients = append(a.mcpClients, clients...)
		return nil
	}
}

// WithMCPServerConfigs connects MCP clients from config definitions.
func WithMCPServerConfigs(servers map[string]config.MCPServerConfig) Option {
	return func(a *Agent) error {
		for name, server := range servers {
			transport := strings.ToLower(strings.TrimSpace(server.Transport))
			if transport == "" {
				transport = "stdio"
			}

			opts := mcpClientOptions(server, a.policyEngine, name)
			switch transport {
			case "stdio":
				if strings.TrimSpace(server.Command) == "" {
					return fmt.Errorf("mcp server %q missing command", name)
				}
				client, err := kmcp.NewClientWithStdioProtocol(server.Command, server.Args, server.Env, server.ProtocolVersion, opts...)
				if err != nil {
					return fmt.Errorf("mcp server %q: %w", name, err)
				}
				a.mcpClients = append(a.mcpClients, client)
			case "http", "streamable-http", "streamablehttp":
				client, err := kmcp.NewClientWithStreamableHTTPProtocol(server.URL, server.ProtocolVersion, opts...)
				if err != nil {
					return fmt.Errorf("mcp server %q: %w", name, err)
				}
				a.mcpClients = append(a.mcpClients, client)
			default:
				return fmt.Errorf("mcp server %q has unsupported transport %q", name, server.Transport)
			}
		}
		return nil
	}
}

// WithPlanner configures an explicit planner graph for deterministic execution.
func WithPlanner(graph *planner.Graph) Option {
	return func(a *Agent) error {
		if graph == nil {
			return errors.New("planner graph is required")
		}
		if err := graph.Validate(); err != nil {
			return err
		}
		a.plannerGraph = graph
		return nil
	}
}

// WithPlannerHandlers registers custom handlers for planner node types.
func WithPlannerHandlers(handlers map[string]planner.Handler) Option {
	return func(a *Agent) error {
		if len(handlers) == 0 {
			return nil
		}
		a.plannerHandlers = copyPlannerHandlers(handlers)
		return nil
	}
}

// WithPlannerIDHandlers registers custom handlers for specific planner node IDs.
// If a handler exists for a node ID, it overrides the type handler.
func WithPlannerIDHandlers(handlers map[string]planner.Handler) Option {
	return func(a *Agent) error {
		if len(handlers) == 0 {
			return nil
		}
		a.plannerIDHandlers = copyPlannerHandlers(handlers)
		return nil
	}
}

// WithPlannerAuditStore attaches an audit store for planner executions.
func WithPlannerAuditStore(store planner.AuditStore) Option {
	return func(a *Agent) error {
		a.plannerAuditStore = store
		return nil
	}
}

// WithPlannerAuditHook attaches an audit hook for planner executions.
func WithPlannerAuditHook(hook func(context.Context, planner.AuditEvent)) Option {
	return func(a *Agent) error {
		a.plannerAuditHook = hook
		return nil
	}
}

// WithMemory attaches a memory backend to the agent.
func WithMemory(memory core.Memory) Option {
	return func(a *Agent) error {
		a.memory = memory
		return nil
	}
}

// WithModel sets the model name used for LLM chat requests.
func WithModel(model string) Option {
	return func(a *Agent) error {
		if strings.TrimSpace(model) == "" {
			return errors.New("model name cannot be empty")
		}
		a.model = model
		return nil
	}
}

// WithMaxIterations sets the maximum number of ReAct loop iterations.
func WithMaxIterations(max int) Option {
	return func(a *Agent) error {
		if max < 1 {
			return errors.New("max iterations must be at least 1")
		}
		a.maxIterations = max
		return nil
	}
}

// WithDisableActionFallback disables legacy "Action:" parsing in the ReAct loop.
func WithDisableActionFallback(disable bool) Option {
	return func(a *Agent) error {
		a.disableActionFallback = disable
		return nil
	}
}

// WithActionFallbackWarning enables log warnings when legacy Action parsing is used.
func WithActionFallbackWarning(enable bool) Option {
	return func(a *Agent) error {
		a.warnOnActionFallback = enable
		return nil
	}
}

// WithPolicyEngine sets a policy engine for tool execution decisions.
func WithPolicyEngine(engine governance.PolicyEngine) Option {
	return func(a *Agent) error {
		a.policyEngine = engine
		return nil
	}
}

// WithGuardrails attaches guardrails to the agent runtime.
func WithGuardrails(gr *guardrails.Guardrails) Option {
	return func(a *Agent) error {
		a.guardrails = gr
		return nil
	}
}

// WithApprovalHook sets a local approval hook for pending policy decisions.
func WithApprovalHook(hook governance.ApprovalHook) Option {
	return func(a *Agent) error {
		a.approvalHook = hook
		return nil
	}
}

// WithEventEmitter attaches a semantic event emitter to the agent.
func WithEventEmitter(emitter core.EventEmitter) Option {
	return func(a *Agent) error {
		if emitter != nil {
			a.eventEmitter = emitter
		}
		return nil
	}
}

// WithAGENTSInstructions sets the AGENTS.md instructions explicitly.
func WithAGENTSInstructions(doc *governance.AgentInstructions) Option {
	return func(a *Agent) error {
		a.agentsDoc = doc
		return nil
	}
}

// ID returns the agent identifier.
func (a *Agent) ID() string { return a.id }

// Role returns the agent role.
func (a *Agent) Role() string { return a.role }

// RoleManifest returns the configured role manifest, if any.
func (a *Agent) RoleManifest() core.RoleManifest {
	return a.roleManifest
}

// Skills returns the agent skills.
func (a *Agent) Skills() []core.Skill {
	return append([]core.Skill(nil), a.skills...)
}

// Tools returns the agent tools.
func (a *Agent) Tools() []core.Tool {
	return append([]core.Tool(nil), a.tools...)
}

// Memory returns the attached memory backend, if any.
func (a *Agent) Memory() core.Memory { return a.memory }

// Run executes the agent loop.
// If a planner graph is configured, it runs the explicit planner; otherwise it uses the emergent ReAct loop.
func (a *Agent) Run(ctx context.Context, input any) (any, error) {
	if a.plannerGraph != nil {
		return a.runPlanner(ctx, input)
	}
	return a.runEmergent(ctx, input)
}

// runEmergent executes the emergent agent loop (ReAct).
// Implements a ReAct Loop: Thought -> Action -> Observation -> Thought -> Final Answer.
// If ConversationMemory is configured, loads previous messages from the session
// and stores new messages after completion.
func (a *Agent) runEmergent(ctx context.Context, input any) (any, error) {
	ctx, runID := core.EnsureRunID(ctx)
	ctx, span := a.tracer.Start(ctx, "Agent.Run")
	defer span.End()
	traceID, spanID := traceIDs(span)
	log := slog.Default()

	inputStr, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("agent currently only supports string input")
	}

	// Add rich agent attributes to span
	span.SetAttributes(telemetry.AgentAttributes(a.id, a.role, a.model, runID, 0, a.maxIterations)...)

	if err := a.checkGuardrailsInput(ctx, log, runID, traceID, spanID, inputStr); err != nil {
		agentErrorCounter.Add(ctx, 1)
		if task, ok := core.TaskFromContext(ctx); ok && task != nil {
			task.Fail(err.Error())
		}
		return nil, err
	}

	// Track task if present
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

	// Get session ID for conversation memory
	sessionID, hasSession := core.SessionID(ctx)
	if a.conversationMemory != nil && !hasSession {
		ctx, sessionID = core.EnsureSessionID(ctx)
		hasSession = true
	}

	// Add session/conversation attributes
	convMsgCount := 0
	convStrategy := ""
	if a.conversationMemory != nil {
		convStrategy = fmt.Sprintf("%T", a.conversationMemory)
	}
	span.SetAttributes(telemetry.SessionAttributes(sessionID, a.conversationMemory != nil, convMsgCount, convStrategy)...)

	log.Info("agent.run.start",
		slog.String("agent_id", a.id),
		slog.String("run_id", runID),
		slog.String("trace_id", traceID),
		slog.String("span_id", spanID),
		slog.String("session_id", sessionID),
	)
	a.emitEvent(ctx, core.EventAgentTaskStarted, map[string]any{
		"run_id":     runID,
		"session_id": sessionID,
	})

	// 1. Construct Initial System Prompt and User Message
	messages := []llm.Message{}

	toolset := a.resolveTools(ctx, log, runID)
	toolDefs := toolDefinitions(toolset)

	// Add rich toolset attributes
	localCount, mcpCount, skillCount := a.countToolsBySource(toolset)
	span.SetAttributes(telemetry.ToolsetAttributes(len(toolset), localCount, mcpCount, skillCount, toolNames(toolset))...)

	log.Info("agent.tools.resolved",
		slog.String("agent_id", a.id),
		slog.String("run_id", runID),
		slog.Int("tool_count", len(toolset)),
		slog.String("tools", strings.Join(toolNames(toolset), ", ")),
	)

	// Construct system prompt with tool instructions if tools are present
	systemPrompt := a.role
	if a.agentsDoc != nil && strings.TrimSpace(a.agentsDoc.Raw) != "" {
		if systemPrompt != "" {
			systemPrompt += "\n\n"
		}
		systemPrompt += "AGENTS.md:\n" + a.agentsDoc.Raw
	}
	if len(toolset) > 0 {
		systemPrompt += "\n\nTools:\n"
		systemPrompt += strings.Join(toolPromptLines(toolset), "\n")
		systemPrompt += "\n"
		if a.disableActionFallback {
			systemPrompt += "\nWhen you need a tool, call it using the tool calling interface. When you are done, respond with the final answer."
		} else {
			systemPrompt += `
To use a tool, please use the following format:
Thought: Do I need to use a tool? Yes
Action: the action to take, should be one of [`
			systemPrompt += strings.Join(toolNames(toolset), ", ")
			systemPrompt += `]
Action Input: the input to the action

If you have a result, or do not need a tool, use:
Final Answer: the final answer to the original input question
`
		}
	}

	if systemPrompt != "" {
		messages = append(messages, llm.Message{Role: llm.RoleSystem, Content: systemPrompt})
	}

	// Load conversation history if configured
	if a.conversationMemory != nil && hasSession {
		historyMessages, err := a.loadConversationHistory(ctx, sessionID)
		if err != nil {
			log.Warn("agent.conversation.load_error",
				slog.String("agent_id", a.id),
				slog.String("session_id", sessionID),
				slog.String("error", err.Error()),
			)
		} else if len(historyMessages) > 0 {
			messages = append(messages, historyMessages...)
			log.Info("agent.conversation.loaded",
				slog.String("agent_id", a.id),
				slog.String("session_id", sessionID),
				slog.Int("history_count", len(historyMessages)),
			)
		}
	}

	// TODO: Retrieve Context from Memory here
	mem := a.resolveMemory(ctx)
	memoryRetrieved := 0
	if mem != nil {
		if memoryContext := a.loadMemoryContext(ctx, mem, inputStr); memoryContext != "" {
			messages = append(messages, llm.Message{Role: llm.RoleSystem, Content: memoryContext})
			memoryRetrieved = 1 // Could count items if we had access
		}
	}
	// Add memory attributes to span
	span.SetAttributes(telemetry.MemoryAttributes(mem != nil, fmt.Sprintf("%T", mem), memoryRetrieved, false)...)

	// Store user message in conversation memory
	if a.conversationMemory != nil && hasSession {
		if err := a.storeConversationMessage(ctx, sessionID, llm.RoleUser, inputStr, ""); err != nil {
			log.Warn("agent.conversation.store_error",
				slog.String("agent_id", a.id),
				slog.String("session_id", sessionID),
				slog.String("error", err.Error()),
			)
		}
	}

	messages = append(messages, llm.Message{Role: llm.RoleUser, Content: inputStr})

	// 2. ReAct Loop
	for i := 0; i < a.maxIterations; i++ {
		a.emitEvent(ctx, core.EventAgentThinking, map[string]any{
			"iteration": i + 1,
		})
		llmStart := time.Now()
		llmCtx, llmSpan := a.tracer.Start(ctx, "Agent.LLM.Chat", trace.WithAttributes(
			attribute.Int("agent.iteration", i+1),
		))
		// Add rich LLM attributes
		llmSpan.SetAttributes(telemetry.LLMAttributes(a.model, "", len(messages), 0)...)

		// Call LLM
		req := llm.ChatRequest{
			Model:    a.model,
			Messages: messages,
		}
		if len(toolDefs) > 0 {
			req.Tools = toolDefs
		}

		resp, err := a.llm.Chat(llmCtx, req)
		llmDurationMs := time.Since(llmStart).Seconds() * 1000

		// Add post-call attributes including tool calls count
		if resp != nil {
			llmSpan.SetAttributes(telemetry.LLMAttributes(a.model, "", len(messages), len(resp.ToolCalls))...)
			llmSpan.SetAttributes(telemetry.LLMUsageAttributes(0, 0, llmDurationMs, "")...)
		}

		llmSpan.End()
		llmLatencyMs.Record(ctx, llmDurationMs)
		if err != nil {
			agentErrorCounter.Add(ctx, 1)
			ke := WrapLLMError(err, a.model)
			if em := GetErrorMetrics(); em != nil {
				em.RecordError(ctx, ke, "agent-llm")
			}
			log.Error("agent.llm.error",
				slog.String("agent_id", a.id),
				slog.String("run_id", runID),
				slog.String("trace_id", traceID),
				slog.String("span_id", spanID),
				slog.String("error", err.Error()),
				slog.String("error_code", string(kerrors.CodeLLMError)),
			)
			a.emitEvent(ctx, core.EventAgentError, map[string]any{
				"run_id": runID,
				"stage":  "llm",
				"error":  err.Error(),
			})
			if task, ok := core.TaskFromContext(ctx); ok && task != nil {
				task.Fail(err.Error())
			}
			return nil, ke
		}

		content := resp.Content
		messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: content})

		if len(resp.ToolCalls) > 0 {
			logDecision(log, decisionPayload{
				AgentID:       a.id,
				RunID:         runID,
				TraceID:       traceID,
				SpanID:        spanID,
				Iteration:     i + 1,
				DecisionType:  "tool_call",
				Rationale:     summarizeToolCallRationale(content),
				InputSummary:  summarizeText(inputStr),
				OutputSummary: summarizeText(content),
			})
			a.handleToolCalls(ctx, log, runID, traceID, spanID, toolset, resp.ToolCalls, &messages, content)
			continue
		}

		// Check for Final Answer
		if strings.Contains(content, "Final Answer:") {
			parts := strings.Split(content, "Final Answer:")
			if len(parts) > 1 {
				finalAnswer := strings.TrimSpace(parts[1])
				finalAnswer = a.applyGuardrailsOutput(ctx, log, runID, traceID, spanID, finalAnswer)
				logDecision(log, decisionPayload{
					AgentID:       a.id,
					RunID:         runID,
					TraceID:       traceID,
					SpanID:        spanID,
					Iteration:     i + 1,
					DecisionType:  "final_answer",
					Rationale:     summarizeFinalRationale(content),
					InputSummary:  summarizeText(inputStr),
					OutputSummary: summarizeText(finalAnswer),
				})
				a.storeMemory(ctx, mem, inputStr, finalAnswer)
				// Store assistant response in conversation memory
				if a.conversationMemory != nil && hasSession {
					if err := a.storeConversationMessage(ctx, sessionID, llm.RoleAssistant, finalAnswer, ""); err != nil {
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
					slog.Int("iterations", i+1),
				)
				a.emitEvent(ctx, core.EventAgentTaskCompleted, map[string]any{
					"run_id": runID,
					"result": finalAnswer,
				})
				if task, ok := core.TaskFromContext(ctx); ok && task != nil {
					task.Complete(finalAnswer)
				}
				return finalAnswer, nil
			}
			logDecision(log, decisionPayload{
				AgentID:       a.id,
				RunID:         runID,
				TraceID:       traceID,
				SpanID:        spanID,
				Iteration:     i + 1,
				DecisionType:  "final_answer",
				Rationale:     summarizeFinalRationale(content),
				InputSummary:  summarizeText(inputStr),
				OutputSummary: summarizeText(content),
			})
			content = a.applyGuardrailsOutput(ctx, log, runID, traceID, spanID, content)
			a.storeMemory(ctx, mem, inputStr, content)
			// Store assistant response in conversation memory
			if a.conversationMemory != nil && hasSession {
				if err := a.storeConversationMessage(ctx, sessionID, llm.RoleAssistant, content, ""); err != nil {
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
				slog.Int("iterations", i+1),
			)
			a.emitEvent(ctx, core.EventAgentTaskCompleted, map[string]any{
				"run_id": runID,
				"result": content,
			})
			if task, ok := core.TaskFromContext(ctx); ok && task != nil {
				task.Complete(content)
			}
			return content, nil
		}

		if a.disableActionFallback && len(toolset) > 0 && strings.TrimSpace(content) != "" {
			logDecision(log, decisionPayload{
				AgentID:       a.id,
				RunID:         runID,
				TraceID:       traceID,
				SpanID:        spanID,
				Iteration:     i + 1,
				DecisionType:  "final_answer",
				Rationale:     summarizeFinalRationale(content),
				InputSummary:  summarizeText(inputStr),
				OutputSummary: summarizeText(content),
			})
			content = a.applyGuardrailsOutput(ctx, log, runID, traceID, spanID, content)
			a.storeMemory(ctx, mem, inputStr, content)
			// Store assistant response in conversation memory
			if a.conversationMemory != nil && hasSession {
				if err := a.storeConversationMessage(ctx, sessionID, llm.RoleAssistant, content, ""); err != nil {
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
				slog.Int("iterations", i+1),
			)
			a.emitEvent(ctx, core.EventAgentTaskCompleted, map[string]any{
				"run_id": runID,
				"result": content,
			})
			if task, ok := core.TaskFromContext(ctx); ok && task != nil {
				task.Complete(content)
			}
			return content, nil
		}

		// Check for Action
		// Simple parsing logic for now.
		// TODO: Make this robust (regex or structured output)
		if !a.disableActionFallback && strings.Contains(content, "Action:") {
			if a.warnOnActionFallback {
				log.Warn("agent.action.fallback",
					slog.String("agent_id", a.id),
					slog.String("run_id", runID),
					slog.String("trace_id", traceID),
					slog.String("span_id", spanID),
					slog.String("note", "legacy action parsing used"),
				)
			}
			logDecision(log, decisionPayload{
				AgentID:       a.id,
				RunID:         runID,
				TraceID:       traceID,
				SpanID:        spanID,
				Iteration:     i + 1,
				DecisionType:  "fallback_action",
				Rationale:     summarizeFallbackRationale(content),
				InputSummary:  summarizeText(inputStr),
				OutputSummary: summarizeText(content),
			})
			lines := strings.Split(content, "\n")
			var action, actionInput string

			for i, line := range lines {
				if strings.HasPrefix(line, "Action:") {
					action = strings.TrimSpace(strings.TrimPrefix(line, "Action:"))
				}
				if strings.HasPrefix(line, "Action Input:") {
					actionInput = strings.TrimSpace(strings.TrimPrefix(line, "Action Input:"))
					if actionInput == "" && i+1 < len(lines) {
						actionInput = strings.TrimSpace(lines[i+1])
					}
				}
			}

			if action != "" {
				log.Info("agent.tool.requested",
					slog.String("agent_id", a.id),
					slog.String("run_id", runID),
					slog.String("tool", action),
					slog.String("action_input", actionInput),
				)
				// Initialize as "Not Found"
				var foundTool core.Tool
				for _, t := range toolset {
					if t.Name() == action {
						foundTool = t
						break
					}
				}

				var observation string
				if foundTool != nil {
					if decision, ok := a.evaluatePolicy(ctx, log, runID, traceID, spanID, action, ""); ok {
						if !decision.IsAllowed() {
							observation = fmt.Sprintf("Policy denied: %s", decision.Reason)
						} else {
							observation = ""
						}
					}
					if observation != "" {
						messages = append(messages, llm.Message{Role: llm.RoleUser, Content: fmt.Sprintf("Observation: %s", observation)})
						continue
					}
					log.Info("agent.tool.found",
						slog.String("agent_id", a.id),
						slog.String("run_id", runID),
						slog.String("tool", action),
					)
					toolStart := time.Now()
					toolCtx, toolSpan := a.tracer.Start(ctx, "Agent.Tool.Call", trace.WithAttributes(
						attribute.String("tool.name", action),
					))
					// Tool execution
					// We treat tool Call input as string for this basic implementation
					res, err := foundTool.Call(toolCtx, actionInput)
					toolSpan.End()
					toolLatencyMs.Record(ctx, time.Since(toolStart).Seconds()*1000, metric.WithAttributes(
						attribute.String("tool.name", action),
					))
					if err != nil {
						ke := WrapToolError(err, action, "")
						if em := GetErrorMetrics(); em != nil {
							em.RecordError(ctx, ke, "agent-tool")
						}
						observation = fmt.Sprintf("Error executing tool: %v", err)
						agentErrorCounter.Add(ctx, 1)
						log.Error("agent.tool.error",
							slog.String("agent_id", a.id),
							slog.String("run_id", runID),
							slog.String("trace_id", traceID),
							slog.String("span_id", spanID),
							slog.String("tool", action),
							slog.String("error", err.Error()),
							slog.String("error_code", string(kerrors.CodeToolFailure)),
						)
						a.emitEvent(ctx, core.EventAgentError, map[string]any{
							"run_id": runID,
							"stage":  "tool",
							"tool":   action,
							"error":  err.Error(),
						})
						if task, ok := core.TaskFromContext(ctx); ok && task != nil {
							task.Fail(err.Error())
						}
					} else {
						observation = fmt.Sprintf("%v", res)
						log.Info("agent.tool.complete",
							slog.String("agent_id", a.id),
							slog.String("run_id", runID),
							slog.String("trace_id", traceID),
							slog.String("span_id", spanID),
							slog.String("tool", action),
						)
					}
				} else {
					observation = fmt.Sprintf("Tool %s not found", action)
					log.Warn("agent.tool.missing",
						slog.String("agent_id", a.id),
						slog.String("run_id", runID),
						slog.String("trace_id", traceID),
						slog.String("span_id", spanID),
						slog.String("tool", action),
					)
				}

				// Append Observation
				msg := fmt.Sprintf("Observation: %s", observation)
				// ReAct paper suggests Observation is next line, often as User or Tool output.
				// We'll treat it as User message to prompt next thought.
				messages = append(messages, llm.Message{Role: llm.RoleUser, Content: msg})
				continue
			}
		}

		// If no tools defined, just return content (single turn behavior)
		if len(toolset) == 0 {
			content = a.applyGuardrailsOutput(ctx, log, runID, traceID, spanID, content)
			a.storeMemory(ctx, mem, inputStr, content)
			// Store assistant response in conversation memory
			if a.conversationMemory != nil && hasSession {
				if err := a.storeConversationMessage(ctx, sessionID, llm.RoleAssistant, content, ""); err != nil {
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
				slog.Int("iterations", i+1),
			)
			a.emitEvent(ctx, core.EventAgentTaskCompleted, map[string]any{
				"run_id": runID,
				"result": content,
			})
			if task, ok := core.TaskFromContext(ctx); ok && task != nil {
				task.Complete(content)
			}
			return content, nil
		}
	}

	agentErrorCounter.Add(ctx, 1)
	ke := WrapTimeoutError(fmt.Errorf("max iterations exceeded"), "agent-loop", a.maxIterations)
	if em := GetErrorMetrics(); em != nil {
		em.RecordError(ctx, ke, "agent-loop")
	}
	log.Error("agent.run.timeout",
		slog.String("agent_id", a.id),
		slog.String("run_id", runID),
		slog.String("trace_id", traceID),
		slog.String("span_id", spanID),
		slog.Int("iterations", a.maxIterations),
		slog.String("error_code", string(kerrors.CodeTimeout)),
	)
	a.emitEvent(ctx, core.EventAgentError, map[string]any{
		"run_id": runID,
		"stage":  "timeout",
		"error":  "max iterations exceeded",
	})
	if task, ok := core.TaskFromContext(ctx); ok && task != nil {
		task.Fail("max iterations exceeded")
	}
	return nil, ke
}

// Close releases MCP client resources if configured.
func (a *Agent) Close() error {
	if len(a.mcpClients) == 0 {
		return nil
	}
	var errs []error
	for _, client := range a.mcpClients {
		if err := client.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("closing mcp clients: %v", errs)
	}
	return nil
}

func (a *Agent) resolveMemory(ctx context.Context) core.Memory {
	if a.memory != nil {
		return a.memory
	}
	mem, _ := core.MemoryFromContext(ctx)
	return mem
}

func (a *Agent) loadMemoryContext(ctx context.Context, mem core.Memory, query string) string {
	if mem == nil {
		return ""
	}

	memStart := time.Now()
	memCtx, memSpan := a.tracer.Start(ctx, "Agent.Memory.Retrieve")
	result, err := mem.Retrieve(memCtx, query)
	if err != nil {
		result, err = mem.Retrieve(memCtx, nil)
		if err != nil {
			memSpan.End()
			memoryLatencyMs.Record(ctx, time.Since(memStart).Seconds()*1000, metric.WithAttributes(
				attribute.String("memory.operation", "retrieve"),
				attribute.String("memory.outcome", "empty"),
			))
			return ""
		}
	}
	memSpan.End()
	memoryLatencyMs.Record(ctx, time.Since(memStart).Seconds()*1000, metric.WithAttributes(
		attribute.String("memory.operation", "retrieve"),
		attribute.String("memory.outcome", "hit"),
	))

	switch value := result.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return ""
		}
		return fmt.Sprintf("Memory context:\n%s", value)
	case []string:
		if len(value) == 0 {
			return ""
		}
		return fmt.Sprintf("Memory context:\n- %s", strings.Join(value, "\n- "))
	case []any:
		if len(value) == 0 {
			return ""
		}
		parts := make([]string, 0, len(value))
		for _, item := range value {
			parts = append(parts, fmt.Sprint(item))
		}
		return fmt.Sprintf("Memory context:\n- %s", strings.Join(parts, "\n- "))
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" {
			return ""
		}
		return fmt.Sprintf("Memory context:\n%s", text)
	}
}

func (a *Agent) storeMemory(ctx context.Context, mem core.Memory, input, output string) {
	if mem == nil {
		return
	}

	memStart := time.Now()
	memCtx, memSpan := a.tracer.Start(ctx, "Agent.Memory.Store")
	entry := fmt.Sprintf("Timestamp: %s\nUser: %s\nAgent: %s", time.Now().UTC().Format(time.RFC3339), input, output)
	if err := mem.Store(memCtx, entry); err != nil {
		memSpan.End()
		ke := WrapMemoryError(err, "store")
		if em := GetErrorMetrics(); em != nil {
			em.RecordError(ctx, ke, "agent-memory")
		}
		agentErrorCounter.Add(ctx, 1)
		slog.Default().Error("agent.memory.store.error",
			slog.String("agent_id", a.id),
			slog.String("run_id", runIDFromContext(ctx)),
			slog.String("trace_id", traceIDFromContext(ctx)),
			slog.String("span_id", spanIDFromContext(ctx)),
			slog.String("error", err.Error()),
			slog.String("error_code", string(kerrors.CodeMemoryError)),
		)
		return
	}
	memSpan.End()
	memoryLatencyMs.Record(ctx, time.Since(memStart).Seconds()*1000, metric.WithAttributes(
		attribute.String("memory.operation", "store"),
		attribute.String("memory.outcome", "ok"),
	))
}

func (a *Agent) resolveTools(ctx context.Context, log *slog.Logger, runID string) []core.Tool {
	tools := append([]core.Tool(nil), a.tools...)

	// Add skill tools (AgentSkills as LLM-callable tools)
	for _, st := range a.skillTools {
		tools = append(tools, st)
	}

	// Add MCP tools
	for _, client := range a.mcpClients {
		list, err := client.ListTools(ctx)
		if err != nil {
			log.Error("agent.mcp.list_tools.error",
				slog.String("agent_id", a.id),
				slog.String("run_id", runID),
				slog.String("error", err.Error()),
			)
			continue
		}
		for _, tool := range list {
			adapter, err := kmcp.NewToolAdapter(tool, client)
			if err != nil {
				log.Error("agent.mcp.tool_adapter.error",
					slog.String("agent_id", a.id),
					slog.String("run_id", runID),
					slog.String("tool", tool.Name),
					slog.String("error", err.Error()),
				)
				continue
			}
			tools = append(tools, adapter)
		}
	}

	// Apply governance tool filter if configured
	if a.toolFilter != nil {
		tools = a.filterToolsByGovernance(ctx, tools)
	}

	return dedupeTools(tools)
}

// filterToolsByGovernance applies the governance tool filter to the tool list.
func (a *Agent) filterToolsByGovernance(ctx context.Context, tools []core.Tool) []core.Tool {
	if a.toolFilter == nil {
		return tools
	}

	filtered := make([]core.Tool, 0, len(tools))
	for _, tool := range tools {
		if a.toolFilter.IsAllowed(ctx, tool.Name()).IsAllowed() {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// skillAllowList is deprecated - tool filtering should use governance.ToolFilter.
// Kept for backward compatibility.
func (a *Agent) skillAllowList() map[string]bool {
	if len(a.skills) == 0 {
		return nil
	}
	allowed := make(map[string]bool)
	for _, skill := range a.skills {
		for _, tool := range skill.AllowedTools {
			if strings.TrimSpace(tool) == "" {
				continue
			}
			allowed[tool] = true
		}
	}
	if len(allowed) == 0 {
		allowed = make(map[string]bool, len(a.skills))
		for _, skill := range a.skills {
			if strings.TrimSpace(skill.Name) == "" {
				continue
			}
			allowed[skill.Name] = true
		}
	}
	return allowed
}

func dedupeTools(tools []core.Tool) []core.Tool {
	if len(tools) < 2 {
		return tools
	}
	seen := make(map[string]bool, len(tools))
	out := make([]core.Tool, 0, len(tools))
	for _, tool := range tools {
		name := tool.Name()
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, tool)
	}
	slices.SortStableFunc(out, func(a, b core.Tool) int {
		if a.Name() == b.Name() {
			return 0
		}
		if a.Name() < b.Name() {
			return -1
		}
		return 1
	})
	return out
}

func toolNames(tools []core.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		if name := tool.Name(); name != "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

func toolPromptLines(tools []core.Tool) []string {
	lines := make([]string, 0, len(tools))
	for _, tool := range tools {
		def := tool.ToolDefinition()
		name := tool.Name()
		desc := strings.TrimSpace(def.Function.Description)
		if desc != "" {
			lines = append(lines, fmt.Sprintf("- %s: %s", name, desc))
		} else {
			lines = append(lines, fmt.Sprintf("- %s", name))
		}
	}
	return lines
}

// countToolsBySource counts tools by their source type (local, MCP, skill).
func (a *Agent) countToolsBySource(tools []core.Tool) (local, mcp, skill int) {
	for _, tool := range tools {
		switch tool.(type) {
		case *skills.SkillTool:
			skill++
		case *kmcp.ToolAdapter:
			mcp++
		default:
			local++
		}
	}
	return
}

// getToolSource returns the source type of a tool ("local", "mcp", or "skill").
func (a *Agent) getToolSource(tool core.Tool) string {
	switch tool.(type) {
	case *skills.SkillTool:
		return "skill"
	case *kmcp.ToolAdapter:
		return "mcp"
	default:
		return "local"
	}
}

func toolDefinitions(tools []core.Tool) []llm.Tool {
	defs := make([]llm.Tool, 0, len(tools))
	for _, tool := range tools {
		def := tool.ToolDefinition()
		if def.Type == "" {
			def.Type = llm.ToolTypeFunction
		}
		if name := strings.TrimSpace(def.Function.Name); name == "" || name != tool.Name() {
			def.Function.Name = tool.Name()
		}
		defs = append(defs, def)
	}
	return defs
}

func skillFromSpec(spec skills.SkillSpec) core.Skill {
	return core.Skill{
		Name:          spec.Name,
		Description:   spec.Description,
		License:       spec.License,
		Compatibility: spec.Compatibility,
		Metadata:      spec.Metadata,
		AllowedTools:  spec.AllowedTools,
		Body:          spec.Body,
	}
}

func (a *Agent) handleToolCalls(ctx context.Context, log *slog.Logger, runID, traceID, spanID string, toolset []core.Tool, calls []llm.ToolCall, messages *[]llm.Message, rationale string) {
	for _, call := range calls {
		toolName := call.Function.Name
		args := strings.TrimSpace(call.Function.Arguments)
		logDecision(log, decisionPayload{
			AgentID:       a.id,
			RunID:         runID,
			TraceID:       traceID,
			SpanID:        spanID,
			DecisionType:  "tool_call",
			Rationale:     summarizeText(rationale),
			OutputSummary: summarizeText(args),
			ToolName:      toolName,
			ToolCallID:    call.ID,
		})
		log.Info("agent.tool.requested",
			slog.String("agent_id", a.id),
			slog.String("run_id", runID),
			slog.String("tool", toolName),
			slog.String("tool_call_id", call.ID),
			slog.String("action_input", args),
		)

		var foundTool core.Tool
		for _, t := range toolset {
			if t.Name() == toolName {
				foundTool = t
				break
			}
		}

		observation := ""
		if foundTool == nil {
			observation = fmt.Sprintf("Tool %s not found", toolName)
			log.Warn("agent.tool.missing",
				slog.String("agent_id", a.id),
				slog.String("run_id", runID),
				slog.String("trace_id", traceID),
				slog.String("span_id", spanID),
				slog.String("tool", toolName),
				slog.String("tool_call_id", call.ID),
			)
		} else {
			if decision, ok := a.evaluatePolicy(ctx, log, runID, traceID, spanID, toolName, call.ID); ok {
				if !decision.IsAllowed() {
					observation = fmt.Sprintf("Policy denied: %s", decision.Reason)
					*messages = append(*messages, llm.Message{
						Role:       llm.RoleTool,
						Content:    observation,
						ToolCallID: call.ID,
					})
					continue
				}
			}
			log.Info("agent.tool.found",
				slog.String("agent_id", a.id),
				slog.String("run_id", runID),
				slog.String("tool", toolName),
				slog.String("tool_call_id", call.ID),
			)
			toolStart := time.Now()

			// Determine tool source
			toolSource := a.getToolSource(foundTool)
			toolCtx, toolSpan := a.tracer.Start(ctx, "Agent.Tool.Call")

			var input any = args
			if parsed := parseToolArguments(args); parsed != nil {
				input = parsed
			}
			res, err := foundTool.Call(toolCtx, input)
			toolDurationMs := time.Since(toolStart).Seconds() * 1000

			// Add rich tool call attributes
			toolSpan.SetAttributes(telemetry.ToolCallAttributes(toolName, call.ID, toolSource, toolDurationMs, err == nil)...)
			toolSpan.SetAttributes(telemetry.ToolCallArgsResult(args, fmt.Sprintf("%v", res), 500)...)

			toolSpan.End()
			toolLatencyMs.Record(ctx, toolDurationMs, metric.WithAttributes(
				attribute.String("tool.name", toolName),
			))
			if err != nil {
				ke := WrapToolError(err, toolName, call.ID)
				if em := GetErrorMetrics(); em != nil {
					em.RecordError(ctx, ke, "agent-tool")
				}
				observation = fmt.Sprintf("Error executing tool: %v", err)
				logDecisionOutcome(log, decisionPayload{
					AgentID:       a.id,
					RunID:         runID,
					TraceID:       traceID,
					SpanID:        spanID,
					DecisionType:  "tool_call",
					OutputSummary: summarizeText(observation),
					ToolName:      toolName,
					ToolCallID:    call.ID,
				}, err)
				agentErrorCounter.Add(ctx, 1)
				log.Error("agent.tool.error",
					slog.String("agent_id", a.id),
					slog.String("run_id", runID),
					slog.String("trace_id", traceID),
					slog.String("span_id", spanID),
					slog.String("tool", toolName),
					slog.String("tool_call_id", call.ID),
					slog.String("error", err.Error()),
					slog.String("error_code", string(kerrors.CodeToolFailure)),
				)
				a.emitEvent(ctx, core.EventAgentError, map[string]any{
					"run_id": runID,
					"stage":  "tool",
					"tool":   toolName,
					"error":  err.Error(),
				})
				if task, ok := core.TaskFromContext(ctx); ok && task != nil {
					task.Fail(err.Error())
				}
			} else {
				observation = fmt.Sprintf("%v", res)
				logDecisionOutcome(log, decisionPayload{
					AgentID:       a.id,
					RunID:         runID,
					TraceID:       traceID,
					SpanID:        spanID,
					DecisionType:  "tool_call",
					OutputSummary: summarizeText(observation),
					ToolName:      toolName,
					ToolCallID:    call.ID,
				}, nil)
				log.Info("agent.tool.complete",
					slog.String("agent_id", a.id),
					slog.String("run_id", runID),
					slog.String("trace_id", traceID),
					slog.String("span_id", spanID),
					slog.String("tool", toolName),
					slog.String("tool_call_id", call.ID),
				)
			}
		}

		*messages = append(*messages, llm.Message{
			Role:       llm.RoleTool,
			Content:    observation,
			ToolCallID: call.ID,
		})
	}
}

func parseToolArguments(raw string) map[string]interface{} {
	if raw == "" {
		return nil
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil
	}
	return decoded
}

func (a *Agent) evaluatePolicy(ctx context.Context, log *slog.Logger, runID, traceID, spanID, toolName, toolCallID string) (governance.Decision, bool) {
	if a.policyEngine == nil {
		return governance.Decision{}, false
	}
	decision := a.policyEngine.Evaluate(ctx, governance.Action{
		Type: governance.ActionTool,
		Name: toolName,
		Metadata: map[string]string{
			"agent_id":     a.id,
			"tool_call_id": toolCallID,
		},
	})
	if decision.IsPending() && a.approvalHook != nil {
		action := governance.Action{
			Type: governance.ActionTool,
			Name: toolName,
			Metadata: map[string]string{
				"agent_id":     a.id,
				"tool_call_id": toolCallID,
			},
		}
		if decision.Reason != "" {
			action.Metadata["policy_reason"] = decision.Reason
		}
		if decision.RuleID != "" {
			action.Metadata["policy_rule_id"] = decision.RuleID
		}
		hookDecision := a.approvalHook.Request(ctx, action)
		if hookDecision.Status != "" || hookDecision.Allowed || hookDecision.Reason != "" {
			decision = hookDecision
		}
	}
	if decision.IsPending() && strings.TrimSpace(decision.Reason) == "" {
		decision.Reason = "approval required"
	}
	if decision.IsDenied() && strings.TrimSpace(decision.Reason) == "" {
		decision.Reason = "blocked by policy"
	}
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(telemetry.PolicyAttributes(true, decision.IsAllowed(), decision.Reason)...)
	if !decision.IsAllowed() {
		event := "agent.policy.denied"
		if decision.IsPending() {
			event = "agent.policy.pending"
		}
		log.Warn(event,
			slog.String("agent_id", a.id),
			slog.String("run_id", runID),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.String("tool", toolName),
			slog.String("rule_id", decision.RuleID),
			slog.String("reason", decision.Reason),
		)
	}
	return decision, true
}

// ToolNames returns the resolved tool names for the agent.
func (a *Agent) ToolNames() []string {
	ctx := context.Background()
	tools := a.resolveTools(ctx, slog.Default(), "tool-names")
	return toolNames(tools)
}

// MCPTools returns the raw MCP tool definitions discovered from configured clients.
func (a *Agent) MCPTools(ctx context.Context) ([]mcpgo.Tool, error) {
	if len(a.mcpClients) == 0 {
		return nil, nil
	}
	seen := make(map[string]bool)
	out := make([]mcpgo.Tool, 0)
	for _, client := range a.mcpClients {
		list, err := client.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		for _, tool := range list {
			if tool.Name == "" || seen[tool.Name] {
				continue
			}
			seen[tool.Name] = true
			out = append(out, tool)
		}
	}
	slices.SortFunc(out, func(a, b mcpgo.Tool) int {
		if a.Name == b.Name {
			return 0
		}
		if a.Name < b.Name {
			return -1
		}
		return 1
	})
	return out, nil
}

var (
	metricsOnce       sync.Once
	agentRunCounter   metric.Int64Counter
	agentErrorCounter metric.Int64Counter
	agentRunLatencyMs metric.Float64Histogram
	llmLatencyMs      metric.Float64Histogram
	toolLatencyMs     metric.Float64Histogram
	memoryLatencyMs   metric.Float64Histogram
)

func initAgentMetrics() {
	metricsOnce.Do(func() {
		meter := otel.Meter("kairos/agent")
		agentRunCounter, _ = meter.Int64Counter("kairos.agent.run.count")
		agentErrorCounter, _ = meter.Int64Counter("kairos.agent.error.count")
		agentRunLatencyMs, _ = meter.Float64Histogram("kairos.agent.run.latency_ms")
		llmLatencyMs, _ = meter.Float64Histogram("kairos.agent.llm.latency_ms")
		toolLatencyMs, _ = meter.Float64Histogram("kairos.agent.tool.latency_ms")
		memoryLatencyMs, _ = meter.Float64Histogram("kairos.agent.memory.latency_ms")
	})
}

func runIDFromContext(ctx context.Context) string {
	if runID, ok := core.RunID(ctx); ok {
		return runID
	}
	return "unknown"
}

func traceIDs(span trace.Span) (string, string) {
	sc := span.SpanContext()
	return sc.TraceID().String(), sc.SpanID().String()
}

func traceIDFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return "unknown"
	}
	return sc.TraceID().String()
}

func (a *Agent) emitEvent(ctx context.Context, eventType core.EventType, payload map[string]any) {
	if a.eventEmitter == nil {
		return
	}
	taskID := ""
	if task, ok := core.TaskFromContext(ctx); ok && task != nil {
		taskID = task.ID
		if task.Goal != "" && payload != nil {
			if _, ok := payload["task_goal"]; !ok {
				payload["task_goal"] = task.Goal
			}
		}
	}
	a.eventEmitter.Emit(ctx, core.NewEvent(eventType, a.id, taskID, payload))
}

func spanIDFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return "unknown"
	}
	return sc.SpanID().String()
}

func mcpClientOptions(server config.MCPServerConfig, policyEngine governance.PolicyEngine, serverName string) []kmcp.ClientOption {
	var opts []kmcp.ClientOption
	if server.TimeoutSeconds != nil && *server.TimeoutSeconds > 0 {
		opts = append(opts, kmcp.WithTimeout(time.Duration(*server.TimeoutSeconds)*time.Second))
	}

	retries := -1
	backoff := time.Duration(0)
	if server.RetryCount != nil {
		retries = *server.RetryCount
	}
	if server.RetryBackoffMs != nil && *server.RetryBackoffMs > 0 {
		backoff = time.Duration(*server.RetryBackoffMs) * time.Millisecond
	}
	if retries >= 0 || backoff > 0 {
		opts = append(opts, kmcp.WithRetry(retries, backoff))
	}

	if server.CacheTTLSeconds != nil && *server.CacheTTLSeconds >= 0 {
		opts = append(opts, kmcp.WithToolCacheTTL(time.Duration(*server.CacheTTLSeconds)*time.Second))
	}
	if policyEngine != nil {
		opts = append(opts, kmcp.WithPolicyEngine(policyEngine))
	}
	if strings.TrimSpace(serverName) != "" {
		opts = append(opts, kmcp.WithServerName(serverName))
	}

	return opts
}

type decisionPayload struct {
	AgentID       string
	RunID         string
	TraceID       string
	SpanID        string
	DecisionType  string
	Rationale     string
	InputSummary  string
	OutputSummary string
	Iteration     int
	ToolName      string
	ToolCallID    string
}

func logDecision(log *slog.Logger, payload decisionPayload) {
	log.Info("agent.decision",
		slog.String("agent_id", payload.AgentID),
		slog.String("run_id", payload.RunID),
		slog.String("trace_id", payload.TraceID),
		slog.String("span_id", payload.SpanID),
		slog.String("decision_type", payload.DecisionType),
		slog.String("rationale", summarizeText(payload.Rationale)),
		slog.String("input_summary", summarizeText(payload.InputSummary)),
		slog.String("output_summary", summarizeText(payload.OutputSummary)),
		slog.Int("iteration", payload.Iteration),
		slog.String("tool", payload.ToolName),
		slog.String("tool_call_id", payload.ToolCallID),
	)
}

func logDecisionOutcome(log *slog.Logger, payload decisionPayload, err error) {
	attrs := []any{
		slog.String("agent_id", payload.AgentID),
		slog.String("run_id", payload.RunID),
		slog.String("trace_id", payload.TraceID),
		slog.String("span_id", payload.SpanID),
		slog.String("decision_type", payload.DecisionType),
		slog.String("output_summary", summarizeText(payload.OutputSummary)),
		slog.String("tool", payload.ToolName),
		slog.String("tool_call_id", payload.ToolCallID),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	log.Info("agent.decision.outcome", attrs...)
}

func summarizeToolCallRationale(content string) string {
	text := strings.TrimSpace(content)
	if text == "" {
		return "tool_call"
	}
	return summarizeText(text)
}

func summarizeFinalRationale(content string) string {
	parts := strings.SplitN(content, "Final Answer:", 2)
	if len(parts) == 0 {
		return "final_answer"
	}
	rationale := strings.TrimSpace(parts[0])
	if rationale == "" {
		return "final_answer"
	}
	return summarizeText(rationale)
}

func summarizeFallbackRationale(content string) string {
	parts := strings.SplitN(content, "Action:", 2)
	if len(parts) == 0 {
		return "fallback_action"
	}
	rationale := strings.TrimSpace(parts[0])
	if rationale == "" {
		return "fallback_action"
	}
	return summarizeText(rationale)
}

func summarizeText(text string) string {
	const maxLen = 400
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= maxLen {
		return trimmed
	}
	return trimmed[:maxLen] + "..."
}

// loadConversationHistory loads previous messages from conversation memory.
func (a *Agent) loadConversationHistory(ctx context.Context, sessionID string) ([]llm.Message, error) {
	if a.conversationMemory == nil {
		return nil, nil
	}

	convMessages, err := a.conversationMemory.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Convert ConversationMessage to llm.Message
	messages := make([]llm.Message, 0, len(convMessages))
	for _, cm := range convMessages {
		msg := llm.Message{
			Role:       llm.Role(cm.Role),
			Content:    cm.Content,
			ToolCallID: cm.ToolCallID,
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// storeConversationMessage saves a message to conversation memory.
func (a *Agent) storeConversationMessage(ctx context.Context, sessionID string, role llm.Role, content, toolCallID string) error {
	if a.conversationMemory == nil {
		return nil
	}

	msg := memory.ConversationMessage{
		SessionID:  sessionID,
		Role:       string(role),
		Content:    content,
		ToolCallID: toolCallID,
	}

	return a.conversationMemory.AppendMessage(ctx, sessionID, msg)
}

// ConversationMemory returns the attached conversation memory, if any.
func (a *Agent) ConversationMemory() memory.ConversationMemory {
	return a.conversationMemory
}
