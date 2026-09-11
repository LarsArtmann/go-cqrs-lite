package metaengine

import (
	"context"
	"encoding/json/v2"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ApplyEncoded processes a JSON-encoded event payload through all queries.
// The eventType identifies which fold to invoke, and payload is JSON bytes
// decoded into each fold's expected event type via its sample.
//
// The apply is a full pipeline citizen — identical to Store.Apply except the
// payload arrives undecoded: it is metered, hook-observed, recorded to the
// attached EventLog, replicated to shadow engines, and counted by the
// synthetic-record advisory (Store.Apply's Doctor caveat applies here too —
// only a Type-only Record can be synthesized, so OnRecord folds see empty
// StreamID/Version; use ApplyEncodedRecord to carry the full Record).
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
	return s.applyWithRecord(ctx, eventType, record.Record{Type: eventType}, rawJSON(payload))
}

// ApplyEncodedRecord is ApplyEncoded with full Record context (ADR-0112):
// Record-aware folds (created via OnRecord) receive rec alongside the
// decoded payload, exactly like Store.ApplyRecord. payload is JSON bytes;
// rec.Type must name the event type the folds listen for. Non-Record-aware
// folds receive only the decoded payload, as usual.
func (s *Store) ApplyEncodedRecord(ctx context.Context, rec record.Record, payload []byte) error {
	if rec.Type == "" {
		return errors.New("metaengine.Store.ApplyEncodedRecord: Record.Type is empty — it must name the event type")
	}

	return s.applyWithRecord(ctx, rec.Type, rec, rawJSON(payload))
}

// rawJSON clones the caller's bytes into a jsontext.Value — the pipeline's
// marker for an undecoded JSON payload (applyFold decodes it per fold via
// the fold's sample). The clone protects the EventLog entry and any queued
// replication job from caller-side mutation after the call returns.
func rawJSON(payload []byte) jsontext.Value {
	return jsontext.Value(slices.Clone(payload))
}

// decodeRawFoldPayload decodes a raw JSON payload into the fold's expected
// event type (via its sample) before invoke, so raw-JSON applies replay
// identically on every dispatch path (primary folds, shadows, replays).
// Decoded payloads pass through unchanged — one type assertion on the struct
// hot path. Folds whose sample is itself a byte slice consume the raw bytes
// undecoded.
func decodeRawFoldPayload(fold Fold, payload any) (any, error) {
	raw, ok := payload.(jsontext.Value)
	if !ok {
		return payload, nil
	}

	if t := derefType(fold.EventSample()); t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return []byte(raw), nil
	}

	decoded, err := decodeFromSample(fold.EventSample(), raw)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", fold.EventType(), err)
	}

	return decoded, nil
}

func decodeFromSample(sample any, payload []byte) (any, error) {
	t := derefType(sample)

	v := reflect.New(t)
	if err := json.Unmarshal(payload, v.Interface()); err != nil {
		return nil, fmt.Errorf("json decode into %s: %w", t.Name(), err)
	}

	return v.Elem().Interface(), nil
}
