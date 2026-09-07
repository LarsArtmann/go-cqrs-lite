// System-backed benchmarking: adapt a system.System onto the bundle-shaped
// Factory so every existing benchkit phase runs against the strategic
// composition layer. Capabilities system does not expose (the bundle's
// kv-backed ReadModels) stay nil — the runner skips those phases with
// recorded warnings instead of silently mis-benchmarking.
package benchkit

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// SystemFactory creates a fresh *system.System for one benchmark run.
// Implementations must return an independently closable system: the adapted
// Factory closes it when the runner closes the bundle.
type SystemFactory func(ctx context.Context) (*system.System, error)

// FactoryFromSystem adapts a SystemFactory onto the bundle-shaped Factory
// contract, so `benchkit.Run` and every phase run unchanged against
// system.New deployments (memory, sqlite, or multi-engine operator configs).
func FactoryFromSystem(sf SystemFactory) Factory {
	return func() (*stack.Bundle, error) {
		sys, err := sf(context.Background())
		if err != nil {
			return nil, fmt.Errorf("system factory: %w", err)
		}

		return AdaptSystem(sys)
	}
}

// AdaptSystem maps a live system.System onto a *stack.Bundle view:
// events, commands, queries, snapshots, and the seekable journal flow from
// the system's stores; closing the returned bundle closes the system.
// Bundle capabilities system does not own (kv ReadModels, bus subscriber,
// command/query journals) stay nil — the runner records those phases as
// skipped, never fabricated.
func AdaptSystem(sys *system.System) (*stack.Bundle, error) {
	if sys == nil {
		return nil, ErrNilSystem
	}

	store := sys.EventStore()
	if store == nil {
		return nil, ErrSystemEventStoreMissing
	}

	bundle, err := stack.New(stack.WithCloser(&closerFunc{fn: sys.Close}))
	if err != nil {
		return nil, fmt.Errorf("bundle for system adapter: %w", err)
	}

	bundle.EventSink = store
	bundle.EventSource = store

	if journal, ok := store.(event.SeekableJournal); ok {
		bundle.Journal = journal
		bundle.SeekableJournal = journal
	}

	if backwards, ok := store.(event.BackwardsSource); ok {
		bundle.BackwardsSource = backwards
	}

	bundle.Publisher = sys.Publisher()
	bundle.CommandSink = sys.CommandStore()
	bundle.CommandSource = sys.CommandStore()
	bundle.QuerySink = sys.QueryStore()
	bundle.QuerySource = sys.QueryStore()
	bundle.SnapshotStore = sys.SnapshotStore()

	return bundle, nil
}

// closerFunc adapts a close function onto io.Closer. It is a struct (not a
// func type) with pointer semantics because Bundle.Close deduplicates closers
// through a map — func values are unhashable and panic as map keys.
type closerFunc struct {
	fn func() error
}

func (f *closerFunc) Close() error { return f.fn() }
