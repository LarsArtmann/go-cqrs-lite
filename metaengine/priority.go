package metaengine

import (
	"context"
	"fmt"
)

// Priority represents an operator's optimization objective for layout planning
// (ADR-0124). The priority weights the cost model's scoring function, influencing
// which engine/layout the planner selects for each query.
//
// The developer expresses zero storage intent. The operator sets priorities to
// tune the deployment. See METAENGINE-LAYOUT-PLANNING-MODEL.md for the full
// design.
type Priority string

const (
	// PriorityWriteSpeed penalizes layouts that rewrite large rows on child
	// mutation. Favors normalized layouts (separate child collections with O(1)
	// inserts) over embedded layouts (read-modify-write on parent).
	PriorityWriteSpeed Priority = "WriteSpeed"

	// PriorityReadSpeed penalizes layouts requiring joins or secondary lookups.
	// Favors embedded/denormalized layouts (single read returns the whole
	// aggregate) over normalized layouts (multi-read joins).
	PriorityReadSpeed Priority = "ReadSpeed"

	// PriorityStorageSpace penalizes data duplication. Favors normalized layouts
	// (one copy of each fact) over embedded layouts (data duplicated across
	// projections).
	PriorityStorageSpace Priority = "StorageSpace"

	// PriorityBalanced applies even weighting across read, write, and storage
	// costs. This is the sane default when the operator has no specific
	// optimization objective.
	PriorityBalanced Priority = "Balanced"
)

// Valid reports whether p is a recognized priority value.
func (p Priority) Valid() bool {
	switch p {
	case PriorityWriteSpeed, PriorityReadSpeed, PriorityStorageSpace, PriorityBalanced:
		return true
	default:
		return false
	}
}

// PriorityWeights maps a priority to cost-type multipliers. Higher weight means
// the cost type is penalized more (the planner avoids expensive options for
// that cost type). The weighted cost is:
//
//	weightedCost = ReadW * readLatency + WriteW * writePenalty + StorageW * storagePenalty
//
// For the initial spike, only ReadW is applied (the cost model only estimates
// read latency). WriteW and StorageW are defined for future use when the cost
// model is extended.
type PriorityWeights struct {
	ReadW    float64
	WriteW   float64
	StorageW float64
}

// Weights returns the cost-type multipliers for this priority.
func (p Priority) Weights() PriorityWeights {
	switch p {
	case PriorityReadSpeed:
		return PriorityWeights{ReadW: 1.5, WriteW: 0.5, StorageW: 1.0}
	case PriorityWriteSpeed:
		return PriorityWeights{ReadW: 0.5, WriteW: 1.5, StorageW: 1.0}
	case PriorityStorageSpace:
		return PriorityWeights{ReadW: 0.8, WriteW: 0.8, StorageW: 2.5}
	default:
		return PriorityWeights{ReadW: 1.0, WriteW: 1.0, StorageW: 1.0}
	}
}

// PriorityConfig holds the operator's priority configuration at three levels
// of specificity (ADR-0124). Resolution order: per-Query (most specific) →
// per-Engine → Global (least specific). The most specific non-empty priority
// wins. Empty config resolves to PriorityBalanced everywhere.
type PriorityConfig struct {
	Global    Priority            `json:"global,omitempty"    yaml:"global,omitempty"`
	PerEngine map[string]Priority `json:"perEngine,omitempty" yaml:"perEngine,omitempty"`
	PerQuery  map[string]Priority `json:"perQuery,omitempty"  yaml:"perQuery,omitempty"`
}

// Resolve returns the effective priority for a given engine and query.
// Resolution order: per-Query → per-Engine → Global → Balanced (default).
func (pc *PriorityConfig) Resolve(engineName, queryName string) Priority {
	if pc == nil {
		return PriorityBalanced
	}

	if p, ok := pc.PerQuery[queryName]; ok && p.Valid() {
		return p
	}

	if p, ok := pc.PerEngine[engineName]; ok && p.Valid() {
		return p
	}

	if pc.Global.Valid() {
		return pc.Global
	}

	return PriorityBalanced
}

// priorityFactor returns the read-cost multiplier for a given priority and
// complexity. This is the spike implementation: it adjusts the cost estimate
// based on the priority, making the planner prefer different complexity classes.
//
// For ReadSpeed: O(1) is strongly preferred (factor < 1), O(N) is penalized.
// For WriteSpeed: O(N) is slightly preferred (simpler write paths).
// For StorageSpace: no complexity-based differentiation (future: storage model).
// For Balanced: no adjustment (factor = 1.0).
func priorityFactor(p Priority, c Complexity) float64 {
	w := p.Weights()

	switch c {
	case ComplexityO1:
		return w.ReadW * 0.8
	case ComplexityOLogN:
		return w.ReadW * 0.9
	case ComplexityON:
		return w.ReadW * 1.3
	case ComplexityONLogN:
		return w.ReadW * 1.5
	case ComplexityODegree:
		return w.ReadW * 2.0
	default:
		return w.ReadW
	}
}

// WithPriorityConfig sets the operator-driven layout priority configuration
// (ADR-0124). The priority weights the cost model's scoring function,
// influencing which engine/layout the planner selects. Resolution order:
// per-Query → per-Engine → Global → Balanced (default).
func WithPriorityConfig(pc *PriorityConfig) planOption {
	return func(c *planConfig) { c.priority = pc }
}

// SetPriority changes the operator's layout priority at runtime and triggers a
// re-plan so queries are re-scored under the new weights (ADR-0124 §5). This is
// the primary runtime API for adjusting layout decisions after Plan() returns.
//
// After SetPriority, call ReplanLayout to see which projections would change
// layout, then ConfirmRebuild to execute rebuilds that exceed the auto-rebuild
// threshold.
func (s *Store) SetPriority(ctx context.Context, pc *PriorityConfig) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("metaengine.Store.SetPriority: %w", err)
	}

	s.mu.Lock()
	s.priorityConfig = pc
	s.mu.Unlock()

	return s.Replan(ctx)
}

// resolvedPriority returns the effective priority for a given engine+query,
// reading from the stored priorityConfig. Returns PriorityBalanced when no
// config is set.
func (s *Store) resolvedPriority(engineName, queryName string) Priority {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.priorityConfig == nil {
		return PriorityBalanced
	}

	return s.priorityConfig.Resolve(engineName, queryName)
}
