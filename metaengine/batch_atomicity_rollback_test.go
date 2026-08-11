package metaengine_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ── Fake transactional engine with failure injection ──
//
// Backed by an in-memory map, but implements metaengine.Transactional with
// real rollback semantics: RunInTx executes fn on a snapshot and only commits
// if fn returns nil. This mirrors the sqliteengine transaction contract
// (transaction.go) without requiring a SQL backend, and lets us inject a
// deterministic MapSet failure to exercise the Store's batch-atomicity
// rollback path in dispatchFolds.

type failingTxEngine struct {
	mu   sync.Mutex
	data map[string]map[string]any // collection → key → value

	// when failOnSet >= 0, the nth MapSet call (0-based, across the whole
	// engine) returns failErr. Use to make a specific fold fail mid-batch.
	failOnSet int
	failErr   error
	setCalls  int
	commits   int
	rollbacks int
}

func newFailingTxEngine(failOnSet int, failErr error) *failingTxEngine {
	return &failingTxEngine{
		data:      make(map[string]map[string]any),
		failOnSet: failOnSet,
		failErr:   failErr,
	}
}

func (e *failingTxEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name: "failing-tx",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}
}

func (e *failingTxEngine) Close() error { return nil }

var (
	_ metaengine.Engine        = (*failingTxEngine)(nil)
	_ metaengine.MapBackend    = (*failingTxEngine)(nil)
	_ metaengine.Transactional = (*failingTxEngine)(nil)
)

func (e *failingTxEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.setCalls++

	if e.failOnSet >= 0 && e.setCalls == e.failOnSet+1 {
		return e.failErr
	}

	if e.data[col] == nil {
		e.data[col] = make(map[string]any)
	}

	e.data[col][fmt.Sprintf("%v", key)] = value

	return nil
}

func (e *failingTxEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	v, ok := e.data[col][fmt.Sprintf("%v", key)]

	return v, ok, nil
}

func (e *failingTxEngine) MapDelete(ctx context.Context, col string, key any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.data[col], fmt.Sprintf("%v", key))

	return nil
}

// RunInTx snapshots the map before fn, then commits only on success.
// On error, the snapshot is restored — real rollback semantics.
// The lock is NOT held while fn runs: fn calls MapSet/MapGet which take
// e.mu themselves. This mirrors how a real engine would run fn against a
// transaction handle while keeping the fake's data map mutex-guarded.
func (e *failingTxEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	e.mu.Lock()

	snap := make(map[string]map[string]any, len(e.data))
	for c, m := range e.data {
		cp := make(map[string]any, len(m))
		for k, v := range m {
			cp[k] = v
		}

		snap[c] = cp
	}

	e.mu.Unlock()

	err := fn(ctx)

	e.mu.Lock()
	defer e.mu.Unlock()

	if err != nil {
		e.data = snap
		e.rollbacks++

		return err
	}

	e.commits++

	return nil
}

// ── Rollback test ──

type rollbackEvent struct {
	ID   string
	Name string
}

type rollbackByID struct {
	ID   string
	Name string
}

type rollbackByName struct {
	ID   string
	Name string
}

type (
	getRollbackByID   struct{ ID string }
	getRollbackByName struct{ Name string }
)

