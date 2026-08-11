package metaengine

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync/atomic"
	"time"
)

// Optional capability interfaces for engines that can measure live latency.
// Local engines (Memory, SQLite, Pebble, DuckDB) do not implement them and
// remain structurally local (NetworkRTT = 0). Remote engines (Postgres, Dgraph,
// future FDB) implement Prober so the runtime can replace the declared RTT
// prior with an honest measurement. Design: docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md.

// Prober measures the current round-trip to the engine's data (e.g. Postgres
// SELECT 1, Dgraph healthcheck, FDB read-version-only transaction). This is the
// honest NetworkRTT — a runtime observation, not a compile-time constant.
type Prober interface {
	Probe(ctx context.Context) (time.Duration, error)
}

// TransactMeasurer measures the round-trip of a real read I/O operation, used
// to build per-operation latency from live traffic instead of cold
// micro-benchmarks. Engines that cannot isolate a single read op need not
// implement this.
type TransactMeasurer interface {
	MeasureTransact(ctx context.Context) (time.Duration, error)
}

// TrackerHost is satisfied by *Calibration and thus by every engine that embeds
// Calibration. ProbeEngine installs live latency trackers through it so that
// Profile() reflects measured values without any per-engine code. Custom
// engines that do not embed Calibration may implement it themselves.
type TrackerHost interface {
	SetRTTTracker(t *LatencyTracker)
	SetReadTracker(t *LatencyTracker)
}

// SampleKind labels which latency axis a sample belongs to.
type SampleKind int

const (
	// SampleRTT is a network round-trip sample feeding NetworkRTT.
	SampleRTT SampleKind = iota
	// SampleRead is a per-read-operation latency sample feeding NsPerRead.
	SampleRead
)

// LatencySample is one measured latency observation pushed to a StatSink.
type LatencySample struct {
	Kind  SampleKind
	Value time.Duration
	At    time.Time
}

// StatSink is the open measurement ingress (ADR-0093 follow-up, P3). Any engine
// — including future external engines that cannot import the probe helper — can
// push live measurements by implementing a sink and recording into a tracker.
// The default no-op sink is returned by NopSink.
type StatSink interface {
	ReportSample(engineName string, s LatencySample)
}

// nopSink discards every sample.
type nopSink struct{}

func (nopSink) ReportSample(string, LatencySample) {}

// NopSink returns a StatSink that ignores all samples.
func NopSink() StatSink { return nopSink{} }

// ProbeOption configures the background probing loop.
type ProbeOption func(*probeConfig)

type probeConfig struct {
	interval      time.Duration
	timeout       time.Duration
	jitter        float64
	ctx           context.Context
	sink          StatSink
	name          string
	window        int
	alpha         float64
	stale         time.Duration
	errorHandler  func(error)
}

// WithProbeInterval sets the time between probes (default 1s).
func WithProbeInterval(d time.Duration) ProbeOption {
	return func(c *probeConfig) {
		if d > 0 {
			c.interval = d
		}
	}
}

