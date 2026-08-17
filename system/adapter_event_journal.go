package system

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ─── SeekableJournal ───

func (a *EventAdapter) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	if a.seqSeek != nil {
		return a.readFromSeq(ctx, afterEventID, limit)
	}

	afterSeq := int64(0)

	if !afterEventID.IsZero() {
		seq, err := a.lookupSeq(ctx, afterEventID)
		if err != nil {
			return nil, fmt.Errorf("event adapter: resolve resume cursor %s: %w", afterEventID, err)
		}

		afterSeq = seq
	}

	values, err := a.Backend.JournalReadFrom(ctx, a.Collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("event adapter: read from: %w", err)
	}

	events, err := a.FromAny(values)
	if err != nil {
		return nil, err
	}

	for i, evt := range events {
		a.seqCache.Set(evt.ID().String(), afterSeq+int64(i)+1)
	}

	return events, nil
}

// readFromSeq is ReadFrom on backends implementing SeqSeekableStreamLog: the
// cursor is a true engine seq token, so every resume is an O(log n) index
// seek instead of an O(offset) positional skip. A zero afterEventID reads
// from the start without any journal scan — the cold-start path of every
// catch-up subscriber.
func (a *EventAdapter) readFromSeq(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	afterSeq := int64(0)

	if !afterEventID.IsZero() {
		token, err := a.lookupSeqToken(ctx, afterEventID)
		if err != nil {
			return nil, fmt.Errorf("event adapter: resolve resume token %s: %w", afterEventID, err)
		}

		afterSeq = token
	}

	entries, err := a.seqSeek.JournalReadFromSeq(ctx, a.Collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("event adapter: read from seq: %w", err)
	}

	events := make([]event.Event, 0, len(entries))

	for _, entry := range entries {
		evt, err := a.decodeValue(entry.Value)
		if err != nil {
			return nil, err
		}

		events = append(events, evt)
	}

	for i, evt := range events {
		a.seqCache.Set(evt.ID().String(), entries[i].Seq)
	}

	return events, nil
}

// lookupSeq returns the global journal sequence number for the given event ID.
// A cache miss triggers one full journal scan that seeds the cache. An ID not
// present in the journal resolves to 0 (read from start) — that is not an
// error, it is the documented cold-start resume behavior.
func (a *EventAdapter) lookupSeq(ctx context.Context, eventID id.EventID) (int64, error) {
	key := eventID.String()

	if seq, ok := a.seqCache.GetIfPresent(key); ok {
		return seq, nil
	}

	all, err := a.Backend.JournalReadAll(ctx, a.Collection)
	if err != nil {
		return 0, fmt.Errorf("event adapter: scan journal for resume cursor: %w", err)
	}

	events, err := a.FromAny(all)
	if err != nil {
		return 0, err
	}

	for i, evt := range events {
		a.seqCache.Set(evt.ID().String(), int64(i+1))
	}

	if seq, ok := a.seqCache.GetIfPresent(key); ok {
		return seq, nil
	}

	return 0, nil
}

// lookupSeqToken resolves an event ID to its engine seq token on a cold
// cache miss. One full pass over JournalReadAllWithSeq seeds the cache with
// true tokens — after this, every subsequent resume (including the caller's
// current one) is an index seek. Unknown IDs resolve to 0 (read from start),
// matching lookupSeq.
func (a *EventAdapter) lookupSeqToken(ctx context.Context, eventID id.EventID) (int64, error) {
	key := eventID.String()

	if seq, ok := a.seqCache.GetIfPresent(key); ok {
		return seq, nil
	}

	entries, err := a.seqSeek.JournalReadAllWithSeq(ctx, a.Collection)
	if err != nil {
		return 0, fmt.Errorf("event adapter: scan journal for resume token: %w", err)
	}

	for _, entry := range entries {
		evt, err := a.decodeValue(entry.Value)
		if err != nil {
			continue
		}

		a.seqCache.Set(evt.ID().String(), entry.Seq)
	}

	if seq, ok := a.seqCache.GetIfPresent(key); ok {
		return seq, nil
	}

	return 0, nil
}
