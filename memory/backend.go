package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/store"
)

// Backend implements store.Backend with an in-memory map.
// Safe for concurrent use. Designed for testing and single-process deployments.
type Backend struct {
	mu   sync.Mutex
	data map[string][]byte
}

// NewBackend creates a new in-memory Backend.
func NewBackend() *Backend {
	return &Backend{data: make(map[string][]byte)}
}

func (b *Backend) Get(_ context.Context, key []byte) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	v, ok := b.data[string(key)]
	if !ok {
		return nil, store.ErrNotFound
	}

	return cloneBytes(v), nil
}

func (b *Backend) Put(_ context.Context, key, value []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.data[string(key)] = cloneBytes(value)

	return nil
}

func (b *Backend) Delete(_ context.Context, key []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.data, string(key))

	return nil
}

func (b *Backend) Scan(_ context.Context, prefix []byte) (store.Iterator, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	p := string(prefix)

	var entries []kvEntry

	for k, v := range b.data {
		if strings.HasPrefix(k, p) {
			entries = append(entries, kvEntry{key: k, value: cloneBytes(v)})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	return &sliceIterator{entries: entries, idx: -1}, nil
}

func (b *Backend) Batch(_ context.Context, fn func(store.Transaction) error) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	return fn(&memTx{data: b.data})
}

func (b *Backend) Close() error { return nil }

var _ store.Backend = (*Backend)(nil)

type kvEntry struct {
	key   string
	value []byte
}

type sliceIterator struct {
	entries []kvEntry
	idx     int
}

func (i *sliceIterator) Next() bool {
	i.idx++

	return i.idx < len(i.entries)
}

func (i *sliceIterator) Key() []byte {
	if i.idx < 0 || i.idx >= len(i.entries) {
		return nil
	}

	return []byte(i.entries[i.idx].key)
}

func (i *sliceIterator) Value() []byte {
	if i.idx < 0 || i.idx >= len(i.entries) {
		return nil
	}

	return i.entries[i.idx].value
}

func (i *sliceIterator) Err() error { return nil }

func (i *sliceIterator) Close() error { return nil }

type memTx struct {
	data map[string][]byte
}

func (t *memTx) Get(key []byte) ([]byte, error) {
	v, ok := t.data[string(key)]
	if !ok {
		return nil, store.ErrNotFound
	}

	return cloneBytes(v), nil
}

func (t *memTx) Put(key, value []byte) error {
	t.data[string(key)] = cloneBytes(value)

	return nil
}

func (t *memTx) Delete(key []byte) error {
	delete(t.data, string(key))

	return nil
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}

	cp := make([]byte, len(b))
	copy(cp, b)

	return cp
}
