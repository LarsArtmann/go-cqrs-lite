package metaengine

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"time"
)

// --- Cost Model Auto-Calibration ---

// calibration holds runtime-calibrated cost overrides for an engine.
// Zero values mean "use the engine's default" (backward compatible).
// Core engines embed this struct to support CalibrateEngine.
type calibration struct {
	nsPerOp    float64
	nsPerRead  float64
	nsPerWrite float64
}

func (c *calibration) setCalibration(op, read, write float64) {
	c.nsPerOp = op
	c.nsPerRead = read
	c.nsPerWrite = write
}

// applyTo overrides the profile's cost fields with calibrated values
// when they are non-zero. Zero values preserve the engine's defaults.
func (c *calibration) applyTo(p *EngineProfile) {
	if c.nsPerOp > 0 {
		p.NsPerOp = c.nsPerOp
	}

	if c.nsPerRead > 0 {
		p.NsPerRead = c.nsPerRead
	}

	if c.nsPerWrite > 0 {
		p.NsPerWrite = c.nsPerWrite
	}
}

// calibratable is an optional interface for engines that support runtime
// cost calibration. CalibrateEngine type-asserts to this interface to
// apply measured timings.
type calibratable interface {
	setCalibration(nsPerOp, nsPerRead, nsPerWrite float64)
}

// CalibrateEngine runs a micro-benchmark to measure the actual per-operation
// cost of an engine, overriding the hardcoded NsPerOp. Call after NewSQLiteEngine
// or NewMemoryEngine to get hardware-accurate cost estimates.
//
//	store, _ := Plan([]Engine{eng}, query)
//	metaengine.CalibrateEngine(eng, 1000)
func CalibrateEngine(eng Engine, iterations int) {
	if iterations <= 0 {
		iterations = 1000
	}

	if mb, ok := eng.(MapBackend); ok {
		ctx := context.Background()

		start := time.Now()

		for i := range iterations {
			_ = mb.MapSet(ctx, "__calibrate", i, i)
		}

		elapsed := time.Since(start)
		writeNs := float64(elapsed.Nanoseconds()) / float64(iterations)

		start = time.Now()

		for i := range iterations {
			_, _, _ = mb.MapGet(ctx, "__calibrate", i)
		}

		elapsed = time.Since(start)
		readNs := float64(elapsed.Nanoseconds()) / float64(iterations)

		// Cleanup
		for i := range iterations {
			_ = mb.MapDelete(ctx, "__calibrate", i)
		}

		if c, ok := eng.(calibratable); ok {
			c.setCalibration((writeNs+readNs)/2, readNs, writeNs)
		}
	}
}

// WithReadCoalescer enables concurrent read coalescing on the Store. When
// multiple goroutines read the same key simultaneously, only one actual
// engine read is performed; the result is shared with all waiters.
func WithReadCoalescer(store *Store, rc *ReadCoalescer) {
	store.coalescer = rc
}

// --- Schema Versioning for Layouts ---

// LayoutVersion tracks schema changes for auto-migration.
type LayoutVersion struct {
	Version int
	Columns []string // columns at this version
}

// MigrateLayout checks if a planned table needs schema migration (new columns
// added since the last plan). On SQLite, this performs ALTER TABLE ADD COLUMN
// for each new column. Returns nil if no migration is needed.
func (e *sqliteEngine) MigrateLayout(collection string, newPlan LayoutPlan) error {
	existing, ok := e.plans[collection]
	if !ok {
		return nil // no existing plan, nothing to migrate
	}

	if PlansColumnCompatible(existing, newPlan) {
		return nil // compatible, no migration needed
	}

	// Find new columns
	existingCols := make(map[string]bool, len(existing.Columns))
	for _, c := range existing.Columns {
		existingCols[c.Name] = true
	}

	ctx := context.Background()

	for _, newCol := range newPlan.Columns {
		if !existingCols[newCol.Name] {
			ddl := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
				QuoteIdent(newPlan.Table), QuoteIdent(newCol.Name), newCol.Type)
			if _, err := e.db.ExecContext(ctx, ddl); err != nil {
				return err //nolint:wrapcheck
			}
		}
	}

	// Update the stored plan
	e.plans[collection] = newPlan

	return nil
}

// --- Checksums ---

// Checksum computes an FNV-1a 64-bit hash of a value's JSON encoding.
// Used for silent-corruption detection: store alongside the value and verify
// on read.
func Checksum(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)

	return h.Sum64()
}

// VerifyChecksum checks that stored data matches its checksum.
func VerifyChecksum(data []byte, expected uint64) bool {
	return Checksum(data) == expected
}

// --- Read Coalescing (Singleflight) ---

// call represents an in-flight or completed read operation.
type readCall struct {
	wg    chan struct{}
	value any
	err   error
}

// ReadCoalescer coalesces concurrent identical reads into a single operation.
// When multiple goroutines request the same key simultaneously, only one
// actual read is performed; the result is shared with all waiters.
type ReadCoalescer struct {
	mu    sync.Mutex
	calls map[string]*readCall
}

// NewReadCoalescer creates a new read coalescer.
func NewReadCoalescer() *ReadCoalescer {
	return &ReadCoalescer{
		calls: make(map[string]*readCall),
	}
}

// Do executes fn if no call for the same key is in flight; otherwise waits
// for the in-flight call and returns its result. The key is an opaque string
// (typically "collection:key").
func (rc *ReadCoalescer) Do(key string, fn func() (any, error)) (any, error) {
	rc.mu.Lock()

	if existing, ok := rc.calls[key]; ok {
		rc.mu.Unlock()
		<-existing.wg

		return existing.value, existing.err
	}

	call := &readCall{wg: make(chan struct{})}
	rc.calls[key] = call
	rc.mu.Unlock()

	call.value, call.err = fn()
	close(call.wg)

	rc.mu.Lock()
	delete(rc.calls, key)
	rc.mu.Unlock()

	return call.value, call.err
}
