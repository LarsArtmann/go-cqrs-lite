package performance_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
)

// P001: repo.Load inside SubscribeAll handler fires.
func TestP001_LoadInSubscribeAll(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func setup() {
	bus.SubscribeAll(func(ctx context.Context, evt Event) error {
		events, _ := repo.Load(ctx, evt.StreamID())
		_ = events
		return nil
	})
}
`,
	})

	findings, err := performance.NewP001Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	if string(findings[0].Rule) != "P001" {
		t.Errorf("expected P001, got %s", findings[0].Rule)
	}
}

// P001: No Load outside handler → no finding.
func TestP001_NoFindingForLoadOutsideHandler(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func loadStream(ctx context.Context, id string) {
	events, _ := repo.Load(ctx, id)
	_ = events
}
`,
	})

	findings, err := performance.NewP001Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}
