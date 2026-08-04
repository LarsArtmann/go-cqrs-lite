package quic

import (
	iroh_ffi "git.coopcloud.tech/decentral1se/iroh-go"
)

// Option configures a QuicTransport.
type Option func(*config)

type config struct {
	alpn     []byte
	presetFn func() iroh_ffi.Preset
	bindAddr string
	relay    bool
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
