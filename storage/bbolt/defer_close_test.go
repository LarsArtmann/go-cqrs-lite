package bbolt

// deferClose silences close errors in defer scope, replacing the bare
// `defer iter.Close()` idiom that silently discards the close error.
//
// Per-module idiom: storage/pebble has its own copy (close_helper.go).
// Cross-module sharing would add a dependency for a 1-line function.
func deferClose(c interface{ Close() error }) { _ = c.Close() }
