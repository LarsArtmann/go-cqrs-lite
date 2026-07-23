package metaengine

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
)

// ApplyEvent processes a real event.Event through all models that listen to it.
// The payload is JSON-decoded into the fold's expected event type via reflection.
//
// For non-JSON encodings (CBOR, raw), decode manually and use Store.Apply with
// the typed payload.
func (s *Store) ApplyEvent(_ context.Context, evt cqrsevent.Event) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	eventType := string(evt.Type())
	payload := evt.Payload()

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

// EventTypes returns all event types the store's models react to.
// This satisfies the projection.Projection interface contract.
func (s *Store) EventTypes() []cqrsevent.Type {
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

	types := make([]string, 0, len(seen))
	for t := range seen {
		types = append(types, t)
	}

	sort.Strings(types)

	result := make([]cqrsevent.Type, len(types))
	for i, t := range types {
		result[i] = cqrsevent.Type(t)
	}

	return result
}

// AsProjection returns a value that implements the projection.Projection interface
// (Name, Handle, EventTypes). Register it with projectionhost.Host:
//
//	host.Register(store.AsProjection("metaengine"))
//
// The projection satisfies projection.Projection via Go's structural typing —
// no import of the projection package is needed.
func (s *Store) AsProjection(name string) ProjectionAdapter {
	return ProjectionAdapter{store: s, name: name}
}

// ProjectionAdapter wraps a Store as a projection.Projection.
type ProjectionAdapter struct {
	store *Store
	name  string
}

func (p ProjectionAdapter) Name() string { return p.name }

func (p ProjectionAdapter) Handle(ctx context.Context, evt cqrsevent.Event) error {
	return p.store.ApplyEvent(ctx, evt)
}

func (p ProjectionAdapter) EventTypes() []cqrsevent.Type {
	return p.store.EventTypes()
}
