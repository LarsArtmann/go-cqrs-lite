package metaengine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// waitFor polls cond until it holds or the deadline expires.
func waitFor(t *testing.T, d time.Duration, cond func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}

		time.Sleep(2 * time.Millisecond)
	}

	return cond()
}

// failingShadowEngine wraps a memory engine whose writes always fail,
// simulating a broken Backup engine.
type failingShadowEngine struct {
	*memoryEngine
	name string
}

func (f *failingShadowEngine) Profile() EngineProfile {
	p := f.memoryEngine.Profile()
	p.Name = f.name

	return p
}

func (f *failingShadowEngine) MapSet(
	ctx context.Context,
	col string,
	key any,
	value any,
) error {
	return errors.New("shadow write failed")
}

// gatedShadowEngine blocks every MapSet until released (or ctx cancelled),
// simulating a hopelessly slow Backup engine for buffer-overflow testing.
type gatedShadowEngine struct {
	*memoryEngine
	name string
	gate chan struct{}
}

func (g *gatedShadowEngine) Profile() EngineProfile {
	p := g.memoryEngine.Profile()
	p.Name = g.name

	return p
}

func (g *gatedShadowEngine) MapSet(
	ctx context.Context,
	col string,
	key any,
	value any,
) error {
	select {
	case <-g.gate:
		return g.memoryEngine.MapSet(ctx, col, key, value)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func mirrorRows(t *testing.T, eng Engine, col string) int {
	t.Helper()

	sb, ok := eng.(ScanBackend)
	if !ok {
		t.Fatal("engine is not a ScanBackend")
	}

	res, err := sb.MapScan(context.Background(), col, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("map scan %s: %v", col, err)
	}

	return len(res.Items)
}

func repApply(t *testing.T, store *Store, n int) {
	repApplyFrom(t, store, 0, n)
}

func repApplyFrom(t *testing.T, store *Store, start, n int) {
	t.Helper()

	ctx := context.Background()

	for i := range n {
		id := i + start
		item := roleItemCreated{ID: fmt.Sprintf("item-%d", id), Name: fmt.Sprintf("n%d", id)}
		if err := store.Apply(ctx, "roleItemCreated", item); err != nil {
			t.Fatalf("apply %d: %v", id, err)
		}
	}
}

// TestReplication_MirrorsAllCollections proves invariant I2: a shadow engine
// receives folds for EVERY matching collection, not just routed ones.
func TestReplication_MirrorsAllCollections(t *testing.T) {
	t.Parallel()

	primary := NewMemoryEngine()
	second := Query[roleFindItem, roleItem](
		"role_items_by_name",
		OnRecord(roleItemCreated{}, func(_ record.Record, e roleItemCreated) (string, roleItem) {
			return e.Name, roleItem{Name: e.Name}
		}),
	)

	store, err := Plan([]Engine{primary}, roleItemQuery(), second)
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	mirror := renamed("mirror")

	if err := store.AddEngine(
		context.Background(), mirror, WithEngineRole(RoleBackup),
	); err != nil {
		t.Fatal(err)
	}

	repApply(t, store, 20)

	if !waitFor(t, 5*time.Second, func() bool {
		return mirrorRows(t, mirror, "role_items") == 20 &&
			mirrorRows(t, mirror, "role_items_by_name") == 20
	}) {
		t.Fatalf(
			"mirror incomplete: role_items=%d role_items_by_name=%d",
			mirrorRows(t, mirror, "role_items"), mirrorRows(t, mirror, "role_items_by_name"),
		)
	}

	status, ok := store.ReplicationStatus("mirror")
	if !ok || status.Stale {
		t.Fatalf("mirror should be healthy, got %+v ok=%v", status, ok)
	}

	if status.Applied != 20 {
		t.Fatalf("applied should be 20, got %d", status.Applied)
	}
}

// TestReplication_FailureIsolated proves invariant I3: a broken shadow never
// fails the primary write path; it goes stale loudly instead.
func TestReplication_FailureIsolated(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngine()}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	broken := &failingShadowEngine{memoryEngine: NewMemoryEngine().(*memoryEngine), name: "broken"}

	if err := store.AddEngine(
		context.Background(), broken, WithEngineRole(RoleBackup),
	); err != nil {
		t.Fatal(err)
	}

	repApply(t, store, 5)

	st, ok := store.ReplicationStatus("broken")
	if !ok || !waitFor(t, 5*time.Second, func() bool {
		st, _ = store.ReplicationStatus("broken")
		return st.Stale
	}) {
		t.Fatalf("broken engine should go stale, got %+v ok=%v", st, ok)
	}

	repApply(t, store, 5)

	if err := store.PromoteEngine(context.Background(), "broken"); err == nil {
		t.Fatal("promoting a stale engine must fail")
	}
}

// TestReplication_BufferOverflowMarksStale fills the replication buffer with a
// gated engine and proves the primary path stays unblocked.
func TestReplication_BufferOverflowMarksStale(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngine()}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	gated := &gatedShadowEngine{
		memoryEngine: NewMemoryEngine().(*memoryEngine), name: "gated", gate: make(chan struct{}),
	}

	if err := store.AddEngine(
		context.Background(), gated, WithEngineRole(RoleBackup),
	); err != nil {
		t.Fatal(err)
	}

	repApply(t, store, replicationBufferJobs+100)

	if !waitFor(t, 5*time.Second, func() bool {
		st, _ := store.ReplicationStatus("gated")
		return st.Stale
	}) {
		t.Fatal("buffer overflow should mark the engine stale")
	}

	repApply(t, store, 10)

	close(gated.gate)
}

