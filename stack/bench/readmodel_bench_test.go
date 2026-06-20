package bench

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/readmodel/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2"
)

type benchKey string

func (k benchKey) String() string { return string(k) }

type benchView struct {
	Name string `json:"name"`
	N    int    `json:"n"`
}

// BenchmarkBundle_ReadModelGet gets a read model through a Store created from
// Bundle.ReadModels. Should be identical to BenchmarkDirect_ReadModelGet.
func BenchmarkBundle_ReadModelGet(b *testing.B) {
	bundle, err := stack.New(
		stack.WithReadModels(kv.NewMemStore()),
	)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = bundle.Close() }()

	store := readmodel.New[benchView, benchKey](
		bundle.ReadModels,
		kv.WithTypedKeyPrefix[benchView, benchKey]("bench:"),
	)

	ctx := context.Background()

	if err := store.Set(ctx, "1", &benchView{Name: "test", N: 42}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for range b.N {
		_, err := store.Get(ctx, "1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDirect_ReadModelGet gets a read model from a Store created directly.
func BenchmarkDirect_ReadModelGet(b *testing.B) {
	backend := kv.NewMemStore()
	defer func() { _ = backend.Close() }()

	store := readmodel.New[benchView, benchKey](
		backend,
		kv.WithTypedKeyPrefix[benchView, benchKey]("bench:"),
	)

	ctx := context.Background()

	if err := store.Set(ctx, "1", &benchView{Name: "test", N: 42}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for range b.N {
		_, err := store.Get(ctx, "1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBundle_FieldAccess proves that accessing a Bundle field is zero-cost
// by comparing it to a direct local variable access.
func BenchmarkBundle_FieldAccess(b *testing.B) {
	store := memory.NewMemoryStore()
	defer func() { _ = store.Close() }()

	bundle, err := stack.New(stack.WithEventStore(store))
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = bundle.Close() }()

	sink := bundle.EventSink

	b.Run("BundleField", func(b *testing.B) {
		for range b.N {
			_ = bundle.EventSink
		}
	})

	b.Run("LocalVar", func(b *testing.B) {
		for range b.N {
			_ = sink
		}
	})
}
