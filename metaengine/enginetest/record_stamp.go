package enginetest

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// stampItemCreated is the event payload used by RunRecordStampTest.
type stampItemCreated struct {
	ID   string
	Name string
}

// stampItemView is the result type whose StreamID and Version fields receive
// Record metadata via auto-stamping.
type stampItemView struct {
	ID       string
	Name     string
	StreamID string
	Version  int64
}

type stampItemQuery struct{}

// RunRecordStampTest verifies that Record metadata (StreamID, Version) is
// correctly stamped into result fields when using AutoInsert through the
// given engine. This proves the Record-aware pipeline works across backends
// (Memory, SQLite, Pebble, DuckDB, Postgres, Badger, ...).
//
// The caller is responsible for closing the engine.
func RunRecordStampTest(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	q := metaengine.Query[stampItemQuery, stampItemView](
		"record-stamp-"+engineName(eng),
		metaengine.AutoInsert[stampItemCreated, stampItemView]("ID"),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	ctx := context.Background()

	rec := record.Record{
		Type:       "stampItemCreated",
		Payload:    []byte(`{"ID":"item-1","Name":"test"}`),
		StreamID:   record.NewStreamRef("Item", "stream-abc"),
		StreamType: "Item",
		Version:    3,
		MetaData: record.CommonMetadata{
			CorrelationID: id.NewCorrelationID(),
			ActorID:       id.NewSystemActor("test"),
		},
	}

	if err := store.ApplyRecord(
		ctx,
		rec,
		stampItemCreated{ID: "item-1", Name: "test"},
	); err != nil {
		t.Fatalf("ApplyRecord: %v", err)
	}

	result, err := store.Execute(stampItemQuery{})
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

	var item stampItemView

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
