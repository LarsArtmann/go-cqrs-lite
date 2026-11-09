package sqliteengine_test

// Stored-bytes pins for the remaining sqliteengine write paths (T1
// follow-up): planned-table writes (execPlannedSet) and MapUpdate re-encodes.
// Same contract as stored_bytes_pin_test.go: raw column reads against literal
// expectations, never derived from the encoder.

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// pinPlanned is a struct, not a map: encoding/json/v2 does not sort map keys,
// so map bytes are iteration-order dependent and un-pinnable. Struct field
// order (declaration order) is the only deterministic shape.
type pinPlanned struct {
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

// TestStoredBytes_PlannedMapSet pins the planned-table path: a collection with
// a LayoutPlan stores the value column as the value's JSON, exactly like the
// base meta_map path. A planned writer that strayed into a different encoding
// (envelope-wrapped, %v stringified) would surface here, not in round-trips.
func TestStoredBytes_PlannedMapSet(t *testing.T) {
	ctx := context.Background()

	db, eng := openPinEngine(t)

	plan := metaengine.BuildLayoutPlan("pin_planned", []string{"status"}, []string{"priority"})
	if err := eng.(metaengine.LayoutPlanApplier).ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("ApplyLayoutPlan: %v", err)
	}

	mb := eng.(metaengine.MapBackend)

	if err := mb.MapSet(ctx, "pin_planned", "k1", pinPlanned{
		Title: "pin", Status: "open", Priority: 3,
	}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	var key, value string

	err := db.QueryRowContext(ctx,
		`SELECT key, value FROM meta_planned_pin_planned`).Scan(&key, &value)
	if err != nil {
		t.Fatalf("query planned row: %v", err)
	}

	if key != "k1" {
		t.Errorf("stored key = %q, want %q", key, "k1")
	}

	// Literal expectation in struct declaration order; the layout columns
	// (status, priority) are extracted into their own columns and do not
	// disturb the value rendering.
	if want := `{"title":"pin","status":"open","priority":3}`; value != want {
		t.Errorf("stored value = %q, want %q", value, want)
	}
}

// TestStoredBytes_MapUpdate_ReEncodes pins the update path's re-encode: the
// row must carry the UPDATED value's JSON — proving MapUpdate persists the
// encoder's rendering of the new value, not a %v of the decoded previous
// value.
func TestStoredBytes_MapUpdate_ReEncodes(t *testing.T) {
	ctx := context.Background()

	db, eng := openPinEngine(t)

	mb := eng.(metaengine.MapBackend)
	updater := eng.(metaengine.MapUpdater)

	if err := mb.MapSet(ctx, "pin-update", "k1", pinValue{Title: "pin", N: 7}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := updater.MapUpdate(ctx, "pin-update", "k1", func(prev any) any {
		return pinValue{Title: "updated", N: 9}
	}); err != nil {
		t.Fatalf("MapUpdate: %v", err)
	}

	var value string

	err := db.QueryRowContext(ctx,
		`SELECT value FROM meta_map WHERE collection = ? AND key = ?`,
		"pin-update", "k1").Scan(&value)
	if err != nil {
		t.Fatalf("query map row: %v", err)
	}

	if want := `{"title":"updated","n":9}`; value != want {
		t.Errorf("stored value = %q, want %q", value, want)
	}
}

// TestStoredBytes_PlannedMapUpdate pins the planned branch of MapUpdate: the
// planned table's value column carries the updated JSON.
func TestStoredBytes_PlannedMapUpdate(t *testing.T) {
	ctx := context.Background()

	db, eng := openPinEngine(t)

	plan := metaengine.BuildLayoutPlan("pin_planned_upd", []string{"status"}, []string{"priority"})
	if err := eng.(metaengine.LayoutPlanApplier).ApplyLayoutPlan(plan); err != nil {
		t.Fatalf("ApplyLayoutPlan: %v", err)
	}

	mb := eng.(metaengine.MapBackend)
	updater := eng.(metaengine.MapUpdater)

	if err := mb.MapSet(ctx, "pin_planned_upd", "k1", pinPlanned{
		Title: "pin", Status: "open", Priority: 1,
	}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := updater.MapUpdate(ctx, "pin_planned_upd", "k1", func(prev any) any {
		return pinPlanned{Title: "updated", Status: "done", Priority: 2}
	}); err != nil {
		t.Fatalf("MapUpdate: %v", err)
	}

	var value string

	err := db.QueryRowContext(ctx,
		`SELECT value FROM meta_planned_pin_planned_upd WHERE key = ?`, "k1").Scan(&value)
	if err != nil {
		t.Fatalf("query planned row: %v", err)
	}

	if want := `{"title":"updated","status":"done","priority":2}`; value != want {
		t.Errorf("stored value = %q, want %q", value, want)
	}
}
