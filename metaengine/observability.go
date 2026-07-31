package metaengine

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// Hooks provides observability callbacks for the Store. All hooks are
// optional — nil hooks are no-ops. Set hooks via Store.WithHooks().
//
// Hooks are designed for zero-overhead when not configured: each hook
// is checked with a nil comparison before invocation, so the hot path
// pays only a single pointer check.
type Hooks struct {
	// OnFold is called after each fold operation with the collection name,
	// event type, fold kind, duration, and any error (nil on success).
	// Use for debug logging, error tracking, and metrics.
	OnFold func(collection, eventType string, kind FoldKind, d time.Duration, err error)

	// OnExecute is called after each Execute/ExecuteCtx call with the
	// collection name, read pattern, duration, and any error (nil on success).
	OnExecute func(collection string, pattern ReadPattern, d time.Duration, err error)

	// SlowQueryThreshold, when > 0, causes OnExecute to be invoked only
	// for queries exceeding this threshold. When 0, all queries invoke it.
	SlowQueryThreshold time.Duration

	// Logger receives debug log lines when debug mode is enabled.
	Logger *log.Logger
}

// StoreHooks is an alias for backward compatibility.
type StoreHooks = Hooks

// WithHooks configures observability hooks on the Store. Pass nil to clear.
//
//	metaengine.Store.WithHooks(store, metaengine.Hooks{
//	    SlowQueryThreshold: 50 * time.Millisecond,
//	    OnExecute: func(col string, pattern ReadPattern, d time.Duration) {
//	        log.Printf("slow query %s: %s took %v", col, pattern, d)
//	    },
//	    OnFold: func(col, evt string, kind FoldKind, d time.Duration) {
//	        log.Printf("[debug] %s: %s → %s (%v)", col, evt, kind, d)
//	    },
//	})
func WithHooks(store *Store, hooks Hooks) {
	store.hooks = &hooks
}

// WithDebug enables debug logging of every fold operation.
func WithDebug(store *Store, logger *log.Logger) {
	WithHooks(store, Hooks{
		Logger: logger,
		OnFold: func(collection, eventType string, kind FoldKind, d time.Duration, err error) {
			logger.Printf("[metaengine] %s: %s → %s (%v)", collection, eventType, kind, d)
		},
	})
}

// WithSlowQueryLog enables slow query detection. Queries exceeding the
// threshold are logged via the provided logger.
func WithSlowQueryLog(store *Store, threshold time.Duration, logger *log.Logger) {
	WithHooks(store, Hooks{
		SlowQueryThreshold: threshold,
		OnExecute: func(col string, pattern ReadPattern, d time.Duration, err error) {
			logger.Printf("[metaengine] slow query %s (%s): %v", col, pattern, d)
		},
	})
}

// --- MetricsRecorder for live metrics ---

// MetricsRecorder collects runtime metrics from the Store.
type MetricsRecorder interface {
	RecordApply(collection, eventType string, kind FoldKind, d time.Duration, err error)
	RecordExecute(collection string, pattern ReadPattern, d time.Duration, resultCount int, err error)
}

// WithMetrics wraps the Store's hooks to forward events to a MetricsRecorder.
func WithMetrics(store *Store, rec MetricsRecorder) {
	hooks := Hooks{
		OnFold: rec.RecordApply,
		OnExecute: func(col string, pattern ReadPattern, d time.Duration) {
			rec.RecordExecute(col, pattern, d, 0)
		},
	}
	if store.hooks != nil {
		hooks.Logger = store.hooks.Logger
		hooks.SlowQueryThreshold = store.hooks.SlowQueryThreshold
	}

	store.hooks = &hooks
}

// --- Plan visualization ---

// DotGraph generates a D2/Graphviz diagram from the PlanResult showing the
// event → fold → ADT → engine → complexity mapping. Returns a string
// suitable for piping to `d2` or any Graphviz renderer.
func (p *PlanResult) DotGraph() string {
	var b strings.Builder

	b.WriteString("digraph metaengine_plan {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=box];\n\n")

	for _, q := range p.Queries {
		// Engine node
		fmt.Fprintf(&b, "  %s [label=\"%s\\n(%s)\", shape=cylinder];\n",
			sanitizeDotID(q.QueryName), q.QueryName, q.EngineName)
		fmt.Fprintf(&b, "  %s_adt [label=\"%s\", shape=hexagon];\n",
			sanitizeDotID(q.QueryName), q.ADT)
		fmt.Fprintf(&b, "  %s -> %s_adt;\n",
			sanitizeDotID(q.QueryName), sanitizeDotID(q.QueryName))
	}

	for _, diag := range p.Diagnostics {
		if diag.Level == DiagLevelDegraded || diag.Level == DiagLevelWarn {
			fmt.Fprintf(&b, "  %s_warn [label=\"%s\", color=orange];\n",
				sanitizeDotID(diag.Query), diag.Message)
		}
	}

	b.WriteString("}\n")

	return b.String()
}

