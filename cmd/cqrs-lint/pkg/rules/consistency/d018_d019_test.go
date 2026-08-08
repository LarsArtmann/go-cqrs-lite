package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestD018_StaleCatalogEntry(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	event.NewEvent("user.created", sid, st, v, p)
}

func _() {
	catalog.NewBuilder("user.deleted", desc)
}
`,
	})

	findings := ruletest.RunDetector(t, consistency.NewD018Detector(ctx))
	ruletest.AssertRule(t, findings, "D018", 1)
}

func TestD018_NoFindingWhenMatching(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	event.NewEvent("user.created", sid, st, v, p)
}

func _() {
	catalog.NewBuilder("user.created", desc)
}
`,
	})

	findings := ruletest.RunDetector(t, consistency.NewD018Detector(ctx))
	ruletest.AssertRule(t, findings, "D018", 0)
}

func TestD019_StaleSpecWithMissingEvents(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	event.NewEvent("user.created", sid, st, v, p)
	event.NewEvent("user.updated", sid, st, v, p)
}

func _() {
	catalog.NewBuilder("user.created", desc)
	catalog.ExportAsyncAPI(reg, "spec.yaml")
}
`,
	})

	findings := ruletest.RunDetector(t, consistency.NewD019Detector(ctx))
	ruletest.AssertRule(t, findings, "D019", 1)
}

func TestD019_NoFindingWhenComplete(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	event.NewEvent("user.created", sid, st, v, p)
}

func _() {
	catalog.NewBuilder("user.created", desc)
	catalog.ExportAsyncAPI(reg, "spec.yaml")
}
`,
	})

	findings := ruletest.RunDetector(t, consistency.NewD019Detector(ctx))
	ruletest.AssertRule(t, findings, "D019", 0)
}

func TestD018_AliasedEventImport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import ev "github.com/larsartmann/go-cqrs-lite/event/v4"

func _() {
	ev.NewEvent("user.created", sid, st, v, p)
}

func _() {
	catalog.NewBuilder("user.created", desc)
}
`,
	})

	findings := ruletest.RunDetector(t, consistency.NewD018Detector(ctx))
	ruletest.AssertRule(t, findings, "D018", 0)
}
