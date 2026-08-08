package pebble_test

// deferClose silences close errors in defer scope, replacing the verbose
// `defer func() { _ = x.Close() }()` idiom.
func deferClose(c interface{ Close() error }) { _ = c.Close() }
