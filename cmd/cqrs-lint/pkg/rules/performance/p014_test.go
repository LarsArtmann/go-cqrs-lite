package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// TestP014_SilentOnSyntaxOnlyLoads pins the typed-path gate from the other
// side: the in-source harness is syntax-only (no TypesInfo), so even an
// ApplyLayout call on a receiver that carries ApplyLayoutPlan stays silent —
// the rule must never guess receiver types from names.
func TestP014_SilentOnSyntaxOnlyLoads(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"engine.go": `package main

type bothPathsEngine struct{}

func (bothPathsEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
	return nil
}

func (bothPathsEngine) ApplyLayoutPlan(plan []byte) error { return nil }

func callIt() error {
	both := bothPathsEngine{}
	return both.ApplyLayout("tasks", nil, nil)
}
`,
	})

	findings := ruletest.RunDetector(t, performance.NewP014Detector(ctx))

	if len(findings) != 0 {
		for _, f := range findings {
			t.Errorf("P014 fired on a syntax-only load: %s", f.Message)
		}
	}
}
