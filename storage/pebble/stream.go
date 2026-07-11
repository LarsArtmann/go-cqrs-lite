package pebble

import (
	"context"
	"io"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// pebbleEventIterator streams events from a Pebble iterator without
// materializing the full slice. NOT goroutine-safe.
type pebbleEventIterator struct {
	iter      *pebble.Iterator
	store     *EventStore
	started   bool
	closed    bool
	firstErr  error
	skipUntil string
	skipping  bool
	limit     int
	yielded   int
}

func (it *pebbleEventIterator) Next() (event.Event, error) {
	if it.closed {
		return nil, io.EOF
	}

	if it.firstErr != nil {
		return nil, it.firstErr
	}

	if it.limit > 0 && it.yielded >= it.limit {
		return nil, io.EOF
	}

	if !it.started {
		it.started = true
		it.iter.First()
	} else {
		it.iter.Next()
	}

	for it.iter.Valid() {
		if it.skipping {
			if journalKeyEventID(it.iter.Key()) == it.skipUntil {
				it.skipping = false
			}

			it.iter.Next()

			continue
		}

		evt, err := it.store.deserializeEvent(it.iter.Value())
		if err != nil {
			it.firstErr = errorfamily.Wrapf(
				it.store.corruptEventErr(string(it.iter.Key()), err),
				errorfamily.Corruption, "pebble.stream_corrupt_event",
				"corrupt event during stream iteration",
			)

			return nil, it.firstErr
		}

		it.yielded++

		return evt, nil
	}

	err := checkIteratorError(it.iter)
	if err != nil {
		it.firstErr = errorfamily.Wrapf(err, errorfamily.Infrastructure, "pebble.stream_iterator",
			"iterator error during stream iteration")

		return nil, it.firstErr
	}

	return nil, io.EOF
}

func (it *pebbleEventIterator) Close() error {
	if it.closed {
		return nil
	}

	it.closed = true

	err := it.iter.Close()
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"pebble.stream.close_iterator",
			"close pebble iterator",
		)
	}

	return nil
}

// newPebbleIterator creates a pebbleEventIterator over the given key range.
func (a *EventStore) newPebbleIterator(
	lowerBound, upperBound []byte,
	skipID string,
	limit int,
) (*pebbleEventIterator, error) {
	iter, err := a.db.NewIter(
		&pebble.IterOptions{
			LowerBound: lowerBound,
			UpperBound: upperBound,
		},
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "pebble.stream_create_iterator",
			"create streaming iterator")
	}

	return &pebbleEventIterator{
		iter:      iter,
		store:     a,
		skipUntil: skipID,
		skipping:  skipID != "",
		limit:     limit,
	}, nil
}

// LoadStream is the streaming equivalent of Load.
func (a *EventStore) LoadStream(
	ctx context.Context,
	ref id.AggregateRef,
) (event.EventIterator, error) {
	_, span := startAggregateSpan(ctx, "pebble.event.load_stream", ref)
	defer span.End()

	lower := a.aggregatePrefix(ref)
	upper := a.aggregateUpperBound(ref)

	return a.newPebbleIterator(lower, upper, "", 0)
}

// LoadStreamFromVersion is the streaming equivalent of LoadFromVersion.
func (a *EventStore) LoadStreamFromVersion(
	ctx context.Context,
	ref id.AggregateRef,
	version event.Version,
) (event.EventIterator, error) {
	_, span := startAggregateSpan(ctx, "pebble.event.load_stream_from_version", ref,
		cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, version.Int()))
	defer span.End()

	lower := a.eventKey(ref, version+1)
	upper := a.aggregateUpperBound(ref)

	return a.newPebbleIterator(lower, upper, "", 0)
}

// ReadStream is the streaming equivalent of ReadAll.
func (a *EventStore) ReadStream(ctx context.Context) (event.EventIterator, error) {
	_, span := cqrsotel.StartSpan(ctx, tracer(), "pebble.journal.read_stream",
		cqrsotel.SpanKindClient)
	defer span.End()

	lower := []byte(a.journalPrefix)
	upper := []byte(a.journalPrefix + "\xff")

	return a.newPebbleIterator(lower, upper, "", 0)
}

// ReadStreamFrom is the streaming equivalent of ReadFrom.
//
// Unlike ReadFrom, this always uses a full-range scan with skip-until.
// The narrowed-range optimization (ULID timestamp - 1min) from ReadFrom
// is NOT applied because the streaming iterator cannot efficiently detect
// "target not found" to trigger a fallback scan. The full-range scan is
// still O(1) memory (iterator-based) — it just scans a few more keys.
// This eliminates the split brain between streaming and slice paths.
func (a *EventStore) ReadStreamFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) (event.EventIterator, error) {
	_, span := cqrsotel.StartSpan(ctx, tracer(), "pebble.journal.read_stream_from",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrInt("limit", limit)))
	defer span.End()

	upper := []byte(a.journalPrefix + "\xff")
	lower := []byte(a.journalPrefix)

	if afterEventID.IsZero() {
		return a.newPebbleIterator(lower, upper, "", limit)
	}

	return a.newPebbleIterator(lower, upper, afterEventID.String(), limit)
}

var (
	_ event.StreamingSource  = (*EventStore)(nil)
	_ event.StreamingJournal = (*EventStore)(nil)
)
