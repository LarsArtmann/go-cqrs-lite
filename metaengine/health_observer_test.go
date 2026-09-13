package metaengine

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// observerRecorder captures health-transition hook calls under a mutex.
type observerRecorder struct {
	mu           sync.Mutex
	quarantined  []string
	reactivated  []string
	probes       []string
	catchUps     []string
	catchUpErr   error
	catchUpCount int
	healthReads  int
}

func (r *observerRecorder) attach(store *Store) {
	WithHooks(store, Hooks{
		OnQuarantined: func(engine string, failures int, lastErr string) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.quarantined = append(r.quarantined, engine)

			// Re-entrancy proof: reading health state from inside the hook
			// must not deadlock — the emit happens after healthMu release.
			r.healthReads = len(store.HealthSnapshot())
		},
		OnReactivated: func(engine string, reason string) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.reactivated = append(r.reactivated, engine+":"+reason)
		},
		OnProbe: func(engine string, err error) {
			r.mu.Lock()
			defer r.mu.Unlock()

			outcome := "ok"
			if err != nil {
				outcome = "fail"
			}

			r.probes = append(r.probes, engine+":"+outcome)
		},
		OnCatchUp: func(engine string, replayed int, err error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.catchUps = append(r.catchUps, engine)
			r.catchUpErr = err
			r.catchUpCount = replayed
		},
	})
}

func slicesClone(s []string) []string {
	if s == nil {
		return nil
	}

	out := make([]string, len(s))
	copy(out, s)

	return out
}

// The quarantine hook fires exactly once per transition, with the engine
// name and failure details — later failures on the quarantined engine do
// not re-fire it.
func TestHealthObserver_QuarantineFiresOncePerTransition(t *testing.T) {
	t.Parallel()

	store, primary, _ := healthTestStore(t)
	rec := &observerRecorder{}
	rec.attach(store)

	if err := store.Apply(context.Background(), "roleItemCreated",
		roleItemCreated{ID: "i1", Name: "n1"}); err != nil {
		t.Fatal(err)
	}

	quarantinePrimary(t, store, primary)

	// Keep failing the already-quarantined engine (reroute target executes
	// fine, but poke recordEngineFailure directly for the same engine).
	store.recordEngineFailure(primary, errors.New("still down"))

	rec.mu.Lock()
	quarantined, reads := slicesClone(rec.quarantined), rec.healthReads
	rec.mu.Unlock()

	if len(quarantined) != 1 || quarantined[0] != "primary" {
		t.Fatalf("OnQuarantined calls = %v, want exactly one for primary", quarantined)
	}

	if reads == 0 {
		t.Fatal("hook did not observe health snapshot — re-entrancy read missing")
	}
}

// Manual reactivation reports reason "manual"; a successful probe-driven
// catch-up rebuild reports "catchup" plus the replayed-event count.
func TestHealthObserver_ReactivationReasons(t *testing.T) {
	t.Parallel()

	store, primary, _ := healthTestStore(t)
	rec := &observerRecorder{}
	rec.attach(store)

	if err := store.Apply(context.Background(), "roleItemCreated",
		roleItemCreated{ID: "i1", Name: "n1"}); err != nil {
		t.Fatal(err)
	}

	quarantinePrimary(t, store, primary)
	primary.armed.Store(false)

	if !store.ReactivateEngine("primary") {
		t.Fatal("reactivate should report true for a quarantined engine")
	}

	// Quarantine again, heal, then let the reprobe loop rebuild via catch-up.
	quarantinePrimary(t, store, primary)
	primary.armed.Store(false)
	primary.healed.Store(true)

	store.reprobeOnce(context.Background())

	rec.mu.Lock()
	defer rec.mu.Unlock()

	wantReactivated := []string{"primary:manual", "primary:catchup"}
	if len(rec.reactivated) != len(wantReactivated) {
		t.Fatalf("OnReactivated = %v, want %v", rec.reactivated, wantReactivated)
	}

	for i, want := range wantReactivated {
		if rec.reactivated[i] != want {
			t.Fatalf("OnReactivated[%d] = %q, want %q (all: %v)", i, rec.reactivated[i], want, rec.reactivated)
		}
	}

	if len(rec.probes) != 1 || rec.probes[0] != "primary:ok" {
		t.Fatalf("OnProbe = %v, want one successful primary probe", rec.probes)
	}

	if len(rec.catchUps) != 1 || rec.catchUps[0] != "primary" {
		t.Fatalf("OnCatchUp = %v, want one primary catch-up", rec.catchUps)
	}

	if rec.catchUpErr != nil {
		t.Fatalf("OnCatchUp err = %v, want nil", rec.catchUpErr)
	}

	if rec.catchUpCount != 1 {
		t.Fatalf("OnCatchUp replayed = %d, want 1 (the applied event)", rec.catchUpCount)
	}
}

// A failed probe reports its error; a refused engine stays quarantined and
// no reactivation or catch-up hook fires.
func TestHealthObserver_ProbeFailureReportsError(t *testing.T) {
	t.Parallel()

	store, primary, _ := healthTestStore(t)
	rec := &observerRecorder{}
	rec.attach(store)

	quarantinePrimary(t, store, primary)
	// Still armed and not healed: probe refuses.
	store.reprobeOnce(context.Background())

	rec.mu.Lock()
	defer rec.mu.Unlock()

	if len(rec.probes) != 1 || rec.probes[0] != "primary:fail" {
		t.Fatalf("OnProbe = %v, want one failed primary probe", rec.probes)
	}

	if len(rec.reactivated) != 0 || len(rec.catchUps) != 0 {
		t.Fatalf("failed probe must not recover: reactivated=%v catchups=%v",
			rec.reactivated, rec.catchUps)
	}
}

// The probe-fallback path: an engine that answers its probe but cannot
// catch up (no EventLog) is reactivated without rebuild, reason
// "probe-fallback".
func TestHealthObserver_ProbeFallbackReason(t *testing.T) {
	t.Parallel()

	primary := &flakyHealthEngine{memoryEngine: NewMemoryEngine().(*memoryEngine), name: "primary"}
	spare := &flakyHealthEngine{memoryEngine: NewMemoryEngine().(*memoryEngine), name: "spare"}

	store, err := Plan([]Engine{primary, spare}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })
	// Deliberately NO WithEventLog: catch-up is structurally impossible.

	rec := &observerRecorder{}
	rec.attach(store)

	quarantinePrimary(t, store, primary)
	primary.armed.Store(false)
	primary.healed.Store(true)

	store.reprobeOnce(context.Background())

	rec.mu.Lock()
	defer rec.mu.Unlock()

	if len(rec.reactivated) != 1 || rec.reactivated[0] != "primary:probe-fallback" {
		t.Fatalf("OnReactivated = %v, want [primary:probe-fallback]", rec.reactivated)
	}

	if len(rec.catchUps) != 1 || rec.catchUpErr == nil {
		t.Fatalf("OnCatchUp = %v (err %v), want one failed attempt", rec.catchUps, rec.catchUpErr)
	}
}
