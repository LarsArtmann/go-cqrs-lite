package consistency_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestD016_TooManyFields(t *testing.T) {
	t.Parallel()

	// Build a struct with 25 fields
	src := "package main\n\ntype BigPayloadCreated struct {\n"
	var srcSb16 strings.Builder
	for i := 0; i < 25; i++ {
		srcSb16.WriteString("\tField  string\n")
	}
	src += srcSb16.String()
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
	var srcSb32 strings.Builder
	for i := 0; i < 5; i++ {
		srcSb32.WriteString("\tField  string\n")
	}
	src += srcSb32.String()
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
