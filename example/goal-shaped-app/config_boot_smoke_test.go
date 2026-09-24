package main

import (
	"context"
	"path/filepath"
	"testing"
)

// TestShippedConfigBoots smoke-runs the repo's actual cqrs.yaml — the file a
// reader copies verbatim. It guards against configs that compile into the
// binary's option surface but fail at startup (the 2026-09-24 lesson: a
// materialized_views block once shipped on a driver without IVM support and
// killed the default run while the unit tests stayed green).
func TestShippedConfigBoots(t *testing.T) {
	// Resolve the shipped config BEFORE moving, then sandbox the run: the
	// config's relative sqlite DSN ("goal.db") must land in a temp dir, while
	// the config file itself is exercised byte-for-byte as shipped.
	configPath, err := filepath.Abs("cqrs.yaml")
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}

	t.Chdir(t.TempDir())

	if err := run(context.Background(), configPath); err != nil {
		t.Fatalf("shipped cqrs.yaml must boot end-to-end: %v", err)
	}
}
