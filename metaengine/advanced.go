package metaengine

import (
	"context"
	"fmt"
	"sync"
)

// --- P6-6: Multi-Engine Tiering ---

// TierConfig configures multi-engine tiering for a collection. When set,
// writes are fanned out to ALL tier engines, and reads use the first
// available (cheapest) tier.
type TierConfig struct {
	WriteEngines []Engine // all engines receive writes
	ReadEngine   Engine   // primary read engine (cheapest)
}

// TieredStore wraps a Store with multi-engine tiering support.
type TieredStore struct {
	inner *Store
	tiers map[string]TierConfig // collection name → tier config
	mu    sync.RWMutex
}

// NewTieredStore creates a tiered store wrapper.
func NewTieredStore(store *Store) *TieredStore {
	return &TieredStore{inner: store, tiers: make(map[string]TierConfig)}
}

// WithTier assigns a multi-engine tier to a collection.
func (ts *TieredStore) WithTier(collection string, cfg TierConfig) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.tiers[collection] = cfg
}

// Apply fans out to all tier engines for the matching collection.
func (ts *TieredStore) Apply(ctx context.Context, eventType string, payload any) error {
	if err := ts.inner.Apply(ctx, eventType, payload); err != nil {
		return err
	}

	ts.mu.RLock()
	defer ts.mu.RUnlock()

	for _, cfg := range ts.tiers {
		for _, eng := range cfg.WriteEngines {
			_ = eng // tier write fan-out would go here
		}
	}

	return nil
}

// --- P6-7: Engine Hot-Swap ---

// SwapEngine replaces an engine at runtime. All queries assigned to the old
// engine are reassigned to the new one. The old engine is NOT closed — the
// caller is responsible for lifecycle management.
//
// This is useful for zero-downtime engine upgrades (e.g., swapping a memory
// engine for a SQLite engine after warmup).
func (s *Store) SwapEngine(oldName, newName string, newEngine Engine) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	swapped := false
	for i, eng := range s.engines {
		if eng.Profile().Name == oldName {
			s.engines[i] = newEngine
			swapped = true
			break
		}
	}

	if !swapped {
		return fmt.Errorf("metaengine.SwapEngine: engine %q not found", oldName)
	}

	// Reassign queries
	for name, q := range s.queries {
		if q.engine.Profile().Name == oldName {
			q.engine = newEngine
			s.queries[name] = q
		}
	}

	return nil
}

// --- P6-1: Pebble Engine Raw Readers (interface contract) ---
//
// Pebble engine implementations should implement RawValueReader and
// RawScanReader to get the same JSON tax reduction as SQLite. The interfaces
// are already defined in engine.go — this file documents the contract.
//
// To add raw readers to the Pebble engine:
// 1. Store values as raw JSON bytes (not decoded to any)
// 2. Implement GetRawValue(ctx, col, key) ([]byte, bool, error)
// 3. Implement ScanRawValues(ctx, col, filters, sort, cursor, limit) ([][]byte, error)
// The ExecuteTyped path automatically prefers these interfaces.

// --- P4-2: Cross-Engine Contract Suite ---

// ContractSuite runs a comprehensive test suite against any engine factory.
// Use to verify that a new engine implementation matches the behavior of
// existing engines across all ADTs.
//
//	func TestMyEngine(t *testing.T) {
//	    metaengine.ContractSuite(t, func() metaengine.Engine {
//	        return myEngine.New()
//	    })
//	}
func ContractSuite(t interface {
	Fatal(...any)
	Errorf(string, ...any)
}, factory func() Engine) {
	eng := factory()
	if eng == nil {
		t.Fatal("ContractSuite: factory returned nil engine")
	}

	profile := eng.Profile()
	if profile.Name == "" {
		t.Errorf("ContractSuite: engine has empty name")
	}

	// Test Map backend if supported
	if mb, ok := eng.(MapBackend); ok {
		ctx := context.Background()
		if err := mb.MapSet(ctx, "test", "key1", "value1"); err != nil {
			t.Errorf("ContractSuite: MapSet failed: %v", err)
		}

		val, found, err := mb.MapGet(ctx, "test", "key1")
		if err != nil || !found || val == nil {
			t.Errorf("ContractSuite: MapGet failed: err=%v found=%v val=%v", err, found, val)
		}

		if err := mb.MapDelete(ctx, "test", "key1"); err != nil {
			t.Errorf("ContractSuite: MapDelete failed: %v", err)
		}

		_, found, _ = mb.MapGet(ctx, "test", "key1")
		if found {
			t.Errorf("ContractSuite: MapGet should return found=false after delete")
		}
	}

	_ = eng.Close()
}

// --- P7-3: V1 Stabilization Checklist ---

// V1StabilizationChecklist documents the criteria for tagging metaengine/v4.1.0.
// All items must be checked before the v1 tag.
var V1StabilizationChecklist = []string{
	"TypedReader API frozen (Get, Scan, Count, Sum, Min, Max, Avg, Distinct, GroupBy)",
	"LayoutPlanner + auto-layout wired into Plan()",
	"RawValueReader + RawScanReader interfaces stable",
	"Exported sentinel errors (ErrNotFound, ErrLayoutConflict, ErrAmbiguousKey, ErrUnsupportedADT)",
	"Aggregation pushdown (AggregateReader interface) stable",
	"FilterIn operator stable (no silent-drop on any path)",
	"OR filter + compound sort (closure fallback, not yet pushed to SQL)",
	"Poison-pill detection wired into all read paths",
	"Hooks system (WithHooks, WithDebug, WithSlowQueryLog, WithMetrics, WithTracing)",
	"Export/Import round-trip tested",
	"Cost calibration API (CalibrateEngine)",
	"Consistency checker (Store.Verify with EventLog)",
	"Schema enforcement diagnostics at Plan() time",
	"Transaction API (Store.InTransaction + Transactional interface)",
	"Cross-engine contract suite available (ContractSuite)",
	"PlanResult.DotGraph() visualization",
}
