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

	afterSeq := a.lookupSeq(ctx, afterEventID)

	values, err := a.Backend.JournalReadFrom(ctx, a.Collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("event adapter: read from: %w", err)
	}

	events, err := a.FromAny(values)
	if err != nil {
		return nil, err
	}

	a.seqCacheMu.Lock()
	for i, evt := range events {
		a.seqCache[evt.ID().String()] = afterSeq + int64(i) + 1
	}
	a.seqCacheMu.Unlock()

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
		afterSeq = a.lookupSeqToken(ctx, afterEventID)
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

	a.seqCacheMu.Lock()
	for i, evt := range events {
		a.seqCache[evt.ID().String()] = entries[i].Seq
	}
	a.seqCacheMu.Unlock()

	return events, nil
}

// lookupSeq returns the global journal sequence number for the given event ID.
func (a *EventAdapter) lookupSeq(ctx context.Context, eventID id.EventID) int64 {
	key := eventID.String()

	a.seqCacheMu.RLock()
	seq, ok := a.seqCache[key]
	a.seqCacheMu.RUnlock()

	if ok {
		return seq
	}

	all, err := a.Backend.JournalReadAll(ctx, a.Collection)
	if err != nil {
		return 0
	}

	events, _ := a.FromAny(all)

	a.seqCacheMu.Lock()
	defer a.seqCacheMu.Unlock()

	for i, evt := range events {
		a.seqCache[evt.ID().String()] = int64(i + 1)
	}

	if seq, ok := a.seqCache[key]; ok {
		return seq
	}

	return 0
}

// lookupSeqToken resolves an event ID to its engine seq token on a cold
// cache miss. One full pass over JournalReadAllWithSeq seeds the cache with
// true tokens — after this, every subsequent resume (including the caller's
// current one) is an index seek. Unknown IDs resolve to 0 (read from start),
// matching lookupSeq.
func (a *EventAdapter) lookupSeqToken(ctx context.Context, eventID id.EventID) int64 {
	key := eventID.String()

	a.seqCacheMu.RLock()
	seq, ok := a.seqCache[key]
	a.seqCacheMu.RUnlock()

	if ok {
		return seq
	}

	entries, err := a.seqSeek.JournalReadAllWithSeq(ctx, a.Collection)
	if err != nil {
		return 0
	}

	a.seqCacheMu.Lock()
	defer a.seqCacheMu.Unlock()

	for _, entry := range entries {
		evt, err := a.decodeValue(entry.Value)
		if err != nil {
			continue
		}

		a.seqCache[evt.ID().String()] = entry.Seq
	}

	if seq, ok := a.seqCache[key]; ok {
		return seq
	}

	return 0
}
