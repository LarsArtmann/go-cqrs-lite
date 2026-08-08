package pebble

// closer is the minimal interface for deferClose.
type closer interface{ Close() error }

// deferClose calls Close and discards the error. It replaces the
// `defer func() { _ = x.Close() }()` idiom for readability.
//
// A mirror copy exists in defer_close_ext_test.go for package pebble_test
// (external tests cannot access unexported symbols). This per-package-scope
// duplication is an unavoidable consequence of Go's visibility rules.
func deferClose(c closer) { _ = c.Close() }
