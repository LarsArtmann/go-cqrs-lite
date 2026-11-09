package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// TestC013_EmbeddedTimeTypedGate pins the F091 Tier-3 typed gate for
// file-location-only candidates: a struct in events.go WITHOUT a payload
// suffix (TimestampsCreated) fires only with structural payload evidence —
// a payload-conventional Type() string method or an event.New payload flow —
// while --typed-info=off keeps the historical heuristic.
func TestC013_EmbeddedTimeTypedGate(t *testing.T) {
	t.Parallel()

	const noEvidence = `package main

import "time"

type TimestampsCreated struct {
	time.Time
	Name string
}
`

	tests := []struct {
		name      string
		source    string
		typedMode string
		want      int
	}{
		{
			name:      "TypeMethodConfirms",
			source:    noEvidence + "\nfunc (TimestampsCreated) Type() string { return \"timestamps.created\" }\n",
			typedMode: "on",
			want:      1,
		},
		{
			name:      "SilentWithoutEvidence",
			source:    noEvidence,
			typedMode: "on",
			want:      0,
		},
		{
			name:      "TypedOffKeepsHistoricalHeuristic",
			source:    noEvidence,
			typedMode: "off",
			want:      1,
		},
		{
			name: "EventNewFlowConfirms",
			source: noEvidence + `
func emit() error {
	_, err := event.New("timestamps.created", id, "Timestamps", 1, &TimestampsCreated{})
	return err
}
`,
			typedMode: "on",
			want:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := analyzer.BuildContextFromSource(t, map[string]string{
				"events.go": tt.source,
			})
			ctx.TypedInfoMode = tt.typedMode

			findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
			ruletest.AssertRule(t, findings, "C013", tt.want)
		})
	}
}
