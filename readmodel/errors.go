package readmodel

import (
	"errors"

	"github.com/larsartmann/go-cqrs-lite/kv/v2"
)

// ErrNotFound is returned by [Store.Get] when no value exists for the given key.
//
// It is the same sentinel as [kv.ErrNotFound]. Get wraps it with key context,
// so callers must use errors.Is(err, readmodel.ErrNotFound) rather than a
// direct equality check.
var ErrNotFound = kv.ErrNotFound

// errNilValue is returned by [Store.Set] when val is nil.
var errNilValue = errors.New("readmodel: Set called with a nil value")
