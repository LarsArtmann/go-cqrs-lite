package pebbleengine

import (
	"context"
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestDriverFactory_CellVersioningRejectedLoudly pins the ADR-0141 config
// contract for engines without versioned cells (issue #131): an operator
// requesting cell_versioning gets a construction failure wrapping
// ErrUnsupportedCellVersioning, never a silently unversioned engine.
func TestDriverFactory_CellVersioningRejectedLoudly(t *testing.T) {
	factory, err := metaengine.LookupDriver("pebble")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	policy := metaengine.RetentionPolicy{MaxVersions: 5}
	_, err = factory(context.Background(), metaengine.DriverConfig{
		DSN:            t.TempDir(),
		CellVersioning: &policy,
	})
	if !errors.Is(err, metaengine.ErrUnsupportedCellVersioning) {
		t.Fatalf("errors.Is(ErrUnsupportedCellVersioning) = false, got %v", err)
	}
}
