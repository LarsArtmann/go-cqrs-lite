package id

import (
	"encoding"
	"fmt"
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

	if len(id.String()) != 36 {
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
		t.Parallel()

		id, err := Parse[AggregateID]("test-id-123")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if id.String() != "test-id-123" {
			t.Errorf("Parse() = %q, want %q", id.String(), "test-id-123")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()

		_, err := Parse[AggregateID]("")
		if err == nil {
			t.Error("Parse() should error on empty string")
		}
	})
}

func TestMustParse(t *testing.T) {
	t.Parallel()

	t.Run("valid string", func(t *testing.T) {
		t.Parallel()

		id := MustParse[AggregateID]("test-id")
		if id.String() != "test-id" {
			t.Errorf("MustParse() = %q, want %q", id.String(), "test-id")
		}
	})

	t.Run("empty string panics", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		var id AggregateID
		if !id.IsEmpty() {
			t.Error("zero value ID should be empty")
		}
	})

	t.Run("non-empty ID", func(t *testing.T) {
		t.Parallel()

		id := MustParse[AggregateID]("test")
		if id.IsEmpty() {
			t.Error("parsed ID should not be empty")
		}
	})
}

func TestIsValid(t *testing.T) {
	t.Parallel()

	t.Run("empty ID", func(t *testing.T) {
		t.Parallel()

		var id AggregateID
		if id.IsValid() {
			t.Error("zero value ID should not be valid")
		}
	})

	t.Run("non-empty ID", func(t *testing.T) {
		t.Parallel()

		id := MustParse[AggregateID]("test")
		if !id.IsValid() {
			t.Error("parsed ID should be valid")
		}
	})
}

func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("equal IDs", func(t *testing.T) {
		t.Parallel()

		a := MustParse[AggregateID]("same")

		b := MustParse[AggregateID]("same")
		if !a.Equal(b) {
			t.Error("equal IDs should be equal")
		}
	})

	t.Run("different IDs", func(t *testing.T) {
		t.Parallel()

		a := MustParse[AggregateID]("a")

		b := MustParse[AggregateID]("b")
		if a.Equal(b) {
			t.Error("different IDs should not be equal")
		}
	})

	t.Run("empty IDs", func(t *testing.T) {
		t.Parallel()

		var a, b AggregateID
		if !a.Equal(b) {
			t.Error("empty IDs should be equal")
		}
	})
}

func TestCompare(t *testing.T) {
	t.Parallel()

	t.Run("less than", func(t *testing.T) {
		t.Parallel()

		a := MustParse[AggregateID]("a")

		b := MustParse[AggregateID]("b")
		if a.Compare(b) != -1 {
			t.Error("a should be less than b")
		}
	})

	t.Run("equal", func(t *testing.T) {
		t.Parallel()

		a := MustParse[AggregateID]("same")

		b := MustParse[AggregateID]("same")
		if a.Compare(b) != 0 {
			t.Error("same IDs should compare equal")
		}
	})

	t.Run("greater than", func(t *testing.T) {
		t.Parallel()

		a := MustParse[AggregateID]("b")

		b := MustParse[AggregateID]("a")
		if a.Compare(b) != 1 {
			t.Error("b should be greater than a")
		}
	})
}

func TestOr(t *testing.T) {
	t.Parallel()

	t.Run("non-empty returns self", func(t *testing.T) {
		t.Parallel()

		id := MustParseAggregateID("primary")

		fallback := MustParseAggregateID("fallback")
		if result := id.Or(fallback); result != id {
			t.Error("non-empty ID should return self")
		}
	})

	t.Run("empty returns fallback", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		fallback := MustParseAggregateID("fallback")
		if result := id.Or(fallback); result != fallback {
			t.Error("empty ID should return fallback")
		}
	})
}

func TestReset(t *testing.T) {
	t.Parallel()

	id := MustParse[AggregateID]("test")
	if id.IsEmpty() {
		t.Error("ID should not be empty before reset")
	}

	id.Reset()

	if !id.IsEmpty() {
		t.Error("ID should be empty after reset")
	}
}

func TestGoString(t *testing.T) {
	t.Parallel()

	id := MustParse[AggregateID]("test")

	gs := id.GoString()
	if gs == "" {
		t.Error("GoString() should not be empty")
	}
}

func TestFormat(t *testing.T) {
	t.Parallel()

	id := MustParse[AggregateID]("test")

	tests := []struct {
		format string
		want   string
	}{
		{"%v", "test"},
		{"%s", "test"},
		{"%q", `"test"`},
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			t.Parallel()

			if got := fmt.Sprintf(tc.format, id); got != tc.want {
				t.Errorf("%s = %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}

func TestJSON(t *testing.T) {
	t.Parallel()

	t.Run("marshal", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		var id AggregateID

		err := json.Unmarshal([]byte(`"test-id"`), &id)
		if err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if id.String() != "test-id" {
			t.Errorf("Unmarshal() = %q, want %q", id.String(), "test-id")
		}
	})

	t.Run("marshal null", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		data, err := json.Marshal(id)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}

		if string(data) != "null" {
			t.Errorf("Marshal() empty = %s, want null", data)
		}
	})

	t.Run("unmarshal null", func(t *testing.T) {
		t.Parallel()

		id := MustParse[AggregateID]("existing")

		err := json.Unmarshal([]byte("null"), &id)
		if err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if !id.IsEmpty() {
			t.Error("Unmarshal(null) should result in empty ID")
		}
	})

	t.Run("unmarshal empty string", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := json.Unmarshal([]byte(`""`), &id)
		if err == nil {
			t.Error("Unmarshal() should error on empty string")
		}
	})

	t.Run("roundtrip", func(t *testing.T) {
		t.Parallel()

		original := New[AggregateID]()

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}

		var restored AggregateID
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if original.String() != restored.String() {
			t.Errorf("roundtrip: %q != %q", original, restored)
		}
	})
}

