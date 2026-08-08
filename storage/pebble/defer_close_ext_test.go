package pebble_test

// deferClose silences close errors in defer scope, replacing the verbose
// `defer func() { _ = x.Close() }()` idiom.
//
// This is a deliberate mirror of the unexported deferClose in close_helper.go.
// External test files (package pebble_test) cannot access unexported symbols
// from package pebble, so a separate copy is structurally required.
func deferClose(c interface{ Close() error }) { _ = c.Close() }
