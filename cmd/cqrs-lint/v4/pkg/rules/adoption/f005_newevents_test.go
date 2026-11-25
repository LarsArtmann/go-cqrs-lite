package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// newEventsOnlyProject emits every event type through event.NewEvents, the
// batch constructor, so the registry sees them only if it reads the slice.
const newEventsOnlyProject = `package main

func _() {
	_, _ = event.NewEvents(sid, st, v, []event.Type{
		"a.one", "a.two", "a.three", "a.four", "a.five", "a.six", "a.seven",
	}, []any{p, p, p, p, p, p, p})
}
`

func TestF005_NewEventsOnlyProject(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{"main.go": newEventsOnlyProject})

	if got := len(ctx.Registry.EventTypesEmitted); got != 7 {
		t.Fatalf("EventTypesEmitted has %d types, want 7", got)
	}

	findings := ruletest.RunDetector(t, adoption.NewF005Detector(ctx))
	ruletest.AssertRule(t, findings, "F005", 1)
}

func TestF005_NewEventsVariableSliceStaysBlind(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{"main.go": `package main

func _(types []event.Type) {
	_, _ = event.NewEvents(sid, st, v, types, []any{p, p, p, p, p, p, p})
}
`})

	findings := ruletest.RunDetector(t, adoption.NewF005Detector(ctx))
	ruletest.AssertRule(t, findings, "F005", 0)
}