func TestBinaryEncoding(t *testing.T) {
	t.Parallel()

	t.Run("marshal", func(t *testing.T) {
		t.Parallel()

		id := MustParse[AggregateID]("test-binary")

		data, err := id.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary() error = %v", err)
		}

		if string(data) != "test-binary" {
			t.Errorf("MarshalBinary() = %q, want %q", string(data), "test-binary")
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := id.UnmarshalBinary([]byte("test-binary"))
		if err != nil {
			t.Fatalf("UnmarshalBinary() error = %v", err)
		}

		if id.String() != "test-binary" {
			t.Errorf("UnmarshalBinary() = %q, want %q", id.String(), "test-binary")
		}
	})

	t.Run("unmarshal empty errors", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := id.UnmarshalBinary([]byte(""))
		if err == nil {
			t.Error("UnmarshalBinary() should error on empty")
		}
	})

	t.Run("interface compliance", func(t *testing.T) {
		t.Parallel()

		var (
			_ encoding.BinaryMarshaler   = AggregateID("")
			_ encoding.BinaryUnmarshaler = (*AggregateID)(nil)
		)
	})
}

func TestTextEncoding(t *testing.T) {
	t.Parallel()

	t.Run("marshal", func(t *testing.T) {
		t.Parallel()

		id := MustParse[AggregateID]("test-text")

		data, err := id.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText() error = %v", err)
		}

		if string(data) != "test-text" {
			t.Errorf("MarshalText() = %q, want %q", string(data), "test-text")
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := id.UnmarshalText([]byte("test-text"))
		if err != nil {
			t.Fatalf("UnmarshalText() error = %v", err)
		}

		if id.String() != "test-text" {
			t.Errorf("UnmarshalText() = %q, want %q", id.String(), "test-text")
		}
	})

	t.Run("unmarshal empty errors", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := id.UnmarshalText([]byte(""))
		if err == nil {
			t.Error("UnmarshalText() should error on empty")
		}
	})

	t.Run("interface compliance", func(t *testing.T) {
		t.Parallel()

		var (
			_ encoding.TextMarshaler   = AggregateID("")
			_ encoding.TextUnmarshaler = (*AggregateID)(nil)
		)
	})
}

func TestSQLValue(t *testing.T) {
	t.Parallel()

	t.Run("value", func(t *testing.T) {
		t.Parallel()

		id := MustParse[AggregateID]("sql-test")

		val, err := id.Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}

		if val != "sql-test" {
			t.Errorf("Value() = %v, want %q", val, "sql-test")
		}
	})

	t.Run("value empty", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		val, err := id.Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}

		if val != "" {
			t.Errorf("Value() empty = %v, want empty", val)
		}
	})
}

func TestSQLScan(t *testing.T) {
	t.Parallel()

	t.Run("scan string", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := id.Scan("scan-test")
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if id.String() != "scan-test" {
			t.Errorf("Scan() = %q, want %q", id.String(), "scan-test")
		}
	})

	t.Run("scan bytes", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := id.Scan([]byte("scan-bytes"))
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if id.String() != "scan-bytes" {
			t.Errorf("Scan() = %q, want %q", id.String(), "scan-bytes")
		}
	})

	t.Run("scan empty string errors", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := id.Scan("")
		if err == nil {
			t.Error("Scan() should error on empty string")
		}
	})

	t.Run("scan empty bytes errors", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := id.Scan([]byte(""))
		if err == nil {
			t.Error("Scan() should error on empty bytes")
		}
	})

	t.Run("scan unsupported type errors", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := id.Scan(123)
		if err == nil {
			t.Error("Scan() should error on unsupported type")
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

	_ = userID.String()
	_ = orderID.String()
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

func TestCorrelationID(t *testing.T) {
	t.Parallel()

	id := NewCorrelationID()
	if id.IsEmpty() {
		t.Error("NewCorrelationID() should not return empty ID")
	}
}

func TestCausationID(t *testing.T) {
	t.Parallel()

	id := NewCausationID()
	if id.IsEmpty() {
		t.Error("NewCausationID() should not return empty ID")
	}
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	id := NewRequestID()
	if id.IsEmpty() {
		t.Error("NewRequestID() should not return empty ID")
	}
}

func TestMustParseID(t *testing.T) {
	t.Parallel()

	t.Run("AggregateID", func(t *testing.T) {
		t.Parallel()

		id := MustParseAggregateID("agg-1")
		if id.String() != "agg-1" {
			t.Errorf("got %q, want %q", id.String(), "agg-1")
		}
	})

	t.Run("EventID", func(t *testing.T) {
		t.Parallel()

		id := MustParseEventID("evt-1")
		if id.String() != "evt-1" {
			t.Errorf("got %q, want %q", id.String(), "evt-1")
		}
	})

	t.Run("UserID", func(t *testing.T) {
		t.Parallel()

		id := MustParseUserID("usr-1")
		if id.String() != "usr-1" {
			t.Errorf("got %q, want %q", id.String(), "usr-1")
		}
	})
}
