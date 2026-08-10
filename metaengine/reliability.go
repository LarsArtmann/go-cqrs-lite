package metaengine

import (
	"context"
	"hash/fnv"
	"sync"
	"time"
)

// --- Cost Model Auto-Calibration ---

// CalibrationCosts holds measured per-operation costs from a calibration run.
// Zero values mean "no override" — the engine's compile-time defaults are used.
// CalibrateEngine builds this from micro-benchmarks; consumers can also
// construct it manually from their own measurements and pass it to
// SetCalibration.
//
// NetworkRTT is the declared PRIOR for a remote engine (same-datacenter
// estimate). It seeds planning before the first live probe; once a live tracker
// has fresh samples, ApplyCalibration replaces it with the measured EWMA. A
// prior is never a fact — it is a fallback, explicitly labelled "prior" or
// "stale" by GetEngineStats when no live measurement backs it.
type CalibrationCosts struct {
	NsPerOp    float64
	NsPerRead  float64
	NsPerWrite float64
	ReadCosts  ReadCosts
	NetworkRTT time.Duration
}

// Calibration holds runtime-calibrated cost overrides for an engine.
// Zero values mean "use the engine's default" (backward compatible).
// Core engines (memoryEngine, sqliteEngine) embed this struct to support
// CalibrateEngine. External engine packages (duckdbengine, pebbleengine,
// pgengine) embed it the same way.
//
// Calibration also hosts optional live latency trackers (see
// METAENGINE-LIVE-LATENCY-MODEL.md). When ProbeEngine installs a tracker, every
// Profile() call — including the planner's plan-time read — reflects the latest
// measured RTT and per-read latency instead of a frozen constant. This is why
// the embedded value type works: engines embed Calibration by value, and the
// pointer-receiver tracker methods are promoted to *Engine, so ProbeEngine can
// wire live measurement into Profile() with zero per-engine code.
type Calibration struct {
	nsPerOp    float64
	nsPerRead  float64
	nsPerWrite float64
	readCosts  ReadCosts
	networkRTT time.Duration // declared prior, replaced by live tracker when fresh

	// Live trackers (optional). Set by ProbeEngine via SetRTTTracker /
	// SetReadTracker. nil = no live measurement; the prior/compile-time values
	// stand and are labelled "prior" by GetEngineStats.
	rtt  *LatencyTracker
	read *LatencyTracker
}

// SetCalibration stores measured cost values. CalibrateEngine calls this
// via the Calibratable interface.
func (c *Calibration) SetCalibration(costs CalibrationCosts) {
	c.nsPerOp = costs.NsPerOp
	c.nsPerRead = costs.NsPerRead
	c.nsPerWrite = costs.NsPerWrite
	c.readCosts = costs.ReadCosts
	c.networkRTT = costs.NetworkRTT
}

// SetRTTTracker installs a live RTT tracker. ProbeEngine calls this on engines
// that embed Calibration. Once installed, ApplyCalibration replaces the declared
// NetworkRTT prior with the tracker's EWMA whenever fresh samples exist.
func (c *Calibration) SetRTTTracker(t *LatencyTracker) { c.rtt = t }

// SetReadTracker installs a live per-read-operation latency tracker. When fresh,
// ApplyCalibration uses its EWMA (in nanoseconds) as NsPerRead.
func (c *Calibration) SetReadTracker(t *LatencyTracker) { c.read = t }

// LiveLatency reports the engine's live measurement state for diagnostics
// (GetEngineStats, Doctor, EXPLAIN). HasRTT/HasRead indicate whether a tracker
// is installed; Fresh indicates whether the RTT tracker specifically has samples
// current enough to drive routing. The WARN rule (rule_live_latency) uses this
// to decide whether estimates rely on a measured value or a prior — RTT is the
// primary routing signal, so freshness is RTT-specific. Read-tracker freshness
// is consumed internally by ApplyCalibration and does not affect the WARN.
type LiveLatency struct {
	RTT     LatencyStats
	Read    LatencyStats
	HasRTT  bool
	HasRead bool
	Fresh   bool
}

