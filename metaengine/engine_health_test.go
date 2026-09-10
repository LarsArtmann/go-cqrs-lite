package metaengine

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// flakyHealthEngine wraps a memory engine whose reads fail with classified
// errors while armed, and whose Probe answers only while healed. Used to
// drive the ADR-0137 quarantine/reroute/reprobe machinery deterministically.
type flakyHealthEngine struct {
	*memoryEngine

	name    string
	armed   atomic.Bool
	healed  atomic.Bool
	expense map[ReadPattern]float64
}

func (e *flakyHealthEngine) Profile() EngineProfile {
	p := e.memoryEngine.Profile()
	p.Name = e.name

	for pattern, cost := range e.expense {
		switch pattern {
		case ReadPointLookup:
			p.ReadCosts.NsPerPointLookup = cost
		case ReadAggregate:
			p.ReadCosts.NsPerAggregate = cost
		case ReadFilteredScan:
			p.ReadCosts.NsPerFilteredScan = cost
		default:
			p.ReadCosts.NsPerScan = cost
		}
	}

	return p
}

func (e *flakyHealthEngine) failIfArmed() error {
	if !e.armed.Load() {
		return nil
	}

	return errorfamily.Newf(errorfamily.Infrastructure, "healthtest.1", "backend unreachable")
}

func (e *flakyHealthEngine) MapGet(
	ctx context.Context,
	collection string,
	key any,
) (any, bool, error) {
	if err := e.failIfArmed(); err != nil {
		return nil, false, err
	}

	return e.memoryEngine.MapGet(ctx, collection, key)
}

func (e *flakyHealthEngine) Probe(_ context.Context) (time.Duration, error) {
	if e.healed.Load() {
		return 0, nil
	}

	return 0, errorfamily.Newf(errorfamily.Infrastructure, "healthtest.2", "probe refused")
}

