package pgengine_test

import (
	"context"
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestPostgresEngine_StreamLogRoundtrip(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

	enginetest.RunStreamLogBackendTest(t, eng)
	enginetest.RunSeqSeekableStreamLogTest(t, eng)
}

func TestPostgresEngine_StreamLogAtomicAppender(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

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
	err := ap.StreamAppendExpected(ctx, "events", "s1", 0, []any{"d"})
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

func TestPostgresEngine_Transactional(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

	enginetest.RunTransactionalTest(t, eng)
}

func TestPostgresEngine_ConcurrentTx(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

	enginetest.RunConcurrentTxTest(t, eng)
}
