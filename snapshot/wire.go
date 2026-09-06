package snapshot

import (
	"encoding/json"
	"time"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// The wire shims below implement the v5 snapshot wire-tag rename: writers
// emit only the honest stream_id/stream_type keys, readers additionally
// accept the pre-v5 aggregateId/aggregateType spellings (decode-only legacy
// fallback, scheduled for deletion at v6). See
// docs/planning/v5-deprecation-sweep.md §4 and the 2026-08-22 data-model
// review (T18/P10).
//
// The fallback covers CBOR as well as JSON: fxamacker/cbor v2.9 falls back
// to the json tag key when no cbor key is present, so the rename moves the
// CBOR map keys too (pinned by TestWire_CBORCarriesNewKeys). Readers of
// pre-v5 bytes decode both spellings via the legacy shadows below.

type snapshotWire struct {
	StreamID   id.StreamID     `json:"stream_id"`
	StreamType id.StreamType   `json:"stream_type"`
	Version    event.Version   `json:"version"`
	State      []byte          `json:"state"`
	Encoding   record.Encoding `json:"encoding,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// snapshotWireLegacy mirrors snapshotWire with the pre-v5 JSON keys.
type snapshotWireLegacy struct {
	StreamID   id.StreamID     `json:"aggregateId"`
	StreamType id.StreamType   `json:"aggregateType"`
	Version    event.Version   `json:"version"`
	State      []byte          `json:"state"`
	Encoding   record.Encoding `json:"encoding,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// UnmarshalJSON decodes a Snapshot from JSON written by any v4.x or v5+
// writer: stream_id/stream_type keys take precedence; aggregateId/
// aggregateType keys are accepted when the stream identity is absent. No
// invariant enforcement happens here — check the decoded value with
// [Snapshot.Validate].
func (s *Snapshot) UnmarshalJSON(data []byte) error {
	return decodeSnapshotWire(data, json.Unmarshal, s)
}

// UnmarshalCBOR is the CBOR twin of [Snapshot.UnmarshalJSON]: fxamacker/cbor
// routes struct decoding through it and applies the same new-keys-first,
// legacy-fallback resolution.
func (s *Snapshot) UnmarshalCBOR(data []byte) error {
	return decodeSnapshotWire(data, codec.CBORDecMode().Unmarshal, s)
}

// decodeSnapshotWire fills dst from wire bytes using the given unmarshaler,
// resolving the pre-v5 aggregate spelling when the stream identity is
// missing from the new keys.
func decodeSnapshotWire(
	data []byte,
	unmarshal func([]byte, any) error,
	dst *Snapshot,
) error {
	var wire snapshotWire
	if err := unmarshal(data, &wire); err != nil {
		return err
	}

	if wire.StreamID.IsZero() || wire.StreamType.IsZero() {
		var legacy snapshotWireLegacy
		if err := unmarshal(data, &legacy); err == nil &&
			!legacy.StreamID.IsZero() && !legacy.StreamType.IsZero() {
			wire.StreamID, wire.StreamType = legacy.StreamID, legacy.StreamType
		}
	}

	*dst = Snapshot{
		StreamID:   wire.StreamID,
		StreamType: wire.StreamType,
		Version:    wire.Version,
		State:      wire.State,
		Encoding:   wire.Encoding,
		CreatedAt:  wire.CreatedAt,
	}

	return nil
}
