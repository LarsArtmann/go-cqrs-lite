package readmodel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/kv/v2"
	"github.com/larsartmann/go-cqrs-lite/readmodel/v2"
)

// todoKey is a branded string key used to verify the fmt.Stringer constraint
// works with named types, not just id.Of branded IDs.
type todoKey string

func (k todoKey) String() string { return string(k) }

// userKey is a second branded key type used to verify key-prefix namespacing
// keeps read models with the same logical key ("1") separate.
type userKey string

func (k userKey) String() string { return string(k) }

type todo struct {
	Title    string `json:"title"`
	Complete bool   `json:"complete"`
}

func newStore(
	t *testing.T,
	opts ...readmodel.Option[todo, todoKey],
) *readmodel.Store[todo, todoKey] {
	t.Helper()

	return readmodel.New[todo, todoKey](kv.NewMemStore(), opts...)
}

func TestStore_GetSet_Roundtrip(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	ctx := context.Background()

	in := &todo{Title: "write tests", Complete: false}
	if err := store.Set(ctx, "1", in); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Title != in.Title || got.Complete != in.Complete {
		t.Fatalf("roundtrip mismatch: got %+v, want %+v", got, in)
	}
}

func TestStore_Get_MissingReturnsErrNotFound(t *testing.T) {
	t.Parallel()

	store := newStore(t)

	_, err := store.Get(context.Background(), "nope")
	if !errors.Is(err, readmodel.ErrNotFound) {
		t.Fatalf("got err=%v, want readmodel.ErrNotFound", err)
	}
	if !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("ErrNotFound must also satisfy kv.ErrNotFound; got %v", err)
	}
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	ctx := context.Background()

	if err := store.Set(ctx, "1", &todo{Title: "x"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := store.Delete(ctx, "1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := store.Get(ctx, "1"); !errors.Is(err, readmodel.ErrNotFound) {
		t.Fatalf("after Delete, Get err=%v, want ErrNotFound", err)
	}

	// Deleting a missing key is a no-op.
	if err := store.Delete(ctx, "missing"); err != nil {
		t.Fatalf("Delete missing: got %v, want nil", err)
	}
}

func TestStore_Has(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	ctx := context.Background()

	if err := store.Set(ctx, "1", &todo{Title: "x"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if has, _ := store.Has(ctx, "1"); !has {
		t.Fatal("Has(1) = false, want true")
	}

	if has, _ := store.Has(ctx, "2"); has {
		t.Fatal("Has(2) = true, want false")
	}
}

func TestStore_Set_NilValue(t *testing.T) {
	t.Parallel()

	store := newStore(t)

	if err := store.Set(context.Background(), "1", nil); err == nil {
		t.Fatal("Set(nil) = nil, want error")
	}
}

func TestStore_Scan_PrefixFiltering(t *testing.T) {
	t.Parallel()

	// Shared backend, two namespaced stores.
	backend := kv.NewMemStore()
	todos := readmodel.New[todo, todoKey](backend, readmodel.WithKeyPrefix[todo, todoKey]("todos:"))

	ctx := context.Background()

	items := []struct {
		key   todoKey
		title string
	}{
		{"a", "alpha"},
		{"b", "beta"},
		{"c", "gamma"},
	}

	for _, it := range items {
		if err := todos.Set(ctx, it.key, &todo{Title: it.title}); err != nil {
			t.Fatalf("Set %s: %v", it.key, err)
		}
	}

	// Scan all todos (empty sub-prefix).
	got, err := todos.Scan(ctx, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("Scan returned %d items, want 3", len(got))
	}

	// MemStore yields in lexicographic key order: a, b, c → alpha, beta, gamma.
	wantTitles := []string{"alpha", "beta", "gamma"}
	for i, w := range wantTitles {
		if got[i].Title != w {
			t.Fatalf("Scan[%d].Title = %q, want %q", i, got[i].Title, w)
		}
	}

	// Sub-prefix scan: only keys starting with "b".
	gotB, err := todos.Scan(ctx, []byte("b"))
	if err != nil {
		t.Fatalf("Scan(b): %v", err)
	}

	if len(gotB) != 1 || gotB[0].Title != "beta" {
		t.Fatalf("Scan(b) = %+v, want [beta]", gotB)
	}
}

func TestStore_KeyPrefix_NamespacesAcrossStores(t *testing.T) {
	t.Parallel()

	backend := kv.NewMemStore()

	type user struct {
		Name string `json:"name"`
	}

	todos := readmodel.New[todo, todoKey](backend, readmodel.WithKeyPrefix[todo, todoKey]("todos:"))
	users := readmodel.New[user, userKey](backend, readmodel.WithKeyPrefix[user, userKey]("users:"))

	ctx := context.Background()

	_ = todos.Set(ctx, "1", &todo{Title: "shared key 1"})
	_ = users.Set(ctx, "1", &user{Name: "Alice"})

	gotTodo, err := todos.Get(ctx, "1")
	if err != nil {
		t.Fatalf("todos.Get: %v", err)
	}

	if gotTodo.Title != "shared key 1" {
		t.Fatalf("todo title = %q, want %q", gotTodo.Title, "shared key 1")
	}
}

func TestStore_WithCodec_CBOR(t *testing.T) {
	t.Parallel()

	store := readmodel.New[todo, todoKey](
		kv.NewMemStore(),
		readmodel.WithCodec[todo, todoKey](codec.CBORCodec{}),
	)

	ctx := context.Background()
	in := &todo{Title: "cbor roundtrip", Complete: true}

	if err := store.Set(ctx, "1", in); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Title != in.Title || !got.Complete {
		t.Fatalf("CBOR roundtrip mismatch: got %+v", got)
	}
}

func TestStore_WithKeyFunc_CustomEncoding(t *testing.T) {
	t.Parallel()

	store := readmodel.New[todo, todoKey](
		kv.NewMemStore(),
		readmodel.WithKeyFunc[todo, todoKey](func(k todoKey) []byte {
			return []byte("custom-" + k.String())
		}),
	)

	ctx := context.Background()

	if err := store.Set(ctx, "1", &todo{Title: "x"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// The raw key in the backend should reflect the custom encoding.
	backend := store.Backend()
	raw, err := backend.Get([]byte("custom-1"))
	if err != nil {
		t.Fatalf("backend Get: %v", err)
	}

	if len(raw) == 0 {
		t.Fatal("custom key encoding not applied: empty value at custom-1")
	}
}
