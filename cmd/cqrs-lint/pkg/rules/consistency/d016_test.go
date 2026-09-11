package consistency_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
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

// TestD016_FieldLimitBoundary pins the exactly-20-fields boundary: the rule
// fires only above the limit, so 20 fields stays silent and 21 fires.
func TestD016_FieldLimitBoundary(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		fields    int
		wantCount int
	}{
		{name: "exactly at limit stays silent", fields: 20, wantCount: 0},
		{name: "one over limit fires", fields: 21, wantCount: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := "package main\n\ntype BigPayloadCreated struct {\n"
			var fieldsSb strings.Builder
			for range tc.fields {
				fieldsSb.WriteString("\tField  string\n")
			}
			src += fieldsSb.String() + "}\n"

			ctx := analyzer.BuildContextFromSource(t, map[string]string{
				"events.go": src,
			})
			findings := ruletest.RunDetector(t, consistency.NewD016Detector(ctx))
			ruletest.AssertRule(t, findings, "D016", tc.wantCount)
		})
	}
}

// TestD016_RegistryPayloadTypeAccepted pins the EventPayloadTypes registry
// acceptance path: a struct whose name carries no payload suffix but that
// the scanner saw as an event.New payload is size-checked like D014/D015.
func TestD016_RegistryPayloadTypeAccepted(t *testing.T) {
	t.Parallel()

	src := "package main\n\ntype AccountState struct {\n"
	var fieldsSb strings.Builder
	for range 25 {
		fieldsSb.WriteString("\tField  string\n")
	}
	src += fieldsSb.String() + "}\n"

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": src,
	})
	ctx.Registry.EventPayloadTypes["AccountState"] = true

	findings := ruletest.RunDetector(t, consistency.NewD016Detector(ctx))
	ruletest.AssertRule(t, findings, "D016", 1)
}
