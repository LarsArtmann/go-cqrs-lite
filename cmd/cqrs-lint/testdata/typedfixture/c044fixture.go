package main

// C044 fixtures: branded-ID .String() in leak positions (BAD — prefixed
// display form reaching keys, SQL args, cursors, wire payloads) and display
// positions (GOOD — logging and message formatting are the intent).

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// C044-BAD-1: SQL argument keyed by the display form (the
// scheduling/sqlstore "no live claim" bug class).
func c044FetchByStream(db queryer, s id.StreamID) {
	db.Query("SELECT * FROM t WHERE id = ?", s.String()) // C044 leak: SQL arg
}

// C044-BAD-2: storage/cursor key built from the display form.
func c044CursorKey(s id.StreamID) string {
	return "u:" + s.String() // C044 leak: cursor key concat
}

// C044-BAD-3: map key from the display form (the system typed-decoder and
// getting-started "found=false forever" bug class).
func c044StoreKey(projection map[string]string, s id.StreamID, v string) {
	projection[s.String()] = v // C044 leak: map key
}

// C044-GOOD-1: slog argument — display is the intent.
func c044LogLoaded(s id.StreamID) {
	slog.Info("loaded stream", "id", s.String())
}

// C044-GOOD-2: stdlib log formatting.
func c044LogPrintf(s id.StreamID) {
	log.Printf("processing %s", s.String())
}

// C044-GOOD-3: fmt message formatting.
func c044FormatMessage(s id.StreamID) string {
	return fmt.Sprintf("task %s", s.String())
}

type queryer interface {
	Query(q string, args ...any)
}
