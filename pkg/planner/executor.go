package planner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jllopis/kairos/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Handler executes a node and can update state.
type Handler func(ctx context.Context, node Node, state *State) (any, error)

// State holds outputs produced during graph execution.
type State struct {
	Last    any
	Outputs map[string]any
}

// NewState creates an initialized execution state.
func NewState() *State {
	return &State{Outputs: make(map[string]any)}
}

// Executor runs a graph using node handlers.
type Executor struct {
	Handlers     map[string]Handler
	HandlersByID map[string]Handler
	AuditStore   AuditStore
	AuditHook    func(ctx context.Context, event AuditEvent)
	RunID        string
	tracer       trace.Tracer
}

// NewExecutor creates an executor with provided handlers.
func NewExecutor(handlers map[string]Handler) *Executor {
	return &Executor{
		Handlers: handlers,
		tracer:   otel.Tracer("kairos/planner"),
	}
}

// AuditEvent captures node execution details for observability.
type AuditEvent struct {
	GraphID    string
	RunID      string
	NodeID     string
	NodeType   string
	Status     string
	Output     any
	Error      string
	StartedAt  time.Time
	FinishedAt time.Time
}

// Execute runs the graph from its start node and returns the final state.
func (e *Executor) Execute(ctx context.Context, graph *Graph, state *State) (*State, error) {
	if graph == nil {
		return nil, fmt.Errorf("graph is nil")
	}
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	if state == nil {
		state = NewState()
	}

	execCtx, execSpan := e.tracer.Start(ctx, "Planner.Execute", trace.WithAttributes(
		telemetry.PlannerAttributes(graph.ID, e.RunID)...,
	))
	defer execSpan.End()

	startID, err := resolveStartNode(graph)
	if err != nil {
		return nil, err
	}

	adjacency, err := buildAdjacency(graph)
	if err != nil {
		return nil, err
	}

	visited := make(map[string]bool)
	currentID := startID
	for currentID != "" {
		if visited[currentID] {
			return nil, fmt.Errorf("cycle detected at node %q", currentID)
		}
		visited[currentID] = true

		node, ok := graph.Nodes[currentID]
		if !ok {
			return nil, fmt.Errorf("node %q not found", currentID)
		}

		handler := e.Handlers[node.Type]
		if e.HandlersByID != nil {
			if byID, ok := e.HandlersByID[node.ID]; ok && byID != nil {
				handler = byID
			}
		}
		if handler == nil {
			return nil, fmt.Errorf("no handler for node type %q", node.Type)
		}

		started := time.Now().UTC()
		if err := e.emitAudit(ctx, AuditEvent{
			GraphID:   graph.ID,
			RunID:     e.RunID,
			NodeID:    node.ID,
			NodeType:  node.Type,
			Status:    "started",
			StartedAt: started,
		}); err != nil {
			return nil, err
		}

		nodeCtx, span := e.tracer.Start(execCtx, "Planner.Node",
			trace.WithAttributes(
				attribute.String("node.id", node.ID),
				attribute.String("node.type", node.Type),
			),
		)
		span.SetAttributes(telemetry.PlannerNodeAttributes(node.ID, node.Type, "started", graph.ID, e.RunID)...)
		if node.Input != nil {
			span.SetAttributes(telemetry.PlannerNodeIO(fmt.Sprint(node.Input), "", 200)...)
		}
		output, err := handler(nodeCtx, node, state)
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(telemetry.PlannerNodeAttributes(node.ID, node.Type, "failed", graph.ID, e.RunID)...)
			span.SetAttributes(telemetry.PlannerNodeIO("", fmt.Sprint(output), 200)...)
			span.End()
			if auditErr := e.emitAudit(ctx, AuditEvent{
				GraphID:    graph.ID,
				RunID:      e.RunID,
				NodeID:     node.ID,
				NodeType:   node.Type,
				Status:     "failed",
				Error:      err.Error(),
				StartedAt:  started,
				FinishedAt: time.Now().UTC(),
			}); auditErr != nil {
				return nil, auditErr
			}
			return nil, fmt.Errorf("node %q failed: %w", node.ID, err)
		}
		span.SetAttributes(telemetry.PlannerNodeAttributes(node.ID, node.Type, "completed", graph.ID, e.RunID)...)
		span.SetAttributes(telemetry.PlannerNodeIO("", fmt.Sprint(output), 200)...)
		span.End()
		if err := e.emitAudit(ctx, AuditEvent{
			GraphID:    graph.ID,
			RunID:      e.RunID,
			NodeID:     node.ID,
			NodeType:   node.Type,
			Status:     "completed",
			Output:     output,
			StartedAt:  started,
			FinishedAt: time.Now().UTC(),
		}); err != nil {
			return nil, err
		}
		state.Outputs[node.ID] = output
		state.Last = output

		next, err := selectNextNode(currentID, adjacency[currentID], graph, state)
		if err != nil {
			return nil, err
		}
		if next == "" {
			break
		}
		currentID = next
	}

	return state, nil
}

