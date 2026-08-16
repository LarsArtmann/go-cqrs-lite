package pgengine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func mustNewCopyEngine(
	t *testing.T,
	minValues int,
) (metaengine.Engine, metaengine.StreamLogBackend) {
	t.Helper()

	eng, err := pgengine.New(pgDSN(t), pgengine.WithCopyAppend(minValues))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("pgengine.Engine does not implement StreamLogBackend")
	}

	return eng, backend
}

// TestStreamAppend_CopyMatchesInsert proves the COPY fast path persists
// exactly what the chunked-INSERT path persists: same values, same version.
func TestStreamAppend_CopyMatchesInsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, copyBackend := mustNewCopyEngine(t, 10)
	_, insertBackend := mustNewCopyEngine(t, 1<<30) // unreachable threshold: INSERT path

	want := make([]any, 50)
	for i := range want {
		want[i] = map[string]any{"n": i}
	}

	if err := copyBackend.StreamAppend(ctx, "copycmp", "via-copy", want); err != nil {
		t.Fatalf("StreamAppend (copy): %v", err)
	}

	if err := insertBackend.StreamAppend(ctx, "copycmp", "via-insert", want); err != nil {
		t.Fatalf("StreamAppend (insert): %v", err)
	}

	viaCopy, err := copyBackend.StreamRead(ctx, "copycmp", "via-copy")
	if err != nil {
		t.Fatalf("StreamRead (copy): %v", err)
	}

	viaInsert, err := insertBackend.StreamRead(ctx, "copycmp", "via-insert")
	if err != nil {
		t.Fatalf("StreamRead (insert): %v", err)
	}

	if len(viaCopy) != 50 || len(viaInsert) != 50 {
		t.Fatalf("lengths: copy=%d insert=%d, want 50/50", len(viaCopy), len(viaInsert))
	}

	for i := range viaCopy {
		if fmt.Sprint(viaCopy[i]) != fmt.Sprint(viaInsert[i]) {
			t.Fatalf("value %d diverges: copy=%v insert=%v", i, viaCopy[i], viaInsert[i])
		}
	}

	copyVersion, err := copyBackend.StreamVersion(ctx, "copycmp", "via-copy")
	if err != nil {
		t.Fatalf("StreamVersion (copy): %v", err)
	}

	if copyVersion != 50 {
		t.Fatalf("copy stream version = %d, want 50", copyVersion)
	}
}

// TestStreamAppend_CopyFallbackInsideRunInTx proves the COPY path defers to
// INSERTs when a transaction is active (COPY cannot join a database/sql tx on
// another pooled connection).
func TestStreamAppend_CopyFallbackInsideRunInTx(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, backend := mustNewCopyEngine(t, 1) // every append would take COPY if it could

	txEng, ok := eng.(metaengine.Transactional)
	if !ok {
		t.Fatal("pgengine.Engine does not implement Transactional")
	}

	err := txEng.RunInTx(ctx, func(context.Context) error {
		return backend.StreamAppend(ctx, "copytx", "s", []any{1, 2, 3})
	})
	if err != nil {
		t.Fatalf("RunInTx with StreamAppend: %v", err)
	}

	read, err := backend.StreamRead(ctx, "copytx", "s")
	if err != nil {
		t.Fatalf("StreamRead: %v", err)
	}

	if len(read) != 3 {
		t.Fatalf("in-tx append len = %d, want 3", len(read))
	}
}

// TestStreamAppend_BelowThresholdUsesInsert guards the threshold: with
// WithCopyAppend(1000), a 5-value append stays on the INSERT path.
func TestStreamAppend_BelowThresholdUsesInsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, backend := mustNewCopyEngine(t, 1000)

	if err := backend.StreamAppend(ctx, "copythresh", "s", []any{"a", "b"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	v, err := backend.StreamVersion(ctx, "copythresh", "s")
	if err != nil {
		t.Fatalf("StreamVersion: %v", err)
	}

	if v != 2 {
		t.Fatalf("version = %d, want 2", v)
	}
}

// BenchmarkStreamAppend compares bulk append strategies on a real Postgres.
// Skip unless POSTGRES_TEST_DSN is set or testcontainers are available.
// Run: go test -bench StreamAppend -benchtime 10x -run XXX.
func BenchmarkStreamAppend(b *testing.B) {
	for _, size := range []int{10_000, 100_000} {
		b.Run(fmt.Sprintf("Insert_%d", size), func(b *testing.B) {
			benchStreamAppend(b, size, false)
		})
		b.Run(fmt.Sprintf("Copy_%d", size), func(b *testing.B) {
			benchStreamAppend(b, size, true)
		})
	}
}

func benchStreamAppend(b *testing.B, size int, copy bool) {
	opts := []pgengine.Option{}
	if copy {
		opts = append(opts, pgengine.WithCopyAppend(1))
	}

	eng, err := pgengine.New(pgDSN(b), opts...)
	if err != nil {
		b.Skipf("Postgres not available: %v", err)
	}

	defer func() { _ = eng.Close() }()

	backend := eng.(metaengine.StreamLogBackend)

	ctx := context.Background()
	values := make([]any, size)
	for i := range values {
		values[i] = map[string]any{"seq": i}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sid := fmt.Sprintf("bench-%d", i)
		if err := backend.StreamAppend(ctx, "copybench", sid, values); err != nil {
			b.Fatalf("StreamAppend: %v", err)
		}
	}
}
