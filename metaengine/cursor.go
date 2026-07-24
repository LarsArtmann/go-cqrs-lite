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
func (c Cursor) String() string {
	if c.Value == nil {
		return ""
	}

	encoded, err := json.Marshal(c.Value)
	if err != nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(encoded)
}

// ParseCursor decodes a cursor string produced by Cursor.String().
// Returns (nil, nil) for an empty string (no cursor — start of stream).
func ParseCursor(s string) (*Cursor, error) {
	if s == "" {
		return nil, nil
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