func resolveStartNode(graph *Graph) (string, error) {
	if graph.Start != "" {
		if _, ok := graph.Nodes[graph.Start]; !ok {
			return "", fmt.Errorf("start node %q not found", graph.Start)
		}
		return graph.Start, nil
	}

	incoming := make(map[string]int)
	for id := range graph.Nodes {
		incoming[id] = 0
	}
	for _, edge := range graph.Edges {
		incoming[edge.To]++
	}

	var candidates []string
	for id, count := range incoming {
		if count == 0 {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no start node found")
	}
	return "", fmt.Errorf("multiple start nodes found")
}

func buildAdjacency(graph *Graph) (map[string][]string, error) {
	adj := make(map[string][]string, len(graph.Nodes))
	for id := range graph.Nodes {
		adj[id] = nil
	}
	for _, edge := range graph.Edges {
		if edge.From == "" || edge.To == "" {
			return nil, fmt.Errorf("edge must include from/to")
		}
		adj[edge.From] = append(adj[edge.From], edge.To)
	}
	return adj, nil
}

func selectNextNode(currentID string, candidates []string, graph *Graph, state *State) (string, error) {
	if len(candidates) == 0 {
		return "", nil
	}
	edges := edgesFrom(graph, currentID)
	if len(edges) == 0 {
		return "", nil
	}

	var fallback string
	for _, edge := range edges {
		cond := strings.TrimSpace(edge.Condition)
		if cond == "" || cond == "default" || cond == "always" {
			if fallback == "" {
				fallback = edge.To
			}
			continue
		}
		ok, err := evaluateCondition(cond, state)
		if err != nil {
			return "", fmt.Errorf("edge condition %q on %q: %w", cond, currentID, err)
		}
		if ok {
			return edge.To, nil
		}
	}
	return fallback, nil
}

func edgesFrom(graph *Graph, from string) []Edge {
	out := make([]Edge, 0)
	for _, edge := range graph.Edges {
		if edge.From == from {
			out = append(out, edge)
		}
	}
	return out
}

func evaluateCondition(condition string, state *State) (bool, error) {
	switch {
	case strings.HasPrefix(condition, "last=="):
		value := strings.TrimSpace(strings.TrimPrefix(condition, "last=="))
		return fmt.Sprint(state.Last) == value, nil
	case strings.HasPrefix(condition, "last!="):
		value := strings.TrimSpace(strings.TrimPrefix(condition, "last!="))
		return fmt.Sprint(state.Last) != value, nil
	case strings.HasPrefix(condition, "last.contains:"):
		value := strings.TrimSpace(strings.TrimPrefix(condition, "last.contains:"))
		return strings.Contains(fmt.Sprint(state.Last), value), nil
	case strings.HasPrefix(condition, "output."):
		return evalOutputCondition(condition, state)
	default:
		return false, fmt.Errorf("unsupported condition")
	}
}

func evalOutputCondition(condition string, state *State) (bool, error) {
	rest := strings.TrimPrefix(condition, "output.")
	if parts := strings.SplitN(rest, "==", 2); len(parts) == 2 {
		return compareOutputPath(parts[0], parts[1], state, true), nil
	}
	if parts := strings.SplitN(rest, "!=", 2); len(parts) == 2 {
		return compareOutputPath(parts[0], parts[1], state, false), nil
	}
	if parts := strings.SplitN(rest, ".contains:", 2); len(parts) == 2 {
		return containsOutputPath(parts[0], parts[1], state), nil
	}
	return false, fmt.Errorf("invalid output condition")
}

func compareOutputPath(path, rawValue string, state *State, equal bool) bool {
	value := strings.TrimSpace(rawValue)
	got, ok := resolveOutputPath(path, state)
	if !ok {
		return false
	}
	if equal {
		return fmt.Sprint(got) == value
	}
	return fmt.Sprint(got) != value
}

func containsOutputPath(path, rawValue string, state *State) bool {
	value := strings.TrimSpace(rawValue)
	got, ok := resolveOutputPath(path, state)
	if !ok {
		return false
	}
	return strings.Contains(fmt.Sprint(got), value)
}

func resolveOutputPath(path string, state *State) (any, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}
	parts := strings.Split(path, ".")
	nodeID := parts[0]
	output, ok := state.Outputs[nodeID]
	if !ok {
		return nil, false
	}
	if len(parts) == 1 {
		return output, true
	}
	current := output
	for _, key := range parts[1:] {
		value, ok := resolveMapValue(current, key)
		if !ok {
			return nil, false
		}
		current = value
	}
	return current, true
}

func resolveMapValue(value any, key string) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		v, ok := typed[key]
		return v, ok
	case map[string]string:
		v, ok := typed[key]
		return v, ok
	default:
		return nil, false
	}
}

func (e *Executor) emitAudit(ctx context.Context, event AuditEvent) error {
	if e.AuditStore != nil {
		if err := e.AuditStore.Record(ctx, event); err != nil {
			return fmt.Errorf("audit store: %w", err)
		}
	}
	if e.AuditHook != nil {
		e.AuditHook(ctx, event)
	}
	return nil
}
