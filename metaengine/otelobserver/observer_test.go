package otelobserver_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/otelobserver/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// flaky wraps a memory engine whose reads fail while armed and whose probe
// answers only while healed — the public-API twin of metaengine's internal
// test fixture, driving quarantine/reprobe/catch-up deterministically.
type flaky struct {
	metaengine.Engine

	name   string
	armed  chan struct{}
	healed chan struct{}
}

func (e *flaky) Profile() metaengine.EngineProfile {
	p := e.Engine.Profile()
	p.Name = e.name

	return p
}

func (e *flaky) MapGet(
	ctx context.Context,
	collection string,
	key any,
) (any, bool, error) {
	select {
	case <-e.armed:
		return nil, false, errorfamily.Newf(
			errorfamily.Infrastructure,
			"obs.1",
			"backend unreachable",
		)
	default:
	}

	mb, ok := e.Engine.(metaengine.MapBackend)
	if !ok {
		tErr := errorfamily.Newf(errorfamily.Infrastructure, "obs.3", "engine lacks MapBackend")
		return nil, false, tErr
	}

	return mb.MapGet(ctx, collection, key)
}

func (e *flaky) Probe(context.Context) (time.Duration, error) {
	select {
	case <-e.healed:
		return 0, nil
	default:
	}

	return 0, errorfamily.Newf(errorfamily.Infrastructure, "obs.2", "probe refused")
}

// ResetEngine delegates to the wrapped engine, keeping flaky catch-up
// capable (interface embedding does not promote EngineResetter).
func (e *flaky) ResetEngine(ctx context.Context) error {
	if r, ok := e.Engine.(metaengine.EngineResetter); ok {
		return r.ResetEngine(ctx) //nolint:wrapcheck // passthrough delegation
	}

	return nil
}

// delegate wraps the engine's optional capability so the wrapper satisfies
// metaengine's capability audits (they type-assert the outermost engine).
func (e *flaky) delegate() (metaengine.MapBackend, bool) {
	mb, ok := e.Engine.(metaengine.MapBackend)

	return mb, ok
}

func (e *flaky) MapSet(ctx context.Context, collection string, key any, value any) error {
	mb, ok := e.delegate()
	if !ok {
		return errorfamily.Newf(errorfamily.Infrastructure, "obs.4", "engine lacks MapBackend")
	}

	return mb.MapSet(ctx, collection, key, value)
}

func (e *flaky) MapDelete(ctx context.Context, collection string, key any) error {
	mb, ok := e.delegate()
	if !ok {
		return errorfamily.Newf(errorfamily.Infrastructure, "obs.4", "engine lacks MapBackend")
	}

	return mb.MapDelete(ctx, collection, key)
}

type itemCreated struct {
	ID   string
	Name string
}

type findItem struct {
	ID string
}

type item struct{ Name string }

func observerTestStore(t *testing.T) (
	store *metaengine.Store,
	primary, spare *flaky,
) {
	t.Helper()

	primary = &flaky{
		Engine: metaengine.NewMemoryEngine(), name: "primary",
		armed: make(chan struct{}), healed: make(chan struct{}),
	}
	spare = &flaky{
		Engine: metaengine.NewMemoryEngine(), name: "spare",
		armed: make(chan struct{}), healed: make(chan struct{}),
	}

	query := metaengine.Query[findItem, item]("items",
		metaengine.OnRecord(itemCreated{}, func(_ record.Record, e itemCreated) (string, item) {
			return e.ID, item{Name: e.Name}
		}),
	)

	var err error

	store, err = metaengine.Plan([]metaengine.Engine{primary, spare}, query)
	if err != nil {
		t.Fatal(err)
	}

	metaengine.WithEventLog(store, metaengine.NewEventLog())
	t.Cleanup(func() { _ = store.Close() })

	for _, qa := range store.Plan().Queries {
		if qa.EngineName != "primary" {
			t.Fatalf("items routed to %q, want primary", qa.EngineName)
		}
	}

	return store, primary, spare
}

// counterValue sums the data points of a counter whose attributes match
// want (attr key → emitted value), failing when the metric was never
// collected.
func counterValue(
	t *testing.T,
	rm *metricdata.ResourceMetrics,
	name string,
	want map[string]string,
) int64 {
	t.Helper()

	total, found := counterValueOrZero(rm, name, want)
	if !found {
		t.Fatalf("metric %q not collected", name)
	}

	return total
}

// counterValueOrZero reads the summed value of a counter's matching data
// points; absent metrics read as (0, false) instead of failing — for
// polling loops.
func counterValueOrZero(
	rm *metricdata.ResourceMetrics,
	name string,
	want map[string]string,
) (int64, bool) {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			data, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}

			var total int64

			for _, dp := range data.DataPoints {
				matches := true

				for k, v := range want {
					got, ok := dp.Attributes.Value(attribute.Key(k))
					if !ok || got.Emit() != v {
						matches = false

						break
					}
				}

				if matches {
					total += dp.Value
				}
			}

			return total, true
		}
	}

	return 0, false
}

