package quic

import (
	iroh_ffi "git.coopcloud.tech/decentral1se/iroh-go"
)

// Option configures a QuicTransport.
type Option func(*config)

// DefaultDedupCapacity is the op-dedup ring capacity used by QuicTransport
// (10K recently-seen op IDs). The dedup regression test
// TestRing_ProductionCapacity10K pins ring behavior at exactly this capacity
// — if you change it here, update that test's capacity in the same commit.
const DefaultDedupCapacity = 10_000

type config struct {
	alpn        []byte
	presetFn    func() iroh_ffi.Preset
	bindAddr    string
	relay       bool
	poolStreams bool
}

func defaultConfig() *config {
	return &config{
		alpn:     DefaultALPN,
		presetFn: iroh_ffi.PresetN0,
		bindAddr: "0.0.0.0:0",
		relay:    false,
	}
}

// WithALPN sets the Application-Layer Protocol Negotiation bytes.
// All nodes in the same cluster must use the same ALPN.
// Defaults to DefaultALPN.
func WithALPN(alpn []byte) Option {
	return func(c *config) { c.alpn = alpn }
}

// WithLocalOnly configures the endpoint for localhost-only operation
// (no relay servers, 127.0.0.1 bind). Ideal for tests and local demos.
func WithLocalOnly() Option {
	return func(c *config) {
		c.presetFn = iroh_ffi.PresetN0DisableRelay
		c.bindAddr = "127.0.0.1:0"
	}
}

// WithRelay enables star-topology relay mode. When enabled, the transport
// forwards received ops to all OTHER connected peers (excluding the sender).
// This allows a coordinator node to relay ops between nodes that are not
// directly connected. Uses a dedup set to prevent echo loops.
func WithRelay() Option {
	return func(c *config) { c.relay = true }
}

// WithBindAddr overrides the bind address for the QUIC endpoint.
// Default is "0.0.0.0:0" (all interfaces, random port).
func WithBindAddr(addr string) Option {
	return func(c *config) { c.bindAddr = addr }
}

// WithStreamPooling enables persistent BiStream pooling. Instead of opening a
// new QUIC bidirectional stream for every Publish (one-stream-per-op), the
// transport maintains a single long-lived BiStream per peer and multiplexes
// ops over it using length-prefix framing.
//
// This eliminates per-op stream creation overhead (stream ID allocation,
// connection-level coordination, finish+read-to-end teardown) and reduces
// latency under high throughput.
//
// Ordering guarantee (verified 2026-08-16): pooled mode delivers ops to a
// given peer in FIRST-IN-FIRST-OUT order. Three mechanisms combine:
//   - QUIC guarantees byte order within a single stream.
//   - The sender serializes the write-frame → read-ack cycle per peer under
//     pc.streamMu, so op N is fully acked before op N+1 is written.
//   - The receiver (handlePooledStream) processes frames strictly sequentially
//     in one loop — decode and dispatch complete before the next frame is read.
//
// Scope: ordering is per (sender, peer) pair. Ops from DIFFERENT peers to the
// same receiver interleave arbitrarily (separate streams), and relayed ops are
// not globally ordered across hops. The default one-stream-per-op mode has NO
// cross-op ordering at all — each op rides its own stream and is handled by a
// concurrent goroutine, so arrival order is scheduler-dependent.
//
// Tradeoff: the per-op ack makes pooled mode head-of-line blocking on a peer
// (a stalled peer delays that peer's subsequent ops, never other peers).
//
// Both the sender and receiver must have pooling enabled — the framing
// protocol is incompatible with the default one-stream-per-op mode.
//
// Default: disabled (one-stream-per-op, backward compatible).
func WithStreamPooling() Option {
	return func(c *config) { c.poolStreams = true }
}
