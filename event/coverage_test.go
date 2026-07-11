package event_test

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	codecpkg "github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestAggregateRef_IsZero(t *testing.T) {
	t.Parallel()

	var ref id.AggregateRef

	if !ref.IsZero() {
		t.Error("expected zero id.AggregateRef to be zero")
	}

	ref = id.NewAggregateRef("User", id.NewAggregateID())

	if ref.IsZero() {
		t.Error("expected non-zero id.AggregateRef to not be zero")
	}
}

func TestAggregateRef_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		ref := id.NewAggregateRef("User", id.NewAggregateID())

		err := ref.Validate()
		if err != nil {
			t.Fatalf("expected valid ref, got: %v", err)
		}
	})

	t.Run("empty type", func(t *testing.T) {
		t.Parallel()

		ref := id.NewAggregateRef("", id.NewAggregateID())

		err := ref.Validate()
		if err == nil {
			t.Fatal("expected error for empty type")
		}
	})

	t.Run("empty ID", func(t *testing.T) {
		t.Parallel()

		ref := id.NewAggregateRef("User", id.AggregateID{})

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

	dec, _ := v.Decrement()
	if dec != 4 {
		t.Errorf("Decrement: got %d, want 4", dec)
	}

	if v != 5 {
		t.Errorf("original mutated: got %d, want 5", v)
	}
}

func TestSchemaVersion_Decrement(t *testing.T) {
	t.Parallel()

	sv := event.SchemaVersion(3)

	decSV, _ := sv.Decrement()
	if decSV != 2 {
		t.Errorf("Decrement: got %d, want 2", decSV)
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
		id.AggregateType("User"), event.Version(1),
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

	inner := errorfamily.NewRejection("test.reject", "rejected")
	wrapped := errorfamily.WrapTransient(inner, "test.transient", "test wrap")

	if wrapped.Family() != errorfamily.Transient {
		t.Errorf("Family = %s, want Transient", wrapped.Family())
	}

	if !errorfamily.IsRetryable(wrapped) {
		t.Error("Transient should be retryable")
	}
}

func TestWithCodec(t *testing.T) {
	t.Parallel()

	c := codecpkg.JSONCodec{}

	evt, err := event.New(
		"test", id.NewAggregateID(), "Test", 1,
		map[string]string{"key": "val"},
		event.WithCodec(c),
	)
	if err != nil {
		t.Fatalf("WithCodec: %v", err)
	}

	if evt.Encoding() != c.Encoding() {
		t.Errorf("Encoding = %s, want %s", evt.Encoding(), c.Encoding())
	}
}