func collect(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}

	return rm
}

// The full ADR-0137 lifecycle — quarantine, manual reactivation, probe,
// catch-up rebuild — lands in the counters with the right attributes.
func TestObserver_CountersRecordTransitions(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	store, primary, _ := observerTestStore(t)

	if _, err := otelobserver.Attach(store, provider.Meter("cqrs")); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	if err := store.Apply(ctx, "itemCreated", itemCreated{ID: "i1", Name: "n1"}); err != nil {
		t.Fatal(err)
	}

	close(primary.armed)

	for range metaengine.DefaultEngineFailureThreshold {
		if _, err := store.Execute(findItem{ID: "i1"}); err == nil {
			t.Fatal("expected classified failure while armed")
		}
	}

	rm := collect(t, reader)
	if got := counterValue(t, &rm, "cqrs.metaengine.quarantine.total",
		map[string]string{"engine": "primary"}); got != 1 {
		t.Fatalf("quarantine.total = %d, want 1", got)
	}

	// Manual reactivation records its reason.
	if !store.ReactivateEngine("primary") {
		t.Fatal("reactivate should report true for a quarantined engine")
	}

	rm = collect(t, reader)
	if got := counterValue(t, &rm, "cqrs.metaengine.reactivate.total",
		map[string]string{"engine": "primary", "reason": "manual"}); got != 1 {
		t.Fatalf("reactivate.total(manual) = %d, want 1", got)
	}

	// Quarantine again, heal, then rebuild: catch-up replays the event and
	// reactivates with reason "catchup".
	for range metaengine.DefaultEngineFailureThreshold {
		if _, err := store.Execute(findItem{ID: "i1"}); err == nil {
			t.Fatal("expected classified failure while still armed")
		}
	}

	close(primary.healed)

	if err := store.CatchUpEngine(ctx, "primary"); err != nil {
		t.Fatal(err)
	}

	rm = collect(t, reader)
	if got := counterValue(t, &rm, "cqrs.metaengine.catchup.total",
		map[string]string{"engine": "primary", "outcome": "ok"}); got != 1 {
		t.Fatalf("catchup.total(ok) = %d, want 1", got)
	}

	if got := counterValue(t, &rm, "cqrs.metaengine.catchup.replayed",
		map[string]string{"engine": "primary"}); got != 1 {
		t.Fatalf("catchup.replayed = %d, want 1 (the applied event)", got)
	}

	if got := counterValue(t, &rm, "cqrs.metaengine.reactivate.total",
		map[string]string{"engine": "primary", "reason": "catchup"}); got != 1 {
		t.Fatalf("reactivate.total(catchup) = %d, want 1", got)
	}
}

// The auto-reprobe loop records probe outcomes: a refused engine keeps
// failing, a healed one recovers without manual intervention.
func TestObserver_ProbeOutcomesFromAutoReprobe(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	store, primary, _ := observerTestStore(t)

	if _, err := otelobserver.Attach(store, provider.Meter("cqrs")); err != nil {
		t.Fatal(err)
	}

	close(primary.armed)

	for range metaengine.DefaultEngineFailureThreshold {
		if _, err := store.Execute(findItem{ID: "x"}); err == nil {
			t.Fatal("expected classified failure while armed")
		}
	}

	stop := store.StartAutoReprobe(context.Background(), 5*time.Millisecond)
	defer stop()

	// Unhealed: probes must record failures.
	deadline := time.Now().Add(5 * time.Second)

	for {
		rm := collect(t, reader)
		if got, _ := counterValueOrZero(&rm, "cqrs.metaengine.probe.total",
			map[string]string{"engine": "primary", "outcome": "fail"}); got > 0 {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("no failed probe recorded within 5s")
		}

		time.Sleep(2 * time.Millisecond)
	}

	// Heal: a successful probe triggers catch-up (no EventLog entries here)
	// and lifts the quarantine.
	close(primary.healed)
	deadline = time.Now().Add(5 * time.Second)

	for {
		if h := store.HealthSnapshot()["primary"]; h.State == metaengine.EngineActive {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("engine not reactivated within 5s after healing")
		}

		time.Sleep(2 * time.Millisecond)
	}

	rm := collect(t, reader)
	if got := counterValue(t, &rm, "cqrs.metaengine.probe.total",
		map[string]string{"engine": "primary", "outcome": "ok"}); got == 0 {
		t.Fatal("no successful probe recorded after healing")
	}
}
