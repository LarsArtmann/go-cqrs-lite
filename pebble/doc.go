// Package pebble provides an embedded key-value event store implementation
// using CockroachDB Pebble. It implements event.Store with optimistic
// concurrency control via per-aggregate locking.
//
// Use NewStore to create a store from an existing *pebble.DB.
//
// # Envelope Format
//
// Events are stored as CBOR-encoded envelopes with canonical (deterministic)
// encoding (RFC 7049). The payload is stored as raw bytes — no base64 encoding.
// For backward compatibility, deserializeEvent reads both CBOR envelopes
// (current) and JSON envelopes (legacy) via format sniffing.
package pebble
