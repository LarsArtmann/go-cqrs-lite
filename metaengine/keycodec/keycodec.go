// Package keycodec provides shared key encoding helpers for the
// LSM-style metaengine backends (Badger, Pebble).
//
// Both engines store their data in a single byte-sorted keyspace using
// length-prefixed records separated by a null byte. The helpers in this
// package produce the standard key shapes used by the MapBackend, SetBackend,
// CounterBackend, MultimapBackend, LogBackend, and StreamLogBackend
// implementations.
//
// Key shapes (all share `sep = "\x00"` as collection separator):
//
//	collectionPrefix   : "m\x00<col>\x00"
//	counterPrefix      : "c\x00<col>\x00"
//	multimapPrefix     : "mm\x00<col>\x00<key>\x00"
//	logPrefix          : "l\x00<col>\x00"
//	vectorPrefix       : "vec\x00<col>\x00"
//	multimapKey        : "mm\x00<col>\x00<key>\x00<seq:%020d>"
//	logKey             : "l\x00<col>\x00<seq:%020d>"
//	vectorKey          : "vec\x00<col>\x00<id>"
//
// Both engines must stay in sync on these key shapes — they share the same
// on-disk layout so that migration between engines is lossless.
package keycodec

import (
	"encoding/binary"
	"encoding/json/v2"
	"fmt"
	"strconv"
)

// Sep is the key separator. Null byte sorts before all printable characters,
// preventing collisions between collection names and keys.
const Sep = "\x00"

// MapKey returns the full key for a (collection, key) pair in the MapBackend.
func MapKey(col, key string) []byte {
	return []byte("m" + Sep + col + Sep + key)
}

// SetKey returns the full key for a (collection, key) pair in the SetBackend.
func SetKey(col, key string) []byte {
	return []byte("s" + Sep + col + Sep + key)
}

// CounterKey returns the full key for a (collection, ckey) pair in the CounterBackend.
func CounterKey(col, ckey string) []byte {
	return []byte("c" + Sep + col + Sep + ckey)
}

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

// VectorKey returns the full key for a (collection, id) pair in the
// VectorBackend. Values are the JSON-encoded embedding dimensions.
func VectorKey(col, id string) []byte {
	return []byte("vec" + Sep + col + Sep + id)
}

// VectorPrefix returns the scan prefix for the vector embeddings of a
// collection.
func VectorPrefix(col string) []byte {
	return []byte("vec" + Sep + col + Sep)
}

// VectorMetaKey returns the metadata key for a (collection, id) pair in the
// VectorBackend. Values are the JSON-encoded Embedding.Metadata map; absent
// metadata is simply not stored. Kept in a separate key family from
// VectorKey so the on-disk vector format is unchanged for engines predating
// metadata-filtered k-NN.
func VectorMetaKey(col, id string) []byte {
	return []byte("vecm" + Sep + col + Sep + id)
}

// VectorMetaPrefix returns the scan prefix for the vector metadata of a
// collection.
func VectorMetaPrefix(col string) []byte {
	return []byte("vecm" + Sep + col + Sep)
}

// GraphEdgeFwdKey returns the forward adjacency key for a directed edge in
// the graphBackend: "edge\x00<col>\x00<from>\x00<to>". A prefix scan over
// "edge\x00<col>\x00<from>\x00" yields the node's outgoing neighbors.
func GraphEdgeFwdKey(col, from, to string) []byte {
	return []byte("edge" + Sep + col + Sep + from + Sep + to)
}

// GraphEdgeFwdPrefix returns the scan prefix for all outgoing edges of a
// node.
func GraphEdgeFwdPrefix(col, from string) []byte {
	return []byte("edge" + Sep + col + Sep + from + Sep)
}

// GraphEdgeRevKey returns the reverse adjacency key for a directed edge:
// "edger\x00<col>\x00<to>\x00<from>". Maintained as a second index so
// undirected traversal is a prefix scan instead of a full collection scan.
// GraphRemoveEdge deletes both key families.
func GraphEdgeRevKey(col, to, from string) []byte {
	return []byte("edger" + Sep + col + Sep + to + Sep + from)
}

// GraphEdgeRevPrefix returns the scan prefix for all incoming edges of a
// node (reverse adjacency).
func GraphEdgeRevPrefix(col, to string) []byte {
	return []byte("edger" + Sep + col + Sep + to + Sep)
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

// StreamKey returns the per-stream entry key for the StreamLogBackend.
// seq is zero-padded to 20 digits so lexicographic byte order matches numeric order.
func StreamKey(col, sid string, seq int64) []byte {
	return fmt.Appendf(nil, "sl%s%s%s%s%s%020d", Sep, col, Sep, sid, Sep, seq)
}

// StreamPrefix returns the scan prefix for all entries of a single stream.
func StreamPrefix(col, sid string) []byte {
	return []byte("sl" + Sep + col + Sep + sid + Sep)
}

// JournalKey returns the global journal index key for the StreamLogBackend.
func JournalKey(col string, gseq int64) []byte {
	return fmt.Appendf(nil, "jl%s%s%s%020d", Sep, col, Sep, gseq)
}

// JournalPrefix returns the scan prefix for the global journal of a collection.
func JournalPrefix(col string) []byte {
	return []byte("jl" + Sep + col + Sep)
}

// JournalSeq parses the seq out of a journal key produced by JournalKey (the
// trailing 20-digit zero-padded decimal). Returns false when the key does not
// end in a parseable seq. Used by the KV engines to attach resume tokens to
// journal entries they are already iterating.
func JournalSeq(key []byte) (int64, bool) {
	const seqLen = 20
	if len(key) < seqLen {
		return 0, false
	}

	seq, err := strconv.ParseInt(string(key[len(key)-seqLen:]), 10, 64)
	if err != nil {
		return 0, false
	}

	return seq, true
}

// StreamSeqKey builds the in-memory map key for per-stream sequence counters.
func StreamSeqKey(col, sid string) string {
	return col + Sep + sid
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
