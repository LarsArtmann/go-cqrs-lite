package main

import (
	"log"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// mustCounter is the demo's one-instrument meter helper: the engine path
// has no ClaimMetrics hooks to bridge (those were sqlstore-specific), so the
// observable surface is the counters the app itself owns.
func mustCounter(m cqrsotel.Meter, name, desc string) cqrsotel.Int64Counter {
	counter, err := m.Int64Counter(name, cqrsotel.CounterMetricWithDescription(desc))
	if err != nil {
		log.Fatalf("meter %s: %v", name, err)
	}

	return counter
}
