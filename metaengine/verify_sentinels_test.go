package metaengine_test

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type verifyNote struct {
	ID   string
	Text string
}

type verifyFind struct {
	ID string
}

func verifyNotesQuery() metaengine.QueryDecl[verifyFind, verifyNote] {
	return metaengine.Query[verifyFind, verifyNote](
		"notes",
		metaengine.OnRecordTyped(
			"note_added",
			verifyNote{},
			func(_ record.Record, e verifyNote) (string, verifyNote) {
				return e.ID, e
			},
		),
	)
}

func newVerifyStore(t *testing.T) *metaengine.Store {
	t.Helper()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		verifyNotesQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	return store
}

// TestVerifySentinels_ExportedAndClassified pins the contract behind issue
// #116: every way Store.Verify can end is a MATCHABLE, CLASSIFIED sentinel, so
// a health/doctor command branches with errors.Is instead of string matching.
func TestVerifySentinels_ExportedAndClassified(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		err    error
		code   string
		family errorfamily.Family
	}{
		{
			"no event log",
			metaengine.ErrNoEventLog,
			"metaengine.no_event_log",
			errorfamily.Rejection,
		},
		{
			"no query decls",
			metaengine.ErrNoQueryDecls,
			"metaengine.no_query_decls",
			errorfamily.Rejection,
		},
		{
			"collection count mismatch",
			metaengine.ErrCollectionCountMismatch,
			"metaengine.collection_count_mismatch",
			errorfamily.Corruption,
		},
		{
			"row-count drift",
			metaengine.ErrVerifyDrift,
			"metaengine.verify_drift",
			errorfamily.Corruption,
		},
	}

	for _, tc := range cases {
		if tc.err == nil {
			t.Fatalf("%s: sentinel is nil", tc.name)
		}

		if got := errorfamily.Code(tc.err); got != tc.code {
			t.Errorf("%s: code = %q, want %q", tc.name, got, tc.code)
		}

		if got := errorfamily.Classify(tc.err); got != tc.family {
			t.Errorf("%s: family = %s, want %s", tc.name, got, tc.family)
		}

		if !errors.Is(tc.err, tc.err) {
			t.Errorf("%s: not matchable via errors.Is", tc.name)
		}
	}
}

// TestVerify_NoEventLog_IsBranchable covers the deployment shape the issue hit:
// under system.New nothing attaches an event log, so Verify is a structural
// no-op. That must be distinguishable from real drift, and must NOT look
// retryable.
func TestVerify_NoEventLog_IsBranchable(t *testing.T) {
	t.Parallel()

	store := newVerifyStore(t)

	if store.EventLogAttached() {
		t.Fatal("fresh store must report no attached event log")
	}

	err := store.Verify(t.Context(), []metaengine.Engine{metaengine.NewMemoryEngine()})
	if !errors.Is(err, metaengine.ErrNoEventLog) {
		t.Fatalf("Verify error = %v, want ErrNoEventLog", err)
	}

	if errors.Is(err, metaengine.ErrVerifyDrift) {
		t.Fatal("the no-op must not read as drift")
	}

	if errorfamily.Classify(err).IsRetryable() {
		t.Error("no-event-log is a structural refusal, it must not be retryable")
	}
}

// TestVerify_DriftIsBranchable pins that real inconsistency is reported as
// ErrVerifyDrift (wrapped with row counts) rather than only by message.
func TestVerify_DriftIsBranchable(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	store := newVerifyStore(t)

	metaengine.WithEventLog(store, metaengine.NewEventLog())

	if !store.EventLogAttached() {
		t.Fatal("EventLogAttached must report the attached log")
	}

	ctx := t.Context()
	if err := store.Apply(ctx, "note_added", verifyNote{ID: "n1", Text: "a"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("memory engine does not implement MapBackend")
	}

	if err := mb.MapSet(ctx, "notes", "rogue", verifyNote{ID: "rogue", Text: "x"}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	// Verify replays the log into a FRESH store; the live store keeps the rogue
	// row, so the row counts disagree.
	err := store.Verify(ctx, []metaengine.Engine{metaengine.NewMemoryEngine()})
	if !errors.Is(err, metaengine.ErrVerifyDrift) {
		t.Fatalf("Verify error = %v, want ErrVerifyDrift", err)
	}

	if errors.Is(err, metaengine.ErrNoEventLog) {
		t.Fatal("drift must not read as a missing event log")
	}
}

// TestBackfill_NoEventLog_IsLoud covers the sibling surprise in issue #116:
// Backfill used to return nil when it had no log to replay from, reporting
// success for work it never did.
func TestBackfill_NoEventLog_IsLoud(t *testing.T) {
	t.Parallel()

	store := newVerifyStore(t)

	err := store.Backfill(t.Context())
	if !errors.Is(err, metaengine.ErrNoEventLog) {
		t.Fatalf("Backfill error = %v, want ErrNoEventLog", err)
	}
}

// TestBackfill_EmptyAttachedLog_IsNoOp makes the distinction explicit: an
// attached-but-empty log means there is genuinely nothing to replay, which is
// success, not the missing-log failure above.
func TestBackfill_EmptyAttachedLog_IsNoOp(t *testing.T) {
	t.Parallel()

	store := newVerifyStore(t)
	metaengine.WithEventLog(store, metaengine.NewEventLog())

	if err := store.Backfill(t.Context()); err != nil {
		t.Fatalf("Backfill with empty log = %v, want nil", err)
	}
}
