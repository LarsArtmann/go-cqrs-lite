package metaengine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// ApplyEncoded processes a JSON-encoded event payload through all models.
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
func (s *Store) ApplyEncoded(eventType string, payload []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, m := range s.models {
		foldIdx, ok := m.foldByEvent[eventType]
		if !ok {
			continue
		}

		fold := m.folds[foldIdx]

		decoded, err := decodeFromSample(fold.EventSample, payload)
		if err != nil {
			return fmt.Errorf("model %q decode %s: %w", m.name, eventType, err)
		}

		if err := s.applyFold(m, fold, decoded); err != nil {
			return fmt.Errorf("model %q fold for %s: %w", m.name, eventType, err)
		}
	}

	return nil
}

func decodeFromSample(sample any, payload []byte) (any, error) {
	t := reflect.TypeOf(sample)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	v := reflect.New(t).Interface()

	if err := json.Unmarshal(payload, v); err != nil {
		return nil, fmt.Errorf("json decode into %s: %w", t.Name(), err)
	}

	return v, nil
}

// EventTypeNames returns all event type names the store's models react to.
// Useful for building projection.Projection adapters.
func (s *Store) EventTypeNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	for _, m := range s.models {
		for _, f := range m.folds {
			if f.Kind != FoldSkip {
				seen[f.EventType] = true
			}
		}
	}

	result := make([]string, 0, len(seen))
	for t := range seen {
		result = append(result, t)
	}

	sort.Strings(result)

	return result
}
