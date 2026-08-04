package api

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestA012_PerModuleProfileSuppressesNonSoftDeleteModule verifies that A012
// evaluates HasSoftDelete per-module: a fold function in a library module
// (no soft-delete events) must NOT fire when an example sub-module has soft-delete.
func TestA012_PerModuleProfileSuppressesNonSoftDeleteModule(t *testing.T) {
	t.Parallel()

	source := `package test

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "created":
		s.Count++
	}
	return s, nil
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/fold.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasSoftDelete: false},
		"/repo/examples/app": {HasSoftDelete: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasSoftDelete: true}

	det := NewA012Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("A012 detect: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for library module (no soft-delete), got %d", len(findings))
	}
}

// TestA012_PerModuleProfileFiresForSoftDeleteModule verifies that A012 still
// fires when the fold IS in a module that has soft-delete events.
func TestA012_PerModuleProfileFiresForSoftDeleteModule(t *testing.T) {
	t.Parallel()

	source := `package test

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "created":
		s.Count++
	}
	return s, nil
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/examples/app/fold.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasSoftDelete: false},
		"/repo/examples/app": {HasSoftDelete: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasSoftDelete: true}

	det := NewA012Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("A012 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for soft-delete module, got %d", len(findings))
	}
}

// TestA012_SingleModuleFallback verifies backward compatibility: a
// single-module project falls back to the primary profile.
func TestA012_SingleModuleFallback(t *testing.T) {
	t.Parallel()

	source := `package test

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "created":
		s.Count++
	}
	return s, nil
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": source,
	})

	ctx.FeatureProfile = analyzer.FeatureProfile{HasSoftDelete: true}

	det := NewA012Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("A012 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for single-module soft-delete project, got %d", len(findings))
	}
}
