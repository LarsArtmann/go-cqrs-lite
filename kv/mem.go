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

func (s *MemStore) Get(_ context.Context, key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	err := s.checkClosed()
	if err != nil {
		return nil, err
	}

	val, ok := s.data[string(key)]
	if !ok {
		return nil, ErrNotFound
	}

	return slices.Clone(val), nil
}

func (s *MemStore) Has(_ context.Context, key []byte) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	err := s.checkClosed()
	if err != nil {
		return false, err
	}

	_, ok := s.data[string(key)]

	return ok, nil
}

func (s *MemStore) Set(_ context.Context, key, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.checkClosed()
	if err != nil {
		return err
	}

	s.data[string(key)] = slices.Clone(value)

	return nil
}

func (s *MemStore) Delete(_ context.Context, key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.checkClosed()
	if err != nil {
		return err
	}

	delete(s.data, string(key))

	return nil
}

// SetIfAbsent atomically sets the value only if the key does not currently
// exist. Returns true if the set succeeded, false if the key already existed.
func (s *MemStore) SetIfAbsent(_ context.Context, key, value []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkClosed(); err != nil {
		return false, err
	}

	k := string(key)
	if _, exists := s.data[k]; exists {
		return false, nil
	}

	s.data[k] = slices.Clone(value)

	return true, nil
}

func (s *MemStore) Batch(_ context.Context) (Batch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	err := s.checkClosed()
	if err != nil {
		return nil, err
	}

	return &memBatch{store: s}, nil //nolint:exhaustruct // zero-value fields are intentional
}

func (s *MemStore) NewIterator(_ context.Context, prefix []byte) (Iterator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	err := s.checkClosed()
	if err != nil {
		return nil, err
	}

	pairs := collectSorted(s.data, prefix)

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
