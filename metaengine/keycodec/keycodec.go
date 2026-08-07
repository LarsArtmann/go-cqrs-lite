// Package keycodec provides shared key encoding helpers for the
// LSM-style metaengine backends (Badger, Pebble).
//
// Both engines store their data in a single byte-sorted keyspace using
// length-prefixed records separated by a null byte. The helpers in this
// package produce the standard key shapes used by the MapBackend, SetBackend,
// CounterBackend, MultimapBackend, LogBackend, GraphBackend, and StreamLogBackend
// implementations.
//
// Key shapes (all share `sep = "\x00"` as collection separator):
//
//	collectionPrefix   : "m\x00<col>\x00"
//	counterPrefix      : "c\x00<col>\x00"
//	multimapPrefix     : "mm\x00<col>\x00<key>\x00"
//	logPrefix          : "l\x00<col>\x00"
//	graphPrefixForward : "g\x00<col>\x00<node>\x00"
//	multimapKey        : "mm\x00<col>\x00<key>\x00<seq:%020d>"
//	logKey             : "l\x00<col>\x00<seq:%020d>"
//	graphEdgeKey       : "g\x00<col>\x00<from>\x00<to>"
//
// Both engines must stay in sync on these key shapes — they share the same
// on-disk layout so that migration between engines is lossless.
package keycodec

import (
	"encoding/binary"
	"encoding/json/v2"
	"fmt"
)

// Sep is the key separator. Null byte sorts before all printable characters,
// preventing collisions between collection names and keys.
const Sep = "\x00"

// CollectionPrefix returns the prefix for Map-prefix scans in the given collection.
func CollectionPrefix(col string) []byte {
	return []byte("m" + Sep + col + Sep)
}

// CounterPrefix returns the prefix for Counter-prefix scans in the given collection.
func CounterPrefix(col string) []byte {
	return []byte("c" + Sep + col + Sep)
}

// MultimapPrefix returns the prefix for Multimap-prefix scans in the given
// (collection, key) pair.
func MultimapPrefix(col, key string) []byte {
	return []byte("mm" + Sep + col + Sep + key + Sep)
}

// LogPrefix returns the prefix for Log-prefix scans in the given collection.
func LogPrefix(col string) []byte {
	return []byte("l" + Sep + col + Sep)
}

// GraphPrefixForward returns the prefix for forward-edge scans in the given
// (collection, node) pair.
func GraphPrefixForward(col, node string) []byte {
	return []byte("g" + Sep + col + Sep + node + Sep)
}

// MultimapKey returns a full key for a (collection, key, seq) tuple used in
// the Multimap and StreamLog backends. seq is zero-padded to 20 digits so
// that lexicographic byte order matches numeric order.
func MultimapKey(col, key string, seq int64) []byte {
	return fmt.Appendf(nil, "mm%s%s%s%s%s%020d", Sep, col, Sep, key, Sep, seq)
}

// LogKey returns a full key for a (collection, seq) tuple used in the Log
// backend. seq is zero-padded to 20 digits so that lexicographic byte order
// matches numeric order.
func LogKey(col string, seq int64) []byte {
	return fmt.Appendf(nil, "l%s%s%s%020d", Sep, col, Sep, seq)
}

// GraphEdgeKey returns a full key for a (collection, from, to) triple used
// in the GraphBackend.
func GraphEdgeKey(col, from, to string) []byte {
	return []byte("g" + Sep + col + Sep + from + Sep + to)
}

// EncodeJSON marshals v to JSON, falling back to fmt.Sprintf("%v", v) on error.
// This is the canonical value-encoding for the LSM engines.
func EncodeJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(fmt.Sprintf("%v", v))
	}

	return b
}

// DecodeJSON unmarshals data into a any value, returning the raw string on
// failure (matches the legacy "leave-as-string" fallback).
func DecodeJSON(data []byte) any {
	var val any
	if err := json.Unmarshal(data, &val); err != nil {
		return string(data)
	}

	return val
}

// EncodeKeyStr is a convenience wrapper that JSON-encodes a key and returns
// the result as a string. Used as the canonical string form of map/multimap keys.
func EncodeKeyStr(key any) string {
	return string(EncodeJSON(key))
}

// EncodeCounterValue encodes an int64 as 8 bytes big-endian.
func EncodeCounterValue(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))

	return b
}

// DecodeCounterValue decodes 8 bytes big-endian into an int64. Returns 0
// when the buffer is shorter than 8 bytes.
func DecodeCounterValue(data []byte) int64 {
	if len(data) < 8 {
		return 0
	}

	return int64(binary.BigEndian.Uint64(data))
}
