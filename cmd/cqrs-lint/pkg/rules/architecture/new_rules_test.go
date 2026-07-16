package architecture_test

import (
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/architecture"
)

// --- E001: Layer violation ---

func TestE001_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE001Detector(ctx))
	assertRule(t, findings, "E001", 0)
}

// --- E002: Circular dependency ---

func TestE002_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE002Detector(ctx))
	assertRule(t, findings, "E002", 0)
}

// --- E002: Positive test — circular dependency ---

func TestE002_DetectsCircularDependency(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	ctx.Packages = []*packages.Package{
		{
			PkgPath: "example.com/app/moduleA",
			Imports: map[string]*packages.Package{
				"example.com/app/moduleB": {PkgPath: "example.com/app/moduleB"},
			},
		},
		{
			PkgPath: "example.com/app/moduleB",
			Imports: map[string]*packages.Package{
				"example.com/app/moduleA": {PkgPath: "example.com/app/moduleA"},
			},
		},
	}
	findings := runDetector(t, architecture.NewE002Detector(ctx))
	assertRule(t, findings, "E002", 1)
}

// --- E003: Positive test — missing module boundary ---

func TestE003_DetectsMixedConcerns(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"domain.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type CreateOrder struct {
	*command.BasicCommand
}

type OrderCreated struct {
	Name string
}

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	return s, nil
}
`,
	})
	findings := runDetector(t, architecture.NewE003Detector(ctx))
	assertRule(t, findings, "E003", 1)
}

// --- E004: Event not in catalog ---

func TestE004_NoFindingOnEmptyRegistry(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE004Detector(ctx))
	assertRule(t, findings, "E004", 0)
}

// --- E005: Command without handler ---

func TestE005_NoFindingOnEmptyRegistry(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE005Detector(ctx))
	assertRule(t, findings, "E005", 0)
}

// --- E006: Event without projection ---

func TestE006_NoFindingOnEmptyRegistry(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE006Detector(ctx))
	assertRule(t, findings, "E006", 0)
}

// --- E007: Query without handler ---

func TestE007_DetectsUnregisteredQuery(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"queries.go": `package main

type GetUserQuery struct {
	ID string
}
`,
	})
	findings := runDetector(t, architecture.NewE007Detector(ctx))
	// The rule checks for structs ending in "Query" or "Request"
	// that aren't registered via RegisterTyped
	assertRule(t, findings, "E007", 1)
}

func TestE007_NoFindingForNonQueryStruct(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": `package main

type User struct {
	ID string
}
`,
	})
	findings := runDetector(t, architecture.NewE007Detector(ctx))
	assertRule(t, findings, "E007", 0)
}
