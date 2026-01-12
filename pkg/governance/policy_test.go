package governance

import (
	"context"
	"testing"
)

func TestRuleSetEvaluate(t *testing.T) {
	rules := []Rule{
		{ID: "deny-mcp", Effect: "deny", Type: ActionMCP, Name: "secrets.*", Reason: "blocked"},
		{ID: "allow-tools", Effect: "allow", Type: ActionTool, Name: "calc.*"},
	}
	engine := NewRuleSet(rules)

	decision := engine.Evaluate(context.Background(), Action{Type: ActionTool, Name: "calc.sum"})
	if !decision.Allowed {
		t.Fatalf("expected allowed")
	}
	decision = engine.Evaluate(context.Background(), Action{Type: ActionMCP, Name: "secrets.read"})
	if decision.Allowed {
		t.Fatalf("expected denied")
	}
	if decision.Reason != "blocked" {
		t.Fatalf("unexpected reason: %s", decision.Reason)
	}
}
