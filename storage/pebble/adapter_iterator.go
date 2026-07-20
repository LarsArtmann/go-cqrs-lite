package pebble

import (
	"slices"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

type pebbleIterator struct {
	iter       *pebble.Iterator
	positioned bool
	closed     bool
}

var _ kv.Iterator = (*pebbleIterator)(nil)

// closeIterator is the shared idempotent Close for both pebbleIterator and
// pebbleEventIterator. Both wrap a *pebble.Iterator, guard against double-close
// via a *bool flag, and wrap the underlying error with errorfamily — differing
// only in the error code and message string.
func closeIterator(closed *bool, iter *pebble.Iterator, errCode, msg string) error {
	if *closed {
		return nil
	}

	*closed = true

	if err := iter.Close(); err != nil {
		return errorfamily.WrapInfrastructure(err, errCode, msg)
	}

	return nil
}

func (it *pebbleIterator) Next() bool {
	if it.closed {
		return false
	}

	if !it.positioned {
		it.positioned = true

		return it.iter.First()
	}

	return it.iter.Next()
}

func (it *pebbleIterator) Key() []byte {
	return slices.Clone(it.iter.Key())
}

func (it *pebbleIterator) Value() []byte {
	return slices.Clone(it.iter.Value())
}

func (it *pebbleIterator) Error() error {
	err := it.iter.Error()
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "pebble.iterator.error",
			"iterator error")
	}

	return nil
}

func (it *pebbleIterator) Close() error {
	return closeIterator(&it.closed, it.iter, "pebble.iterator.close", "close iterator")
}
