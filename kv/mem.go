package kv

import (
	"bytes"
	"context"
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
	return &MemStore{ //nolint:exhaustruct // zero-value fields are intentional
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

// withRLock acquires s's read lock for the duration of fn, returning ErrClosed
// if the store has already been shut down. The read-lock idom duplicates across
// every read-side method (Get, Has, Batch, NewIterator); centralising it here
// removes the four-line `RLock/defer/checkEarly-return` preamble from each
// public method in exchange for one closure allocation per call. MemStore is
// the in-memory test backend, so closure allocation overhead is acceptable.
func (s *MemStore) withRLock(fn func()) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkClosed(); err != nil {
		return err
	}

	fn()

	return nil
}

// withLock acquires s's write lock for the duration of fn, returning ErrClosed
// if the store has already been shut down. See [withRLock] for the rationale.
func (s *MemStore) withLock(fn func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkClosed(); err != nil {
		return err
	}

	fn()

	return nil
}

func (s *MemStore) Get(_ context.Context, key []byte) ([]byte, error) {
	var (
		val   []byte
		found bool
	)

	err := s.withRLock(func() {
		v, ok := s.data[string(key)]
		if !ok {
			return
		}

		val = slices.Clone(v)
		found = true
	})
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, ErrNotFound
	}

	return val, nil
}

func (s *MemStore) Has(_ context.Context, key []byte) (bool, error) {
	var ok bool

	err := s.withRLock(func() { _, ok = s.data[string(key)] })
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (s *MemStore) Set(_ context.Context, key, value []byte) error {
	return s.withLock(func() { s.data[string(key)] = slices.Clone(value) })
}

func (s *MemStore) Delete(_ context.Context, key []byte) error {
	return s.withLock(func() { delete(s.data, string(key)) })
}

// SetIfAbsent atomically sets the value only if the key does not currently
// exist. Returns true if the set succeeded, false if the key already existed.
func (s *MemStore) SetIfAbsent(_ context.Context, key, value []byte) (bool, error) {
	var inserted bool

	err := s.withLock(func() {
		k := string(key)
		if _, exists := s.data[k]; exists {
			return
		}

		s.data[k] = slices.Clone(value)
		inserted = true
	})
	if err != nil {
		return false, err
	}

	return inserted, nil
}

func (s *MemStore) Batch(_ context.Context) (Batch, error) {
	var batch *memBatch

	err := s.withRLock(func() {
		batch = &memBatch{store: s} //nolint:exhaustruct // zero-value ops/closed are intentional
	})
	if err != nil {
		return nil, err
	}

	return batch, nil
}

func (s *MemStore) NewIterator(_ context.Context, prefix []byte) (Iterator, error) {
	var pairs []memKV

	err := s.withRLock(func() { pairs = collectSorted(s.data, prefix) })
	if err != nil {
		return nil, err
	}

	return &memIterator{pairs: pairs}, nil //nolint:exhaustruct // zero-value fields are intentional
}

func (s *MemStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	s.data = nil

	return nil
}

// ── helpers ──────────────────────────────────────────────────

type memKV struct {
	key   []byte
	value []byte
}

func collectSorted(data map[string][]byte, prefix []byte) []memKV {
	var pairs []memKV

	for k, v := range data {
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

	return pairs
}

// ── memIterator ──────────────────────────────────────────────

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