func healthTestStore(t *testing.T) (store *Store, primary, spare *flakyHealthEngine) {
	t.Helper()

	primary = &flakyHealthEngine{memoryEngine: NewMemoryEngine().(*memoryEngine), name: "primary"}
	spare = &flakyHealthEngine{
		memoryEngine: NewMemoryEngine().(*memoryEngine),
		name:         "spare",
		expense:      map[ReadPattern]float64{ReadPointLookup: 1_000_000},
	}

	store, err := Plan([]Engine{primary, spare}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	WithEventLog(store, NewEventLog())
	t.Cleanup(func() { _ = store.Close() })

	for _, qa := range store.Plan().Queries {
		if qa.EngineName != "primary" {
			t.Fatalf(
				"role_items routed to %q, want primary (spare must lose planning)",
				qa.EngineName,
			)
		}
	}

	return store, primary, spare
}

// The full ADR-0137 happy path: classified storm quarantines the engine,
// execution reroutes around it, reactivation restores the planned route.
func TestEngineHealth_QuarantineRerouteReactivate(t *testing.T) {
	t.Parallel()

	store, primary, _ := healthTestStore(t)
	ctx := context.Background()

	if err := store.Apply(
		ctx,
		"roleItemCreated",
		roleItemCreated{ID: "i1", Name: "n1"},
	); err != nil {
		t.Fatal(err)
	}

	primary.armed.Store(true)

	for i := range DefaultEngineFailureThreshold {
		if _, err := store.Execute(roleFindItem{ID: "i1"}); err == nil {
			t.Fatalf("execute %d: expected classified failure while armed", i)
		}
	}

	if h := store.HealthSnapshot()["primary"]; h.State != EngineQuarantined {
		t.Fatalf("primary state = %q, want quarantined (health: %+v)", h.State, h)
	}

	// Quarantined: the plan is untouched but execution reroutes to the spare.
	if _, err := store.Execute(roleFindItem{ID: "i1"}); err != nil {
		t.Fatalf("post-quarantine execute should reroute, got: %v", err)
	}

	for _, qa := range store.Plan().Queries {
		if qa.EngineName != "primary" {
			t.Fatalf("plan mutated by quarantine: %q now on %q", qa.QueryName, qa.EngineName)
		}
	}

	// Stats + Doctor surface the deactivation.
	found := false

	for _, es := range store.GetEngineStats(ctx) {
		if es.Name == "primary" {
			found = true

			if es.Health.State != EngineQuarantined ||
				es.Health.ConsecutiveFailures < DefaultEngineFailureThreshold {
				t.Errorf("stats health = %+v, want quarantined with failures", es.Health)
			}
		}
	}

	if !found {
		t.Fatal("primary missing from engine stats")
	}

	if doc := store.Doctor(ctx); !containsAll(doc, "QUARANTINED", "primary") {
		t.Errorf("doctor report lacks quarantine line:\n%s", doc)
	}

	// Heal + explicit reactivation restores the planned (and now healthy) route.
	primary.armed.Store(false)

	if !store.ReactivateEngine("primary") {
		t.Fatal("reactivate should report true for a quarantined engine")
	}

	item, err := store.Execute(roleFindItem{ID: "i1"})
	if err != nil {
		t.Fatalf("execute after reactivation: %v", err)
	}

	if role, ok := item.(roleItem); !ok || role.Name != "n1" {
		t.Errorf("post-reactivation result = %+v, want the seeded item from primary", item)
	}
}

// Rejection-class failures never quarantine: client mistakes fail loudly
// forever instead of triggering failover.
func TestEngineHealth_RejectionNeverQuarantines(t *testing.T) {
	t.Parallel()

	rejecting := &rejectingHealthEngine{
		memoryEngine: NewMemoryEngine().(*memoryEngine),
		name:         "reject",
	}

	store, err := Plan([]Engine{rejecting}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	for range 5 {
		if _, err := store.Execute(roleFindItem{ID: "x"}); err == nil {
			t.Fatal("expected the rejection error to surface")
		}
	}

	if h := store.HealthSnapshot()["reject"]; h.State == EngineQuarantined ||
		h.ConsecutiveFailures != 0 {
		t.Errorf("rejection failures must not count toward health: %+v", h)
	}
}

// With no healthy alternative, a quarantined engine's error still surfaces —
// fail loud, never silently wrong.
func TestEngineHealth_NoCandidateFailsLoudly(t *testing.T) {
	t.Parallel()

	solo := &flakyHealthEngine{memoryEngine: NewMemoryEngine().(*memoryEngine), name: "solo"}

	store, err := Plan([]Engine{solo}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	solo.armed.Store(true)

	for i := range DefaultEngineFailureThreshold + 2 {
		if _, err := store.Execute(roleFindItem{ID: "x"}); err == nil {
			t.Fatalf("execute %d: expected the engine error to surface with no reroute target", i)
		}
	}

	if h := store.HealthSnapshot()["solo"]; h.State != EngineQuarantined {
		t.Fatalf("solo state = %q, want quarantined", h.State)
	}
}

// The reprobe loop reactivates a quarantined engine once its Prober answers.
func TestEngineHealth_AutoReprobeReactivates(t *testing.T) {
	t.Parallel()

	store, primary, _ := healthTestStore(t)
	ctx := context.Background()
	_ = store.Apply(ctx, "roleItemCreated", roleItemCreated{ID: "i2", Name: "n2"})

	primary.armed.Store(true)

	for range DefaultEngineFailureThreshold {
		if _, err := store.Execute(roleFindItem{ID: "i2"}); err == nil {
			t.Fatal("expected failure while armed")
		}
	}

	primary.armed.Store(false)
	primary.healed.Store(true)

	stop := store.StartAutoReprobe(ctx, 10*time.Millisecond)
	defer stop()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if h := store.HealthSnapshot()["primary"]; h.State == EngineActive {
			if _, err := store.Execute(roleFindItem{ID: "i2"}); err != nil {
				t.Fatalf("execute after auto-reprobe: %v", err)
			}

			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("auto-reprobe did not reactivate the healed engine in time")
}

// SetEngineFailureThreshold is honored: one classified failure quarantines.
func TestEngineHealth_ThresholdOverride(t *testing.T) {
	t.Parallel()

	store, primary, _ := healthTestStore(t)
	store.SetEngineFailureThreshold(1)

	primary.armed.Store(true)

	if _, err := store.Execute(roleFindItem{ID: "z"}); err == nil {
		t.Fatal("expected failure while armed")
	}

	if h := store.HealthSnapshot()["primary"]; h.State != EngineQuarantined {
		t.Fatalf("state after single failure with threshold 1 = %q, want quarantined", h.State)
	}
}

// rejectingHealthEngine fails reads with a Rejection-family error — the
// family that must never count toward engine health.
type rejectingHealthEngine struct {
	*memoryEngine

	name string
}

func (e *rejectingHealthEngine) Profile() EngineProfile {
	p := e.memoryEngine.Profile()
	p.Name = e.name

	return p
}

func (e *rejectingHealthEngine) MapGet(
	_ context.Context,
	_ string,
	_ any,
) (any, bool, error) {
	return nil, false, errorfamily.Newf(errorfamily.Rejection, "healthtest.3", "bad request shape")
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}

	return true
}
