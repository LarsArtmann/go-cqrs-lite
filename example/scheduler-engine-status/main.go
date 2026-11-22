// Command scheduler-engine-status is the v5-forward timer demo: durable
// deadlines via scheduling/engine.TimerStore over a metaengine DueClaimer
// engine (ADR-0142 — "all durable I/O rides metaengine"), with OTel counters
// on /metrics and a JSON /status snapshot carrying a cross-restart claim
// rate anchor. The deprecated sqlstore path (with its ClaimMetrics hooks) is
// demonstrated by example/scheduler-otel-status until v5 removes it.
//
//	go run .            # then: curl localhost:8080/status | jq
//	                    #       curl localhost:8080/metrics | grep cqrs_scheduler
package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	cqrsprom "github.com/larsartmann/go-cqrs-lite/prometheus/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// reminder is the timer payload: timers are typed, not opaque blobs.
type reminder struct {
	OrderID string `json:"orderId"`
	Note    string `json:"note"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	prov, err := cqrsprom.Setup()
	if err != nil {
		return err
	}

	meter := prov.AsMeterProvider().Meter("cqrs/scheduler")
	fired := mustCounter(meter, "cqrs_scheduler_timers_fired", "Timers dispatched by the engine-path scheduler")

	// The engine IS the storage layer (ADR-0142): any metaengine engine
	// implementing DueClaimer serves as the timer backend. The in-memory
	// engine keeps the demo dependency-light; a deployed service swaps in
	// sqlite/postgres/... by registering the driver and changing one line.
	eng := metaengine.NewMemoryEngine()
	if !metaengine.SupportsDueClaims(eng) {
		return errors.New("memory engine lost its DueClaimer capability — demo cannot run")
	}

	store, err := engine.NewTimerStore[reminder](
		eng,
		engine.WithCollection("demo_timers"),
		engine.WithOwner("demo"),
	)
	if err != nil {
		return err
	}

	stats := &runStats{startedAt: time.Now()}

	scheduler := scheduling.New(
		store,
		func(_ context.Context, t scheduling.Timer[reminder]) error {
			atomic.AddInt64(&stats.fired, 1)
			fired.Add(ctx, 1)
			log.Printf("fired %s: order %s — %s", t.ID.Get(), t.Payload.OrderID, t.Payload.Note)

			return nil
		},
		scheduling.WithLogger(slog.Default()),
	)

	// Seed staggered reminders so /status shows live activity.
	for _, t := range seedTimers() {
		if err := store.Schedule(ctx, t); err != nil {
			return err
		}
	}

	go func() {
		if err := scheduler.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("scheduler: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", prov.Handler())
	mux.HandleFunc("/status", stats.handler())

	log.Printf("scheduler-engine-status listening on %s (GET /status, GET /metrics)", addr())

	server := &http.Server{Addr: addr(), Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// runStats is the /status snapshot source: the engine path has no
// ClaimMetrics surface (that was sqlstore-specific) — the fired counter and
// the StartedAt anchor are the honest observable state.
type runStats struct {
	startedAt time.Time
	fired     int64
}

type statusSnapshot struct {
	StartedAt      string  `json:"startedAt"`
	TimersFired    int64   `json:"timersFired"`
	FiredPerMinute float64 `json:"firedPerMinute"`
}

func (s *runStats) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		fired := atomic.LoadInt64(&s.fired)
		minutes := time.Since(s.startedAt).Minutes()

		rate := 0.0
		if minutes > 0 {
			rate = float64(fired) / minutes
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.MarshalWrite(w, statusSnapshot{
			StartedAt:      s.startedAt.Format(time.RFC3339),
			TimersFired:    fired,
			FiredPerMinute: rate,
		}); err != nil {
			log.Printf("status encode: %v", err)
		}
	}
}

func seedTimers() []scheduling.Timer[reminder] {
	now := time.Now()

	return []scheduling.Timer[reminder]{
		{ID: scheduling.MustParseTimerID("demo-immediate"), FireAt: now.Add(-time.Second),
			Payload: reminder{OrderID: "order-1", Note: "fires immediately"}},
		{ID: scheduling.MustParseTimerID("demo-5s"), FireAt: now.Add(5 * time.Second),
			Payload: reminder{OrderID: "order-2", Note: "fires after five seconds"}},
		{ID: scheduling.MustParseTimerID("demo-30s"), FireAt: now.Add(30 * time.Second),
			Payload: reminder{OrderID: "order-3", Note: "fires after thirty seconds"}},
	}
}

// addr prefers ADDR (e.g. ":8181") so the demo can coexist with whatever
// already owns :8080 on a busy machine.
func addr() string {
	if a := os.Getenv("ADDR"); a != "" {
		return a
	}

	return ":8080"
}
