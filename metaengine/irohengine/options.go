package irohengine

import (
	"strconv"
	"sync/atomic"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Option configures a replicated engine.
type Option func(*config)

type config struct {
	namespace string
	author    string
	transport Transport
}

const defaultAuthor = "default"

// WithNamespace sets the logical namespace for this node's writes.
// Maps to iroh-docs' NamespaceId in a real Iroh integration.
func WithNamespace(ns string) Option {
	return func(c *config) { c.namespace = ns }
}

// WithAuthor sets the author identity for this node.
// Each author signs its own writes; counter increments are tracked per-author
// for PN-Counter semantics. Maps to iroh-docs' AuthorId.
func WithAuthor(author string) Option {
	return func(c *config) { c.author = author }
}

// WithTransport sets the transport layer for cross-node replication.
// Use NewNetwork().Join(nodeID) to create a transport backed by the in-process
// P2P simulator. A custom Transport (e.g. real Iroh FFI) can be injected here.
func WithTransport(t Transport) Option {
	return func(c *config) { c.transport = t }
}

func defaultConfig() *config {
	return &config{
		author: defaultAuthor,
	}
}

var opSeq atomic.Uint64

func nextOpID() string {
	return time.Now().Format("20060102-150405.000000000") + "-" +
		strconv.FormatUint(opSeq.Add(1), 10)
}

// Replicated wraps a local engine with CRDT replication. The local engine
// handles all reads (full query power retained); CRDT-safe writes are applied
// locally AND published to the transport for cross-node convergence.
//
// ReplicationLag and NetworkRTT in the resulting EngineProfile are MEASURED
// from actual delivery traffic — never hardcoded. The transport records real
// delivery and convergence times; Profile() returns P99 convergence as lag and
// 2× P50 delivery as RTT.
//
// Supported CRDT-safe operations:
//   - MapSet (LWW-Map: latest timestamp wins)
//   - MapDelete (LWW tombstone)
//   - SetAdd (OR-Set: add-only)
//   - CounterIncrement (PN-Counter: per-author increments)
//   - MultiAdd (OR-Set per key)
//   - LogAppend (per-author append-only)
//
// Non-CRDT operations (MapUpdate, MapScan, Graph, Vector, Search, Spatial)
// execute locally and do NOT replicate. This matches the CALM theorem constraint:
// only monotonic operations converge without coordination.
func Replicated(local metaengine.Engine, opts ...Option) metaengine.Engine {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	eng := &replicatedEngine{
		local:      local,
		cfg:        cfg,
		timestamps: make(map[string]time.Time),
	}

	if cfg.transport != nil {
		if lp, ok := cfg.transport.(LatencyProvider); ok {
			eng.latency = lp
		}
		_ = cfg.transport.Subscribe(eng.applyRemote)
	}

	return eng
}
