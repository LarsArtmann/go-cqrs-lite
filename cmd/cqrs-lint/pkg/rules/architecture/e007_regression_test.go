package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// TestE007_PerTypeRegistrationNotPackageWide verifies that E007 checks each
// query type individually rather than suppressing the entire package. With
// the old PackagesWithRegistration heuristic, registering one query would
// suppress ALL query findings in the same package — even unregistered ones.
// The per-type check fires for DeleteUserQuery while GetUserQuery is suppressed.
func TestE007_PerTypeRegistrationNotPackageWide(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

type GetUserQuery struct{}

func (GetUserQuery) Type() string { return "get_user" }

type DeleteUserQuery struct{}

func (DeleteUserQuery) Type() string { return "delete_user" }

func setup(disp *Dispatcher) {
	query.RegisterTyped[GetUserQuery, any](disp, handler)
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE007Detector(ctx))
	ruletest.AssertRule(t, findings, "E007", 1)
}
