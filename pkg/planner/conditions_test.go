package planner

import "testing"

func TestEvaluateConditionAdvanced(t *testing.T) {
	state := NewState()
	state.Last = "alpha-beta"
	state.Outputs["node1"] = map[string]any{
		"status": "ok",
		"meta": map[string]any{
			"region": "EMEA",
		},
	}

	cases := []struct {
		cond string
		want bool
	}{
		{"last.contains:beta", true},
		{"last.contains:gamma", false},
		{"output.node1.status==ok", true},
		{"output.node1.status!=ok", false},
		{"output.node1.meta.region==EMEA", true},
		{"output.node1.meta.region!=EMEA", false},
		{"output.node1.status.contains:ok", true},
		{"output.node1.status.contains:fail", false},
	}

	for _, tc := range cases {
		got, err := evaluateCondition(tc.cond, state)
		if err != nil {
			t.Fatalf("condition %q error: %v", tc.cond, err)
		}
		if got != tc.want {
			t.Fatalf("condition %q expected %v, got %v", tc.cond, tc.want, got)
		}
	}
}
