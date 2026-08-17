package sqlstore_test

import (
	"context"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// TestSQLiteTimerStore_ActorSurvivesRoundtrip locks the guarantee that
// Timer.Actor (the "kind:raw" audit-trail attribution) survives SQL
// persistence — a timer scheduled by a user must still name that user when it
// fires after a restart.
func TestSQLiteTimerStore_ActorSurvivesRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, _ := newSQLiteStore[testPayload](t)

	due := time.Now().Add(-time.Minute)

	err := store.Schedule(ctx, scheduling.Timer[testPayload]{
		ID:      "actor-roundtrip",
		FireAt:  due,
		Payload: testPayload{Action: "cancel", Amount: 3},
		Actor:   "user:01HK1540X0841Y0A6BSX1VKR99",
	})
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	timers, err := store.Due(ctx, time.Now())
	if err != nil {
		t.Fatalf("Due: %v", err)
	}

	if len(timers) != 1 {
		t.Fatalf("expected 1 due timer, got %d", len(timers))
	}

	if got, want := timers[0].Actor, "user:01HK1540X0841Y0A6BSX1VKR99"; got != want {
		t.Errorf("actor lost through SQL round-trip: got %q, want %q", got, want)
	}

	if timers[0].Payload != (testPayload{Action: "cancel", Amount: 3}) {
		t.Errorf("payload mismatch: got %+v", timers[0].Payload)
	}
}

// TestSQLiteTimerStore_LegacyBarePayloadRowsStillDecode proves rows written
// by pre-actor versions (payload column holds the bare JSON of P) still load
// with an empty actor — the envelope switch is backward compatible.
func TestSQLiteTimerStore_LegacyBarePayloadRowsStillDecode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, db := newSQLiteStore[testPayload](t)

	due := time.Now().Add(-time.Minute).UTC().Format("2006-01-02T15:04:05.000000000Z07:00")

	// Write a legacy row exactly as v4.2.0 did: bare payload JSON.
	_, err := db.ExecContext(
		ctx,
		`INSERT INTO timers (id, fire_at, payload) VALUES (?, ?, ?)`,
		"legacy-timer", due, `{"action":"remind","amount":7}`,
	)
	if err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	timers, err := store.Due(ctx, time.Now())
	if err != nil {
		t.Fatalf("Due: %v", err)
	}

	if len(timers) != 1 {
		t.Fatalf("expected 1 due timer, got %d", len(timers))
	}

	if timers[0].Payload != (testPayload{Action: "remind", Amount: 7}) {
		t.Errorf("legacy payload mismatch: got %+v", timers[0].Payload)
	}

	if timers[0].Actor != "" {
		t.Errorf("legacy row should decode with empty actor, got %q", timers[0].Actor)
	}
}

// TestSQLiteTimerStore_NonObjectLegacyPayloadDecodes guards the decoder's
// legacy fallback for payload types that are not JSON objects (strings,
// numbers) — the envelope probe must not reject them.
func TestSQLiteTimerStore_NonObjectLegacyPayloadDecodes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, db := newSQLiteStore[string](t)

	due := time.Now().Add(-time.Minute).UTC().Format("2006-01-02T15:04:05.000000000Z07:00")

	_, err := db.ExecContext(
		ctx,
		`INSERT INTO timers (id, fire_at, payload) VALUES (?, ?, ?)`,
		"string-payload", due, `"just-a-string"`,
	)
	if err != nil {
		t.Fatalf("seed string-payload row: %v", err)
	}

	timers, err := store.Due(ctx, time.Now())
	if err != nil {
		t.Fatalf("Due: %v", err)
	}

	if len(timers) != 1 || timers[0].Payload != "just-a-string" {
		t.Fatalf("string payload round-trip: got %+v (len %d)", timers, len(timers))
	}
}

// TestSQLiteTimerStore_CorruptPayloadClassifiedAsCorruption proves a row whose
// payload column no longer decodes as P surfaces as a Corruption-family error
// naming the offending timer — operators can distinguish a rotting payload
// column from an infrastructure outage.
func TestSQLiteTimerStore_CorruptPayloadClassifiedAsCorruption(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, db := newSQLiteStore[testPayload](t)

	due := time.Now().Add(-time.Minute).UTC().Format("2006-01-02T15:04:05.000000000Z07:00")

	_, err := db.ExecContext(
		ctx,
		`INSERT INTO timers (id, fire_at, payload) VALUES (?, ?, ?)`,
		"corrupt-timer", due, `{"action":42}`, // P declares Action string; number is a type mismatch
	)
	if err != nil {
		t.Fatalf("seed corrupt row: %v", err)
	}

	_, err = store.Due(ctx, time.Now())
	if err == nil {
		t.Fatal("expected decode failure for corrupt payload row, got nil")
	}

	if got := errorfamily.Classify(err); got != errorfamily.Corruption {
		t.Errorf("corrupt payload must classify as Corruption, got %v", got)
	}
}
