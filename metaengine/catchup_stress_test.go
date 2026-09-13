package metaengine

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// tickInput addresses the single "total" key of the ticks collection.
type tickInput struct{ Key string }

// roleTick is a pure-counter event; every tick addresses the single
// "total" key. roleTickSeed plants the counter cell so the update fold
// never faces a missing key.
type (
	roleTick     struct{ Key string }
	roleTickSeed struct{}
)

func ticksQuery() any {
	return Query[tickInput, int](
		"ticks",
		OnRecord(roleTickSeed{}, func(_ record.Record, _ roleTickSeed) (string, int) {
			return "total", 0
		}),
		OnRecord(roleTick{}, func(_ record.Record, _ roleTick, prev int) int {
			return prev + 1
		}),
	)
}

// catchupStressStore is healthTestStore with the ticks query planned too, so
// both engines fold a non-idempotent counter alongside the idempotent map.
func catchupStressStore(t *testing.T) (store *Store, primary, spare *flakyHealthEngine) {
	t.Helper()

	primary = &flakyHealthEngine{memoryEngine: NewMemoryEngine().(*memoryEngine), name: "primary"}
	spare = &flakyHealthEngine{
		memoryEngine: NewMemoryEngine().(*memoryEngine),
		name:         "spare",
		expense:      map[ReadPattern]float64{ReadPointLookup: 1_000_000},
	}

	store, err := Plan([]Engine{primary, spare}, roleItemQuery(), ticksQuery())
	if err != nil {
		t.Fatal(err)
	}

	WithEventLog(store, NewEventLog())
	t.Cleanup(func() { _ = store.Close() })

	return store, primary, spare
}

// Sequential tests cannot see the stale-snapshot hole: it opens only when
// the EventLog keeps growing WHILE CatchUpEngine replays (writes fail over
// to the spare and still record into the shared log). Writers hammer the
// store for the whole rebuild and past reactivation; the rebuilt primary
// must then fold EVERY event exactly once — misses and double-applies both
// show up as a counter mismatch.
func TestEngineHealth_CatchUpUnderConcurrentApplies(t *testing.T) {
	t.Parallel()

	store, primary, spare := catchupStressStore(t)
	ctx := context.Background()

	if err := store.Apply(
		ctx,
		"roleItemCreated",
		roleItemCreated{ID: "seed", Name: "seed"},
	); err != nil {
		t.Fatal(err)
	}

	if err := store.Apply(ctx, "roleTickSeed", roleTickSeed{}); err != nil {
		t.Fatal(err)
	}

	quarantinePrimary(t, store, primary)

	// Every quarantined-period apply logs a reroute WARN; tens of thousands
	// of them serialize the storm on stderr syscalls and turn the race
	// window into a slog benchmark. Discard logs for the duration.
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(prevLogger) })

	const writers = 8

	var (
		wg        sync.WaitGroup
		stop      = make(chan struct{})
		errs      = make(chan error, writers)
		created   atomic.Int64
		ticked    atomic.Int64
		nextIndex atomic.Int64
	)

	for range writers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-stop:
					return
				default:
				}

				i := nextIndex.Add(1)

				if i%2 == 0 {
					id := fmt.Sprintf("w%d", i)
					if err := store.Apply(
						ctx,
						"roleItemCreated",
						roleItemCreated{ID: id, Name: id},
					); err != nil {
						errs <- fmt.Errorf("apply roleItemCreated: %w", err)

						return
					}

					created.Add(1)
				} else {
					if err := store.Apply(ctx, "roleTick", roleTick{Key: "total"}); err != nil {
						errs <- fmt.Errorf("apply roleTick: %w", err)

						return
					}

					ticked.Add(1)
				}
			}
		}()
	}

	// The rebuild runs INSIDE the write storm: wait until the log is
	// demonstrably growing, then block in CatchUpEngine while writers keep
	// appending — the stabilize loop's passes race fresh appends, and
	// reactivation lands with writers still in flight.
	for store.eventLog.Len() < 32 {
		runtime.Gosched()
	}

	if err := store.CatchUpEngine(ctx, "primary"); err != nil {
		t.Fatalf("CatchUpEngine under concurrent applies: %v", err)
	}

	close(stop)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("writer failed: %v", err)
	}

	primary.armed.Store(false)

	if h := store.HealthSnapshot()["primary"]; h.State != EngineActive {
		t.Fatalf("primary state = %q, want active after catch-up", h.State)
	}

	total, _, err := primary.MapGet(ctx, "ticks", "total")
	if err != nil {
		t.Fatalf("read primary ticks: %v", err)
	}

	if got, ok := total.(int); !ok || got != int(ticked.Load()) {
		t.Fatalf(
			"primary ticks = %v, want exactly %d — an event was missed or double-folded during catch-up",
			total,
			ticked.Load(),
		)
	}

	spareTotal, _, err := spare.MapGet(ctx, "ticks", "total")
	if err != nil {
		t.Fatalf("read spare ticks: %v", err)
	}

	// The spare folds exactly the quarantined-period events (reroute) and
	// nothing after reactivation (folds return to the planned engine), so
	// its count is timing-dependent — but the writers were guaranteed to be
	// applying while quarantined (the log-length spin above), so it must
	// have ingested a strict subset that ends before the primary's total.
	if got, ok := spareTotal.(int); !ok || got < 1 || got > int(ticked.Load()) {
		t.Fatalf("spare ticks = %v, want 1..%d (failover path ingested nothing or overcounted)", spareTotal, ticked.Load())
	}

	if _, ok, _ := primary.MapGet(ctx, "role_items", "seed"); !ok {
		t.Fatal("primary must hold the pre-quarantine seed after rebuild")
	}
}
