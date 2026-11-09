package metaengine

import (
	"io"
	"log"
	"slices"
	"testing"
	"time"
)

// Merge chains callbacks from both sides (receiver first), keeps scalar
// fields from the receiver when set, and adopts the other side's nil
// fields untouched.
func TestHooksMerge_ChainsCallbacksAndKeepsScalars(t *testing.T) {
	t.Parallel()

	var calls []string

	base := Hooks{
		Logger:             log.New(io.Discard, "", 0),
		SlowQueryThreshold: 50 * time.Millisecond,
		OnExecute: func(collection string, _ ReadPattern, _ time.Duration, _ error) {
			calls = append(calls, "base:"+collection)
		},
	}

	other := Hooks{
		OnExecute: func(collection string, _ ReadPattern, _ time.Duration, _ error) {
			calls = append(calls, "other:"+collection)
		},
		OnQuarantined: func(engine string, _ int, _ string) {
			calls = append(calls, "quarantined:"+engine)
		},
	}

	merged := base.Merge(other)

	if merged.Logger != base.Logger {
		t.Fatal("Merge must keep the receiver's logger")
	}

	if merged.SlowQueryThreshold != base.SlowQueryThreshold {
		t.Fatal("Merge must keep the receiver's slow-query threshold")
	}

	if merged.OnExecute == nil || merged.OnQuarantined == nil {
		t.Fatal("Merge must chain both OnExecute sides and adopt OnQuarantined")
	}

	merged.OnExecute("items", ReadPointLookup, 0, nil)
	merged.OnQuarantined("primary", 3, "down")

	want := []string{"base:items", "other:items", "quarantined:primary"}
	if !slices.Equal(calls, want) {
		t.Fatalf("callback order = %v, want %v", calls, want)
	}
}

// Merging into the zero hooks must yield exactly the other side.
func TestHooksMerge_ZeroBaseAdoptsOther(t *testing.T) {
	t.Parallel()

	fired := false
	other := Hooks{OnCatchUp: func(string, int, error) { fired = true }}

	merged := Hooks{}.Merge(other)
	merged.OnCatchUp("primary", 1, nil)

	if !fired {
		t.Fatal("adopted OnCatchUp did not fire")
	}
}

// CurrentHooks returns the zero value when nothing is configured and the
// live hooks otherwise.
func TestStoreCurrentHooks(t *testing.T) {
	t.Parallel()

	store, _, _ := healthTestStore(t)

	if got := store.CurrentHooks(); got.OnFold != nil || got.OnExecute != nil {
		t.Fatalf("CurrentHooks on a fresh store = %+v, want zero value", got)
	}

	want := Hooks{OnProbe: func(string, error) {}}
	WithHooks(store, want)

	got := store.CurrentHooks()
	if got.OnProbe == nil {
		t.Fatal("CurrentHooks did not return the configured OnProbe hook")
	}
}
