package bench_test

import (
	"testing"

	bboltengine "github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	pebbleengine "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// routing_regression_test.go pins planner routing against REAL engine
// profiles (memory + bbolt + pebble). The core module's selection tests use
// synthetic profiles — they prove the math but cannot catch a profile-side
// regression (e.g. an engine losing its ReadCosts, or a recalibration
// accidentally making a KV engine win point lookups against memory).
//
// The exact per-pattern constant pins follow the layout-matrix convention:
// a recalibration that intentionally moves a number updates this test in the
// same commit (see TODO_LIST / ADR-0133 for the calibration protocol).

// TestRealProfiles_ReadCostsPinned pins the per-pattern read costs of the
// real KV engines via NsForRead (medians of 3, measured 2026-08-30).
func TestRealProfiles_ReadCostsPinned(t *testing.T) {
	bboltEng, err := bboltengine.NewBboltEngine("")
	if err != nil {
		t.Skipf("bbolt not available: %v", err)
	}
	defer func() { _ = bboltEng.Close() }()

	pebbleEng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		t.Skipf("pebble not available: %v", err)
	}
	defer func() { _ = pebbleEng.Close() }()

	tests := []struct {
		name    string
		profile metaengine.EngineProfile
		pattern metaengine.ReadPattern
		want    float64
	}{
		{"bbolt point lookup", bboltEng.Profile(), metaengine.ReadPointLookup, 750},
		{"bbolt filtered scan", bboltEng.Profile(), metaengine.ReadFilteredScan, 620},
		{"bbolt aggregate", bboltEng.Profile(), metaengine.ReadAggregate, 100},
		{"bbolt scan", bboltEng.Profile(), metaengine.ReadScan, 660},
		{"pebble point lookup", pebbleEng.Profile(), metaengine.ReadPointLookup, 700},
		{"pebble filtered scan", pebbleEng.Profile(), metaengine.ReadFilteredScan, 830},
		{"pebble aggregate", pebbleEng.Profile(), metaengine.ReadAggregate, 125},
		{"pebble scan", pebbleEng.Profile(), metaengine.ReadScan, 700},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.profile.NsForRead(tt.pattern); got != tt.want {
				t.Errorf(
					"NsForRead(%s) = %.0f, want %.0f — a recalibration moved this constant; update this pin in the same commit with fresh bench medians",
					tt.pattern,
					got,
					tt.want,
				)
			}
		})
	}
}

// TestRealProfiles_PointLookupRoutesToMemory plans a point-lookup query
// across memory + both KV engines and pins that memory wins: its point-lookup
// cost (NsPerOp fallback, ~500ns) must stay below the KV engines' calibrated
// NsPerPointLookup (bbolt 750, pebble 700). A recalibration that pushes a KV
// engine's point lookup below memory's — or a profile regression that drops
// ReadCosts so the scalar fallback (1300/1500-era numbers) resurfaces — fails
// here instead of silently flipping production routing.
func TestRealProfiles_PointLookupRoutesToMemory(t *testing.T) {
	bboltEng, err := bboltengine.NewBboltEngine("")
	if err != nil {
		t.Skipf("bbolt not available: %v", err)
	}

	pebbleEng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		t.Skipf("pebble not available: %v", err)
	}

	// NOTE: Store.Close() closes every engine handed to Plan() — no engine
	// defers here, a second Close panics on pebble.

	type findUser struct {
		UserID string
	}

	type userUpserted struct {
		UserID string
	}

	type userRow struct {
		UserID string
		Status string
	}

	findUserQuery := metaengine.Query[findUser, userRow](
		"route_find_user",
		metaengine.OnRecord(
			userUpserted{},
			func(_ record.Record, e userUpserted) userRow {
				return userRow{UserID: e.UserID, Status: "active"}
			},
		),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine(), bboltEng, pebbleEng},
		findUserQuery,
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer func() { _ = store.Close() }()

	plan := store.Plan()
	if plan == nil {
		t.Fatal("Plan() returned nil plan")
	}

	var assigned string
	found := false

	for _, q := range plan.Queries {
		if q.QueryName == "route_find_user" {
			assigned = q.EngineName
			found = true
		}
	}

	if !found {
		t.Fatal("planned query route_find_user missing from plan")
	}

	if assigned != "memory" {
		t.Errorf(
			"point lookup routed to %q, want \"memory\" — KV point-lookup constants or memory profile drifted; re-run the calibration benches before updating this pin",
			assigned,
		)
	}
}
