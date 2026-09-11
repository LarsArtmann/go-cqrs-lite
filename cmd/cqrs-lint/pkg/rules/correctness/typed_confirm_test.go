package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// --- C013 view branch: file-only candidates need a serialization point ---

// TestC013_TypedTier_ViewFileCandidateJSONTagGate: a struct in views.go
// without a view suffix fires only when the struct carries a json tag (the
// serialization reflection point that marks it as a served view);
// --typed-info=off keeps the historical heuristic.
func TestC013_TypedTier_ViewFileCandidateJSONTagGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		typedMode string
		withTag   bool
		want      int
	}{
		{name: "SilentWithoutJSONTag", typedMode: "on", want: 0},
		{name: "JSONTagConfirms", typedMode: "on", withTag: true, want: 1},
		{name: "TypedOffKeepsHistoricalHeuristic", typedMode: "off", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := ""
			if tt.withTag {
				tag = " `json:\"openedAt\"`"
			}

			ctx := analyzer.BuildContextFromSource(t, map[string]string{
				"views.go": "package views\n\nimport \"time\"\n\ntype AccountSummary struct {\n\tOpenedAt time.Time" + tag + "\n}\n",
			})
			ctx.TypedInfoMode = tt.typedMode

			findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
			ruletest.AssertRule(t, findings, "C013", tt.want)
		})
	}
}

// TestC013_TypedOff_ViewFileCandidateKeepsHistoricalHeuristic pins the
// fallback: without the typed tier the file location alone fires the view
// branch.
func TestC013_TypedOff_ViewFileCandidateKeepsHistoricalHeuristic(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"views.go": `package views

import "time"

type AccountSummary struct {
	OpenedAt time.Time
}
`,
	})
	ctx.TypedInfoMode = "off"

	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 1)
}

// --- C035: weak candidates need live map-field evidence ---

// TestC035_TypedTier_WeakCandidateSilentWithoutMapUse: a generic STORE-suffix
// struct whose map field is never referenced anywhere stays silent under the
// typed tier — inert DTO, not live shared state.
func TestC035_TypedTier_WeakCandidateSilentWithoutMapUse(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

type SessionStore struct {
	Sessions map[string]string
}
`,
	})
	ctx.TypedInfoMode = "on"

	findings := ruletest.RunDetector(t, correctness.NewC035Detector(ctx))
	ruletest.AssertRule(t, findings, "C035", 0)
}

// TestC035_TypedTier_MapUseConfirmsWeakCandidate: a selector reference to the
// map field anywhere in the analyzed files confirms the weak candidate.
func TestC035_TypedTier_MapUseConfirmsWeakCandidate(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

type SessionStore struct {
	Sessions map[string]string
}

func lookup(s *SessionStore, key string) string {
	return s.Sessions[key]
}
`,
	})
	ctx.TypedInfoMode = "on"

	findings := ruletest.RunDetector(t, correctness.NewC035Detector(ctx))
	ruletest.AssertRule(t, findings, "C035", 1)
}

// TestC035_TypedOff_WeakCandidateKeepsHistoricalHeuristic pins the fallback:
// without the typed tier the generic suffix alone fires.
func TestC035_TypedOff_WeakCandidateKeepsHistoricalHeuristic(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

type SessionStore struct {
	Sessions map[string]string
}
`,
	})
	ctx.TypedInfoMode = "off"

	findings := ruletest.RunDetector(t, correctness.NewC035Detector(ctx))
	ruletest.AssertRule(t, findings, "C035", 1)
}

// TestC035_TypedTier_StrongSuffixNeverNeedsEvidence: an explicit read-model
// name fires under the typed tier even when the map field is never referenced
// — strong candidates skip the confirmation gate (C008 Tier-2 pattern).
func TestC035_TypedTier_StrongSuffixNeverNeedsEvidence(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

type UserProjection struct {
	Users map[string]string
}
`,
	})
	ctx.TypedInfoMode = "on"

	findings := ruletest.RunDetector(t, correctness.NewC035Detector(ctx))
	ruletest.AssertRule(t, findings, "C035", 1)
}

// TestC035_TypedTier_FileCandidateNeedsMapUse: a struct in handler.go without
// any read-model suffix is a file-location candidate — gated like the weak
// suffixes.
func TestC035_TypedTier_FileCandidateNeedsMapUse(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

type Draft struct {
	Values map[string]string
}
`,
	})
	ctx.TypedInfoMode = "on"

	findings := ruletest.RunDetector(t, correctness.NewC035Detector(ctx))
	ruletest.AssertRule(t, findings, "C035", 0)
}
