package correctness_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

// C022: _ = ctx fires.
func TestC022_ContextDiscarded(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func Handle(ctx context.Context, evt Event) error {
	_ = ctx
	return process(evt)
}
`,
	})

	findings, err := correctness.NewC022Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	if string(findings[0].Rule) != "C022" {
		t.Errorf("expected C022, got %s", findings[0].Rule)
	}
}

// C022: No finding when ctx is used.
func TestC022_NoFindingWhenContextUsed(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func Handle(ctx context.Context, evt Event) error {
	return process(ctx, evt)
}
`,
	})

	findings, err := correctness.NewC022Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}
