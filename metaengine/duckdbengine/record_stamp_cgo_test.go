//go:build cgo

package duckdbengine_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestDuckDB_RecordStamping verifies that Record metadata (StreamID, Version)
// is correctly stamped into result fields when using AutoInsert through the
// DuckDB engine. This proves the Record-aware pipeline works on columnar OLAP
// backends, not just SQLite (SQL), Pebble (LSM), or in-memory.
func TestDuckDB_RecordStamping(t *testing.T) {
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
		"duckdb-record-items",
		metaengine.AutoInsert[itemCreated, itemView]("ID"),
	)

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	defer metaengine.DeferClose(eng)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	defer store.Close()

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

	jv, ok := scan.Items[0].(metaengine.JSONValue)
	if !ok {
		t.Fatalf("expected JSONValue, got %T", scan.Items[0])
	}

	var item itemView
	if err := json.Unmarshal(jv, &item); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
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
