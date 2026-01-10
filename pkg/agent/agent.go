package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Agent is an LLM-driven agent implementation.
type Agent struct {
	id            string
	role          string
	skills        []core.Skill
	tools         []core.Tool
	memory        core.Memory
	llm           llm.Provider
	tracer        trace.Tracer
	maxIterations int
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
		maxIterations: 10, // default
	}

	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, err
		}
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

// WithSkills assigns semantic skills to the agent.
func WithSkills(skills []core.Skill) Option {
	return func(a *Agent) error {
		a.skills = append([]core.Skill(nil), skills...)
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

// WithMemory attaches a memory backend to the agent.
func WithMemory(memory core.Memory) Option {
	return func(a *Agent) error {
		a.memory = memory
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

// ID returns the agent identifier.
func (a *Agent) ID() string { return a.id }

// Role returns the agent role.
func (a *Agent) Role() string { return a.role }

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
// Implements a ReAct Loop: Thought -> Action -> Observation -> Thought -> Final Answer.
func (a *Agent) Run(ctx context.Context, input any) (any, error) {
	ctx, span := a.tracer.Start(ctx, "Agent.Run")
	defer span.End()

	inputStr, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("agent currently only supports string input")
	}

	// 1. Construct Initial System Prompt and User Message
	messages := []llm.Message{}

	// Construct system prompt with tool instructions if tools are present
	systemPrompt := a.role
	if len(a.tools) > 0 {
		systemPrompt += "\n\nYou have access to the following tools:\n"
		for _, t := range a.tools {
			systemPrompt += fmt.Sprintf("- %s: (Capability)\n", t.Name()) // TODO: add description to Tool interface if needed
		}
		systemPrompt += `
To use a tool, please use the following format:
Thought: Do I need to use a tool? Yes
Action: the action to take, should be one of [`
		toolNames := make([]string, len(a.tools))
		for i, t := range a.tools {
			toolNames[i] = t.Name()
		}
		systemPrompt += strings.Join(toolNames, ", ")
		systemPrompt += `]
Action Input: the input to the action

If you have a result, or do not need a tool, use:
Final Answer: the final answer to the original input question
`
	}

	if systemPrompt != "" {
		messages = append(messages, llm.Message{Role: llm.RoleSystem, Content: systemPrompt})
	}

	// TODO: Retrieve Context from Memory here

	messages = append(messages, llm.Message{Role: llm.RoleUser, Content: inputStr})

	// 2. ReAct Loop
	for i := 0; i < a.maxIterations; i++ {
		// Call LLM
		req := llm.ChatRequest{
			Model:    "default",
			Messages: messages,
		}

		resp, err := a.llm.Chat(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("llm chat failed: %w", err)
		}

		content := resp.Content
		messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: content})

		// Check for Final Answer
		if strings.Contains(content, "Final Answer:") {
			parts := strings.Split(content, "Final Answer:")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1]), nil
			}
			return content, nil
		}

		// Check for Action
		// Simple parsing logic for now.
		// TODO: Make this robust (regex or structured output)
		if strings.Contains(content, "Action:") {
			lines := strings.Split(content, "\n")
			var action, actionInput string

			for _, line := range lines {
				if strings.HasPrefix(line, "Action:") {
					action = strings.TrimSpace(strings.TrimPrefix(line, "Action:"))
				}
				if strings.HasPrefix(line, "Action Input:") {
					actionInput = strings.TrimSpace(strings.TrimPrefix(line, "Action Input:"))
				}
			}

			if action != "" {
				// Initialize as "Not Found"
				var foundTool core.Tool
				for _, t := range a.tools {
					if t.Name() == action {
						foundTool = t
						break
					}
				}

				var observation string
				if foundTool != nil {
					// Tool execution
					// We treat tool Call input as string for this basic implementation
					res, err := foundTool.Call(ctx, actionInput)
					if err != nil {
						observation = fmt.Sprintf("Error executing tool: %v", err)
					} else {
						observation = fmt.Sprintf("%v", res)
					}
				} else {
					observation = fmt.Sprintf("Tool %s not found", action)
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
		if len(a.tools) == 0 {
			return content, nil
		}
	}

	return nil, fmt.Errorf("agent exceeded max iterations (%d) without final answer", a.maxIterations)
}
