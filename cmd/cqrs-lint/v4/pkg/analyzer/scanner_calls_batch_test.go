package analyzer_test

import (
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

func newEventsSource(typesArg string) map[string]string {
	return map[string]string{
		"main.go": `package main

const (
	Renamed   event.Type = "room.renamed"
	AliasType            = Renamed
)

func _() {
	_, _ = event.NewEvents(sid, st, v, ` + typesArg + `, []any{p, p, p, p, p, p, p})
}
`,
	}
}

func emittedTypes(ctx *analyzer.AnalysisContext) []string {
	types := make([]string, 0, len(ctx.Registry.EventTypesEmitted))
	for eventType := range ctx.Registry.EventTypesEmitted {
		types = append(types, eventType)
	}

	slices.Sort(types)

	return types
}

func TestScanCallExpr_NewEvents_LiteralSlice(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, newEventsSource(
		`[]event.Type{"a.one", "a.two", "a.three", "a.four", "a.five", Renamed, event.Type("a.six"), AliasType}`,
	))

	want := []string{"a.five", "a.four", "a.one", "a.six", "a.three", "a.two", "room.renamed"}
	if got := emittedTypes(ctx); !slices.Equal(got, want) {
		t.Fatalf("emitted = %v, want %v", got, want)
	}

	emission, ok := ctx.Registry.EventTypesEmitted["a.two"]
	if !ok || emission.File != "main.go" || emission.Line == 0 {
		t.Errorf("a.two emission = %+v (present %v)", emission, ok)
	}
}

func TestScanCallExpr_NewEvents_ArrayLiteral(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, newEventsSource(
		`[...]event.Type{"a.one", "a.two"}`,
	))

	if got := emittedTypes(ctx); len(got) != 2 {
		t.Errorf("emitted = %v, want two types", got)
	}
}

func TestScanCallExpr_NewEvents_UnresolvableArgs(t *testing.T) {
	t.Parallel()

	for name, typesArg := range map[string]string{
		"variable slice":    `types`,
		"function call":     `eventTypes()`,
		"built-in append":   `append(types, "a.one")`,
		"empty literal":     `[]event.Type{}`,
		"non-constant elem": `[]event.Type{typeName()}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := analyzer.BuildContextFromSource(t, newEventsSource(typesArg))

			if got := emittedTypes(ctx); len(got) != 0 {
				t.Errorf("emitted = %v, want none", got)
			}
		})
	}
}

func TestScanCallExpr_NewEvents_ForeignPackageIgnored(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	_, _ = other.NewEvents(sid, st, v, []event.Type{"a.one"}, nil)
}
`,
	})

	if got := emittedTypes(ctx); len(got) != 0 {
		t.Errorf("emitted = %v, want none", got)
	}
}
