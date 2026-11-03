package main

// C044 fixtures: branded-ID .String() in leak positions (BAD — prefixed
// display form reaching keys, SQL args, cursors, wire payloads) and display
// positions (GOOD — logging and message formatting are the intent).

import (
	"fmt"
	"log"
	"log/slog"
	"strings"

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

// C044-BAD-4: ActorID's String() drops the actor kind — the boundary form is
// PrefixedString()/ParseActorID (AGENTS contract 21e).
func c044ActorKey(a id.ActorID) string {
	return a.String() // C044 leak: kind-dropping display form
}

// C044-BAD-5: StreamRef's String() is the display composite — the storage
// key is StreamKey() (bare form, the 2026-10-17 disk-truth fix).
func c044RefKey(r id.StreamRef) string {
	return r.String() // C044 leak: display composite as key
}

// C044-GOOD-4: ULID-backed IDs have marker-less brands — String() returns
// the BARE value there, so the rule stays silent (FP control).
func c044BareEventID(e id.EventID) string {
	return e.String()
}

// C044-GOOD-5: nested inside a display call's arguments — whitelisted at
// any depth, not just direct arguments.
func c044NestedDisplay(s id.StreamID) {
	slog.Info("loaded stream", "id", strings.ToUpper(s.String()))
}

// C044-GOOD-6: the documented round-trip law — ParseStreamID strips the
// display prefix, so ParseStreamID(id.String()) == id.
func c044RoundTrip(s id.StreamID) (id.StreamID, error) {
	return id.ParseStreamID(s.String())
}

type queryer interface {
	Query(q string, args ...any)
}
