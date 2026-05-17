package storage

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cockroachdb/pebble"
)

// unmarshalFromIter unmarshals the current iterator value into dest.
// Returns true if successful, false if unmarshal failed (logs warning and continues).
func unmarshalFromIter[T any](iter *pebble.Iterator, logger *slog.Logger, dest *T) bool {
	err := json.Unmarshal(iter.Value(), dest)
	if err != nil {
		logger.Warn(
			"failed to unmarshal, skipping",
			slog.String("key", string(iter.Key())),
			slog.String("error", err.Error()),
		)

		return false
	}

	return true
}

func newPrefixIter(db *pebble.DB, prefix string) (*pebble.Iterator, error) {
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: []byte(prefix + "\xff"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator for prefix %q: %w", prefix, err)
	}

	return iter, nil
}

// handleIteratorError checks an iterator for errors and returns an appropriate error.
func handleIteratorError(iter *pebble.Iterator, prefix string) error {
	err := iter.Error()
	if err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}

	return nil
}
