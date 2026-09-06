package snapshot

import (
	"encoding/json"
	"time"

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
// CBOR is unaffected: canonical CBOR (fxamacker/cbor) keys structs by Go
// field name, so those wire bytes never carried the aggregate vocabulary.

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
	var wire snapshotWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	if wire.StreamID.IsZero() || wire.StreamType.IsZero() {
		var legacy snapshotWireLegacy
		if err := json.Unmarshal(data, &legacy); err == nil &&
			!legacy.StreamID.IsZero() && !legacy.StreamType.IsZero() {
			wire.StreamID, wire.StreamType = legacy.StreamID, legacy.StreamType
		}
	}

	*s = Snapshot{
		StreamID:   wire.StreamID,
		StreamType: wire.StreamType,
		Version:    wire.Version,
		State:      wire.State,
		Encoding:   wire.Encoding,
		CreatedAt:  wire.CreatedAt,
	}

	return nil
}
