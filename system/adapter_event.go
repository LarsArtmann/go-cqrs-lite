package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/maypok86/otter/v2"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// seqCacheCapacity bounds the event-ID → seq cache. 4096 covers resume
// cursors for thousands of concurrent subscribers while keeping the cache
// in the L1/L2-friendly range; evictions only cost a re-seed scan.
const seqCacheCapacity = 4096

// EventAdapterOption tunes an EventAdapter at construction time.
type EventAdapterOption func(*EventAdapter)

// WithSerialization enables event serialization for persistent engines
// (SQLite, Pebble). Each event is wrapped in a JSON envelope that carries the
// raw payload bytes (base64) plus its original [codec.Encoding] tag, so
// payloads round-trip byte-identical: CBOR events are NOT re-encoded, and
// [event.DecodePayloadAuto] sees the original encoding on read. Pinned by
// TestEventAdapter_CBORPayloadRoundTripSQLite. For the Memory engine, this
// option should NOT be set — events are stored as direct pointers.
func WithSerialization() EventAdapterOption {
	return func(a *EventAdapter) { a.Serialize = true }
}

// WithRacySave opts in to the racy check-then-append Save fallback for
// backends that implement neither [metaengine.AtomicAppender] nor
// [metaengine.Transactional]. Without it such a backend fails closed
// ([ErrRacySaveRefused]). Under concurrency two writers can both pass the
// version check and both append, corrupting the stream version sequence —
// use only for single-threaded or externally-serialized backends. The
// source-of-truth role refuses such engines at construction regardless
// ([ErrEventSaveNotAtomic]).
func WithRacySave() EventAdapterOption {
	return func(a *EventAdapter) { a.allowRacySave = true }
}

// EventAdapter wraps a [metaengine.StreamLogBackend] as an [event.Store].
type EventAdapter struct {
	AdapterCore[event.Event]

	// temporal is non-nil when the backend implements StreamTemporalReader,
	// enabling efficient version-bounded reads without loading the full stream.
	temporal metaengine.StreamTemporalReader

	// seqSeek is non-nil when the backend implements SeqSeekableStreamLog.
	// ReadFrom then resumes on engine seq tokens (O(log n) index seeks)
	// instead of journal positions (O(offset) OFFSET scans), and the seq
	// cache below stores true tokens rather than positions.
	seqSeek metaengine.SeqSeekableStreamLog

	// byID is non-nil when the backend implements EventByIDBackend (#32):
	// LoadByEventID then resolves single events by their globally unique ID
	// instead of forcing callers into sequential journal scans.
	byID metaengine.EventByIDBackend

	// seqCache maps event IDs to journal positions or engine seq tokens.
	// Bounded (default 4096 entries, LRU eviction via otter) so long-running
	// processes cannot grow it without limit; otter is safe for concurrent
	// use, so no external mutex is needed.
	seqCache *otter.Cache[string, int64]

	// allowRacySave opts into the racy check-then-append Save fallback for
	// backends with neither AtomicAppender nor Transactional (see WithRacySave).
	allowRacySave bool
}

// NewEventAdapter creates an event.Store backed by a StreamLogBackend.
func NewEventAdapter(
	backend metaengine.StreamLogBackend,
	collection string,
	opts ...EventAdapterOption,
) *EventAdapter {
	a := &EventAdapter{seqCache: otter.Must(&otter.Options[string, int64]{
		MaximumSize: seqCacheCapacity,
	})}
	a.AdapterCore = AdapterCore[event.Event]{
		Backend:    backend,
		Collection: collection,
		Noun:       "event",
		Encode:     a.encodeEvent,
		Decode:     a.decodeEvent,
		IDOf:       func(evt event.Event) string { return evt.ID().String() },
	}

	for _, opt := range opts {
		opt(a)
	}

	// Detect StreamTemporalReader support for efficient version-bounded reads.
	if tr, ok := backend.(metaengine.StreamTemporalReader); ok {
		a.temporal = tr
	}

	// Detect token-resumable journal reads: ReadFrom switches from
	// position-based OFFSET skips to O(log n) seq-token index seeks.
	if ss, ok := backend.(metaengine.SeqSeekableStreamLog); ok {
		a.seqSeek = ss
	}

	// Detect single-event-by-ID resolution (EventByIDBackend, #32).
	if eb, ok := backend.(metaengine.EventByIDBackend); ok {
		a.byID = eb
	}

	return a
}

