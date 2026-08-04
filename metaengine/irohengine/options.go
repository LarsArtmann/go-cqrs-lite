package irohengine

import (
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Option configures a replicated engine.
type Option func(*config)

type config struct {
	namespace      string
	author         string
	transport      Transport
	replicationLag time.Duration
	networkRTT     time.Duration
}

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

// WithReplicationLag sets the expected replication lag for the engine profile.
// Diagnostic-only; does not affect latency estimation. Defaults to 100ms.
func WithReplicationLag(d time.Duration) Option {
	return func(c *config) { c.replicationLag = d }
}

// WithNetworkRTT sets the round-trip time for the engine profile's cost model.
// Additive latency: total = compute + NetworkRTT. Defaults to 50ms to model
// a realistic P2P relay scenario.
func WithNetworkRTT(d time.Duration) Option {
	return func(c *config) { c.networkRTT = d }
}

const (
	defaultReplicationLag = 100 * time.Millisecond
	defaultNetworkRTT     = 50 * time.Millisecond
	defaultAuthor         = "default"
)

func defaultConfig() *config {
	return &config{
		author:         defaultAuthor,
		replicationLag: defaultReplicationLag,
		networkRTT:     defaultNetworkRTT,
	}
}

// Replicated wraps a local engine with CRDT replication. The local engine
// handles all reads (full query power retained); CRDT-safe writes are applied
// locally AND published to the transport for cross-node convergence.
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
		local: local,
		cfg:   cfg,
	}

	if cfg.transport != nil {
		_ = cfg.transport.Subscribe(eng.applyRemote)
	}

	return eng
}
