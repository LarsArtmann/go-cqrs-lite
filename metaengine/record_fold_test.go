package metaengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestOnRecord_Insert(t *testing.T) {
	t.Parallel()

	type evt struct{ ID, Name string }
	type result struct{ ID, Name, StreamID string }
	type query struct{ ID string }

	q := metaengine.Query[query, result](
		"onrecord_insert",
		metaengine.OnRecord(evt{}, func(rec record.Record, e evt) (string, result) {
			_, sid := rec.StreamID.Split()
			return e.ID, result{ID: e.ID, Name: e.Name, StreamID: sid}
		}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	rec := record.Record{
		Type:       "evt",
		StreamID:   record.NewStreamRef("User", "user-42"),
		StreamType: "User",
		Version:    1,
	}
	payload := evt{ID: "user-42", Name: "Alice"}

	if err := store.ApplyRecord(ctx, rec, payload); err != nil {
		t.Fatalf("ApplyRecord: %v", err)
	}

	got, err := metaengine.ExecuteTyped[query, result](ctx, store, query{ID: "user-42"})
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}
	if got.StreamID != "user-42" {
		t.Errorf("StreamID = %q, want %q", got.StreamID, "user-42")
	}
	if got.Name != "Alice" {
		t.Errorf("Name = %q, want %q", got.Name, "Alice")
	}
}

func TestOnRecord_LegacyApplyStillWorks(t *testing.T) {
	t.Parallel()

	type evt struct{ ID string }
	type result struct{ ID string }
	type query struct{}

	q := metaengine.Query[query, result](
		"onrecord_legacy",
		metaengine.On(evt{}, func(e evt) (string, result) {
			return e.ID, result{ID: e.ID}
		}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "evt", evt{ID: "legacy-1"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	rec := record.Record{
		Type:    "evt",
		StreamID: record.NewStreamRef("User", "user-1"),
		Version: 2,
	}
	if err := store.ApplyRecord(ctx, rec, evt{ID: "record-1"}); err != nil {
		t.Fatalf("ApplyRecord: %v", err)
	}
}

func TestOnRecord_Count(t *testing.T) {
	t.Parallel()

	type evt struct{ Status string }
	type query struct{}

	q := metaengine.Query[query, map[string]int64](
		"onrecord_count",
		metaengine.OnRecord(evt{}, func(rec record.Record, e evt) metaengine.Delta {
			return metaengine.Delta{e.Status: +1}
		}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	rec := record.Record{
		Type:    "evt",
		StreamID: record.NewStreamRef("User", "u1"),
		Version: 1,
	}
	if err := store.ApplyRecord(ctx, rec, evt{Status: "open"}); err != nil {
		t.Fatalf("ApplyRecord: %v", err)
	}

	result, err := metaengine.ExecuteTyped[query, map[string]int64]](
		ctx, store, query{},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}
	if result["open"] != 1 {
		t.Errorf("expected open=1, got %v", result)
	}
}
