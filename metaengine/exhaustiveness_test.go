package metaengine

import (
	"errors"
	"testing"
)

// TestApplyFoldExhaustiveness verifies that every concrete Fold type is handled
// in the applyFold type switch. If a new fold type is added without updating the
// switch, this test fails — preventing silent fallthrough to the default
// errUnknownFoldKind error path.
//
// The test creates one fold of each registered FoldKind via the On constructor,
// then passes each through a mirror of the applyFold type switch. The mirror's
// default case fails the test.
//
// Additionally, the test asserts that the number of created folds matches
// len(AllFoldKinds()), so adding a new FoldKind constant without a corresponding
// fold type + test entry is also caught.
func TestApplyFoldExhaustiveness(t *testing.T) {
	t.Parallel()

	type event struct{ ID string }

	folds := []Fold{
		On(event{}, func(e event) (string, int) { return e.ID, 1 }),          // insertFold
		On(event{}, func(e event, prev int) int { return prev + 1 }),         // updateFold
		On(event{}, Remove[int]()),                                           // removeFold
		On(event{}, func(e event) Delta { return Delta{e.ID: 1} }),           // countFold
		On(event{}, func(e event) Edge { return Edge{From: e.ID, To: "x"} }), // edgeFold
		On(
			event{},
			func(e event) string { return e.ID },
		), // setFold (default in classifySingleReturn)
		On(event{}, func(e event) Skip { return Skip{} }),                       // skipFold
		On(event{}, func(e event) MultiEntry { return MultiEntry{Key: e.ID} }),  // multiInsertFold
		On(event{}, func(e event) Append { return Append{Value: e.ID} }),        // appendFold
		On(event{}, func(e event) Embedding { return Embedding{ID: e.ID} }),     // vectorFold
		On(event{}, func(e event) IndexedText { return IndexedText{ID: e.ID} }), // searchFold
		On(event{}, func(e event) Point { return Point{ID: e.ID, X: 1, Y: 2} }), // spatialFold
	}

	if len(folds) != len(AllFoldKinds()) {
		t.Fatalf("fold count mismatch: created %d folds but AllFoldKinds() returns %d — "+
			"add the new fold type to this test and to applyFold's type switch",
			len(folds), len(AllFoldKinds()))
	}

	// Build a set of all FoldKinds we created, then verify every registered
	// FoldKind is represented.
	seen := make(map[FoldKind]bool, len(folds))
	for _, f := range folds {
		seen[f.Kind()] = true
	}

	for _, kind := range AllFoldKinds() {
		if !seen[kind] {
			t.Errorf("FoldKind %q has no corresponding fold in the test — "+
				"add it to the folds slice", kind)
		}
	}

	// Mirror the applyFold type switch. The default case must never execute.
	for _, f := range folds {
		switch f.(type) {
		case *insertFold:
		case *updateFold:
		case *removeFold:
		case *countFold:
		case *edgeFold:
		case *setFold:
		case *skipFold:
		case *multiInsertFold:
		case *appendFold:
		case *vectorFold:
		case *searchFold:
		case *spatialFold:
		default:
			t.Fatalf("unhandled fold type %T (FoldKind=%s) — "+
				"add this case to applyFold's type switch in store.go",
				f, f.Kind())
		}
	}
}

// TestApplyFoldWrapsErrorWithApplyError verifies that applyFold wraps errors
// with ApplyError, providing structured context (query name, event type, fold
// kind) for debugging.
func TestApplyFoldWrapsErrorWithApplyError(t *testing.T) {
	t.Parallel()

	type event struct{ ID string }

	eng := NewMemoryEngine()
	store, err := Plan(
		[]Engine{eng},
		Query[event, map[string]int](
			"test_exhaustiveness",
			On(event{}, func(e event) (string, int) { return e.ID, 1 }),
		),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// Apply with a payload type that doesn't match the fold's expected event
	// type — this triggers a fold error that should be wrapped in ApplyError.
	ctx := t.Context()
	err = store.Apply(ctx, "event", "wrong payload type")
	if err == nil {
		t.Fatal("expected error from fold with mismatched payload type")
	}

	var applyErr *ApplyError
	if !errors.As(err, &applyErr) {
		t.Fatalf("expected error to be *ApplyError, got %T: %v", err, err)
	}

	if applyErr.Query != "test_exhaustiveness" {
		t.Errorf("ApplyError.Query = %q, want %q", applyErr.Query, "test_exhaustiveness")
	}

	if applyErr.EventType != "event" {
		t.Errorf("ApplyError.EventType = %q, want %q", applyErr.EventType, "event")
	}

	if applyErr.FoldKind != FoldInsert {
		t.Errorf("ApplyError.FoldKind = %q, want %q", applyErr.FoldKind, FoldInsert)
	}

	if applyErr.Cause == nil {
		t.Error("ApplyError.Cause should not be nil")
	}
}
