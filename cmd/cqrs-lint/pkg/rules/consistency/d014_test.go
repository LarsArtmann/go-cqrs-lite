package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
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
