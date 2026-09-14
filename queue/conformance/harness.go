package conformance

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// Payload is the suite's fixed task payload type. Engines instantiate
// their generic store over it when registering the suite; the JSON codec
// round-trips it like any domain type.
type Payload struct {
	Cmd string `json:"cmd"`
}

// Harness supplies the engine under test. Every field is required: the
// suite has no optional semantics.
type Harness struct {
	// NewStore returns a FRESH, EMPTY store. Called once per subtest —
	// the implementation decides what fresh means (new file, truncated
	// tables, throwaway database) but MUST guarantee no state survives
	// from a previous subtest.
	NewStore func(t *testing.T) queue.Store[Payload]

	// Backdate rewinds one task's created_at by d, so the aging pins can
	// run in milliseconds instead of days. White-box by necessity —
	// production code has no such path; engines implement it with a
	// direct UPDATE against their tasks table.
	Backdate func(t *testing.T, s queue.Store[Payload], id task.ID, d time.Duration)
}

// Run executes the full conformance suite against the harness's engine.
// It is the ONLY registration surface: engines call it from one test
// function and inherit every pin, present and future.
func Run(t *testing.T, h Harness) {
	t.Helper()

	if h.NewStore == nil || h.Backdate == nil {
		t.Fatal("conformance: Harness.NewStore and Harness.Backdate are both required")
	}

	s := &suite{h: h}

	t.Run("Lifecycle", s.runLifecycle)
	t.Run("Claims", s.runClaims)
	t.Run("Retry", s.runRetry)
	t.Run("Dedup", s.runDedup)
	t.Run("Journal", s.runJournal)
	t.Run("Reads", s.runReads)
}

// suite carries the harness through the pin methods.
type suite struct {
	h Harness
}

// openEnv wires a fresh store for one subtest, closed on cleanup.
func (s *suite) openEnv(t *testing.T) *env {
	t.Helper()

	store := s.h.NewStore(t)
	t.Cleanup(func() { _ = store.Close() })

	return &env{store: store}
}

// env is one subtest's store handle.
type env struct {
	store queue.Store[Payload]
}

// enqueue is the one-line happy path every pin starts from.
func (e *env) enqueue(t *testing.T, n task.New[Payload]) task.Task[Payload] {
	t.Helper()

	got, err := e.store.Enqueue(t.Context(), n)
	if err != nil {
		t.Fatalf("enqueue %q: %v", n.Type, err)
	}

	return got
}

// claim is the one-line claim helper.
func (e *env) claim(t *testing.T, owner string) queue.Claim[Payload] {
	t.Helper()

	c, err := e.store.ClaimDue(t.Context(), owner, time.Minute)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	return c
}

// mustError asserts op fails with an error matching want via errors.Is.
func mustError(t *testing.T, op string, err, want error) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected error %v, got nil", op, want)
	}

	if !errors.Is(err, want) {
		t.Fatalf("%s: error = %v, want %v", op, err, want)
	}
}
