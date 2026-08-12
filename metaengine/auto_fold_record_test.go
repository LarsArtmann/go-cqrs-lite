package metaengine_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestAutoFold_RecordAware_Insert verifies that AutoInsert stamps Record
// metadata (StreamID, Version, CorrelationID) into result fields that are not
// covered by event field mappings.
func TestAutoFold_RecordAware_Insert(t *testing.T) {
	t.Parallel()

	type productCreated struct {
		ID    string
		Name  string
		Price int64
	}

	type productView struct {
		ID            string
		Name          string
		Price         int64
		StreamID      string // not in event → stamped from Record
		Version       int64  // not in event → stamped from Record
		CorrelationID string // not in event → stamped from Record
	}

	q := metaengine.Query[struct {
		ID string
	}, productView](
		"products",
		metaengine.AutoInsert[productCreated, productView]("ID"),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	rec := record.Record{
		Type:       "productCreated",
		Payload:    []byte(`{"ID":"p1","Name":"Widget","Price":999}`),
		StreamID:   record.NewStreamRef("Product", "prod-123"),
		StreamType: "Product",
		Version:    5,
		MetaData: record.CommonMetadata{
			CorrelationID: "corr-abc",
			ActorID:       "user-xyz",
		},
	}

	var payload productCreated
	if err := json.Unmarshal(rec.Payload, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if err := store.ApplyRecord(context.Background(), rec, payload); err != nil {
		t.Fatalf("ApplyRecord: %v", err)
	}

	result, err := metaengine.ExecuteTyped[struct {
		ID string
	}, productView](context.Background(), store, struct {
		ID string
	}{"p1"})
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result.StreamID != "Product/prod-123" {
		t.Errorf("StreamID = %q, want %q", result.StreamID, "Product/prod-123")
	}

	if result.Version != 5 {
		t.Errorf("Version = %d, want 5", result.Version)
	}

	if result.CorrelationID != "corr-abc" {
		t.Errorf("CorrelationID = %q, want %q", result.CorrelationID, "corr-abc")
	}

	if result.Name != "Widget" {
		t.Errorf("Name = %q, want Widget (event mapping)", result.Name)
	}

	if result.Price != 999 {
		t.Errorf("Price = %d, want 999 (event mapping)", result.Price)
	}
}

// TestAutoFold_RecordAware_Update verifies that AutoUpdate stamps Record
// metadata into result fields during partial updates.
func TestAutoFold_RecordAware_Update(t *testing.T) {
	t.Parallel()

	type productCreated struct {
		ID    string
		Name  string
		Price int64
	}

	type productUpdated struct {
		ID    string
		Name  string
		Price int64
	}

	type productView struct {
		ID       string
		Name     string
		Price    int64
		StreamID string // stamped from Record
		Version  int64  // stamped from Record
	}

	q := metaengine.Query[struct {
		ID string
	}, productView](
		"products",
		metaengine.AutoInsert[productCreated, productView]("ID"),
		metaengine.AutoUpdate[productUpdated, productView]("ID"),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	// Insert
	insertRec := record.Record{
		Type:       "productCreated",
		StreamID:   record.NewStreamRef("Product", "prod-1"),
		StreamType: "Product",
		Version:    1,
	}

	if err := store.ApplyRecord(context.Background(), insertRec, productCreated{
		ID: "p1", Name: "Original", Price: 100,
	}); err != nil {
		t.Fatalf("ApplyRecord insert: %v", err)
	}

	// Update — new StreamID + Version
	updateRec := record.Record{
		Type:       "productUpdated",
		StreamID:   record.NewStreamRef("Product", "prod-1"),
		StreamType: "Product",
		Version:    2,
	}

	if err := store.ApplyRecord(context.Background(), updateRec, productUpdated{
		ID: "p1", Name: "Updated", Price: 200,
	}); err != nil {
		t.Fatalf("ApplyRecord update: %v", err)
	}

	result, err := metaengine.ExecuteTyped[struct {
		ID string
	}, productView](context.Background(), store, struct {
		ID string
	}{"p1"})
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result.Name != "Updated" {
		t.Errorf("Name = %q, want Updated", result.Name)
	}

	if result.Price != 200 {
		t.Errorf("Price = %d, want 200", result.Price)
	}

	if result.Version != 2 {
		t.Errorf("Version = %d, want 2 (updated by record)", result.Version)
	}
}

// TestAutoFold_RecordAware_NoMetadataFields verifies that auto-folds work
// correctly when the result type has NO metadata fields — no stamping occurs,
// behavior is identical to the pre-Record-aware path.
func TestAutoFold_RecordAware_NoMetadataFields(t *testing.T) {
	t.Parallel()

	type simpleEvent struct {
		ID   string
		Name string
	}

	type simpleView struct {
		ID   string
		Name string
	}

	q := metaengine.Query[struct {
		ID string
	}, simpleView](
		"simple",
		metaengine.AutoInsert[simpleEvent, simpleView]("ID"),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	rec := record.Record{
		Type:       "simpleEvent",
		StreamID:   record.NewStreamRef("Thing", "t1"),
		StreamType: "Thing",
		Version:    1,
	}

	if err := store.ApplyRecord(context.Background(), rec, simpleEvent{
		ID: "s1", Name: "Test",
	}); err != nil {
		t.Fatalf("ApplyRecord: %v", err)
	}

	result, err := metaengine.ExecuteTyped[struct {
		ID string
	}, simpleView](context.Background(), store, struct {
		ID string
	}{"s1"})
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result.Name != "Test" {
		t.Errorf("Name = %q, want Test", result.Name)
	}
}
