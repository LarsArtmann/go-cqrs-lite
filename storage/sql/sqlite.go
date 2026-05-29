package sql

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// ParseSQLiteTimestamp parses a SQLite timestamp string using multiple layouts.
func ParseSQLiteTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
	} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, event.WrapCorruption(
		ErrUnsupportedTimestamp,
		"storage.unsupported_timestamp",
		"unsupported timestamp format: "+s,
	)
}
