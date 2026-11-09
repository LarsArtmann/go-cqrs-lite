package sqlstore

// The timer payload envelope and its decoders: the versioned payload-column
// format, the due-row decoder, and the legacy-shape probe. Split from store.go
// (file-size ratchet).

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"time"

	codec "github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// timerEnvelopeVersion is the payload-column format version. v1 wraps the
// payload so [scheduling.Timer.Actor] survives SQL persistence; v0 (legacy)
// rows hold the bare JSON of P and decode with an empty actor.
const timerEnvelopeVersion = 1

// timerEnvelope is the versioned payload-column format: the INNER value of
// the gcdc wrapper (ADR-0044) — {"$":"gcdc","enc":"json","dat":<this>} on
// every new write; v1-legacy rows hold this struct's bare JSON, v0-legacy
// rows hold the bare JSON of P, and both still decode (UnwrapDecode's
// fallback hands them through untouched):
//
//	{"v":1,"actor":"user:01JXYZ...","payload":<P>}
//
// Detection requires BOTH a "v":1 integer AND a "payload" key, so a legacy
// payload whose own JSON happens to carry a "v" field is not misread — only
// a legacy payload that is itself an object with exactly these two keys
// (v=1 + payload) would be, which is outside the contract.
type timerEnvelope[P any] struct {
	Version int    `json:"v"`
	Actor   string `json:"actor,omitzero"`
	Payload P      `json:"payload"`
}

// decodeDueTimer turns one due row into a Timer. Every failure branch here
// is Corruption in the stored data, not a reason to drop the whole batch.
func decodeDueTimer[P any](
	rawID string,
	payload []byte,
	fireAt time.Time,
) (scheduling.Timer[P], error) {
	envelope, err := decodeTimerPayload[P](rawID, payload)
	if err != nil {
		return scheduling.Timer[P]{}, err
	}

	timerID, err := scheduling.ParseTimerID(rawID)
	if err != nil {
		return scheduling.Timer[P]{}, errorfamily.WrapCorruption(
			err,
			"scheduling.sqlstore.parse_timer_id",
			"parse timer ID "+rawID,
		)
	}

	actor, err := id.ParseActorID(envelope.Actor)
	if err != nil {
		return scheduling.Timer[P]{}, errorfamily.WrapCorruption(
			err,
			"scheduling.sqlstore.parse_actor",
			"parse actor for timer "+rawID,
		)
	}

	return scheduling.Timer[P]{
		ID:      timerID,
		FireAt:  fireAt,
		Payload: envelope.Payload,
		Actor:   actor,
	}, nil
}

// decodeTimerPayload decodes a payload-column blob for the timer with the
// given ID (used for error context). v1 rows carry the full envelope (actor +
// payload); legacy rows hold the bare JSON of P and decode with an empty
// actor. A probe requiring BOTH a "v":1 integer AND a "payload" key
// distinguishes the shapes; non-object legacy payloads (strings, arrays) fail
// the probe and decode directly as P. Decode failures are classified as
// corruption — a row whose payload no longer matches P is a corrupt row.
func decodeTimerPayload[P any](id string, data []byte) (timerEnvelope[P], error) {
	// Unwrap the gcdc envelope when present (new writes); legacy rows come
	// back unchanged with the JSON fallback codec (decode-old-wrap-new).
	_, inner := codec.UnwrapDecode(data, codec.JSONCodec{})
	data = inner

	var probe struct {
		Version *int            `json:"v"`
		Payload *jsontext.Value `json:"payload"`
	}

	if err := json.Unmarshal(data, &probe); err == nil &&
		probe.Version != nil && *probe.Version == timerEnvelopeVersion && probe.Payload != nil {
		var env timerEnvelope[P]
		if err := json.Unmarshal(data, &env); err != nil {
			return timerEnvelope[P]{}, errorfamily.WrapCorruption(
				err,
				"scheduling.sqlstore.unmarshal_envelope",
				"unmarshal timer envelope for "+id,
			)
		}

		return env, nil
	}

	var p P

	if err := json.Unmarshal(data, &p); err != nil {
		return timerEnvelope[P]{}, errorfamily.WrapCorruption(
			err,
			"scheduling.sqlstore.unmarshal_legacy_payload",
			"unmarshal legacy timer payload for "+id,
		)
	}

	return timerEnvelope[P]{Version: 0, Actor: "", Payload: p}, nil
}
