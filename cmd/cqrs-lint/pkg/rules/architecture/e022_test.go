package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// --- E022: declared version ladder with a missing rung ---

// The gap class: declared at v3 with ops only from v1 — stored v2 events
// never upcast.
func TestE022_FiresOnLadderGap(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

func declare() {
	_ = schema.Event("user.created", 3,
		schema.RenameField("user.created", 1, "name", "displayName"),
	)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE022Detector(ctx))
	ruletest.AssertRule(t, findings, "E022", 1)
}

// A continuous ladder stays silent: ops cover every version below current.
func TestE022_ContinuousLadderSilent(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

func declare() {
	_ = schema.Event("user.created", 3,
		schema.RenameField("user.created", 1, "name", "displayName"),
		schema.AddField("user.created", 2, "rank", 0),
	)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE022Detector(ctx))
	ruletest.AssertRule(t, findings, "E022", 0)
}

// Current version 1 has nothing below to migrate — never fires.
func TestE022_FirstVersionSilent(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

func declare() {
	_ = schema.Event("user.created", 1)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE022Detector(ctx))
	ruletest.AssertRule(t, findings, "E022", 0)
}

// The builder form's ladder is tracked too: .Event[T]("t", 3) with one op
// from v2 leaves v1 missing.
func TestE022_BuilderFormTracked(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

func declare() {
	_ = system.Schemas().Event[UserCreated]("user.created", 3,
		schema.RemoveField("user.created", 2, "legacyFlag"),
	)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE022Detector(ctx))
	ruletest.AssertRule(t, findings, "E022", 1)
}

// Const-referenced versions stay untracked: the rule stays silent rather
// than guessing the constant's value.
func TestE022_ConstVersionSilent(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"schema.go": `package main

const currentVersion = 5

func declare() {
	_ = schema.Event("user.created", currentVersion)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE022Detector(ctx))
	ruletest.AssertRule(t, findings, "E022", 0)
}
