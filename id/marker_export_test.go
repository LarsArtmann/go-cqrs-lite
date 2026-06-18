package id_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// TestExportedMarkers_DownstreamUsable verifies the phantom marker types are
// exported so downstream packages (e.g. cqrs-htmx) can reference them as type
// parameters for go-branded-id's BrandNamer, JSON formatters, and other
// type-parameterized tooling. If a marker were unexported this file would not
// compile from an external package.
func TestExportedMarkers_DownstreamUsable(t *testing.T) {
	t.Parallel()

	// A downstream package can construct branded IDs straight from the markers,
	// using them as type arguments exactly as BrandNamer / JSON tooling requires.
	userID := id.New[id.UserMarker]()
	corrID := id.New[id.CorrelationMarker]()
	reqID := id.New[id.RequestMarker]()

	if userID.String() == "" {
		t.Error("UserMarker ID has empty string representation")
	}

	if corrID.String() == "" {
		t.Error("CorrelationMarker ID has empty string representation")
	}

	if reqID.String() == "" {
		t.Error("RequestMarker ID has empty string representation")
	}

	// The convenience constructors are unaffected by the rename.
	if id.NewUserID().String() == "" {
		t.Error("NewUserID produced empty ID")
	}
}
