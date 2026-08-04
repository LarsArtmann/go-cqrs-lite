package pgengine_test

import (
	"context"
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
)

func TestPostgresEngine_StreamLogRoundtrip(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()

	slb, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("pgEngine does not implement StreamLogBackend")
	}

	// Append to two streams.
	if err := slb.StreamAppend(ctx, "events", "s1", []any{"e1", "e2", "e3"}); err != nil {
		t.Fatalf("StreamAppend s1: %v", err)
	}

	if err := slb.StreamAppend(ctx, "events", "s2", []any{"e4"}); err != nil {
		t.Fatalf("StreamAppend s2: %v", err)
	}

	// Verify StreamRead.
	values, err := slb.StreamRead(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamRead s1: %v", err)
	}

	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}

	// Verify StreamVersion.
	ver, err := slb.StreamVersion(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamVersion s1: %v", err)
	}

	if ver != 3 {
		t.Fatalf("expected version 3, got %d", ver)
	}

	// Verify JournalReadAll (cross-stream ordering).
	journal, err := slb.JournalReadAll(ctx, "events")
	if err != nil {
		t.Fatalf("JournalReadAll: %v", err)
	}

	if len(journal) != 4 {
		t.Fatalf("expected 4 journal entries, got %d", len(journal))
	}

	// Verify JournalReadFrom (position-based resumption).
	from2, err := slb.JournalReadFrom(ctx, "events", 2, 0)
	if err != nil {
		t.Fatalf("JournalReadFrom: %v", err)
	}

	if len(from2) != 2 {
		t.Fatalf("expected 2 entries after seq 2, got %d", len(from2))
	}
}

func TestPostgresEngine_StreamLogAtomicAppender(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()

	ap, ok := eng.(metaengine.AtomicAppender)
	if !ok {
		t.Fatal("pgEngine does not implement AtomicAppender")
	}

	// Append at version 0 → succeeds.
	if err := ap.StreamAppendExpected(ctx, "events", "s1", 0, []any{"a", "b"}); err != nil {
		t.Fatalf("StreamAppendExpected v0: %v", err)
	}

	// Append at version 2 → succeeds.
	if err := ap.StreamAppendExpected(ctx, "events", "s1", 2, []any{"c"}); err != nil {
		t.Fatalf("StreamAppendExpected v2: %v", err)
	}

	// Append at version 0 (stale) → fails with ErrVersionConflict.
	err = ap.StreamAppendExpected(ctx, "events", "s1", 0, []any{"d"})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}

	// Verify final state.
	slb := eng.(metaengine.StreamLogBackend)
	ver, err := slb.StreamVersion(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamVersion: %v", err)
	}

	if ver != 3 {
		t.Fatalf("expected version 3, got %d", ver)
	}
}
