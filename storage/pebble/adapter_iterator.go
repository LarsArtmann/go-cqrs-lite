package pebble

import (
	"slices"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
)

type pebbleIterator struct {
	iter       *pebble.Iterator
	positioned bool
	closed     bool
}

var _ kv.Iterator = (*pebbleIterator)(nil)

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
	if it.closed {
		return nil
	}

	it.closed = true

	err := it.iter.Close()
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "pebble.iterator.close",
			"close iterator")
	}

	return nil
}
