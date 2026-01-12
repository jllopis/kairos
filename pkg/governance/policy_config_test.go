package governance

import (
	"context"
	"testing"

	"github.com/jllopis/kairos/pkg/config"
)

func TestRuleSetFromConfig(t *testing.T) {
	cfg := config.GovernanceConfig{
		Policies: []config.PolicyRuleConfig{
			{
				ID:     "deny-tools",
				Effect: "deny",
				Type:   "tool",
				Name:   "danger.*",
				Reason: "blocked",
			},
		},
	}
	engine := RuleSetFromConfig(cfg)
	decision := engine.Evaluate(context.Background(), Action{Type: ActionTool, Name: "danger.rm"})
	if decision.Allowed {
		t.Fatalf("expected denied decision")
	}
	if decision.RuleID != "deny-tools" {
		t.Fatalf("unexpected rule id: %s", decision.RuleID)
	}
}
