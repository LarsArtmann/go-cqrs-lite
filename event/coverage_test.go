package event_test

import (
	"testing"

	codecpkg "github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestAggregateRef_IsZero(t *testing.T) {
	t.Parallel()

	var ref event.AggregateRef

	if !ref.IsZero() {
		t.Error("expected zero AggregateRef to be zero")
	}

	ref = event.NewAggregateRef("User", id.NewAggregateID())

	if ref.IsZero() {
		t.Error("expected non-zero AggregateRef to not be zero")
	}
}

func TestAggregateRef_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		ref := event.NewAggregateRef("User", id.NewAggregateID())

		err := ref.Validate()
		if err != nil {
			t.Fatalf("expected valid ref, got: %v", err)
		}
	})

	t.Run("empty type", func(t *testing.T) {
		t.Parallel()

		ref := event.NewAggregateRef("", id.NewAggregateID())

		err := ref.Validate()
		if err == nil {
			t.Fatal("expected error for empty type")
		}
	})

	t.Run("empty ID", func(t *testing.T) {
		t.Parallel()

		ref := event.NewAggregateRef("User", id.AggregateID{})

		err := ref.Validate()
		if err == nil {
			t.Fatal("expected error for empty ID")
		}
	})
}

func TestCheckpoint_String(t *testing.T) {
	t.Parallel()

	cp := event.Checkpoint{EventID: id.NewEventID()}

	s := cp.String()

	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestVersion_Decrement(t *testing.T) {
	t.Parallel()

	v := event.Version(5)

	if v.Decrement() != 4 {
		t.Errorf("Decrement: got %d, want 4", v.Decrement())
	}

	if v != 5 {
		t.Errorf("original mutated: got %d, want 5", v)
	}
}

func TestSchemaVersion_Decrement(t *testing.T) {
	t.Parallel()

	sv := event.SchemaVersion(3)

	if sv.Decrement() != 2 {
		t.Errorf("Decrement: got %d, want 2", sv.Decrement())
	}
}

func TestSchemaVersion_IsPositive(t *testing.T) {
	t.Parallel()

	if event.SchemaVersion(0).IsPositive() {
		t.Error("0 should not be positive")
	}

	if !event.SchemaVersion(1).IsPositive() {
		t.Error("1 should be positive")
	}
}

func TestSchemaVersion_Cmp(t *testing.T) {
	t.Parallel()

	sv := event.SchemaVersion(5)

	if sv.Cmp(3) != 1 {
		t.Errorf("5.Cmp(3) = %d, want 1", sv.Cmp(3))
	}

	if sv.Cmp(5) != 0 {
		t.Errorf("5.Cmp(5) = %d, want 0", sv.Cmp(5))
	}

	if sv.Cmp(10) != -1 {
		t.Errorf("5.Cmp(10) = %d, want -1", sv.Cmp(10))
	}
}

func TestImmutableEvent_String(t *testing.T) {
	t.Parallel()

	evt, err := event.NewEvent(
		event.Type("user.created"), id.NewAggregateID(),
		event.AggregateType("User"), event.Version(1),
		[]byte(`{"name":"test"}`),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	s := evt.String()

	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestWrapTransient(t *testing.T) {
	t.Parallel()

	inner := event.NewRejection("test.reject", "rejected")
	wrapped := event.WrapTransient(inner, "test.transient", "test wrap")

	if wrapped.Family() != event.Transient {
		t.Errorf("Family = %s, want Transient", wrapped.Family())
	}

	if !event.IsRetryable(wrapped) {
		t.Error("Transient should be retryable")
	}
}

func TestWithNewCodec(t *testing.T) {
	t.Parallel()

	c := codecpkg.JSONCodec{}

	evt, err := event.New(
		"test", id.NewAggregateID(), "Test", 1,
		map[string]string{"key": "val"},
		event.WithNewCodec(c),
	)
	if err != nil {
		t.Fatalf("WithNewCodec: %v", err)
	}

	if evt.Encoding() != c.Encoding() {
		t.Errorf("Encoding = %s, want %s", evt.Encoding(), c.Encoding())
	}
}
