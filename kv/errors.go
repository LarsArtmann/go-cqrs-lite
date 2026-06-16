package kv

import "errors"

// ErrNotFound is returned when a Get operation finds no value for the key.
var ErrNotFound = errors.New("kv: key not found")

// ErrClosed is returned when an operation is attempted on a closed store,
// iterator, or batch.
var ErrClosed = errors.New("kv: store closed")
