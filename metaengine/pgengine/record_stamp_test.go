package pgengine_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestPostgres_RecordStamping verifies that Record metadata (StreamID, Version)
// is correctly stamped into result fields when using AutoInsert through the
// Postgres engine. This proves the Record-aware pipeline works on PG JSONB +
// B-tree backends, completing cross-engine coverage alongside SQLite, Pebble,
// and DuckDB.
func TestPostgres_RecordStamping(t *testing.T) {
	t.Parallel()

	type itemCreated struct {
		ID   string
		Name string
	}

	type itemView struct {
		ID       string
		Name     string
		StreamID string
		Version  int64
	}

	type itemQuery struct{}

	q := metaengine.Query[itemQuery, itemView](
		"pg-record-items",
		metaengine.AutoInsert[itemCreated, itemView]("ID"),
	)

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	defer eng.Close()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	ctx := context.Background()

	rec := record.Record{
		Type:       "itemCreated",
		Payload:    []byte(`{"ID":"item-1","Name":"test"}`),
		StreamID:   record.NewStreamRef("Item", "stream-abc"),
		StreamType: "Item",
		Version:    3,
		MetaData: record.CommonMetadata{
			CorrelationID: "corr-123",
			ActorID:       "user-456",
		},
	}

	err = store.ApplyRecord(ctx, rec, itemCreated{ID: "item-1", Name: "test"})
	if err != nil {
		t.Fatalf("ApplyRecord: %v", err)
	}

	result, err := store.Execute(itemQuery{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	scan, ok := result.(metaengine.ScanResult)
	if !ok {
		t.Fatalf("expected ScanResult, got %T", result)
	}

	if len(scan.Items) == 0 {
		t.Fatal("no items in store")
	}

	var item itemView
	switch v := scan.Items[0].(type) {
	case metaengine.JSONValue:
		if err := json.Unmarshal(v, &item); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
	case map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}

		if err := json.Unmarshal(data, &item); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
	default:
		t.Fatalf("unexpected item type: %T", v)
	}

	if item.StreamID != "Item/stream-abc" {
		t.Errorf("StreamID: got %q, want %q", item.StreamID, "Item/stream-abc")
	}

	if item.Version != 3 {
		t.Errorf("Version: got %d, want 3", item.Version)
	}

	if item.ID != "item-1" {
		t.Errorf("ID: got %q, want %q", item.ID, "item-1")
	}
}
