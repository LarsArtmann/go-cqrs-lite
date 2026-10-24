package sqlstore_test

// A18 migration pin: timer payloads ride the shared gcdc envelope
// (codec.WrapEncode/UnwrapDecode) on every new write, while BOTH legacy
// formats (the v1 hand-rolled envelope {"v":1,"actor":...,"payload":...}
// and the v0 bare payload JSON) still decode — one envelope system, no
// journal rewrite.

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

type legacyV1Envelope struct {
	Version int    `json:"v"`
	Actor   string `json:"actor,omitzero"`
	Payload any    `json:"payload"`
}

func TestTimerPayloadEnvelope_MigrationOldToNew(t *testing.T) {
	t.Parallel()

	store, db := newSQLiteStore[testPayload](t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	// A v1-legacy row: the hand-rolled envelope, generated with the real
	// serializer (never hand-written wire bytes).
	v1Payload, err := json.Marshal(legacyV1Envelope{
		Version: 1,
		Payload: map[string]any{"action": "legacy-v1", "amount": 11},
	})
	if err != nil {
		t.Fatalf("marshal v1 legacy envelope: %v", err)
	}

	legacyID := scheduling.MustParseTimerID("legacy-v1")

	if err := insertTimerRow(ctx, db, legacyID, now.Add(-time.Second), string(v1Payload)); err != nil {
		t.Fatalf("insert v1 legacy row: %v", err)
	}

	// A v0-legacy row: the bare payload JSON.
	v0Payload, err := json.Marshal(testPayload{Action: "legacy-v0", Amount: 7})
	if err != nil {
		t.Fatalf("marshal v0 legacy payload: %v", err)
	}

	v0ID := scheduling.MustParseTimerID("legacy-v0")

	if err := insertTimerRow(ctx, db, v0ID, now.Add(-time.Second), string(v0Payload)); err != nil {
		t.Fatalf("insert v0 legacy row: %v", err)
	}

	// A NEW row through the store (gcdc-wrapped).
	newID := scheduling.MustParseTimerID("gcdc-new")

	if err := store.Schedule(ctx, scheduling.Timer[testPayload]{
		ID:      newID,
		FireAt:  now.Add(-time.Second),
		Payload: testPayload{Action: "new", Amount: 42},
	}); err != nil {
		t.Fatalf("schedule new-format timer: %v", err)
	}

	// The new write IS gcdc-wrapped (the envelope magic is on disk).
	var stored string

	if err := db.QueryRowContext(ctx,
		`SELECT payload FROM timers WHERE id = ?`, newID.Get()).Scan(&stored); err != nil {
		t.Fatalf("read stored payload: %v", err)
	}

	if !strings.Contains(stored, `"gcdc"`) {
		t.Fatalf("new write must carry the gcdc envelope magic, got: %s", stored)
	}

	// ALL THREE formats decode through one Due() call.
	due, err := store.Due(ctx, now)
	if err != nil {
		t.Fatalf("Due across mixed formats: %v", err)
	}

	byAction := map[string]int{}

	for _, timer := range due {
		byAction[timer.Payload.Action] = timer.Payload.Amount
	}

	if byAction["legacy-v1"] != 11 {
		t.Fatalf("v1 legacy row did not decode: %+v", byAction)
	}

	if byAction["legacy-v0"] != 7 {
		t.Fatalf("v0 legacy row did not decode: %+v", byAction)
	}

	if byAction["new"] != 42 {
		t.Fatalf("new gcdc row did not decode: %+v", byAction)
	}
}

func insertTimerRow(ctx context.Context, db *sql.DB, id scheduling.TimerID, fireAt time.Time, payload string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO timers (id, fire_at, payload) VALUES (?, ?, ?)`,
		id.Get(), fireAt.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"), payload)

	return err
}
