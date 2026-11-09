package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestD014_DetectsMissingJSONTags(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserCreated struct {
	Name  string
	Email string
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD014Detector(ctx))
	ruletest.AssertRule(t, findings, "D014", 2)
}

func TestD014_NoFindingWhenJSONTagsPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserCreated struct {
	Name  string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD014Detector(ctx))
	ruletest.AssertRule(t, findings, "D014", 0)
}

func TestD014_NoFindingForNonEventStruct(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

type Config struct {
	Port string
	Host string
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD014Detector(ctx))
	ruletest.AssertRule(t, findings, "D014", 0)
}

func TestD014_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD014Detector(ctx))
	ruletest.AssertRule(t, findings, "D014", 0)
}

// TestD014_RegistryPayloadTypeAccepted pins the EventPayloadTypes registry
// acceptance path (parity with D016's registry test): a struct whose name
// carries no payload suffix but that the scanner saw as an event.New payload
// is tag-checked like a conventional payload struct.
func TestD014_RegistryPayloadTypeAccepted(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type AccountState struct {
	Balance string
	Owner   string
}
`,
	})
	ctx.Registry.EventPayloadTypes["AccountState"] = true

	findings := ruletest.RunDetector(t, consistency.NewD014Detector(ctx))
	ruletest.AssertRule(t, findings, "D014", 2)
}
