package pebble

import (
	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"
)

func init() { //nolint:gochecknoinits // package-wide registration of pebble driver error classification, must run before any store operation
	// Register pebble driver error classification so that a leaked
	// pebble.ErrNotFound (if a code path forgets to translate it) classifies
	// as Rejection instead of defaulting to Transient. Every call site in
	// this package already catches ErrNotFound locally and translates it to
	// kv.ErrNotFound or treats it as a non-error, so this is defense-in-depth.
	errorfamily.RegisterClassification(pebble.ErrNotFound, errorfamily.Rejection)
}
