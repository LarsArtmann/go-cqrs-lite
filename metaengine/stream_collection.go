package metaengine

import (
	"context"
	"errors"
	"fmt"
)

// errCollectionNotFound is returned by StreamCollection when no query owns
// the requested collection name.
var errCollectionNotFound = errors.New("metaengine.StreamCollection: collection not found")

// errNoScanBackend is returned by StreamCollection when the collection's
// engine implements neither StreamingScan nor ScanBackend.
var errNoScanBackend = errors.New(
	"metaengine.StreamCollection: engine supports neither streaming nor scan",
)

// StreamCollection iterates every row in a collection, calling fn once per
// row. It prefers the assigned engine's StreamingScan capability (O(1)
// memory per row for unsorted scans) and falls back to ScanBackend.MapScan,
// which materializes the rows first.
//
// The collection's engine reference is resolved under a brief read lock; the
// iteration itself runs without holding the store lock, so concurrent writes
// may be observed mid-stream (no snapshot isolation) and fn may safely call
// back into the Store.
//
// Iteration stops at the first error: engine errors are wrapped with the
// collection name, fn errors are returned unchanged.
func (s *Store) StreamCollection(
	ctx context.Context,
	collection string,
	fn func(row any) error,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("metaengine.StreamCollection: %w", err)
	}

	eng, ok := s.collectionEngine(collection)
	if !ok {
		return fmt.Errorf("%w: %q", errCollectionNotFound, collection)
	}

	s.meter.IncRead()

	if ss, ok := eng.(StreamingScan); ok {
		for row, err := range ss.StreamScan(ctx, collection, nil, nil) {
			if err != nil {
				return fmt.Errorf("stream %s: %w", collection, err)
			}

			if err := fn(row); err != nil {
				return err
			}
		}

		return nil
	}

	sb, ok := eng.(ScanBackend)
	if !ok {
		return fmt.Errorf("%w: %q", errNoScanBackend, collection)
	}

	result, err := sb.MapScan(ctx, collection, nil, nil, nil, 0)
	if err != nil {
		return fmt.Errorf("stream %s: %w", collection, err)
	}

	for _, row := range result.Items {
		if err := fn(row); err != nil {
			return err
		}
	}

	return nil
}