func TestBatchAtomicity_RollbackOnSecondFoldFailure(t *testing.T) {
	t.Parallel()

	// Set returns an error on the second MapSet call (fold for by_name
	// runs after by_id because queries are sorted by name).
	eng := newFailingTxEngine(1, errors.New("simulated fold failure"))

	q1 := metaengine.Query[getRollbackByID, rollbackByID]("by_id",
		metaengine.OnRecord(rollbackEvent{},
			func(_ record.Record, e rollbackEvent) (string, rollbackByID) {
				return e.ID, rollbackByID(e)
			}))

	q2 := metaengine.Query[getRollbackByName, rollbackByName]("by_name",
		metaengine.OnRecord(rollbackEvent{},
			func(_ record.Record, e rollbackEvent) (string, rollbackByName) {
				return e.Name, rollbackByName(e)
			}))

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q1, q2)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Apply an event: by_id fold writes first (index 0), by_name fold fails
	// (index 1). The failing tx must roll back by_id's write too.
	err = store.Apply(ctx, "rollbackEvent", rollbackEvent{ID: "r1", Name: "alpha"})
	if err == nil {
		t.Fatal("Apply should fail when a fold in the batch fails")
	}

	// Neither collection should contain the partial write — the whole batch
	// rolled back atomically.
	if _, ok, _ := eng.MapGet(ctx, "by_id", "r1"); ok {
		t.Error("by_id/r1 present after rollback — batch was not atomic")
	}

	if _, ok, _ := eng.MapGet(ctx, "by_name", "alpha"); ok {
		t.Error("by_name/alpha present after rollback — batch was not atomic")
	}

	// The Store must report the batch error, not a poisoned collection.
	if poisoned := store.IsPoisoned("by_id"); poisoned != nil {
		t.Errorf("by_id should not be poisoned after fold error (only panics poison): %v", poisoned)
	}

	if poisoned := store.IsPoisoned("by_name"); poisoned != nil {
		t.Errorf("by_name should not be poisoned after fold error: %v", poisoned)
	}

	// The engine must have observed exactly one rollback.
	if eng.rollbacks != 1 {
		t.Errorf("expected 1 rollback, got %d (commits=%d)", eng.rollbacks, eng.commits)
	}

	// Recoverability: a subsequent Apply with the failure cleared must
	// succeed and commit both folds.
	eng.mu.Lock()
	eng.failOnSet = -1
	eng.mu.Unlock()

	err = store.Apply(ctx, "rollbackEvent", rollbackEvent{ID: "r2", Name: "beta"})
	if err != nil {
		t.Fatalf("Apply after clearing failure: %v", err)
	}

	byID, err := metaengine.ExecuteTyped[getRollbackByID, rollbackByID](
		ctx, store, getRollbackByID{ID: "r2"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped by_id: %v", err)
	}

	if byID.Name != "beta" {
		t.Errorf("by_id Name = %q, want %q", byID.Name, "beta")
	}

	byName, err := metaengine.ExecuteTyped[getRollbackByName, rollbackByName](
		ctx, store, getRollbackByName{Name: "beta"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped by_name: %v", err)
	}

	if byName.ID != "r2" {
		t.Errorf("by_name ID = %q, want %q", byName.ID, "r2")
	}
}

// ── Commit semantics: successful batch commits both folds ──

func TestBatchAtomicity_CommitOnAllFoldsSuccess(t *testing.T) {
	t.Parallel()

	eng := newFailingTxEngine(-1, nil) // never fails

	q1 := metaengine.Query[getRollbackByID, rollbackByID]("by_id",
		metaengine.OnRecord(rollbackEvent{},
			func(_ record.Record, e rollbackEvent) (string, rollbackByID) {
				return e.ID, rollbackByID(e)
			}))

	q2 := metaengine.Query[getRollbackByName, rollbackByName]("by_name",
		metaengine.OnRecord(rollbackEvent{},
			func(_ record.Record, e rollbackEvent) (string, rollbackByName) {
				return e.Name, rollbackByName(e)
			}))

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q1, q2)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	err = store.Apply(context.Background(), "rollbackEvent", rollbackEvent{ID: "c1", Name: "gamma"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if eng.commits != 1 {
		t.Errorf("expected 1 commit, got %d", eng.commits)
	}

	if eng.rollbacks != 0 {
		t.Errorf("expected 0 rollbacks, got %d", eng.rollbacks)
	}
}

// ── Non-transactional engine: no rollback, error still propagates ──

// noTxnFailingEngine wraps the map backend but does NOT expose RunInTx:
// it embeds a MapBackend interface value rather than the concrete type,
// so method promotion does not surface the Transactional implementation.
type noTxnFailingEngine struct {
	metaengine.MapBackend
}

// Profile returns the same profile so planning routes queries the same way.
func (e *noTxnFailingEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name: "no-txn-failing",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}
}

func (e *noTxnFailingEngine) Close() error { return nil }

var (
	_ metaengine.Engine     = (*noTxnFailingEngine)(nil)
	_ metaengine.MapBackend = (*noTxnFailingEngine)(nil)
)

// Compile-time negative assertion: must NOT implement Transactional.
var _ = func() bool {
	var e noTxnFailingEngine
	_, ok := any(&e).(metaengine.Transactional)

	return !ok
}()

func TestBatchAtomicity_NonTransactionalEngineNoRollback(t *testing.T) {
	t.Parallel()

	// failOnSet = 1: the by_name fold fails. Without Transactional, the
	// by_id write stays (documented limitation: per-engine atomicity only
	// when the engine implements Transactional).
	inner := newFailingTxEngine(1, errors.New("simulated fold failure"))
	eng := &noTxnFailingEngine{MapBackend: inner}

	q1 := metaengine.Query[getRollbackByID, rollbackByID]("by_id",
		metaengine.OnRecord(rollbackEvent{},
			func(_ record.Record, e rollbackEvent) (string, rollbackByID) {
				return e.ID, rollbackByID(e)
			}))

	q2 := metaengine.Query[getRollbackByName, rollbackByName]("by_name",
		metaengine.OnRecord(rollbackEvent{},
			func(_ record.Record, e rollbackEvent) (string, rollbackByName) {
				return e.Name, rollbackByName(e)
			}))

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q1, q2)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	err = store.Apply(context.Background(), "rollbackEvent", rollbackEvent{ID: "n1", Name: "delta"})
	if err == nil {
		t.Fatal("Apply should still fail when a fold fails (no rollback, error propagates)")
	}

	// Documented limitation: the earlier fold's write remains.
	if _, ok, _ := inner.MapGet(context.Background(), "by_id", "n1"); !ok {
		t.Error("by_id/n1 should remain for non-transactional engine (documented non-atomicity)")
	}
}
