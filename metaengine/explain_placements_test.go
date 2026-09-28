package metaengine

import (
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type placementItemCreated struct{ Name string }

// TestQueryPlacements pins the introspection accessor: every planned query
// reports its assigned engine, ADT, the DECLARED volume hint, and the plan's
// estimated latency when a volume exists. Unplanned stores still report the
// declared hints with an empty engine.
func TestQueryPlacements(t *testing.T) {
	t.Parallel()

	type in struct{ Key string }
	type row struct{ Name string }

	highVolume := Query[in, row](
		"high_volume",
		OnRecord(
			placementItemCreated{},
			func(_ record.Record, _ placementItemCreated) (string, row) {
				return "seed", row{Name: "seed"}
			},
		),
		Volume(1_000),
	)
	plain := Query[in, row](
		"plain",
		OnRecord(
			placementItemCreated{},
			func(_ record.Record, e placementItemCreated) (string, row) {
				return "seed", row(e)
			},
		),
	)

	store, err := Plan([]Engine{NewMemoryEngine()}, highVolume, plain)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	placements := store.QueryPlacements()
	if len(placements) != 2 {
		t.Fatalf("placements = %d, want 2", len(placements))
	}

	if !slices.IsSortedFunc(placements, func(a, b QueryPlacement) int {
		return cmpStrings(a.QueryName, b.QueryName)
	}) {
		t.Fatalf("placements must be sorted by name, got %v", placements)
	}

	byName := map[string]QueryPlacement{}
	for _, p := range placements {
		byName[p.QueryName] = p
	}

	hv := byName["high_volume"]
	if hv.Engine != "memory" {
		t.Errorf("high_volume engine = %q, want memory", hv.Engine)
	}
	if hv.Volume != 1_000 {
		t.Errorf("high_volume volume = %d, want 1000 (the declared hint)", hv.Volume)
	}
	if hv.EstimatedLatencyMs <= 0 {
		t.Errorf(
			"high_volume est latency = %v, want > 0 when a volume hint exists",
			hv.EstimatedLatencyMs,
		)
	}

	p := byName["plain"]
	if p.Engine != "memory" {
		t.Errorf("plain engine = %q, want memory", p.Engine)
	}
	if p.Volume != 0 {
		t.Errorf("plain volume = %d, want 0 (no hint declared)", p.Volume)
	}
}

func cmpStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
