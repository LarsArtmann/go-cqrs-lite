package sync

import (
	"encoding/json"
	"testing"
	"time"
)

func TestOperationType_Constants(t *testing.T) {
	if OpCreate != "create" {
		t.Errorf("OpCreate = %q, want %q", OpCreate, "create")
	}

	if OpUpdate != "update" {
		t.Errorf("OpUpdate = %q, want %q", OpUpdate, "update")
	}

	if OpDelete != "delete" {
		t.Errorf("OpDelete = %q, want %q", OpDelete, "delete")
	}
}

func TestOperationType_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		op   OperationType
		want bool
	}{
		{"create is valid", OpCreate, true},
		{"update is valid", OpUpdate, true},
		{"delete is valid", OpDelete, true},
		{"unknown is invalid", OperationType("unknown"), false},
		{"empty is invalid", OperationType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.op.Valid(); got != tt.want {
				t.Errorf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMustNewOperation(t *testing.T) {
	payload := map[string]string{"name": "test"}
	before := time.Now().UTC()

	op := MustNewOperation(OperationID("op-1"), OpCreate, MustParseNodeID("node-a"), payload)

	if op.ID != OperationID("op-1") {
		t.Errorf("ID = %q, want %q", op.ID, "op-1")
	}

	if op.Type != OpCreate {
		t.Errorf("Type = %q, want %q", op.Type, OpCreate)
	}

	if op.NodeID != NodeID("node-a") {
		t.Errorf("NodeID = %q, want %q", op.NodeID, "node-a")
	}

	if op.Timestamp.Before(before) {
		t.Error("Timestamp should be >= creation time")
	}

	if op.VectorClock == nil {
		t.Error("VectorClock should not be nil")
	}

	if len(op.VectorClock) != 0 {
		t.Errorf("VectorClock should be empty, got %d entries", len(op.VectorClock))
	}

	if op.Payload["name"] != "test" {
		t.Errorf("Payload = %v, want name=test", op.Payload)
	}
}

func TestNewOperation_WithDifferentTypes(t *testing.T) {
	t.Run("string payload", func(t *testing.T) {
		op := MustNewOperation(OperationID("op-1"), OpCreate, MustParseNodeID("node-a"), "hello")
		if op.Payload != "hello" {
			t.Errorf("Payload = %q, want %q", op.Payload, "hello")
		}
	})

	t.Run("int payload", func(t *testing.T) {
		op := MustNewOperation(OperationID("op-2"), OpUpdate, MustParseNodeID("node-b"), 42)
		if op.Payload != 42 {
			t.Errorf("Payload = %d, want 42", op.Payload)
		}
	})

	t.Run("struct payload", func(t *testing.T) {
		type Item struct {
			Name string `json:"name"`
		}

		op := MustNewOperation(
			OperationID("op-3"),
			OpDelete,
			MustParseNodeID("node-c"),
			Item{Name: "item1"},
		)
		if op.Payload.Name != "item1" {
			t.Errorf("Payload.Name = %q, want %q", op.Payload.Name, "item1")
		}
	})
}

func TestOperation_Serialize_Deserialize(t *testing.T) {
	type TestPayload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := MustNewOperation(OperationID("op-1"), OpUpdate, MustParseNodeID("node-a"), TestPayload{
		Name:  "test",
		Value: 42,
	})
	original.VectorClock.Increment(NodeID("node-a"))
	original.VectorClock.Increment(NodeID("node-b"))

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Serialize() returned empty data")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("serialized data is not valid JSON: %v", err)
	}

	deserialized, err := DeserializeOperation[TestPayload](data)
	if err != nil {
		t.Fatalf("DeserializeOperation() error: %v", err)
	}

	if deserialized.ID != original.ID {
		t.Errorf("ID = %q, want %q", deserialized.ID, original.ID)
	}

	if deserialized.Type != original.Type {
		t.Errorf("Type = %q, want %q", deserialized.Type, original.Type)
	}

	if deserialized.NodeID != original.NodeID {
		t.Errorf("NodeID = %q, want %q", deserialized.NodeID, original.NodeID)
	}

	if deserialized.Payload.Name != "test" {
		t.Errorf("Payload.Name = %q, want %q", deserialized.Payload.Name, "test")
	}

	if deserialized.Payload.Value != 42 {
		t.Errorf("Payload.Value = %d, want 42", deserialized.Payload.Value)
	}

	if !deserialized.VectorClock.Equal(original.VectorClock) {
		t.Errorf("VectorClock = %v, want %v", deserialized.VectorClock, original.VectorClock)
	}
}

func TestDeserializeOperation_InvalidJSON(t *testing.T) {
	_, err := DeserializeOperation[string]([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestDeserializeOperation_EmptyJSON(t *testing.T) {
	op, err := DeserializeOperation[string]([]byte("{}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if op.ID != "" {
		t.Errorf("expected empty ID, got %q", op.ID)
	}

	if op.Payload != "" {
		t.Errorf("expected empty payload, got %q", op.Payload)
	}
}

func TestOperation_RoundTrip_PreservesAllFields(t *testing.T) {
	type ComplexPayload struct {
		Tags  []string `json:"tags"`
		Count int      `json:"count"`
	}

	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	original := &Operation[ComplexPayload]{
		ID:        OperationID("complex-op"),
		Type:      OpCreate,
		NodeID:    MustParseNodeID("node-x"),
		Timestamp: now,
		VectorClock: VectorClock{
			NodeID("node-x"): 5,
			NodeID("node-y"): 3,
		},
		Payload: ComplexPayload{
			Tags:  []string{"a", "b", "c"},
			Count: 99,
		},
	}

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	result, err := DeserializeOperation[ComplexPayload](data)
	if err != nil {
		t.Fatalf("DeserializeOperation() error: %v", err)
	}

	if result.ID != original.ID {
		t.Errorf("ID mismatch: %q vs %q", result.ID, original.ID)
	}

	if result.Type != original.Type {
		t.Errorf("Type mismatch: %q vs %q", result.Type, original.Type)
	}

	if result.NodeID != original.NodeID {
		t.Errorf("NodeID mismatch: %q vs %q", result.NodeID, original.NodeID)
	}

	if result.Payload.Count != original.Payload.Count {
		t.Errorf("Count mismatch: %d vs %d", result.Payload.Count, original.Payload.Count)
	}

	if len(result.Payload.Tags) != len(original.Payload.Tags) {
		t.Errorf(
			"Tags length mismatch: %d vs %d",
			len(result.Payload.Tags),
			len(original.Payload.Tags),
		)
	}

	if !result.VectorClock.Equal(original.VectorClock) {
		t.Errorf("VectorClock mismatch: %v vs %v", result.VectorClock, original.VectorClock)
	}
}

func TestNewOperation_EmptyID(t *testing.T) {
	t.Parallel()

	_, err := NewOperation(OperationID(""), OpCreate, MustParseNodeID("node-a"), "test")
	if err == nil {
		t.Fatal("expected error for empty operation ID")
	}
}

func TestNewOperation_EmptyNodeID(t *testing.T) {
	t.Parallel()

	_, err := NewOperation(OperationID("op-1"), OpCreate, NodeID(""), "test")
	if err == nil {
		t.Fatal("expected error for empty node ID")
	}
}

func TestNewOperation_InvalidType(t *testing.T) {
	t.Parallel()

	_, err := NewOperation(OperationID("op-1"), OperationType("bogus"), MustParseNodeID("node-a"), "test")
	if err == nil {
		t.Fatal("expected error for invalid operation type")
	}
}

func TestMustNewOperation_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty ID")
		}
	}()

	MustNewOperation(OperationID(""), OpCreate, MustParseNodeID("node-a"), "test")
}
