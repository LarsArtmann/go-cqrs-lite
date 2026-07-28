package metaengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"maps"
	"reflect"
	"slices"
)

// ApplyEncoded processes a JSON-encoded event payload through all queries.
// The eventType identifies which fold to invoke, and payload is JSON bytes
// that will be decoded into the fold's expected event type via reflection.
//
// For non-JSON encodings (CBOR, etc.), decode manually and use Store.Apply.
//
// Example with event.Event:
//
//	err := store.ApplyEncoded(string(evt.Type()), evt.Payload())
//
// To integrate with projection.Projection, create a thin adapter:
//
//	type projectionAdapter struct{ store *metaengine.Store }
//	func (p *projectionAdapter) Handle(_ context.Context, evt event.Event) error {
//	    return p.store.ApplyEncoded(string(evt.Type()), evt.Payload())
//	}
func (s *Store) ApplyEncoded(ctx context.Context, eventType string, payload []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		q := s.queries[name]

		foldIdx, ok := q.foldByEvent[eventType]
		if !ok {
			continue
		}

		fold := q.folds[foldIdx]

		decoded, err := decodeFromSample(fold.EventSample, payload)
		if err != nil {
			return fmt.Errorf("query %q decode %s: %w", q.name, eventType, err)
		}

		if err := s.applyFold(ctx, q, fold, decoded); err != nil {
			return fmt.Errorf("query %q fold for %s: %w", q.name, eventType, err)
		}
	}

	return nil
}

func decodeFromSample(sample any, payload []byte) (any, error) {
	t := derefType(sample)

	v := reflect.New(t)
	if err := json.Unmarshal(payload, v.Interface()); err != nil {
		return nil, fmt.Errorf("json decode into %s: %w", t.Name(), err)
	}

	return v.Elem().Interface(), nil
}


