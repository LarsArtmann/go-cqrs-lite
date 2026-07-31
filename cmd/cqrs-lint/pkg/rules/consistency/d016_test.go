package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestD016_TooManyFields(t *testing.T) {
	t.Parallel()

	// Build a struct with 25 fields
	src := "package main\n\ntype BigPayloadCreated struct {\n"
	for i := 0; i < 25; i++ {
		src += "\tField  string\n"
	}
	src += "}\n"

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": src,
	})
	findings := ruletest.RunDetector(t, consistency.NewD016Detector(ctx))
	ruletest.AssertRule(t, findings, "D016", 1)
}

func TestD016_UnderLimit(t *testing.T) {
	t.Parallel()

	src := "package main\n\ntype SmallCreated struct {\n"
	for i := 0; i < 5; i++ {
		src += "\tField  string\n"
	}
	src += "}\n"

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": src,
	})
	findings := ruletest.RunDetector(t, consistency.NewD016Detector(ctx))
	ruletest.AssertRule(t, findings, "D016", 0)
}

func TestD016_NotPayloadStruct(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

type Config struct {
	Field1  string
	Field2  string
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD016Detector(ctx))
	ruletest.AssertRule(t, findings, "D016", 0)
}
