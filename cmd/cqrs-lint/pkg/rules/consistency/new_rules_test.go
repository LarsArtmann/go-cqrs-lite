package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
)

// --- D003: Inconsistent logging library ---

func TestD003_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, consistency.NewD003Detector(ctx))
	assertRule(t, findings, "D003", 0)
}

// --- D004: Inconsistent JSON key casing ---

func TestD004_DetectsMixedCasing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": `package main

type User struct {
	FirstName string ` + "`json:\"first_name\"`" + `
	LastName  string ` + "`json:\"lastName\"`" + `
}
`,
	})
	findings := runDetector(t, consistency.NewD004Detector(ctx))
	assertRule(t, findings, "D004", 1)
}

func TestD004_NoFindingForConsistentCasing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": `package main

type User struct {
	FirstName string ` + "`json:\"first_name\"`" + `
	LastName  string ` + "`json:\"last_name\"`" + `
}
`,
	})
	findings := runDetector(t, consistency.NewD004Detector(ctx))
	assertRule(t, findings, "D004", 0)
}

// --- D005: Stale documentation version ---

func TestD005_NoCrashOnEmptyRoot(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, consistency.NewD005Detector(ctx))
	assertRule(t, findings, "D005", 0)
}
