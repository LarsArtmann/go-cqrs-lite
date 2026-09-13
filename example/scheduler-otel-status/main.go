// Command scheduler-otel-status is the runnable wiring example for
// scheduling/sqlstore claim observability: ClaimMetrics hooks feed OTel
// counters (exposed on /metrics via the OTel→Prometheus bridge) while the
// built-in Metrics() snapshot serves /status as JSON — including the
// StartedAt anchor for cross-restart claim rates.
//
//	go run .            # then: curl localhost:8080/status | jq
//	                    #       curl localhost:8080/metrics | grep cqrs_scheduler
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	cqrsprom "github.com/larsartmann/go-cqrs-lite/prometheus/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	// One call: MeterProvider + Prometheus HTTP handler, CQRS views applied.
	prov, err := cqrsprom.Setup()
	if err != nil {
		return err
	}

	recorder, err := newClaimRecorder(prov.AsMeterProvider().Meter("cqrs/scheduler"))
	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite", "file:scheduler-otel-status?mode=memory&cache=shared")
	if err != nil {
		return err
	}

	store, err := sqlstore.NewClaimingSQLiteStore[struct{}](
		ctx, db, 0, sqlstore.WithClaimMetrics[struct{}](recorder),
	)
	if err != nil {
		return err
	}

	if err := store.Schedule(ctx, dueTimer("demo-timer")); err != nil {
		return err
	}

	go pollLoop(ctx, store)

	mux := http.NewServeMux()
	mux.Handle("/metrics", prov.Handler())
	mux.HandleFunc("/status", statusHandler(store))

	log.Println("scheduler-otel-status listening on :8080 (GET /status, GET /metrics)")

	//nolint:wrapcheck // top-level server error
	return http.ListenAndServe(":8080", mux)
}

// statusSnapshot pairs the store's claim counters with a live claim rate
// derived from the StartedAt anchor — the cross-restart-rate pattern.
type statusSnapshot struct {
	sqlstore.ClaimMetricsSnapshot
	ClaimedPerMinute float64 `json:"claimedPerMinute"`
}

func statusHandler(store *sqlstore.ClaimingTimerStore[struct{}]) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		snap := store.Metrics()

		minutes := time.Since(snap.StartedAt).Minutes()
		rate := 0.0
		if minutes > 0 {
			rate = float64(snap.ClaimedTimers) / minutes
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(statusSnapshot{
			ClaimMetricsSnapshot: snap,
			ClaimedPerMinute:     rate,
		}); err != nil {
			log.Printf("status encode: %v", err)
		}
	}
}

// pollLoop claims due timers forever — the activity both surfaces observe.
func pollLoop(ctx context.Context, store *sqlstore.ClaimingTimerStore[struct{}]) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			timers, err := store.Due(ctx, time.Now())
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				log.Printf("Due: %v", err)

				continue
			}

			for _, tm := range timers {
				if err := store.MarkFired(ctx, tm.ID); err != nil {
					log.Printf("MarkFired %s: %v", tm.ID, err)
				}
			}
		}
	}
}

func dueTimer(id string) scheduling.Timer[struct{}] {
	return scheduling.Timer[struct{}]{
		ID:     scheduling.MustParseTimerID(id),
		FireAt: time.Now().Add(-time.Second),
	}
}
