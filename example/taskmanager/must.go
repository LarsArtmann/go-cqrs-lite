// Panic-on-error helpers for unrecoverable errors (initialization, tests).
//
// Local copies of the go-must helpers: go-must is a private repository the
// module proxy cannot serve past v0.1.0, and a private require in any
// workspace module breaks every workspace-wide go command on machines
// without credentials (CI module loading exits 128 at the auth prompt).
// An example must stay buildable for anonymous consumers.
package taskmanager

// Must panics if err is non-nil and returns v otherwise.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

// Check panics if err is non-nil. Use for fire-and-forget error checks
// where the error has no associated value to return.
func Check(err error) {
	if err != nil {
		panic(err)
	}
}
