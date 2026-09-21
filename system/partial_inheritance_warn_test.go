package system_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── G-T10 warn-first fixture: evolution covers create/update/delete; the
// projection's own samples deliberately omit the delete tombstone. ──

type PartialCreated struct {
	ID    string
	Title string
}

type PartialUpdated struct {
	ID    string
	Title string
}

type PartialDeleted struct {
	ID string
}

type PartialView struct {
	ID    string
	Title string
}

func partialDomain() system.DomainConfig {
	tasks := system.OnEvolution(
		system.OnEvolution(
			system.Evolve[PartialView]("partial_views"),
			"partial.created", PartialCreated{},
		),
		"partial.updated", PartialUpdated{},
	)
	evolution := system.OnEvolution(tasks, "partial.deleted", PartialDeleted{})

	return system.DomainConfig{
		Evolutions: []system.EvolutionSpec{evolution.Done()},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[PartialView]("partial_views").
				On("partial.created", PartialCreated{}).
				On("partial.updated", PartialUpdated{}).
				Done(),
		},
	}
}

// captureSlog redirects the default logger into a buffer for the test.
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}

	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return buf
}

// TestSystem_PartialSampleProjectionWarns pins the G-T10 warn-first guard: a
// projection with its own samples that do not cover its matching Evolution's
// event types warns loudly, naming the missing types and the ghost-row
// consequence. Behavior is unchanged (the samples still win).
func TestSystem_PartialSampleProjectionWarns(t *testing.T) {
	buf := captureSlog(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := partialDomain()
	domain.Commands = nil

	sys, err := system.New(ctx, domain, tombstoneDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	out := buf.String()
	if !strings.Contains(out, "partial_views") ||
		!strings.Contains(out, "partial.deleted") ||
		!strings.Contains(out, "ghost rows") {
		t.Fatalf("expected partial-inheritance warning naming projection and missing tombstone, got:\n%s", out)
	}
}

// TestSystem_CoveringSamplesDoNotWarn pins the quiet side: samples covering
// every evolution event type (or no matching evolution at all) stay silent.
func TestSystem_CoveringSamplesDoNotWarn(t *testing.T) {
	buf := captureSlog(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := partialDomain()
	domain.Commands = nil
	domain.Projections = []system.ProjectionDeclaration{
		system.Lookup[PartialView]("partial_views").
			On("partial.created", PartialCreated{}).
			On("partial.updated", PartialUpdated{}).
			On("partial.deleted", PartialDeleted{}).
			Done(),
		system.Lookup[PartialView]("inherited_views").Done(),
	}

	sys, err := system.New(ctx, domain, tombstoneDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	if out := buf.String(); strings.Contains(out, "do not cover") {
		t.Fatalf("covering samples must not warn, got:\n%s", out)
	}
}
