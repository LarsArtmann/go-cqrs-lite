package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ErrTrimUnavailable names the missing-capability class: the configured event
// store's backend implements neither (or not the requested one) of the
// metaengine trim capabilities. nil-surface facts, not miswires: the System
// simply was not built over an erasable journal.
var ErrTrimUnavailable = errors.New("system: journal trimming is not available on the configured event store")

// snapshotsCollection is the system's snapshot namespace on the engine's
// snapshot table — the same name buildSnapshotStore wires (roles.go) and
// TrimStream/TrimJournalBefore co-trim against.
const snapshotsCollection = "snapshots"

// trimBackend resolves the stream-log backend and its journal collection from
// the wired event store. Both trim surfaces are maintenance-only: they read
// the immutable adapter wiring (set once in New), so no System lock is taken.
func (s *System) trimBackend() (metaengine.StreamLogBackend, string, error) {
	ad, ok := s.eventStore.(*EventAdapter)
	if !ok {
		return nil, "", fmt.Errorf("%w: event store %T is not the engine adapter", ErrTrimUnavailable, s.eventStore)
	}

	return ad.Backend, ad.Collection, nil
}

// TrimStream erases one stream's journal rows and its snapshot (#118,
// metaengine.StreamTrimmer): the GDPR-erasure maintenance primitive. The
// deletion DOMAIN event must already have been emitted and folded — this call
// settles the layer below (the bytes), after which the stream loads as
// never-written and replay never sees it. Returns the number of journal rows
// removed.
func (s *System) TrimStream(ctx context.Context, ref id.StreamRef) (int64, error) {
	backend, collection, err := s.trimBackend()
	if err != nil {
		return 0, err
	}

	trimmer, ok := backend.(metaengine.StreamTrimmer)
	if !ok {
		return 0, fmt.Errorf("%w: backend %T implements metaengine.StreamTrimmer nowhere", ErrTrimUnavailable, backend)
	}

	// Sticky-key resolution: a legacy-keyed stream's rows must be erased
	// under the key they actually live in (same rule as Save).
	sid, err := resolveStreamKey(ctx, backend, collection, ref)
	if err != nil {
		return 0, fmt.Errorf("system: trim %s: resolve stream key: %w", ref, err)
	}

	removed, err := trimmer.TrimStream(ctx, collection, snapshotsCollection, sid)
	if err != nil {
		return removed, fmt.Errorf("system: trim %s: %w", ref, err)
	}

	return removed, nil
}

// TrimJournalBefore erases every journal row older than cutoff plus the
// co-trimmed snapshots (#118, metaengine.JournalSweeper): the age-sweep
// maintenance primitive for retention windows and bounded journals. The
// operator owns the correctness precondition: remaining rows must still fold
// to honest state. Returns the number of journal rows removed.
func (s *System) TrimJournalBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	backend, collection, err := s.trimBackend()
	if err != nil {
		return 0, err
	}

	sweeper, ok := backend.(metaengine.JournalSweeper)
	if !ok {
		return 0, fmt.Errorf("%w: backend %T implements metaengine.JournalSweeper nowhere", ErrTrimUnavailable, backend)
	}

	removed, err := sweeper.TrimJournalBefore(ctx, collection, snapshotsCollection, cutoff)
	if err != nil {
		return removed, fmt.Errorf("system: sweep journal before %s: %w", cutoff.Format(time.RFC3339), err)
	}

	return removed, nil
}
