package correctness_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC013_DetectsTimeTimeInEventPayload(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package domain

import "time"

type UserCreatedPayload struct {
	Name      string    ` + "`json:\"name\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 1)
}

func TestC013_DetectsPointerTimeTime(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package domain

import "time"

type OrderShippedEvent struct {
	ShippedAt *time.Time ` + "`json:\"shipped_at\"`" + `
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 1)
}

func TestC013_DetectsByPayloadSuffix(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"battle.go": `package commands

import "time"

type BattleCreatedPayload struct {
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 1)
}

func TestC013_DoesNotFlagNonEventStructs(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package model

import "time"

type User struct {
	Name      string    ` + "`json:\"name\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 0)
}

func TestC013_RespectsAllowPragma(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package domain

import "time"

type UserCreatedPayload struct {
	//cqrs-lint:allow-time-time
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 0)
}

func TestC013_NoTimeTimeNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package domain

type UserCreatedEvent struct {
	Name string ` + "`json:\"name\"`" + `
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 0)
}

// Ensure runDetector and assertRule are available in this test package.
// They are defined in rules_test.go in the same package.

var _ = context.Background // keep import for clarity
