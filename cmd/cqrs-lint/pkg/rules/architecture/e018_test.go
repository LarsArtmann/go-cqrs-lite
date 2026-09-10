package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// --- E018: Projection handles an event type that nothing emits ---

func TestE018_FiresOnGhostSubscriptionType(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
		"proj.go": `package main

func subscribe(bus *Bus) {
	bus.Subscribe("user.creted", handler)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE018Detector(ctx))
	ruletest.AssertRule(t, findings, "E018", 1)
}

// The registered form (NewProjection with an event-type slice) is scanned the
// same way as Subscribe-style calls.
func TestE018_FiresOnNewProjectionRegistration(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
		"proj.go": `package main

func build() {
	_ = projection.NewProjection("users", handler, []event.Type{"user.creted"})
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE018Detector(ctx))
	ruletest.AssertRule(t, findings, "E018", 1)
}

func TestE018_NoFindingWhenTypeIsEmitted(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
		"proj.go": `package main

func subscribe(bus *Bus) {
	bus.Subscribe("user.created", handler)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE018Detector(ctx))
	ruletest.AssertRule(t, findings, "E018", 0)
}

// A cataloged type counts as provided: events imported from another service
// are declared via catalog.Event, not emitted locally.
func TestE018_NoFindingWhenTypeIsCataloged(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
		"catalog.go": `package main

func register() {
	catalog.Event("billing.invoice.paid", "Invoice was settled")
}
`,
		"proj.go": `package main

func subscribe(bus *Bus) {
	bus.Subscribe("billing.invoice.paid", handler)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE018Detector(ctx))
	ruletest.AssertRule(t, findings, "E018", 0)
}

// With zero detected emissions the rule stays silent: the emission scanner
// only sees string-literal event.New calls, so an empty producer set cannot
// distinguish a typo from emission sites it failed to parse.
func TestE018_NoFindingWhenNothingIsEmitted(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

func subscribe(bus *Bus) {
	bus.Subscribe("user.creted", handler)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE018Detector(ctx))
	ruletest.AssertRule(t, findings, "E018", 0)
}

// One finding per ghost subscription site: the scanner records each distinct
// consumed type, and each is independently actionable.
func TestE018_ReportsEachGhostType(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
		"proj.go": `package main

func subscribe(bus *Bus) {
	bus.Subscribe("user.creted", handler)
	bus.Subscribe("user.dleted", auditHandler)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE018Detector(ctx))
	ruletest.AssertRule(t, findings, "E018", 2)
}
