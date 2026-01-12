package governance

import (
	"context"
	"path"
	"strings"

	"github.com/jllopis/kairos/pkg/config"
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
	Status  DecisionStatus
}

// PolicyEngine evaluates actions.
type PolicyEngine interface {
	Evaluate(ctx context.Context, action Action) Decision
}

// ApprovalHook can request a human decision for a policy action.
type ApprovalHook interface {
	Request(ctx context.Context, action Action) Decision
}

// Rule defines a single policy rule.
type Rule struct {
	ID     string
	Effect string // allow, deny, or pending
	Type   ActionType
	Name   string // glob pattern, optional
	Reason string
}

// DecisionStatus captures the policy outcome.
type DecisionStatus string

const (
	DecisionStatusAllow   DecisionStatus = "allow"
	DecisionStatusDeny    DecisionStatus = "deny"
	DecisionStatusPending DecisionStatus = "pending"
)

// RuleSet evaluates rules in order.
type RuleSet struct {
	Rules           []Rule
	DefaultDecision Decision
}

// NewRuleSet creates a rule set with a default allow decision.
func NewRuleSet(rules []Rule) *RuleSet {
	return &RuleSet{
		Rules:           append([]Rule(nil), rules...),
		DefaultDecision: Decision{Allowed: true, Status: DecisionStatusAllow},
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
		decision := Decision{Reason: rule.Reason, RuleID: rule.ID}
		switch strings.ToLower(rule.Effect) {
		case "deny":
			decision.Status = DecisionStatusDeny
		case "pending":
			decision.Status = DecisionStatusPending
		default:
			decision.Status = DecisionStatusAllow
		}
		decision.Allowed = decision.Status == DecisionStatusAllow
		return decision
	}
	return r.DefaultDecision
}

// IsAllowed returns true when the decision permits the action.
func (d Decision) IsAllowed() bool {
	if d.Status == "" {
		return d.Allowed
	}
	return d.Status == DecisionStatusAllow
}

// IsPending returns true when the decision requires approval.
func (d Decision) IsPending() bool {
	return d.Status == DecisionStatusPending
}

// IsDenied returns true when the decision forbids the action.
func (d Decision) IsDenied() bool {
	if d.Status == "" {
		return !d.Allowed
	}
	return d.Status == DecisionStatusDeny
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

// RuleSetFromConfig builds a rule set from config rules.
func RuleSetFromConfig(cfg config.GovernanceConfig) *RuleSet {
	if len(cfg.Policies) == 0 {
		return NewRuleSet(nil)
	}
	rules := make([]Rule, 0, len(cfg.Policies))
	for _, rule := range cfg.Policies {
		if strings.TrimSpace(rule.ID) == "" {
			rule.ID = "rule"
		}
		rules = append(rules, Rule{
			ID:     rule.ID,
			Effect: rule.Effect,
			Type:   ActionType(strings.ToLower(rule.Type)),
			Name:   rule.Name,
			Reason: rule.Reason,
		})
	}
	return NewRuleSet(rules)
}
