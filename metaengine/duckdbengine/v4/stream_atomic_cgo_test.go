//go:build cgo

package duckdbengine_test

import (
	"context"
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// A batch whose second value is rejected must leave the stream untouched:
// the first value must not survive on its own.
func TestStreamAppend_BatchIsAtomic(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)
	ctx := context.Background()

	// One typed envelope whose event id already exists in the stream.
	seed := typedEnvelope(t, duckTypedEventID1, duckTypedStreamID, 1)
	dup := typedEnvelope(t, duckTypedEventID1, duckTypedStreamID, 2)

	if err := eng.(metaengine.StreamLogBackend).StreamAppend(ctx, "atomic", "s-append", []any{seed}); err != nil {
		t.Fatalf("seed append: %v", err)
	}

	before, err := eng.(metaengine.StreamLogBackend).StreamVersion(ctx, "atomic", "s-append")
	if err != nil {
		t.Fatalf("StreamVersion: %v", err)
	}

	batch := []any{"committed-only-with-the-rest", dup}
	if err := eng.(metaengine.StreamLogBackend).StreamAppend(ctx, "atomic", "s-append", batch); !errors.Is(
		err, metaengine.ErrVersionConflict,
	) {
		t.Fatalf("append error = %v, want metaengine.ErrVersionConflict", err)
	}

	after, readErr := eng.(metaengine.StreamLogBackend).StreamVersion(ctx, "atomic", "s-append")
	if readErr != nil {
		t.Fatalf("StreamVersion: %v", readErr)
	}

	if after != before {
		t.Fatalf("version after a rejected batch = %d, want %d: the batch was not rolled back", after, before)
	}
}

// Same contract for the optimistic-concurrency append: the rejected batch must
// not move the stream version.
func TestStreamAppendExpected_BatchIsAtomic(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)
	ctx := context.Background()

	ap := eng.(metaengine.AtomicAppender)
	slb := eng.(metaengine.StreamLogBackend)

	if err := ap.StreamAppendExpected(ctx, "atomic", "s-expected", 0, []any{"a"}); err != nil {
		t.Fatalf("append at v0: %v", err)
	}

	dup := typedEnvelope(t, duckTypedEventID1, duckTypedStreamID, 1)
	batch := []any{dup, dup}

	if err := ap.StreamAppendExpected(ctx, "atomic", "s-expected", 1, batch); !errors.Is(
		err, metaengine.ErrVersionConflict,
	) {
		t.Fatalf("append error = %v, want metaengine.ErrVersionConflict", err)
	}

	ver, err := slb.StreamVersion(ctx, "atomic", "s-expected")
	if err != nil {
		t.Fatalf("StreamVersion: %v", err)
	}

	if ver != 1 {
		t.Fatalf("version after a rejected batch = %d, want 1: the batch was not rolled back", ver)
	}

	vals, err := slb.StreamRead(ctx, "atomic", "s-expected")
	if err != nil {
		t.Fatalf("StreamRead: %v", err)
	}

	if len(vals) != 1 {
		t.Fatalf("stream length = %d, want 1", len(vals))
	}
}
