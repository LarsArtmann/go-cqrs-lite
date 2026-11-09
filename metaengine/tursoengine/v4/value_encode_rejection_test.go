package tursoengine_test

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// The turso engine delegates its storage core to sqliteengine. These tests pin
// that the delegation preserves the stored-value contract end to end through
// the embedded libSQL driver: a non-JSON-representable value must fail the
// write with a Rejection carrying the sqliteengine codes — never silently
// store its %v rendering. The codes are asserted as the string contracts they
// are (sqliteengine's constants are unexported): the map write path carries
// sqliteengine.encode_value, the stream codec path carries
// metaengine.encode_stream_value.

func TestTurso_MapSet_RejectsUnrepresentableValue(t *testing.T) {
	t.Parallel()

	eng := mustNewTursoEngine(t)
	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)

	err := mb.MapSet(ctx, "maps", "k", map[string]any{"bad": make(chan int)})
	assertDelegatedRejection(t, err, "MapSet", "sqliteengine.encode_value")
}

func TestTurso_StreamAppend_RejectsUnrepresentableValueAtomically(t *testing.T) {
	t.Parallel()

	eng := mustNewTursoEngine(t)
	ctx := context.Background()
	sl := eng.(metaengine.StreamLogBackend)

	err := sl.StreamAppend(ctx, "streams", "s-1", []any{"ok", make(chan int)})
	assertDelegatedRejection(t, err, "StreamAppend", "metaengine.encode_stream_value")

	version, err := sl.StreamVersion(ctx, "streams", "s-1")
	if err != nil {
		t.Fatalf("StreamVersion: %v", err)
	}

	if version != 0 {
		t.Fatalf("stream version = %d after a rejected batch, want 0", version)
	}
}

func assertDelegatedRejection(t *testing.T, err error, surface, wantCode string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s with a non-representable value must fail, got nil", surface)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("%s error %v classified as %v, want Rejection", surface, err, family)
	}

	if code := errorfamily.Code(err); code != wantCode {
		t.Fatalf("%s error code = %q, want %q (%v)", surface, code, wantCode, err)
	}
}
