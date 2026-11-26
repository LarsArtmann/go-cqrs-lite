package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// Stamping the first schema version is what D013 asks for and what cqrs-gen
// derives by default. With no second version there is nothing to upcast from,
// so F005 must not follow D013 like a trap.
func TestF005_FirstVersionOnlyIsNotAnEvolution(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	event.New("user.created", sid, st, v, p, event.WithSchemaVersion(1))
	event.New("user.updated", sid, st, v, p, event.WithSchemaVersion(event.SchemaVersion(1)))
}
`,
	})

	ruletest.AssertRule(t, ruletest.RunDetector(t, adoption.NewF005Detector(ctx)), "F005", 0)
}

// A variable may name any version, so it counts as an evolving schema.
func TestF005_VariableVersionIsAnEvolution(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _(version event.SchemaVersion) {
	event.New("user.created", sid, st, v, p, event.WithSchemaVersion(version))
}
`,
	})

	ruletest.AssertRule(t, ruletest.RunDetector(t, adoption.NewF005Detector(ctx)), "F005", 1)
}
