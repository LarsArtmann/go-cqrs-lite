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

func TestNewOperation(t *testing.T) {
	payload := map[string]string{"name": "test"}
	before := time.Now().UTC()

	op := NewOperation("op-1", OpCreate, "node-a", payload)

	if op.ID != "op-1" {
		t.Errorf("ID = %q, want %q", op.ID, "op-1")
	}

	if op.Type != OpCreate {
		t.Errorf("Type = %q, want %q", op.Type, OpCreate)
	}

	if op.NodeID != "node-a" {
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
		op := NewOperation("op-1", OpCreate, "node-a", "hello")
		if op.Payload != "hello" {
			t.Errorf("Payload = %q, want %q", op.Payload, "hello")
		}
	})

	t.Run("int payload", func(t *testing.T) {
		op := NewOperation("op-2", OpUpdate, "node-b", 42)
		if op.Payload != 42 {
			t.Errorf("Payload = %d, want 42", op.Payload)
		}
	})

	t.Run("struct payload", func(t *testing.T) {
		type Item struct {
			Name string `json:"name"`
		}

		op := NewOperation("op-3", OpDelete, "node-c", Item{Name: "item1"})
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

	original := NewOperation("op-1", OpUpdate, "node-a", TestPayload{
		Name:  "test",
		Value: 42,
	})
	original.VectorClock.Increment("node-a")
	original.VectorClock.Increment("node-b")

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
		ID:        "complex-op",
		Type:      OpCreate,
		NodeID:    "node-x",
		Timestamp: now,
		VectorClock: VectorClock{
			"node-x": 5,
			"node-y": 3,
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