// sanitizeDotID makes a string safe for use as a DOT graph node identifier.
func sanitizeDotID(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}

		return '_'
	}, s)
}

// --- Cost accuracy reporter ---

// CostReport compares estimated latency against actual measured latency
// for each query in a PlanResult.
type CostReport struct {
	Query              string
	EstimatedLatencyNs float64 // ns
	ActualLatency      float64 // ns
	DriftPercent       float64 // (actual - estimated) / estimated * 100
}

// CostAccuracyReporter measures actual query latency and compares it
// against the planner's estimates. Use to detect cost model drift.
type CostAccuracyReporter struct {
	mu       sync.Mutex
	measured map[string][]time.Duration // query name → recent latencies
	maxLen   int
}

// NewCostAccuracyReporter creates a reporter that keeps the last N
// latency measurements per query for drift analysis.
func NewCostAccuracyReporter(maxPerQuery int) *CostAccuracyReporter {
	if maxPerQuery <= 0 {
		maxPerQuery = 100
	}

	return &CostAccuracyReporter{
		measured: make(map[string][]time.Duration),
		maxLen:   maxPerQuery,
	}
}

// Record stores a latency measurement for a query.
func (r *CostAccuracyReporter) Record(query string, d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.measured[query] = append(r.measured[query], d)
	if len(r.measured[query]) > r.maxLen {
		r.measured[query] = r.measured[query][len(r.measured[query])-r.maxLen:]
	}
}

// Report generates a cost accuracy report comparing measured latencies
// against the plan's estimates.
func (r *CostAccuracyReporter) Report(plan *PlanResult) []CostReport {
	r.mu.Lock()
	defer r.mu.Unlock()

	var reports []CostReport

	for _, q := range plan.Queries {
		measurements, ok := r.measured[q.QueryName]
		if !ok || len(measurements) == 0 {
			continue
		}

		var total time.Duration
		for _, d := range measurements {
			total += d
		}

		actualNs := float64(total.Nanoseconds()) / float64(len(measurements))
		estimatedNs := q.Cost.EstimatedLatencyMs * 1e6

		drift := 0.0
		if estimatedNs > 0 {
			drift = (actualNs - estimatedNs) / estimatedNs * 100
		}

		reports = append(reports, CostReport{
			Query:              q.QueryName,
			EstimatedLatencyNs: estimatedNs,
			ActualLatency:      actualNs,
			DriftPercent:       drift,
		})
	}

	return reports
}

// --- Tracing hook interface ---

// Tracer is a minimal tracing interface that avoids importing OTel.
// Implementations can bridge to OTel, Jaeger, or any tracing system.
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// Span represents a tracing span.
type Span interface {
	End()
	SetAttribute(key string, value any)
}

// WithTracing wraps Store hooks to create spans for Apply/Execute.
func WithTracing(store *Store, tracer Tracer) {
	// Tracing is implemented via the existing hook system.
	// OnFold creates a span for each fold; OnExecute creates a span for each query.
	hooks := Hooks{
		OnFold: func(collection, eventType string, kind FoldKind, d time.Duration) {
			ctx, span := tracer.StartSpan(
				context.Background(),
				"metaengine.fold."+collection,
			)
			span.SetAttribute("collection", collection)
			span.SetAttribute("event", eventType)
			span.SetAttribute("kind", string(kind))
			span.SetAttribute("duration_ms", d.Milliseconds())
			span.End()

			_ = ctx
		},
		OnExecute: func(collection string, pattern ReadPattern, d time.Duration) {
			_, span := tracer.StartSpan(
				context.Background(),
				"metaengine.execute."+collection,
			)
			span.SetAttribute("collection", collection)
			span.SetAttribute("pattern", string(pattern))
			span.SetAttribute("duration_ms", d.Milliseconds())
			span.End()
		},
	}
	store.hooks = &hooks
}
