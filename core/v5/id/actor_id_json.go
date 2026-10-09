package id

import (
	"encoding/json/v2"
	"fmt"
)

// MarshalJSON serializes ActorID as its prefixed string form ("kind:raw"),
// or null/empty if zero. This is self-describing: ParseActorID can recover
// the full ActorID from the JSON value.
func (a ActorID) MarshalJSON() ([]byte, error) {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	if a.IsZero() {
		return json.Marshal("")
	}

	return json.Marshal(a.PrefixedString())
}

// UnmarshalJSON deserializes ActorID from its prefixed string form.
// Accepts "kind:raw" or empty string (which yields the zero ActorID).
func (a *ActorID) UnmarshalJSON(data []byte) error {
	var s string

	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("ActorID: %w", err)
	}

	parsed, err := ParseActorID(s)
	if err != nil {
		return err
	}

	*a = parsed

	return nil
}
