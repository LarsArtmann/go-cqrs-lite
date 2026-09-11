package irohengine

import (
	"context"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// A reset delegates to the local engine's EngineResetter and keeps the LWW
// barriers, so stale in-flight peer writes (older timestamps) are rejected
// after the reset while replayed writes (fresh timestamps) apply normally.
func TestResetEngine_ResetsLocalAndKeepsLWWBarriers(t *testing.T) {
	t.Parallel()

	local := metaengine.NewMemoryEngine()
	eng := Replicated(local).(*replicatedEngine)
	ctx := context.Background()

	mb := eng.local.(metaengine.MapBackend)
	if err := mb.MapSet(ctx, "tasks", "t1", "old"); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	// A pre-reset replicated write records its LWW timestamp.
	eng.recordLWW("tasks", "t1", time.Now().Add(-time.Minute))

	if err := eng.ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if _, ok, err := mb.MapGet(ctx, "tasks", "t1"); err != nil || ok {
		t.Fatalf("local map must be empty after reset (ok=%v err=%v)", ok, err)
	}

	// Stale peer write after the reset: older than the barrier → rejected.
	if eng.isLWWNewer("tasks", "t1", time.Now().Add(-2*time.Minute)) {
		t.Fatal("a write older than the recorded LWW barrier must be rejected after a reset")
	}

	// A fresh replay write: newer than every barrier → accepted.
	if !eng.isLWWNewer("tasks", "t1", time.Now()) {
		t.Fatal("a fresh post-reset write must be accepted")
	}
}

// The wrapper must fail loudly when the local engine cannot be reset — a
// replicated reset is only as revertible as its local half.
func TestResetEngine_FailsWhenLocalNotResettable(t *testing.T) {
	t.Parallel()

	eng := Replicated(&nonResettableLocal{}).(*replicatedEngine)

	err := eng.ResetEngine(context.Background())
	if err == nil {
		t.Fatal("expected an error when the local engine lacks EngineResetter")
	}
}

type nonResettableLocal struct{}

func (l *nonResettableLocal) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{Name: "non-resettable"}
}

func (l *nonResettableLocal) Close() error { return nil }
