package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

type listUsersPayload struct {
	Filter string `json:"filter"`
	Limit  int    `json:"limit"`
}

func TestTypedQueryStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryQueryStore()

	ts := query.NewTypedQueryStore[listUsersPayload](store, codec.JSONCodec{})

	err := ts.SaveQuery(ctx, query.TypedQuery[listUsersPayload]{
		Type:    "user.list",
		Payload: listUsersPayload{Filter: "active", Limit: 10},
	})
	if err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}

	loaded, err := ts.LoadQueries(ctx, time.Time{})
	if err != nil {
		t.Fatalf("LoadQueries: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 query, got %d", len(loaded))
	}

	if loaded[0].Payload.Filter != "active" {
		t.Errorf("Filter = %q, want %q", loaded[0].Payload.Filter, "active")
	}

	if loaded[0].Payload.Limit != 10 {
		t.Errorf("Limit = %d, want %d", loaded[0].Payload.Limit, 10)
	}

	if loaded[0].Type != "user.list" {
		t.Errorf("Type = %q, want %q", loaded[0].Type, "user.list")
	}
}

func TestTypedQueryStore_NilCodecDefaultsToJSON(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryQueryStore()

	ts := query.NewTypedQueryStore[listUsersPayload](store, nil)

	err := ts.SaveQuery(ctx, query.TypedQuery[listUsersPayload]{
		Type:    "user.list",
		Payload: listUsersPayload{Filter: "nil codec"},
	})
	if err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}

	loaded, err := ts.LoadQueries(ctx, time.Time{})
	if err != nil {
		t.Fatalf("LoadQueries: %v", err)
	}

	if loaded[0].Payload.Filter != "nil codec" {
		t.Errorf("Filter = %q, want %q", loaded[0].Payload.Filter, "nil codec")
	}
}
