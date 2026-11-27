package kv_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// blockingStore reads the value, then holds the Get before returning it, so
// a Set can land between a Get's store read and its cache insert.
type blockingStore struct {
	mu      sync.Mutex
	values  map[string]*testView
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *blockingStore) Get(_ context.Context, key stringKey) (*testView, error) {
	s.mu.Lock()
	val, ok := s.values[key.String()]
	copied := *val
	s.mu.Unlock()

	if !ok {
		return nil, kv.ErrNotFound
	}

	s.once.Do(func() { close(s.entered) })
	<-s.release

	return &copied, nil
}

func (s *blockingStore) Set(_ context.Context, key stringKey, val *testView) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copied := *val
	s.values[key.String()] = &copied

	return nil
}

func (*blockingStore) Delete(context.Context, stringKey) error { return nil }

func (*blockingStore) Scan(context.Context, []byte) ([]*testView, error) { return nil, nil }

// A write that lands while another Get is loading the old value from the store
// must not be hidden by that Get caching what it read.
func TestCacheGetDuringSetDoesNotPinStaleValue(t *testing.T) {
	store := &blockingStore{
		values:  map[string]*testView{},
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}

	key := stringKey("k1")

	if err := store.Set(context.Background(), key, &testView{Name: "old"}); err != nil {
		t.Fatal(err)
	}

	cache, err := kv.NewCacheForViewStore[testView, stringKey](store)
	if err != nil {
		t.Fatal(err)
	}

	loaded := make(chan *testView, 1)
	go func() {
		v, _ := cache.Get(context.Background(), key)
		loaded <- v
	}()

	<-store.entered

	if err := cache.Set(context.Background(), key, &testView{Name: "new"}); err != nil {
		t.Fatal(err)
	}

	close(store.release)

	select {
	case <-loaded:
	case <-time.After(5 * time.Second):
		t.Fatal("blocked Get did not return")
	}

	for i := range 3 {
		got, err := cache.Get(context.Background(), key)
		if err != nil {
			t.Fatal(err)
		}

		if got.Name != "new" {
			t.Fatalf("read %d after Set: got %q, want the value Set wrote", i, got.Name)
		}
	}
}