// LiveLatency returns the live measurement snapshot for this engine. It is
// promoted to every engine that embeds Calibration, so GetEngineStats can read
// it through a type assertion without per-engine code.
//
// Fresh is RTT-specific: true only when the RTT tracker exists and has current
// samples. A stale or missing RTT tracker yields Fresh=false even if the read
// tracker is fresh — this prevents a read-only tracker from suppressing the
// "routing on prior RTT" WARN.
func (c *Calibration) LiveLatency() LiveLatency {
	out := LiveLatency{}
	if c.rtt != nil {
		out.HasRTT = true
		stats, fresh := c.rtt.Live()
		out.RTT = stats
		out.Fresh = fresh
	}
	if c.read != nil {
		out.HasRead = true
		stats, _ := c.read.Live()
		out.Read = stats
	}

	return out
}

// ApplyCalibration overrides the profile's cost fields with calibrated values
// when they are non-zero, then layers live measurements on top when they are
// fresh. The precedence is:
//
//  1. Engine compile-time defaults (set by the engine before calling this).
//  2. SetCalibration priors (cold micro-benchmark / operator override).
//  3. Live tracker EWMA (runtime observation) — the most honest value, used
//     only when fresh; otherwise the prior stands and is labelled stale.
//
// Engines call this inside their Profile() method.
func (c *Calibration) ApplyCalibration(p *EngineProfile) {
	if c.nsPerOp > 0 {
		p.NsPerOp = c.nsPerOp
	}

	if c.nsPerRead > 0 {
		p.NsPerRead = c.nsPerRead
	}

	if c.nsPerWrite > 0 {
		p.NsPerWrite = c.nsPerWrite
	}

	if c.readCosts.NsPerPointLookup > 0 {
		p.ReadCosts.NsPerPointLookup = c.readCosts.NsPerPointLookup
	}

	if c.readCosts.NsPerFilteredScan > 0 {
		p.ReadCosts.NsPerFilteredScan = c.readCosts.NsPerFilteredScan
	}

	if c.readCosts.NsPerAggregate > 0 {
		p.ReadCosts.NsPerAggregate = c.readCosts.NsPerAggregate
	}

	if c.readCosts.NsPerScan > 0 {
		p.ReadCosts.NsPerScan = c.readCosts.NsPerScan
	}

	// Declared RTT prior (operator/calibration override of the compile-time value).
	if c.networkRTT > 0 {
		p.NetworkRTT = c.networkRTT
	}

	// Live RTT measurement replaces the prior when fresh. This is the whole
	// point of the live-latency model: Profile() returns the current network
	// distance, not a compile-time guess.
	if c.rtt != nil {
		if stats, fresh := c.rtt.Live(); fresh {
			p.NetworkRTT = stats.EWMA
		}
	}

	// Live per-read latency replaces the calibrated NsPerRead when fresh.
	if c.read != nil {
		if stats, fresh := c.read.Live(); fresh {
			p.NsPerRead = float64(stats.EWMA.Nanoseconds())
		}
	}
}

// Calibratable is an optional interface for engines that support runtime
// cost calibration. CalibrateEngine type-asserts to this interface to
// apply measured timings. External engine packages embed Calibration to
// implement this interface without writing boilerplate.
type Calibratable interface {
	SetCalibration(costs CalibrationCosts)
}

// CalibrateEngine runs a micro-benchmark to measure the actual per-operation
// cost of an engine, overriding the hardcoded cost constants. Call after
// constructing any engine that implements Calibratable (memory, SQLite, DuckDB,
// Pebble, Postgres) to get hardware-accurate cost estimates.
//
//	store, _ := Plan([]Engine{eng}, query)
//	metaengine.CalibrateEngine(eng, 1000)
//
// For engines with per-read-pattern costs (e.g. DuckDB's 4000x span between
// point lookups and vectorized scans), run the engine-specific calibration
// benchmarks and construct CalibrationCosts manually with ReadCosts fields:
//
//	metaengine.CalibrateEngine(engine, 1000) // sets NsPerOp/Read/Write
//	// Then override ReadCosts from calibration_bench_test results:
//	if c, ok := engine.(metaengine.Calibratable); ok {
//	    c.SetCalibration(metaengine.CalibrationCosts{
//	        ReadCosts: metaengine.ReadCosts{
//	            NsPerPointLookup: 50_000,
//	            NsPerFilteredScan: 450,
//	            NsPerAggregate:   150,
//	            NsPerScan:        1_000,
//	        },
//	    })
//	}
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

		if c, ok := eng.(Calibratable); ok {
			c.SetCalibration(CalibrationCosts{
				NsPerOp:    (writeNs + readNs) / 2,
				NsPerRead:  readNs,
				NsPerWrite: writeNs,
			})
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