// Compile-time assertions.
var (
	_ event.Store           = (*EventAdapter)(nil)
	_ event.Journal         = (*EventAdapter)(nil)
	_ event.SeekableJournal = (*EventAdapter)(nil)
)

// ─── EventSink ───

func (a *EventAdapter) Save(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	// Sticky stream key (bare for fresh streams; legacy display-form key
	// when older rows exist — see resolveStreamKey).
	sid, err := resolveStreamKey(ctx, a.Backend, a.Collection, ref)
	if err != nil {
		return fmt.Errorf("event adapter: resolve stream key: %w", err)
	}

	//art-dupl:accept same-shape ToAny+wrap — adapter-specific types
	values, err := a.ToAny(events)
	if err != nil {
		return err
	}

	if ap, ok := a.Backend.(metaengine.AtomicAppender); ok {
		if err := ap.StreamAppendExpected(
			ctx,
			a.Collection,
			sid,
			int64(expectedVersion),
			values,
		); err != nil {
			if errors.Is(err, metaengine.ErrVersionConflict) {
				return event.ErrVersionConflict
			}

			return fmt.Errorf("event adapter: save: %w", err)
		}

		return nil
	}

	if tx, ok := a.Backend.(metaengine.Transactional); ok {
		return tx.RunInTx(ctx, func(ctx context.Context) error {
			current, err := a.Backend.StreamVersion(ctx, a.Collection, sid)
			if err != nil {
				return fmt.Errorf("event adapter: stream version: %w", err)
			}

			if err := event.CheckVersionConflict(int(current), expectedVersion); err != nil {
				return err
			}

			return a.Backend.StreamAppend(ctx, a.Collection, sid, values)
		})
	}

	if !a.allowRacySave {
		return fmt.Errorf(
			"%w: backend %T implements neither metaengine.AtomicAppender nor metaengine.Transactional; have the backend implement one of them for concurrent safety, or pass WithRacySave() for a deliberately single-threaded store",
			ErrRacySaveRefused,
			a.Backend,
		)
	}

	current, err := a.Backend.StreamVersion(ctx, a.Collection, sid)
	if err != nil {
		return fmt.Errorf("event adapter: stream version: %w", err)
	}

	if err := event.CheckVersionConflict(int(current), expectedVersion); err != nil {
		return err
	}

	return a.Backend.StreamAppend(ctx, a.Collection, sid, values)
}

func (a *EventAdapter) AppendBatch(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
) error {
	//art-dupl:accept same-shape ToAny+wrap — adapter-specific types
	values, err := a.ToAny(events)
	if err != nil {
		return err
	}

	// Sticky stream key: append where the stream's rows already live.
	sid, err := resolveStreamKey(ctx, a.Backend, a.Collection, ref)
	if err != nil {
		return fmt.Errorf("event adapter: resolve stream key: %w", err)
	}

	return a.Backend.StreamAppend(ctx, a.Collection, sid, values)
}

// ─── EventSource ───

func (a *EventAdapter) Load(ctx context.Context, ref id.StreamRef) ([]event.Event, error) {
	// art-dupl:accept command/event adapters share the resolveStreamKey prologue by design
	key, err := resolveStreamKey(ctx, a.Backend, a.Collection, ref)
	if err != nil {
		return nil, fmt.Errorf("event adapter: resolve stream key: %w", err)
	}

	return a.LoadStream(ctx, key)
}

// ReadAll is promoted from AdapterCore and satisfies event.Journal.
