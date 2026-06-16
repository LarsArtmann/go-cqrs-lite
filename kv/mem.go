package kv

import (
	"bytes"
	"slices"
	"sync"
)

// MemStore is an in-memory implementation of [Store].
// It is safe for concurrent use.
// Keys are stored in a map and sorted on iteration.
type MemStore struct {
	mu     sync.RWMutex
	data   map[string][]byte
	closed bool
}

// NewMemStore creates a new empty [MemStore].
func NewMemStore() *MemStore {
	return &MemStore{
		data: make(map[string][]byte),
	}
}

// compile-time interface check.
var _ Store = (*MemStore)(nil)

func (s *MemStore) checkClosed() error {
	if s.closed {
		return ErrClosed
	}

	return nil
}

func (s *MemStore) Get(key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	val, ok := s.data[string(key)]
	if !ok {
		return nil, ErrNotFound
	}

	return slices.Clone(val), nil
}

func (s *MemStore) Has(key []byte) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkClosed(); err != nil {
		return false, err
	}

	_, ok := s.data[string(key)]

	return ok, nil
}

func (s *MemStore) Set(key, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkClosed(); err != nil {
		return err
	}

	s.data[string(key)] = slices.Clone(value)

	return nil
}

func (s *MemStore) Delete(key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkClosed(); err != nil {
		return err
	}

	delete(s.data, string(key))

	return nil
}

func (s *MemStore) Batch() (Batch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	return &memBatch{store: s}, nil
}

func (s *MemStore) NewIterator(prefix []byte) (Iterator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	var pairs []memKV

	for k, v := range s.data {
		bk := []byte(k)
		if len(prefix) == 0 || bytes.HasPrefix(bk, prefix) {
			pairs = append(pairs, memKV{
				key:   slices.Clone(bk),
				value: slices.Clone(v),
			})
		}
	}

	slices.SortFunc(pairs, func(a, b memKV) int {
		return bytes.Compare(a.key, b.key)
	})

	return &memIterator{pairs: pairs}, nil
}

func (s *MemStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	s.data = nil

	return nil
}

// ── memIterator ──────────────────────────────────────────────

type memKV struct {
	key   []byte
	value []byte
}

type memIterator struct {
	pairs  []memKV
	idx    int
	err    error
	closed bool
}

var _ Iterator = (*memIterator)(nil)

func (it *memIterator) Next() bool {
	if it.closed {
		return false
	}

	it.idx++

	return it.idx <= len(it.pairs)
}

func (it *memIterator) Key() []byte {
	if it.idx == 0 || it.idx > len(it.pairs) {
		return nil
	}

	return it.pairs[it.idx-1].key
}

func (it *memIterator) Value() []byte {
	if it.idx == 0 || it.idx > len(it.pairs) {
		return nil
	}

	return it.pairs[it.idx-1].value
}

func (it *memIterator) Error() error {
	return it.err
}

func (it *memIterator) Close() error {
	it.closed = true
	it.pairs = nil

	return nil
}

// ── memBatch ─────────────────────────────────────────────────

type batchOp struct {
	isDelete bool
	key      []byte
	value    []byte
}

type memBatch struct {
	store  *MemStore
	ops    []batchOp
	closed bool
}

var _ Batch = (*memBatch)(nil)

func (b *memBatch) Set(key, value []byte) error {
	if b.closed {
		return ErrClosed
	}

	b.ops = append(b.ops, batchOp{
		key:   slices.Clone(key),
		value: slices.Clone(value),
	})

	return nil
}

func (b *memBatch) Delete(key []byte) error {
	if b.closed {
		return ErrClosed
	}

	b.ops = append(b.ops, batchOp{
		isDelete: true,
		key:      slices.Clone(key),
	})

	return nil
}

func (b *memBatch) Commit() error {
	if b.closed {
		return ErrClosed
	}

	defer b.Close()

	b.store.mu.Lock()
	defer b.store.mu.Unlock()

	if err := b.store.checkClosed(); err != nil {
		return err
	}

	for _, op := range b.ops {
		if op.isDelete {
			delete(b.store.data, string(op.key))
		} else {
			b.store.data[string(op.key)] = op.value
		}
	}

	return nil
}

func (b *memBatch) Close() error {
	b.closed = true
	b.ops = nil

	return nil
}
