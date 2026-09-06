//go:build integration

package storage_test

// Encrypted-snapshot-at-rest verification against real PostgreSQL:
//
//  1. snapshots saved through a SnapshotStateCodec store are CIPHERTEXT in
//     the `state` column (the plaintext marker must be absent, the envelope
//     key id present);
//  2. with the rotation codec + rewrite-on-read, the first load of a
//     retired-key snapshot re-encrypts it under the active key IN THE
//     DATABASE (write-back migration) without ever returning corrupt state.

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

func TestPostgresSnapshotEncryption_AtRestAndRotation(t *testing.T) {
	db := pgDB(t)

	rawStore, err := storage.NewSQLSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewSQLSnapshotStore: %v", err)
	}

	oldKey, keyErr := encryption.GenerateKey()
	if keyErr != nil {
		t.Fatalf("GenerateKey (old): %v", keyErr)
	}

	newKey, keyErr := encryption.GenerateKey()
	if keyErr != nil {
		t.Fatalf("GenerateKey (new): %v", keyErr)
	}

	oldCipher, err := encryption.NewAES256GCM(oldKey)
	if err != nil {
		t.Fatalf("NewAES256GCM (old): %v", err)
	}

	newCipher, err := encryption.NewAES256GCM(newKey)
	if err != nil {
		t.Fatalf("NewAES256GCM (new): %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const plaintextMarker = `{"hp":4242,"secret":"rotation-probe"}`

	legacy, err := encryption.SnapshotStateCodec(oldCipher, "key-2025")
	if err != nil {
		t.Fatalf("SnapshotStateCodec: %v", err)
	}

	legacyStore, err := snapshot.NewTransformedStore(rawStore, legacy.Protect, legacy.Restore)
	if err != nil {
		t.Fatalf("NewTransformedStore: %v", err)
	}

	streamID := idtest.ParseStreamID(t, "01HGW5FPJPYK5RE8ACZDesWMY2")
	ref := id.NewStreamRef("User", streamID)

	snap, err := snapshot.NewSnapshot(
		ref,
		event.Version(7),
		[]byte(plaintextMarker),
		record.EncodingJSON,
	)
	if err != nil {
		t.Fatalf("NewSnapshot: %v", err)
	}

	if err = legacyStore.Save(ctx, snap); err != nil {
		t.Fatalf("save encrypted snapshot: %v", err)
	}

	stored := snapshotColumnState(t, db, streamID)

	if strings.Contains(stored, "rotation-probe") {
		t.Fatalf("snapshot state column contains PLAINTEXT: %s", stored)
	}

	if got := envelopeKeyID(t, stored); got != "key-2025" {
		t.Fatalf("snapshot state column is not a key-2025 envelope: %s", stored)
	}

	resolver := encryption.NewStaticKeyResolver(map[encryption.KeyID]encryption.Decrypter{
		"key-2025": oldCipher,
		"key-2026": newCipher,
	})

	rotating, err := encryption.RotatingSnapshotStateCodec("key-2026", newCipher, resolver)
	if err != nil {
		t.Fatalf("RotatingSnapshotStateCodec: %v", err)
	}

	migrating, err := snapshot.NewRewritingTransformedStore(rawStore, snapshot.StateTransforms{
		Protect:      rotating.Protect,
		Restore:      rotating.Restore,
		NeedsRewrite: rotating.NeedsRewrite,
		Reencrypt:    rotating.Reencrypt,
	})
	if err != nil {
		t.Fatalf("NewRewritingTransformedStore: %v", err)
	}

	loaded, err := migrating.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load through migrating store: %v", err)
	}

	if string(loaded.State) != plaintextMarker {
		t.Errorf("restored state = %q, want original plaintext", loaded.State)
	}

	rotated := snapshotColumnState(t, db, streamID)

	if strings.Contains(rotated, "rotation-probe") {
		t.Errorf("post-migration state column contains PLAINTEXT: %s", rotated)
	}

	if got := envelopeKeyID(t, rotated); got != "key-2026" {
		t.Errorf("write-back did not re-encrypt under the active key: %s", rotated)
	}

	reread, err := migrating.Load(ctx, ref)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	if string(reread.State) != plaintextMarker {
		t.Errorf("second restored state = %q, want original plaintext", reread.State)
	}
}

// pgQuerier is the subset of *sql.DB the raw-column probes need.
type pgQuerier interface {
	QueryRow(query string, args ...any) *sql.Row
}

// envelopeKeyID parses the stored envelope JSON and returns its key id,
// asserting the column really holds an envelope.
func envelopeKeyID(t *testing.T, raw string) string {
	t.Helper()

	var env struct {
		Version string `json:"v"`
		KeyID   string `json:"kid"`
	}

	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("stored state is not an envelope JSON object: %s", raw)
	}

	return env.KeyID
}

// snapshotColumnState reads the raw state column straight from the database,
// bypassing every store wrapper.
func snapshotColumnState(t *testing.T, db pgQuerier, streamID id.StreamID) string {
	t.Helper()

	const query = `SELECT state FROM snapshots WHERE stream_id = $1`

	var state []byte

	err := db.QueryRow(query, streamID.String()).Scan(&state)
	if err != nil {
		t.Fatalf("read raw snapshot state column: %v", err)
	}

	return string(state)
}
