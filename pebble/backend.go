package pebble

import (
	"context"
	"errors"
	"slices"

	pebbledb "github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/core/store"
)

// Backend implements store.Backend using CockroachDB Pebble.
type Backend struct {
	db      *pebbledb.DB
	syncOpt *pebbledb.WriteOptions
}

// NewBackend creates a Pebble-backed Backend from an existing *pebble.DB.
// Writes are synchronous by default (data survives process crash).
// Pass nil for db to create an uninitialized backend (must call Open before use).
func NewBackend(db *pebbledb.DB) *Backend {
	return &Backend{db: db, syncOpt: pebbledb.Sync}
}

// WithAsyncWrites disables sync writes for higher throughput at the cost of
// durability. Use only when data loss on crash is acceptable.
func (b *Backend) WithAsyncWrites() *Backend {
	b.syncOpt = pebbledb.NoSync

	return b
}

func (b *Backend) Get(_ context.Context, key []byte) ([]byte, error) {
	val, closer, err := b.db.Get(key)
	if err != nil {
		if errors.Is(err, pebbledb.ErrNotFound) {
			return nil, store.ErrNotFound
		}

		return nil, err
	}
	defer closer.Close()

	cp := make([]byte, len(val))
	copy(cp, val)

	return cp, nil
}

func (b *Backend) Put(_ context.Context, key, value []byte) error {
	return b.db.Set(key, value, b.syncOpt)
}

func (b *Backend) Delete(_ context.Context, key []byte) error {
	return b.db.Delete(key, b.syncOpt)
}

func (b *Backend) Scan(_ context.Context, prefix []byte) (store.Iterator, error) {
	iter, err := b.db.NewIter(&pebbledb.IterOptions{
		LowerBound: prefix,
		UpperBound: successor(prefix),
	})
	if err != nil {
		return nil, err
	}

	return &pebbleIterator{iter: iter, started: false}, nil
}

func (b *Backend) Batch(_ context.Context, fn func(store.Transaction) error) error {
	batch := b.db.NewBatch()
	defer batch.Close()

	tx := &pebbleTx{batch: batch}

	err := fn(tx)
	if err != nil {
		return err
	}

	return batch.Commit(b.syncOpt)
}

func (b *Backend) Close() error {
	if b.db != nil {
		return b.db.Close()
	}

	return nil
}

var _ store.Backend = (*Backend)(nil)

func successor(prefix []byte) []byte {
	ub := make([]byte, len(prefix))
	copy(ub, prefix)

	for _, v := range slices.Backward(ub) {
		v++
		if v != 0 {
			return ub
		}
	}

	return []byte{0xff, 0xff, 0xff, 0xff}
}

type pebbleIterator struct {
	iter    *pebbledb.Iterator
	started bool
}

func (i *pebbleIterator) Next() bool {
	if !i.started {
		i.started = true
		i.iter.First()
	} else {
		i.iter.Next()
	}

	return i.iter.Valid()
}

func (i *pebbleIterator) Key() []byte { return i.iter.Key() }

func (i *pebbleIterator) Value() []byte { return i.iter.Value() }

func (i *pebbleIterator) Err() error { return i.iter.Error() }

func (i *pebbleIterator) Close() error { return i.iter.Close() }

type pebbleTx struct {
	batch *pebbledb.Batch
}

func (t *pebbleTx) Get(key []byte) ([]byte, error) {
	val, closer, err := t.batch.Get(key)
	if err != nil {
		if errors.Is(err, pebbledb.ErrNotFound) {
			return nil, store.ErrNotFound
		}

		return nil, err
	}
	defer closer.Close()

	cp := make([]byte, len(val))
	copy(cp, val)

	return cp, nil
}

func (t *pebbleTx) Put(key, value []byte) error {
	return t.batch.Set(key, value, nil)
}

func (t *pebbleTx) Delete(key []byte) error {
	return t.batch.Delete(key, nil)
}
