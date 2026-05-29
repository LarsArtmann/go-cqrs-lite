package event

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id"
)

// StreamKey returns the canonical key for an event stream,
// combining aggregate type and aggregate ID.
func StreamKey(aggregateType AggregateType, aggregateID id.AggregateID) string {
	return AggregateRef{Type: aggregateType, ID: aggregateID}.String()
}

// StreamLoader loads events as a stream rather than loading all into memory at once.
// This is essential for large aggregates or projection replay where memory usage
// must be bounded. SQL implementations can use cursor-based fetching (e.g., sql.Rows).
type StreamLoader interface {
	// LoadStream returns a stream of events for a single aggregate, ordered by version.
	LoadStream(
		ctx context.Context,
		ref AggregateRef,
	) (EventStream, error)
}

// EventStream yields events one at a time.
//
// Usage:
//
//	stream, err := store.LoadStream(ctx, "Order", orderID)
//	if err != nil { ... }
//	defer stream.Close()
//
//	for {
//	    evt, ok := stream.Next()
//	    if !ok {
//	        break
//	    }
//	    // process evt
//	}
//	if err := stream.Err(); err != nil {
//	    // handle error
//	}
type EventStream interface {
	// Next returns the next event and true, or a zero Event and false when
	// the stream is exhausted or an error occurs. Call Err() after Next
	// returns false to distinguish EOF from an actual error.
	Next() (Event, bool)

	// Err returns any error encountered during iteration.
	Err() error

	// Close releases resources associated with the stream.
	Close() error
}

// sliceStream is a simple in-memory EventStream backed by a []Event slice.
type sliceStream struct {
	events []Event
	idx    int
	err    error
}

// NewSliceStream creates an EventStream from a pre-loaded slice.
// Used by in-memory implementations, tests, and as a fallback for Store wrappers.
func NewSliceStream(events []Event) EventStream {
	return &sliceStream{events: events}
}

func (s *sliceStream) Next() (Event, bool) {
	if s.err != nil || s.idx >= len(s.events) {
		return nil, false
	}

	evt := s.events[s.idx]
	s.idx++

	return evt, true
}

func (s *sliceStream) Err() error {
	return s.err
}

func (s *sliceStream) Close() error {
	s.events = nil

	return nil
}

// StoreStreamAdapter wraps a Store to provide StreamLoader semantics.
// It loads all events via Store.Load and streams them from memory.
// Useful for stores that do not natively support cursor-based streaming.
type StoreStreamAdapter struct {
	store Store
}

// NewStoreStreamAdapter creates a StreamLoader that delegates to the given Store.
func NewStoreStreamAdapter(store Store) *StoreStreamAdapter {
	return &StoreStreamAdapter{store: store}
}

// LoadStream implements StreamLoader by loading all events and yielding them sequentially.
func (a *StoreStreamAdapter) LoadStream(
	ctx context.Context,
	ref AggregateRef,
) (EventStream, error) {
	events, err := a.store.Load(ctx, ref)
	if err != nil {
		return nil, err
	}

	return NewSliceStream(events), nil
}
