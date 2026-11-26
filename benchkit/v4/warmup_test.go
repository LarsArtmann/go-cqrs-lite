package benchkit

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	storagemem "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// versionCheckingStore enforces the Save contract the way the SQL stores do:
// an appended event must carry expectedVersion+1 (and consecutive versions
// within one call). The in-memory store accepts any version, which is how a
// warmup stream with a hole in its versions went unnoticed.
type versionCheckingStore struct {
	event.Store
}

func (s *versionCheckingStore) Save(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
	expected event.Version,
) error {
	for i, evt := range events {
		if got, want := evt.Version(), expected.Add(uint(i+1)); got != want {
			return fmt.Errorf("expected version %d, got %d", want, got)
		}
	}

	return s.Store.Save(ctx, ref, events, expected)
}

// TestRun_WarmupStreamVersionsAreConsecutive pins the warmup stream: every
// warmup event must carry the version after the previous one. The second
// warmup event carried current+2 until 2026-11-25, so `--warmup 2` failed on
// every store that checks versions while the lenient in-memory store hid it.
func TestRun_WarmupStreamVersionsAreConsecutive(t *testing.T) {
	t.Parallel()

	factory := func() (*stack.Bundle, error) {
		return stack.New(stack.WithEventStore(&versionCheckingStore{Store: storagemem.NewMemoryStore()}))
	}

	for _, warmup := range []int{1, 2, 5, 50} {
		result, err := Run(t.Context(), Config{
			Profile:        ProfileDev,
			PayloadSize:    64,
			Warmup:         warmup,
			SkipMetaEngine: true,
		}, factory)
		if err != nil {
			t.Fatalf("Run with Warmup=%d: %v", warmup, err)
		}

		if result.WarmupEvents != warmup {
			t.Errorf("Warmup=%d: WarmupEvents = %d", warmup, result.WarmupEvents)
		}
	}
}
