package metaengine

import (
	"context"
	"errors"
	"testing"
)

// quarantinePrimary drives the classified failure storm that quarantines the
// primary engine without touching its (fold-)write path.
func quarantinePrimary(t *testing.T, store *Store, primary *flakyHealthEngine) {
	t.Helper()

	primary.armed.Store(true)

	for range DefaultEngineFailureThreshold {
		if _, err := store.Execute(roleFindItem{ID: "i1"}); err == nil {
			t.Fatal("expected classified failure while armed")
		}
	}

	if h := store.HealthSnapshot()["primary"]; h.State != EngineQuarantined {
		t.Fatalf("primary state = %q, want quarantined", h.State)
	}
}

// ADR-0137 write failover: while the primary is quarantined, folds reroute to
// the healthy spare instead of failing — the write path keeps ingesting, the
// quarantined engine's collections just go stale until catch-up.
func TestEngineHealth_FoldWriteFailover(t *testing.T) {
	t.Parallel()

	store, primary, spare := healthTestStore(t)
	ctx := context.Background()

	if err := store.Apply(ctx, "roleItemCreated", roleItemCreated{ID: "i1", Name: "pre"}); err != nil {
		t.Fatal(err)
	}

	quarantinePrimary(t, store, primary)

	// A fold during quarantine must succeed and land on the failover engine.
	if err := store.Apply(ctx, "roleItemCreated", roleItemCreated{ID: "i2", Name: "during"}); err != nil {
		t.Fatalf("apply during quarantine should fail over, got: %v", err)
	}

	mb, _, err := spare.MapGet(ctx, "role_items", "i2")
	if err != nil || mb == nil {
		t.Fatalf("failover engine must hold the quarantined-period fold (val=%v err=%v)", mb, err)
	}

	// The rerouted read sees it too.
	item, err := store.Execute(roleFindItem{ID: "i2"})
	if err != nil {
		t.Fatalf("execute during quarantine: %v", err)
	}

	if role, ok := item.(roleItem); !ok || role.Name != "during" {
		t.Fatalf("rerouted read = %+v, want the during-quarantine item", item)
	}

	primary.armed.Store(false)
}

// CatchUpEngine is the recovery half: reset + EventLog replay into EXACTLY
// the quarantined engine, quarantine lifted only after a clean rebuild, and
// the rebuilt engine then serves the full history on its planned route.
func TestEngineHealth_CatchUpEngineRebuildsAndReactivates(t *testing.T) {
	t.Parallel()

	store, primary, spare := healthTestStore(t)
	ctx := context.Background()

	if err := store.Apply(ctx, "roleItemCreated", roleItemCreated{ID: "i1", Name: "pre"}); err != nil {
		t.Fatal(err)
	}

	quarantinePrimary(t, store, primary)

	if err := store.Apply(ctx, "roleItemCreated", roleItemCreated{ID: "i2", Name: "during"}); err != nil {
		t.Fatal(err)
	}

	primary.armed.Store(false)

	// Catch-up rebuilds the primary from the log (reset + replay) and lifts
	// the quarantine only afterwards.
	if err := store.CatchUpEngine(ctx, "primary"); err != nil {
		t.Fatalf("CatchUpEngine: %v", err)
	}

	if h := store.HealthSnapshot()["primary"]; h.State != EngineActive {
		t.Fatalf("primary state after catch-up = %q, want active", h.State)
	}

	// The rebuilt primary serves BOTH events — pre-quarantine and the one
	// folded onto the spare while the primary was down.
	for id, want := range map[string]string{"i1": "pre", "i2": "during"} {
		item, err := store.Execute(roleFindItem{ID: id})
		if err != nil {
			t.Fatalf("execute %s after catch-up: %v", id, err)
		}

		if role, ok := item.(roleItem); !ok || role.Name != want {
			t.Errorf("post-catch-up read %s = %+v, want %q", id, item, want)
		}
	}

	// The spare keeps its failover copy (harmless: it is only read when the
	// plan routes to it, and then it is consistent), and its copy was never
	// replayed onto again — no double-apply.
	if _, ok, _ := spare.MapGet(ctx, "role_items", "i2"); !ok {
		t.Fatal("spare copy should survive catch-up (it is the failover copy)")
	}
}

// CatchUpEngine refuses engines that are not quarantined — the rebuild is the
// quarantine-recovery path, not a general-purpose re-fold.
func TestEngineHealth_CatchUpEngineRequiresQuarantine(t *testing.T) {
	t.Parallel()

	store, _, _ := healthTestStore(t)

	err := store.CatchUpEngine(context.Background(), "primary")
	if err == nil {
		t.Fatal("expected an error for a non-quarantined engine")
	}
}

// The auto-reprobe completion prefers a consistent catch-up: a healed engine
// comes back REBUILT, not merely reactivated with stale state.
func TestEngineHealth_ReprobeCatchesUpBeforeReactivation(t *testing.T) {
	t.Parallel()

	store, primary, _ := healthTestStore(t)
	ctx := context.Background()

	if err := store.Apply(ctx, "roleItemCreated", roleItemCreated{ID: "i1", Name: "pre"}); err != nil {
		t.Fatal(err)
	}

	quarantinePrimary(t, store, primary)

	if err := store.Apply(ctx, "roleItemCreated", roleItemCreated{ID: "i2", Name: "during"}); err != nil {
		t.Fatal(err)
	}

	// Heal, then run one reprobe pass (the loop body StartAutoReprobe ticks).
	primary.armed.Store(false)
	primary.healed.Store(true)

	store.reprobeOnce(ctx)

	if h := store.HealthSnapshot()["primary"]; h.State != EngineActive {
		t.Fatalf("primary state after reprobe = %q, want active via catch-up", h.State)
	}

	item, err := store.Execute(roleFindItem{ID: "i2"})
	if err != nil {
		t.Fatalf("execute after reprobe catch-up: %v", err)
	}

	if role, ok := item.(roleItem); !ok || role.Name != "during" {
		t.Fatalf("post-reprobe read = %+v, want the during-quarantine item (rebuilt)", item)
	}
}

// The fallback: when catch-up is structurally impossible (no EventLog),
// CatchUpEngine reports ErrCatchUpUnsupported and a healed engine still
// reactivates the plain way (pre-failover behavior) — loudly.
func TestEngineHealth_ReprobeFallsBackWithoutEventLog(t *testing.T) {
	t.Parallel()

	store, primary, spare := healthTestStore(t)
	ctx := context.Background()
	_ = store // its engines are reused below without the EventLog

	bare, err := Plan([]Engine{primary, spare}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = bare.Close() })

	quarantinePrimary(t, bare, primary)

	if err := bare.CatchUpEngine(ctx, "primary"); err == nil || !errors.Is(err, ErrCatchUpUnsupported) {
		t.Fatalf("CatchUpEngine without EventLog = %v, want ErrCatchUpUnsupported", err)
	}

	if h := bare.HealthSnapshot()["primary"]; h.State != EngineQuarantined {
		t.Fatal("a failed catch-up must leave the engine quarantined")
	}

	primary.armed.Store(false)
	primary.healed.Store(true)

	bare.reprobeOnce(ctx)

	if h := bare.HealthSnapshot()["primary"]; h.State != EngineActive {
		t.Fatalf("primary state after fallback reprobe = %q, want active", h.State)
	}
}
