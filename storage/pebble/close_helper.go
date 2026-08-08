package pebble

// closer is the minimal interface for deferClose.
type closer interface{ Close() error }

// deferClose calls Close and discards the error. It replaces the
// `defer func() { _ = x.Close() }()` idiom for readability.
func deferClose(c closer) { _ = c.Close() }
