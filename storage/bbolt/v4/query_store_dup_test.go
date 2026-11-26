package bbolt

import (
	"context"
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// A duplicate request ID is the caller's mistake: it must surface as
// Conflict (as on pebble), not as Infrastructure.
func TestQueryStore_SaveQueryDuplicateIsConflict(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newTestBackend(t).QueryStore()

	q, err := query.NewPersistedQuery("user.search", []byte(`{}`),
		query.WithQueryID(id.NewRequestID()))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.SaveQuery(ctx, q); err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}

	err = store.SaveQuery(ctx, q)
	if !errors.Is(err, query.ErrDuplicateQuery) {
		t.Fatalf("duplicate SaveQuery: err = %v, want ErrDuplicateQuery", err)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Conflict {
		t.Fatalf("duplicate SaveQuery: family = %v, want Conflict (err: %v)", family, err)
	}
}
