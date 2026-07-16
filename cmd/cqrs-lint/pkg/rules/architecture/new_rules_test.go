package architecture_test

import (
	"testing"

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

// --- E003: Missing module boundary ---

func TestE003_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE003Detector(ctx))
	assertRule(t, findings, "E003", 0)
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
