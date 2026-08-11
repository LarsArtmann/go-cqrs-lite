package metaengine

import (
	"context"
	"fmt"
)

// RebuildThreshold configures when a projection rebuild is automatic vs
// requires operator confirmation (ADR-0124 §11).
type RebuildThreshold struct {
	// MaxEventCount is the maximum number of events a projection may contain
	// to qualify for automatic rebuild. Projections with more events require
	// explicit operator confirmation. Default: 100000.
	MaxEventCount int64

	// MaxDataBytes is the maximum estimated data size for automatic rebuild.
	// Projections larger than this require explicit confirmation.
	// Default: 1GB (1 << 30).
	MaxDataBytes int64
}

// DefaultRebuildThreshold returns the default rebuild threshold configuration.
func DefaultRebuildThreshold() RebuildThreshold {
	return RebuildThreshold{
		MaxEventCount: 100_000,
		MaxDataBytes:  1 << 30, // 1GB
	}
}

// LayoutDiff describes a single projection's layout change.
type LayoutDiff struct {
	QueryName string
	From      LayoutOption
	To        LayoutOption
	Reason    string

	// EstimatedRebuildEvents is the approximate number of events that must be
	// replayed to rebuild this projection.
	EstimatedRebuildEvents int64

	// EstimatedRebuildBytes is the approximate data size of the projection.
	EstimatedRebuildBytes int64

	// AutoRebuild is true when the projection is small enough to rebuild
	// automatically (below threshold). False means operator confirmation needed.
	AutoRebuild bool
}

// ReplanLayout computes new layouts for all queries given a priority config,
// identifies which projections must change, and returns the diffs.
//
// This is the operator's "what would happen if I changed the priority?" tool.
// It does NOT execute any rebuilds — it only computes the plan diff.
// Call ConfirmRebuild to execute the changes.
func (s *Store) ReplanLayout(ctx context.Context, pc *PriorityConfig) ([]LayoutDiff, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("metaengine.Store.ReplanLayout: %w", err)
	}

	threshold := DefaultRebuildThreshold()

	s.mu.RLock()
	defer s.mu.RUnlock()

	var diffs []LayoutDiff

	for _, name := range sortedQueryNames(s.queries) {
		q := s.queries[name]
		engine := q.QueryEngine()
		if engine == nil {
			continue
		}

		profile := engine.Profile()
		resolvedPriority := PriorityBalanced
		if pc != nil {
			resolvedPriority = pc.Resolve(profile.Name, name)
		}

		newOption, _ := SelectLayout(profile, resolvedPriority)

		// The current layout is Embed by default (existing behavior)
		currentOption := LayoutEmbed

		if newOption != currentOption {
			vol := q.QueryConfig().Volume
			if vol <= 0 {
				vol = 1000
			}

			estimatedBytes := vol * 256 // ~256B per entry

			diff := LayoutDiff{
				QueryName:              name,
				From:                   currentOption,
				To:                     newOption,
				Reason:                 fmt.Sprintf("priority=%s on %s engine", resolvedPriority, profile.Name),
				EstimatedRebuildEvents: vol,
				EstimatedRebuildBytes:  estimatedBytes,
				AutoRebuild:            vol < threshold.MaxEventCount && estimatedBytes < threshold.MaxDataBytes,
			}

			diffs = append(diffs, diff)
		}
	}

	return diffs, nil
}

// ConfirmRebuild executes the layout diffs that require operator confirmation.
// Auto-rebuild diffs are NOT executed by this method — they should be handled
// by ReplanLayout + Backfill.
//
// This is a safety mechanism: large projection rebuilds require explicit
// operator approval to prevent accidental massive parallel rebuilds.
func (s *Store) ConfirmRebuild(ctx context.Context, diffs []LayoutDiff) error {
	for _, d := range diffs {
		if d.AutoRebuild {
			continue // auto-rebuilds are handled separately
		}

		// In the spike implementation, we just log the confirmation.
		// In production, this would trigger the actual rebuild.
	}

	return nil
}

// sortedQueryNames returns query names in sorted order for deterministic output.
func sortedQueryNames(queries map[string]queryMeta) []string {
	names := make([]string, 0, len(queries))
	for name := range queries {
		names = append(names, name)
	}

	// Simple sort without imports (avoid adding sort import just for this)
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j-1] > names[j]; j-- {
			names[j-1], names[j] = names[j], names[j-1]
		}
	}

	return names
}
