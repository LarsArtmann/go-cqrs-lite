package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// TestC013_EmbeddedTime_TypeMethodConfirmsFileCandidate pins the F091 Tier-3
// typed gate: a struct in events.go WITHOUT a payload suffix (TimestampsCreated)
// is a file-location-only candidate — it fires when a Type() string method
// (the payload-conventional method shape) provides the structural evidence.
func TestC013_EmbeddedTime_TypeMethodConfirmsFileCandidate(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "time"

type TimestampsCreated struct {
	time.Time
	Name string
}

func (TimestampsCreated) Type() string { return "timestamps.created" }
`,
	})
	ctx.TypedInfoMode = "on"

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 1)
}

// TestC013_EmbeddedTime_SilentWithoutEvidence is the suppression half of the
// typed gate: the same file-location-only candidate with NO payload evidence
// (no event.New flow, no Type() method) stays silent — a struct that merely
// lives in events.go is not proof of being an event payload.
func TestC013_EmbeddedTime_SilentWithoutEvidence(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "time"

type TimestampsCreated struct {
	time.Time
	Name string
}
`,
	})
	ctx.TypedInfoMode = "on"

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 0)
}

// TestC013_EmbeddedTime_TypedOffKeepsHistoricalHeuristic pins the fallback:
// with --typed-info=off (and on syntax-only loads) the historical heuristic
// fires on the file-location candidate alone.
func TestC013_EmbeddedTime_TypedOffKeepsHistoricalHeuristic(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "time"

type TimestampsCreated struct {
	time.Time
	Name string
}
`,
	})
	ctx.TypedInfoMode = "off"

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 1)
}

// TestC013_EmbeddedTime_EventNewFlowConfirmsFileCandidate: the other evidence
// channel — the scanner saw the struct flow into an event.New payload
// position — confirms the file-location-only candidate under the typed tier.
func TestC013_EmbeddedTime_EventNewFlowConfirmsFileCandidate(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "time"

type TimestampsCreated struct {
	time.Time
	Name string
}

func emit() error {
	_, err := event.New("timestamps.created", id, "Timestamps", 1, &TimestampsCreated{})
	return err
}
`,
	})
	ctx.TypedInfoMode = "on"

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 1)
}
