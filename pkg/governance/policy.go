package governance

import (
	"context"
	"path"
	"strings"
)

// ActionType describes the type of action to evaluate.
type ActionType string

const (
	ActionTool  ActionType = "tool"
	ActionAgent ActionType = "agent"
	ActionMCP   ActionType = "mcp"
)

// Action describes a decision target for policy evaluation.
type Action struct {
	Type     ActionType
	Name     string
	Metadata map[string]string
}

// Decision captures the outcome of a policy evaluation.
type Decision struct {
	Allowed bool
	Reason  string
	RuleID  string
}

// PolicyEngine evaluates actions.
type PolicyEngine interface {
	Evaluate(ctx context.Context, action Action) Decision
}

// Rule defines a single policy rule.
type Rule struct {
	ID     string
	Effect string // allow or deny
	Type   ActionType
	Name   string // glob pattern, optional
	Reason string
}

// RuleSet evaluates rules in order.
type RuleSet struct {
	Rules           []Rule
	DefaultDecision Decision
}

// NewRuleSet creates a rule set with a default allow decision.
func NewRuleSet(rules []Rule) *RuleSet {
	return &RuleSet{
		Rules:           append([]Rule(nil), rules...),
		DefaultDecision: Decision{Allowed: true},
	}
}

// Evaluate checks rules in order and returns the first match.
func (r *RuleSet) Evaluate(_ context.Context, action Action) Decision {
	for _, rule := range r.Rules {
		if rule.Type != "" && rule.Type != action.Type {
			continue
		}
		if rule.Name != "" && !matchPattern(rule.Name, action.Name) {
			continue
		}
		decision := Decision{
			Allowed: strings.ToLower(rule.Effect) != "deny",
			Reason:  rule.Reason,
			RuleID:  rule.ID,
		}
		if rule.Effect == "" {
			decision.Allowed = true
		}
		return decision
	}
	return r.DefaultDecision
}

func matchPattern(pattern, value string) bool {
	if pattern == "" {
		return true
	}
	ok, err := path.Match(pattern, value)
	if err == nil && ok {
		return true
	}
	return pattern == value
}
