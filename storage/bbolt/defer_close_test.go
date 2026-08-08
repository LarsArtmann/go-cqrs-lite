package bbolt

// deferClose silences close errors in defer scope, replacing the bare
// `defer iter.Close()` idiom that silently discards the close error.
func deferClose(c interface{ Close() error }) { _ = c.Close() }
