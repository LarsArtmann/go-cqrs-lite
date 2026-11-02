package compat

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func TestConflictsIdempotentUnion(t *testing.T) {
	t.Parallel()

	// Merging a catalog with itself, or with a copy, must not report
	// divergence: an idempotent union is the normal hub case.
	cat := catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField))

	if conflicts := Conflicts(cat, cat); len(conflicts) != 0 {
		t.Fatalf("self-union must be conflict-free, got %v", conflicts)
	}

	if conflicts := Conflicts(cat, catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField))); len(conflicts) != 0 {
		t.Fatalf("identical declarations must be conflict-free, got %v", conflicts)
	}
}

func TestConflictsVersionMismatch(t *testing.T) {
	t.Parallel()

	left := catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField))
	right := catalogWithEvents(eventMsg("order_placed", "2.0.0", oneField))

	conflicts := Conflicts(left, right)

	if len(conflicts) != 1 {
		t.Fatalf("want one conflict, got %v", conflicts)
	}

	if conflicts[0].Kind != ConflictVersionMismatch {
		t.Fatalf("kind = %q, want %q", conflicts[0].Kind, ConflictVersionMismatch)
	}

	if conflicts[0].Path != "order_placed" {
		t.Fatalf("path = %q, want the message ID", conflicts[0].Path)
	}
}

func TestConflictsSchemaMismatch(t *testing.T) {
	t.Parallel()

	left := catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField))
	right := catalogWithEvents(eventMsg("order_placed", "1.0.0", twoFields))

	conflicts := Conflicts(left, right)

	if len(conflicts) != 1 || conflicts[0].Kind != ConflictSchemaMismatch {
		t.Fatalf("want one schema-mismatch conflict, got %v", conflicts)
	}
}

func TestConflictsKindMismatch(t *testing.T) {
	t.Parallel()

	left := catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField))
	right := catalogWithEvents(catalog.Message{
		Kind:      catalog.CommandMessage,
		ID:        "order_placed",
		Name:      "order_placed",
		Version:   "1.0.0",
		Direction: catalog.Receives,
		Schema:    oneField,
	})

	conflicts := Conflicts(left, right)

	if len(conflicts) != 1 || conflicts[0].Kind != ConflictKindMismatch {
		t.Fatalf("want one kind-mismatch conflict, got %v", conflicts)
	}
}

func TestConflictsServiceMismatch(t *testing.T) {
	t.Parallel()

	left := catalogWithEvents()
	right := catalogWithEvents()
	right.Services[0].Version = "9.9.9"

	conflicts := Conflicts(left, right)

	if len(conflicts) != 1 || conflicts[0].Kind != ConflictServiceMismatch {
		t.Fatalf("want one service-metadata-mismatch conflict, got %v", conflicts)
	}

	if conflicts[0].String() == "" {
		t.Fatal("Conflict.String() must not be empty")
	}
}

func TestConflictsDisjointAndNil(t *testing.T) {
	t.Parallel()

	left := catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField))
	right := catalogWithEvents(eventMsg("order_shipped", "1.0.0", oneField))

	if conflicts := Conflicts(left, right); len(conflicts) != 0 {
		t.Fatalf("disjoint catalogs cannot conflict, got %v", conflicts)
	}

	if conflicts := Conflicts(nil, nil); len(conflicts) != 0 {
		t.Fatalf("nil catalogs cannot conflict, got %v", conflicts)
	}
}