// TestPromoteEngine_Cutover runs the full Migration runbook: add mirror,
// backfill, live traffic, promote, retire the primary — reads still work and
// the mirror was complete.
func TestPromoteEngine_Cutover(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngine()}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	WithEventLog(store, NewEventLog())

	repApply(t, store, 10)

	mirror := renamed("mirror")

	if err := store.AddEngine(
		context.Background(), mirror, WithEngineRole(RoleMigration),
	); err != nil {
		t.Fatal(err)
	}

	if err := store.Backfill(context.Background()); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	if got := mirrorRows(t, mirror, "role_items"); got != 10 {
		t.Fatalf("backfill should populate the mirror with 10 rows, got %d", got)
	}

	repApplyFrom(t, store, 10, 10)

	if !waitFor(t, 5*time.Second, func() bool {
		return mirrorRows(t, mirror, "role_items") == 20
	}) {
		t.Fatal("mirror did not catch up with live traffic")
	}

	if err := store.PromoteEngine(context.Background(), "mirror"); err != nil {
		t.Fatalf("promote: %v", err)
	}

	if role, _ := store.EngineRole("mirror"); role != RoleActive {
		t.Fatalf("promoted engine should be Active, got %q", role)
	}

	if err := store.RemoveEngine(context.Background(), "memory"); err != nil {
		t.Fatalf("retire primary: %v", err)
	}

	got, err := ExecuteTyped[roleFindItem, roleItem](
		context.Background(), store, roleFindItem{ID: "item-15"},
	)
	if err != nil {
		t.Fatalf("execute after cutover: %v", err)
	}

	if got.Name != "n15" {
		t.Fatalf("read after cutover returned %+v", got)
	}
}

// TestRegisterQuery_ReplicatesToShadows proves the task snapshot refreshes: a
// query registered AFTER a shadow engine exists still replicates to it.
func TestRegisterQuery_ReplicatesToShadows(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngine()}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	mirror := renamed("mirror")

	if err := store.AddEngine(
		context.Background(), mirror, WithEngineRole(RoleBackup),
	); err != nil {
		t.Fatal(err)
	}

	late := Query[roleFindItem, roleItem](
		"late_items",
		OnRecord(roleItemCreated{}, func(_ record.Record, e roleItemCreated) (string, roleItem) {
			return "latest-" + e.ID, roleItem{Name: e.Name}
		}),
	)

	if err := store.RegisterQuery(late); err != nil {
		t.Fatal(err)
	}

	repApply(t, store, 3)

	if !waitFor(t, 5*time.Second, func() bool {
		return mirrorRows(t, mirror, "late_items") == 3
	}) {
		t.Fatal("late-registered query did not replicate to the shadow")
	}
}

// TestReplication_ConcurrentWithPrimary runs primary applies concurrently with
// active replication. Run with -race.
func TestReplication_ConcurrentWithPrimary(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngine()}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	mirror := renamed("mirror")

	if err := store.AddEngine(
		context.Background(), mirror, WithEngineRole(RoleBackup),
	); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	const goroutines = 8
	const iterations = 50

	wg.Add(goroutines + 1)

	go func() { // replication observer keeps status reads racing with applies
		defer wg.Done()
		for range 100 {
			_, _ = store.ReplicationStatus("mirror")
		}
	}()

	for g := range goroutines {
		go func() {
			defer wg.Done()

			for i := range iterations {
				id := fmt.Sprintf("g%d-i%d", g, i)
				item := roleItemCreated{ID: id, Name: id}
				if err := store.Apply(ctx, "roleItemCreated", item); err != nil {
					t.Errorf("apply: %v", err)
					return
				}
			}
		}()
	}

	wg.Wait()

	if !waitFor(t, 5*time.Second, func() bool {
		return mirrorRows(t, mirror, "role_items") == goroutines*iterations
	}) {
		t.Fatalf(
			"mirror lost writes: want %d, got %d",
			goroutines*iterations, mirrorRows(t, mirror, "role_items"),
		)
	}
}
