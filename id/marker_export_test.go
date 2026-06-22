package id_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// TestExportedMarkers_DownstreamUsable verifies the phantom marker types are
// exported so downstream packages (e.g. cqrs-htmx) can reference them as type
// parameters for go-branded-id's BrandNamer, JSON formatters, and other
// type-parameterized tooling. If a marker were unexported this file would not
// compile from an external package.
func TestExportedMarkers_DownstreamUsable(t *testing.T) {
	t.Parallel()

	markers := []struct {
		name string
		id   string
	}{
		{"AggregateMarker", id.New[id.AggregateMarker]().String()},
		{"UserMarker", id.New[id.UserMarker]().String()},
		{"CorrelationMarker", id.New[id.CorrelationMarker]().String()},
		{"RequestMarker", id.New[id.RequestMarker]().String()},
		{"CausationMarker", id.New[id.CausationMarker]().String()},
		{"ClientMarker", id.New[id.ClientMarker]().String()},
		{"CommandMarker", id.New[id.CommandMarker]().String()},
		{"EventMarker", id.New[id.EventMarker]().String()},
	}

	for _, m := range markers {
		if m.id == "" {
			t.Errorf("%s produced empty string representation", m.name)
		}
	}
}
