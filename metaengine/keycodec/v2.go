package keycodec

import "fmt"

// The v2 key family: same shapes as v1 with an "2"-suffixed family tag and
// ESCAPED user segments (Escape). Escaping makes every join injective — no
// segment content can forge a separator, so ("a", "b\x00c") and ("a\x00b",
// "c") now produce distinct keys in different families. The v1 builders and
// their byte ranges stay untouched so data written before the v2 adoption
// remains readable for the dual-read window (one release, per the plan's
// G3 drop gate); new writes go to v2 ranges only.

// MapKeyV2 returns the full v2 key for a (collection, key) pair.
func MapKeyV2(col, key string) []byte {
	return []byte("m2" + Sep + Escape(col) + Sep + Escape(key))
}

// SetKeyV2 returns the full v2 key for a (collection, key) pair.
func SetKeyV2(col, key string) []byte {
	return []byte("s2" + Sep + Escape(col) + Sep + Escape(key))
}

// CounterKeyV2 returns the full v2 key for a (collection, ckey) pair.
func CounterKeyV2(col, ckey string) []byte {
	return []byte("c2" + Sep + Escape(col) + Sep + Escape(ckey))
}

// CollectionPrefixV2 returns the v2 scan prefix for a collection's map keys.
func CollectionPrefixV2(col string) []byte {
	return []byte("m2" + Sep + Escape(col) + Sep)
}

// CounterPrefixV2 returns the v2 scan prefix for a collection's counters.
func CounterPrefixV2(col string) []byte {
	return []byte("c2" + Sep + Escape(col) + Sep)
}

// MultimapPrefixV2 returns the v2 scan prefix for a (collection, key) pair.
func MultimapPrefixV2(col, key string) []byte {
	return []byte("mm2" + Sep + Escape(col) + Sep + Escape(key) + Sep)
}

// LogPrefixV2 returns the v2 scan prefix for a collection's log keys.
func LogPrefixV2(col string) []byte {
	return []byte("l2" + Sep + Escape(col) + Sep)
}

// MultimapKeyV2 returns a full v2 key for a (collection, key, seq) tuple.
func MultimapKeyV2(col, key string, seq int64) []byte {
	return fmt.Appendf(nil, "mm2%s%s%s%s%s%020d", Sep, Escape(col), Sep, Escape(key), Sep, seq)
}

// LogKeyV2 returns a full v2 key for a (collection, seq) tuple.
func LogKeyV2(col string, seq int64) []byte {
	return fmt.Appendf(nil, "l2%s%s%s%020d", Sep, Escape(col), Sep, seq)
}

// VectorKeyV2 returns the full v2 key for a (collection, id) pair.
func VectorKeyV2(col, id string) []byte {
	return []byte("vec2" + Sep + Escape(col) + Sep + Escape(id))
}

// VectorPrefixV2 returns the v2 scan prefix for a collection's embeddings.
func VectorPrefixV2(col string) []byte {
	return []byte("vec2" + Sep + Escape(col) + Sep)
}

// VectorMetaKeyV2 returns the v2 metadata key for a (collection, id) pair.
func VectorMetaKeyV2(col, id string) []byte {
	return []byte("vecm2" + Sep + Escape(col) + Sep + Escape(id))
}

// VectorMetaPrefixV2 returns the v2 scan prefix for a collection's vector metadata.
func VectorMetaPrefixV2(col string) []byte {
	return []byte("vecm2" + Sep + Escape(col) + Sep)
}

// GraphEdgeFwdKeyV2 returns the v2 forward adjacency key for a directed edge.
func GraphEdgeFwdKeyV2(col, from, to string) []byte {
	return []byte("edge2" + Sep + Escape(col) + Sep + Escape(from) + Sep + Escape(to))
}

// GraphEdgeFwdPrefixV2 returns the v2 scan prefix for a node's outgoing edges.
func GraphEdgeFwdPrefixV2(col, from string) []byte {
	return []byte("edge2" + Sep + Escape(col) + Sep + Escape(from) + Sep)
}

// GraphEdgeRevKeyV2 returns the v2 reverse adjacency key for a directed edge.
func GraphEdgeRevKeyV2(col, to, from string) []byte {
	return []byte("edger2" + Sep + Escape(col) + Sep + Escape(to) + Sep + Escape(from))
}

// GraphEdgeRevPrefixV2 returns the v2 scan prefix for a node's incoming edges.
func GraphEdgeRevPrefixV2(col, to string) []byte {
	return []byte("edger2" + Sep + Escape(col) + Sep + Escape(to) + Sep)
}

// StreamKeyV2 returns the v2 per-stream entry key for the StreamLogBackend.
func StreamKeyV2(col, sid string, seq int64) []byte {
	return fmt.Appendf(nil, "sl2%s%s%s%s%s%020d", Sep, Escape(col), Sep, Escape(sid), Sep, seq)
}

// StreamPrefixV2 returns the v2 scan prefix for a single stream's entries.
func StreamPrefixV2(col, sid string) []byte {
	return []byte("sl2" + Sep + Escape(col) + Sep + Escape(sid) + Sep)
}

// JournalKeyV2 returns the v2 global journal index key.
func JournalKeyV2(col string, gseq int64) []byte {
	return fmt.Appendf(nil, "jl2%s%s%s%020d", Sep, Escape(col), Sep, gseq)
}

// JournalPrefixV2 returns the v2 scan prefix for a collection's journal.
func JournalPrefixV2(col string) []byte {
	return []byte("jl2" + Sep + Escape(col) + Sep)
}

// StreamSeqKeyV2 builds the v2 in-memory map key for per-stream counters.
func StreamSeqKeyV2(col, sid string) string {
	return Escape(col) + Sep + Escape(sid)
}
