package metaengine

import (
	"context"
	"fmt"
)

// CollectionStats holds per-collection row count and engine metadata.
type CollectionStats struct {
	Name       string
	EngineName string
	RowCount   int64
}

// Stats returns row counts and engine metadata for every collection. For
// engines that implement AggregateReader, COUNT(*) is pushed to SQL. For
// other engines (e.g. memory), a full scan is performed. Collections that
// cannot be counted are reported with RowCount = -1.
func (s *Store) Stats(ctx context.Context) ([]CollectionStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CollectionStats, 0, len(s.queries))

	for name, q := range s.queries {
		stats := CollectionStats{
			Name:       name,
			EngineName: q.engine.Profile().Name,
		}

		// Try AggregateReader (SQL COUNT pushdown).
		if ar, ok := q.engine.(AggregateReader); ok {
			n, err := ar.Aggregate(ctx, name, AggregateCount, "", nil)
			if err == nil {
				stats.RowCount = int64(n)
			} else {
				stats.RowCount = -1
			}
		} else if sb, ok := q.engine.(ScanBackend); ok {
			// Closure-based count via full scan.
			rows, err := sb.MapScan(ctx, name, nil, nil, nil, 0)
			if err == nil {
				stats.RowCount = int64(len(rows))
			} else {
				stats.RowCount = -1
			}
		} else {
			stats.RowCount = -1
		}

		result = append(result, stats)
	}

	return result, nil
}

// HealthCheck verifies that all engines are responsive. Engines that implement
// the HealthChecker interface are pinged; non-implementing engines are assumed
// healthy. Returns the first unhealthy engine's error, or nil if all are healthy.
func (s *Store) HealthCheck(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, eng := range s.engines {
		if hc, ok := eng.(HealthChecker); ok {
			if err := hc.HealthCheck(ctx); err != nil {
				return fmt.Errorf("metaengine: engine %s health check: %w", eng.Profile().Name, err)
			}
		}
	}

	return nil
}

// HealthChecker is an optional engine capability for liveness/readiness probes.
// Engines that interact with external systems (SQLite, Pebble) should implement
// this so consumers can wire Kubernetes-style health checks.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}
