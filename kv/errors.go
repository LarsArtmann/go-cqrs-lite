package kv

import errorfamily "github.com/larsartmann/go-error-family"

// ErrNotFound is returned when a Get operation finds no value for the key.
var ErrNotFound = errorfamily.NewRejection("kv.not_found", "kv: key not found")

// ErrClosed is returned when an operation is attempted on a closed store,
// iterator, or batch.
var ErrClosed = errorfamily.NewInfrastructure("kv.closed", "kv: store closed")