// WithProbeTimeout sets the per-probe deadline (default 5s). A probe that times
// out is dropped (not recorded) so a hung connection cannot poison the EWMA
// with an inflated sample.
func WithProbeTimeout(d time.Duration) ProbeOption {
	return func(c *probeConfig) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// WithProbeJitter sets the fractional randomization of the probe interval
// (default 0.2 = +/-20%) to desynchronize probes across engines.
func WithProbeJitter(f float64) ProbeOption {
	return func(c *probeConfig) {
		if f >= 0 && f < 1 {
			c.jitter = f
		}
	}
}

// WithProbeContext binds the probing loop to a context (default
// context.Background). Cancelling it (or calling the returned stop function)
// stops probing.
func WithProbeContext(ctx context.Context) ProbeOption {
	return func(c *probeConfig) {
		if ctx != nil {
			c.ctx = ctx
		}
	}
}

// WithProbeSink wires a sink that receives every probe sample from this loop.
// The name labels samples so a single shared sink can demultiplex multiple engines.
func WithProbeSink(name string, sink StatSink) ProbeOption {
	return func(c *probeConfig) {
		c.sink = sink
		c.name = name
	}
}

// WithProbeWindow sets the sample window size for the latency trackers created
// by ProbeEngine (default 512). Larger windows smooth more but respond slower.
func WithProbeWindow(n int) ProbeOption {
	return func(c *probeConfig) {
		if n > 0 {
			c.window = n
		}
	}
}

// WithProbeAlpha sets the EWMA smoothing factor for the latency trackers created
// by ProbeEngine (default 0.1). Higher means more responsive to recent samples.
func WithProbeAlpha(a float64) ProbeOption {
	return func(c *probeConfig) {
		if a > 0 && a <= 1 {
			c.alpha = a
		}
	}
}

// WithProbeStale sets how long after the last sample the trackers created by
// ProbeEngine consider the measurement stale (default 30s). Zero disables
// staleness (always fresh once sampled).
func WithProbeStale(d time.Duration) ProbeOption {
	return func(c *probeConfig) {
		c.stale = d
	}
}

// WithProbeErrorHandler sets a callback invoked for every failed probe (network
// error, timeout, engine unreachable). When unset, failures are logged at Debug
// level via slog. Use this to wire custom observability (e.g., incrementing a
// Prometheus counter, sending to an alerting channel). The handler is called
// from the probe goroutine — it must not block.
func WithProbeErrorHandler(fn func(error)) ProbeOption {
	return func(c *probeConfig) {
		if fn != nil {
			c.errorHandler = fn
		}
	}
}

// ProbeHandle controls a background probe loop and exposes failure telemetry.
// Call Stop to halt probing and wait for the goroutine to exit. Use Failures to
// inspect how many probes failed (network errors, timeouts).
type ProbeHandle struct {
	stop     func()
	failures atomic.Int64
}

// Stop halts the probe loop and waits for the goroutine to exit. Safe to call
// multiple times.
func (h *ProbeHandle) Stop() {
	if h != nil && h.stop != nil {
		h.stop()
	}
}

// Failures returns the number of probes that returned an error since the loop
// started. Probes that succeed or are never attempted (local engine no-op) do
// not increment this counter.
func (h *ProbeHandle) Failures() int64 {
	if h == nil {
		return 0
	}
	return h.failures.Load()
}

// ProbeEngine starts a background loop that measures the live RTT (and, when the
// engine implements TransactMeasurer, per-read latency) of an engine and feeds
// it into Profile() through the engine's embedded Calibration. It returns a
// ProbeHandle whose Stop method halts the loop and waits for it to exit.
//
// For engines that implement neither Prober nor TransactMeasurer (all local
// engines), ProbeEngine is a no-op and returns a ProbeHandle whose Stop does
// nothing — calling it unconditionally is always safe.
//
// 	ph := metaengine.ProbeEngine(pg,
// 	    metaengine.WithProbeInterval(time.Second),
// 	)
// 	defer ph.Stop()
func ProbeEngine(eng Engine, opts ...ProbeOption) *ProbeHandle {
	c := probeConfig{
		interval: time.Second,
		timeout:  5 * time.Second,
		jitter:   0.2,
		ctx:      context.Background(),
		sink:     nil,
		window:   defaultTrackerWindow,
		alpha:    defaultTrackerAlpha,
		stale:    defaultStaleAfter,
	}
	for _, opt := range opts {
		opt(&c)
	}

	prober, hasProbe := eng.(Prober)
	measurer, hasMeasure := eng.(TransactMeasurer)
	if hasProbe && !eng.Profile().IsRemote() {
		hasProbe, prober = false, nil
	}
	if !hasProbe && !hasMeasure {
		return &ProbeHandle{}
	}

	var rtt, read *LatencyTracker
	if hasProbe {
		rtt = NewLatencyTracker(
			WithTrackerWindow(c.window),
			WithTrackerAlpha(c.alpha),
			WithStaleAfter(c.stale),
			WithTrackerSink(c.name, c.sink),
		)
	}
	if hasMeasure {
		read = NewLatencyTracker(
			WithTrackerWindow(c.window),
			WithTrackerAlpha(c.alpha),
			WithStaleAfter(c.stale),
		)
	}

	if host, ok := eng.(TrackerHost); ok {
		if rtt != nil {
			host.SetRTTTracker(rtt)
		}
		if read != nil {
			host.SetReadTracker(read)
		}
	}

	ctx, cancel := context.WithCancel(c.ctx)
	done := make(chan struct{})

	handle := &ProbeHandle{}

	go func() {
		defer close(done)
		runProbeLoop(ctx, c, prober, hasProbe, measurer, hasMeasure, rtt, read, &handle.failures)
	}()

	handle.stop = func() {
		cancel()
		<-done
	}

	return handle
}

func runProbeLoop(
	ctx context.Context,
	c probeConfig,
	prober Prober, hasProbe bool,
	measurer TransactMeasurer, hasMeasure bool,
	rtt, read *LatencyTracker,
	failures *atomic.Int64,
) {
	handleError := func(stage string, err error) {
		failures.Add(1)
		if c.errorHandler != nil {
			c.errorHandler(fmt.Errorf("probe %s: %w", stage, err))
			return
		}
		slog.Debug("metaengine: probe failed",
			"stage", stage, "engine", c.name, "err", err)
	}

	probeOnce := func() {
		pctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()

		if hasProbe {
			if d, err := prober.Probe(pctx); err == nil {
				rtt.Record(d)
			} else {
				handleError("rtt", err)
			}
		}
		if hasMeasure {
			if d, err := measurer.MeasureTransact(pctx); err == nil {
				read.Record(d)
			} else {
				handleError("transact", err)
			}
		}
	}

	probeOnce()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(jitteredInterval(c.interval, c.jitter)):
			probeOnce()
		}
	}
}

func jitteredInterval(base time.Duration, jitter float64) time.Duration {
	if jitter <= 0 {
		return base
	}

	factor := 1 + jitter*(2*rand.Float64()-1)

	return time.Duration(float64(base) * factor)
}
