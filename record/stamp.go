package record

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// Stamp is a timestamp whose presence is explicit: a zero time.Time can no
// longer masquerade as "recorded at epoch" (review P7). The zero Stamp means
// "not recorded"; NewStamp records a known time.
//
// The fields are unexported so an inconsistent state (a time without the
// known flag) cannot be constructed by accident — the only producers are
// NewStamp and JSON decoding of a marshaled Stamp.
type Stamp struct {
	at    time.Time
	known bool
}

// NewStamp records a known timestamp. The zero Stamp (no constructor call)
// means "unknown / not stamped".
func NewStamp(at time.Time) Stamp {
	return Stamp{at: at, known: true}
}

// IsZero reports whether no timestamp was recorded.
func (s Stamp) IsZero() bool { return !s.known }

// Time returns the recorded timestamp. It is the zero time.Time when IsZero
// is true — check IsZero before trusting the value.
func (s Stamp) Time() time.Time { return s.at }

// String renders the stamp for logs: the RFC 3339 timestamp when known,
// "unknown" when not recorded.
func (s Stamp) String() string {
	if !s.known {
		return unknownStr
	}

	return s.at.Format(time.RFC3339Nano)
}

// MarshalJSON encodes a known stamp as {"at":"<RFC3339Nano>"} and an unknown
// stamp as null, keeping the presence distinction lossless across JSON.
func (s Stamp) MarshalJSON() ([]byte, error) {
	if !s.known {
		return []byte("null"), nil
	}

	data, err := json.Marshal(stampWire{At: s.at})
	if err != nil {
		return nil, fmt.Errorf("record.Stamp: marshal: %w", err)
	}

	return data, nil
}

// UnmarshalJSON decodes both marshaled forms: null (and {}) yield the zero
// Stamp; {"at":...} yields a known stamp.
func (s *Stamp) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*s = Stamp{at: time.Time{}, known: false}

		return nil
	}

	var wire stampWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return fmt.Errorf("record.Stamp: %w", err)
	}

	if wire.At.IsZero() {
		*s = Stamp{at: time.Time{}, known: false}

		return nil
	}

	*s = NewStamp(wire.At)

	return nil
}

// stampWire is the JSON shape of a known Stamp.
type stampWire struct {
	At time.Time `json:"at"`
}
