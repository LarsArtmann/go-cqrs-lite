package consistency_test //nolint:dupl // test boilerplate

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestD015_DetectsNullablePointerFields(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserCreated struct {
	Name  *string
	Email *string
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD015Detector(ctx))
	ruletest.AssertRule(t, findings, "D015", 2)
}

func TestD015_NoFindingForValueFields(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserCreated struct {
	Name  string
	Email string
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD015Detector(ctx))
	ruletest.AssertRule(t, findings, "D015", 0)
}

func TestD015_NoFindingForNonEventStruct(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

type Config struct {
	Host *string
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD015Detector(ctx))
	ruletest.AssertRule(t, findings, "D015", 0)
}

func TestD015_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD015Detector(ctx))
	ruletest.AssertRule(t, findings, "D015", 0)
}
