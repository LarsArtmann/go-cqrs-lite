package api_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
)

func runDetector(t *testing.T, det finding.Detector) []finding.Finding {
	t.Helper()
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("detector %s: %v", det.Name(), err)
	}

	return findings
}

func assertRule(t *testing.T, findings []finding.Finding, ruleID string, wantCount int) {
	t.Helper()
	count := 0
	for _, f := range findings {
		if string(f.Rule) == ruleID {
			count++
		}
	}
	if count != wantCount {
		t.Errorf("rule %s: got %d findings, want %d", ruleID, count, wantCount)
		for _, f := range findings {
			t.Logf("  finding: %s %s: %s", f.Rule, f.Severity, f.Message)
		}
	}
}

// --- A002: event.NewEvent with json.Marshal ---

func TestA002_DetectsNewEventWithJSONMarshal(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import (
	"encoding/json"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func createEvent(type_ string, id string, streamType string, ver event.Version, payload any) {
	_ = event.NewEvent(type_, id, streamType, ver, json.Marshal(payload))
}
`,
	})
	findings := runDetector(t, api.NewA002Detector(ctx))
	assertRule(t, findings, "A002", 1)
}

// --- A006: Adapter layer wrapping ---

func TestA006_DetectsAdapterMethods(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"adapter.go": `package main

func (a *Adapter) WrapEvent() {}
func (a *Adapter) UnwrapEvent() {}
`,
	})
	findings := runDetector(t, api.NewA006Detector(ctx))
	assertRule(t, findings, "A006", 2)
}

// --- A008: Parallel type system ---

func TestA008_DetectsDuplicateType(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": `package types

type StreamID string
type Version uint64
`,
	})
	findings := runDetector(t, api.NewA008Detector(ctx))
	assertRule(t, findings, "A008", 2)
}

func TestA008_NoFindingInEventPackage(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"event/types.go": `package event

type StreamID string
`,
	})
	findings := runDetector(t, api.NewA008Detector(ctx))
	assertRule(t, findings, "A008", 0)
}

// --- A017: Repository with generic snapshot options is NOT flagged

func TestA017_NoFindingWithGenericSnapshotStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"infra.go": `package main

func setup() {
	repo, _ := decider.NewRepository(
		store, bus, deciderInstance,
		decider.WithSnapshotStore[MyState](snapStore),
		decider.WithSnapshotStrategy[MyState](strategy),
	)
	_ = repo
}
`,
	})
	findings := runDetector(t, api.NewA017Detector(ctx))
	assertRule(t, findings, "A017", 0)
}

func TestA017_DetectsMissingSnapshotStrategy(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"infra.go": `package main

func setup() {
	repo, _ := decider.NewRepository(store, bus, deciderInstance)
	_ = repo
}
`,
	})
	findings := runDetector(t, api.NewA017Detector(ctx))
	assertRule(t, findings, "A017", 1)
}
