package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// --- E021: emitted event type without a schema declaration ---

// The typo class: the emitter writes "user.creted" while the declaration
// covers "user.created" — the emitted type silently skips the evolution
// machinery.
func TestE021_FiresOnEmittedButUndeclared(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

func declare() {
	_ = schema.EventOf[UserCreated]("user.created", 2)
}
`,
		"emit.go": `package main

func emit() {
	_ = event.New("user.creted", id, "User", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE021Detector(ctx))
	ruletest.AssertRule(t, findings, "E021", 1)
}

// The system.Schemas() builder form through a local variable — the fluent
// API's primary shape.
func TestE021_BuilderVariableFormCounts(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

func declare() {
	schemas := system.Schemas()
	schemas.Event[UserCreated]("user.created", 2)
	_ = schemas
}
`,
		"emit.go": `package main

func emit() {
	_ = event.New("user.creted", id, "User", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE021Detector(ctx))
	ruletest.AssertRule(t, findings, "E021", 1)
}

// The direct chained form: system.Schemas().Event[T](...).
func TestE021_ChainedBuilderFormCounts(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

func declare() {
	_ = system.Schemas().Event[UserCreated]("user.created", 2)
}
`,
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE021Detector(ctx))
	ruletest.AssertRule(t, findings, "E021", 0)
}

func TestE021_NoFindingWhenDeclared(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

func declare() {
	_ = schema.Event("user.created", 2)
}
`,
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE021Detector(ctx))
	ruletest.AssertRule(t, findings, "E021", 0)
}

// Zero declarations → silence: projects that have not adopted schema
// declarations are not coached into them (E018's zero-silence convention).
func TestE021_SilentWithoutDeclarations(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE021Detector(ctx))
	ruletest.AssertRule(t, findings, "E021", 0)
}

// catalog.Event counts as declared (E018 lockstep): externally imported
// events surfaced in governance are not schema-declaration gaps.
func TestE021_CatalogDeclarationCounts(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

func declare() {
	_ = schema.Event("user.created", 2)
}
`,
		"catalog.go": `package main

func catalog() {
	_ = catalog.Event[ImportedHappened]("import.happened", catalog.Receives)
}
`,
		"emit.go": `package main

func emit() {
	_ = event.New("import.happened", id, "Import", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE021Detector(ctx))
	ruletest.AssertRule(t, findings, "E021", 0)
}

// Constant-referenced declarations resolve through the post-pass, matching
// the emitter's constant on the resolved VALUE, not the identifier.
func TestE021_ConstantDeclarationResolves(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

const evtUserCreated event.Type = "user.created"

func declare() {
	_ = schema.EventOf[UserCreated](evtUserCreated, 2)
}
`,
		"emit.go": `package main

func emit() {
	_ = event.New(evtUserCreated, id, "User", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE021Detector(ctx))
	ruletest.AssertRule(t, findings, "E021", 0)
}
