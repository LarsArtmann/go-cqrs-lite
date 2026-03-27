package id

import (
	"testing"

	"github.com/go-json-experiment/json"
)

func TestNew(t *testing.T) {
	t.Parallel()

	id := New[AggregateID]()

	if id.IsEmpty() {
		t.Error("New() should not return empty ID")
	}
	if !id.IsValid() {
		t.Error("New() should return valid ID")
	}
	if len(id.String()) != 36 { // UUID v4 length
		t.Errorf("ID string length = %d, want 36", len(id.String()))
	}
}

func TestNewWithPrefix(t *testing.T) {
	t.Parallel()

	id := NewWithPrefix[AggregateID]("user")

	if id.IsEmpty() {
		t.Error("NewWithPrefix() should not return empty ID")
	}
	if len(id.String()) < 6 {
		t.Error("NewWithPrefix() should return ID longer than prefix")
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("valid string", func(t *testing.T) {
		id, err := Parse[AggregateID]("test-id-123")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if id.String() != "test-id-123" {
			t.Errorf("Parse() = %q, want %q", id.String(), "test-id-123")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := Parse[AggregateID]("")
		if err == nil {
			t.Error("Parse() should error on empty string")
		}
	})
}

func TestMustParse(t *testing.T) {
	t.Parallel()

	t.Run("valid string", func(t *testing.T) {
		id := MustParse[AggregateID]("test-id")
		if id.String() != "test-id" {
			t.Errorf("MustParse() = %q, want %q", id.String(), "test-id")
		}
	})

	t.Run("empty string panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustParse() should panic on empty string")
			}
		}()
		_ = MustParse[AggregateID]("")
	})
}

func TestString(t *testing.T) {
	t.Parallel()

	id := MustParse[AggregateID]("my-id")
	if id.String() != "my-id" {
		t.Errorf("String() = %q, want %q", id.String(), "my-id")
	}
}

func TestIsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty ID", func(t *testing.T) {
		var id AggregateID
		if !id.IsEmpty() {
			t.Error("zero value ID should be empty")
		}
	})

	t.Run("non-empty ID", func(t *testing.T) {
		id := MustParse[AggregateID]("test")
		if id.IsEmpty() {
			t.Error("parsed ID should not be empty")
		}
	})
}

func TestIsValid(t *testing.T) {
	t.Parallel()

	t.Run("empty ID", func(t *testing.T) {
		var id AggregateID
		if id.IsValid() {
			t.Error("zero value ID should not be valid")
		}
	})

	t.Run("non-empty ID", func(t *testing.T) {
		id := MustParse[AggregateID]("test")
		if !id.IsValid() {
			t.Error("parsed ID should be valid")
		}
	})
}

func TestJSON(t *testing.T) {
	t.Parallel()

	t.Run("marshal", func(t *testing.T) {
		id := MustParse[AggregateID]("test-id")
		data, err := json.Marshal(id)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		if string(data) != `"test-id"` {
			t.Errorf("Marshal() = %s, want %q", data, `"test-id"`)
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		var id AggregateID
		err := json.Unmarshal([]byte(`"test-id"`), &id)
		if err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if id.String() != "test-id" {
			t.Errorf("Unmarshal() = %q, want %q", id.String(), "test-id")
		}
	})

	t.Run("unmarshal empty string", func(t *testing.T) {
		var id AggregateID
		err := json.Unmarshal([]byte(`""`), &id)
		if err == nil {
			t.Error("Unmarshal() should error on empty string")
		}
	})
}

func TestTypeSafety(t *testing.T) {
	t.Parallel()

	type userBrand struct{}
	type orderBrand struct{}

	type UserID = Of[userBrand]
	type OrderID = Of[orderBrand]

	userID := New[UserID]()
	orderID := New[OrderID]()

	// These should compile as different types
	_ = userID.String()
	_ = orderID.String()

	// This would not compile (uncomment to verify):
	// userID = orderID
}

func TestAggregateID(t *testing.T) {
	t.Parallel()

	id := NewAggregateID()
	if id.IsEmpty() {
		t.Error("NewAggregateID() should not return empty ID")
	}

	parsed, err := ParseAggregateID("test-agg-id")
	if err != nil {
		t.Fatalf("ParseAggregateID() error = %v", err)
	}
	if parsed.String() != "test-agg-id" {
		t.Errorf("ParseAggregateID() = %q, want %q", parsed.String(), "test-agg-id")
	}
}

func TestEventID(t *testing.T) {
	t.Parallel()

	id := NewEventID()
	if id.IsEmpty() {
		t.Error("NewEventID() should not return empty ID")
	}
}

func TestUserID(t *testing.T) {
	t.Parallel()

	id := NewUserID()
	if id.IsEmpty() {
		t.Error("NewUserID() should not return empty ID")
	}
}
