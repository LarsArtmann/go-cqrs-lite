package metaengine

import (
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
)

// String serializes the cursor into a URL-safe base64 string suitable for
// HTTP query parameters, headers, or API responses.
//
// The encoding is JSON wrapped in base64 (RawURLEncoding, no padding).
// An empty string represents a nil cursor (start of stream).
//
// String implements fmt.Stringer for debugging and best-effort encoding. A
// marshal failure is swallowed and returns "" — which ParseCursor interprets
// as "start of stream", silently resetting pagination. If you must not lose
// the cursor (e.g. persisting across requests), call [Cursor.Encode] instead
// and surface its error.
func (c Cursor) String() string {
	s, _ := c.Encode()
	return s
}

// Encode is the error-returning counterpart to [Cursor.String]. It returns the
// same URL-safe base64 string, plus any marshal error so the caller can refuse
// to silently reset pagination. Encode is the correct choice whenever the
// cursor crosses a process or request boundary.
func (c Cursor) Encode() (string, error) {
	if c.Value == nil {
		return "", nil
	}

	encoded, err := json.Marshal(c.Value)
	if err != nil {
		return "", fmt.Errorf("metaengine.Cursor.Encode: marshal cursor value: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

// ParseCursor decodes a cursor string produced by Cursor.String().
// Returns (nil, nil) for an empty string (no cursor — start of stream).
func ParseCursor(s string) (*Cursor, error) {
	if s == "" {
		return nil, nil //nolint:nilnil // empty cursor = start of stream (nil result, nil error); documented contract
	}

	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("metaengine.ParseCursor: invalid base64 encoding: %w", err)
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("metaengine.ParseCursor: invalid JSON payload: %w", err)
	}

	return &Cursor{Value: v}, nil
}
