package id

import (
	"encoding"
	"encoding/json"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	id := New[AggregateID]()

	if id.IsZero() {
		t.Error("New() should not return zero ID")
	}

	if id.IsZero() {
		t.Error("New() should return non-zero ID")
	}

	if len(id.String()) != 26 {
		t.Errorf("ID string length = %d, want 26 (ULID)", len(id.String()))
	}
}

func TestNewWithPrefix(t *testing.T) {
	t.Parallel()

	id := NewWithPrefix[AggregateID]("user")

	if id.IsZero() {
		t.Error("NewWithPrefix() should not return zero ID")
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

	tests := []struct {
		input    string
		expected string
	}{
		{"test-id", "test-id"},
		{"my-id", "my-id"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			id := MustParse[AggregateID](tc.input)
			if id.String() != tc.expected {
				t.Errorf("MustParse() = %q, want %q", id.String(), tc.expected)
			}
		})
	}

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

func TestIsZero(t *testing.T) {
	t.Parallel()

	t.Run("zero ID", func(t *testing.T) {
		t.Parallel()

		var id AggregateID
		if !id.IsZero() {
			t.Error("zero value ID should be zero")
		}
	})

	t.Run("non-zero ID", func(t *testing.T) {
		t.Parallel()

		id := MustParse[AggregateID]("test")
		if id.IsZero() {
			t.Error("parsed ID should not be zero")
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

	t.Run("zero IDs", func(t *testing.T) {
		t.Parallel()

		var a, b AggregateID
		if !a.Equal(b) {
			t.Error("zero IDs should be equal")
		}
	})
}

func TestCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		aStr string
		bStr string
		want int
		desc string
	}{
		{"less than", "a", "b", -1, "a should be less than b"},
		{"equal", "same", "same", 0, "same IDs should compare equal"},
		{"greater than", "b", "a", 1, "b should be greater than a"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := MustParse[AggregateID](tc.aStr)
			b := MustParse[AggregateID](tc.bStr)

			got, err := a.Compare(b)
			if err != nil {
				t.Fatalf("Compare() error = %v", err)
			}

			if got != tc.want {
				t.Error(tc.desc)
			}
		})
	}
}

func TestOr(t *testing.T) {
	t.Parallel()

	t.Run("non-zero returns self", func(t *testing.T) {
		t.Parallel()

		id := MustParseAggregateID("primary")

		fallback := MustParseAggregateID("fallback")
		if result := id.Or(fallback); !result.Equal(id) {
			t.Error("non-zero ID should return self")
		}
	})

	t.Run("zero returns fallback", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		fallback := MustParseAggregateID("fallback")
		if result := id.Or(fallback); !result.Equal(fallback) {
			t.Error("zero ID should return fallback")
		}
	})
}

func TestReset(t *testing.T) {
	t.Parallel()

	id := MustParse[AggregateID]("test")
	if id.IsZero() {
		t.Error("ID should not be zero before reset")
	}

	id.Reset()

	if !id.IsZero() {
		t.Error("ID should be zero after reset")
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

		if !id.IsZero() {
			t.Error("Unmarshal(null) should result in zero ID")
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

		err = json.Unmarshal(data, &restored)
		if err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if original.String() != restored.String() {
			t.Errorf("roundtrip: %q != %q", original, restored)
		}
	})
}

func TestEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		testValue     string
		marshal       func(string) ([]byte, error)
		unmarshal     func(*AggregateID, []byte) error
		marshalName   string
		unmarshalName string
	}{
		{
			name:      "binary",
			testValue: "test-binary",
			marshal: func(id string) ([]byte, error) {
				return MustParseAggregateID(id).MarshalBinary()
			},
			unmarshal: func(id *AggregateID, data []byte) error {
				return id.UnmarshalBinary(data)
			},
			marshalName:   "MarshalBinary",
			unmarshalName: "UnmarshalBinary",
		},
		{
			name:      "text",
			testValue: "test-text",
			marshal: func(id string) ([]byte, error) {
				return MustParseAggregateID(id).MarshalText()
			},
			unmarshal: func(id *AggregateID, data []byte) error {
				return id.UnmarshalText(data)
			},
			marshalName:   "MarshalText",
			unmarshalName: "UnmarshalText",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				data, err := tc.marshal(tc.testValue)
				if err != nil {
					t.Fatalf("%s() error = %v", tc.marshalName, err)
				}

				if string(data) != tc.testValue {
					t.Errorf("%s() = %q, want %q", tc.marshalName, string(data), tc.testValue)
				}
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()

				var id AggregateID

				err := tc.unmarshal(&id, []byte(tc.testValue))
				if err != nil {
					t.Fatalf("%s() error = %v", tc.unmarshalName, err)
				}

				if id.String() != tc.testValue {
					t.Errorf("%s() = %q, want %q", tc.unmarshalName, id.String(), tc.testValue)
				}
			})
		})
	}

	t.Run("binary interface compliance", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		var (
			_ encoding.BinaryMarshaler   = id
			_ encoding.BinaryUnmarshaler = &id
		)
	})

	t.Run("text interface compliance", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		var (
			_ encoding.TextMarshaler   = id
			_ encoding.TextUnmarshaler = &id
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

	t.Run("value zero", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		val, err := id.Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}

		if val != nil {
			t.Errorf("Value() zero = %v, want nil", val)
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

	t.Run("scan unsupported type", func(t *testing.T) {
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
	if id.IsZero() {
		t.Error("NewAggregateID() should not return zero ID")
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
	if id.IsZero() {
		t.Error("NewEventID() should not return zero ID")
	}
}

func TestUserID(t *testing.T) {
	t.Parallel()

	id := NewUserID()
	if id.IsZero() {
		t.Error("NewUserID() should not return zero ID")
	}
}

func TestCorrelationID(t *testing.T) {
	t.Parallel()

	id := NewCorrelationID()
	if id.IsZero() {
		t.Error("NewCorrelationID() should not return zero ID")
	}
}

func TestCausationID(t *testing.T) {
	t.Parallel()

	id := NewCausationID()
	if id.IsZero() {
		t.Error("NewCausationID() should not return zero ID")
	}
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	id := NewRequestID()
	if id.IsZero() {
		t.Error("NewRequestID() should not return zero ID")
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
