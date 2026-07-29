package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
)

// --- D006: Missing errorfamily classification (regression) ---

func TestD006_DetectsErrorsNew(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"svc.go": `package main

import "errors"

func boom() error {
	return errors.New("boom")
}
`,
	})
	findings := runDetector(t, consistency.NewD006Detector(ctx))
	assertRule(t, findings, "D006", 1)
}

func TestD006_DetectsFmtErrorfWithoutWrap(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"svc.go": `package main

import "fmt"

func boom() error {
	return fmt.Errorf("failed: %s", "x")
}
`,
	})
	findings := runDetector(t, consistency.NewD006Detector(ctx))
	assertRule(t, findings, "D006", 1)
}

// fmt.Errorf WITH %w wraps an error and preserves its classification — not flagged.
func TestD006_NoFindingForFmtErrorfWithWrap(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"svc.go": `package main

import (
	"errors"
	"fmt"
)

var ErrSentinel = errors.New("sentinel")

func boom() error {
	return fmt.Errorf("wrap: %w", ErrSentinel)
}
`,
	})
	findings := runDetector(t, consistency.NewD006Detector(ctx))
	assertRule(t, findings, "D006", 0)
}

// Package-level sentinel errors (var ErrXxx = errors.New(...)) are matched by
// errors.Is, not classified — must NOT be flagged.
func TestD006_NoFindingForPackageLevelSentinel(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"errors.go": `package main

import "errors"

var ErrNotFound = errors.New("not found")
`,
	})
	findings := runDetector(t, consistency.NewD006Detector(ctx))
	assertRule(t, findings, "D006", 0)
}
