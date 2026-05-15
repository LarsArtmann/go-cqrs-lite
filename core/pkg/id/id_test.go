package id

import (
	"encoding"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/oklog/ulid/v2"
)

const testULID = "01HK1549P84T9XF8R94E960633"

func TestNew(t *testing.T) {
	t.Parallel()

	id := New[AggregateID]()

	if id.IsZero() {
		t.Error("New() should not return zero ID")
	}

	if len(id.String()) != ulid.EncodedSize {
		t.Errorf("ID string length = %d, want %d", len(id.String()), ulid.EncodedSize)
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("valid string", func(t *testing.T) {
		t.Parallel()

		id, err := Parse[AggregateID](testULID)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if id.String() != testULID {
			t.Errorf("Parse() = %q, want %q", id.String(), testULID)
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
		{testULID, testULID},
		{"01HK154ANGZHV2ZW0X3SKSNEN2", "01HK154ANGZHV2ZW0X3SKSNEN2"},
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

		id := MustParse[AggregateID](testULID)
		if id.IsZero() {
			t.Error("parsed ID should not be zero")
		}
	})
}

func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("equal IDs", func(t *testing.T) {
		t.Parallel()

		a := MustParse[AggregateID](testULID)

		b := MustParse[AggregateID](testULID)
		if !a.Equal(b) {
			t.Error("equal IDs should be equal")
		}
	})

	t.Run("different IDs", func(t *testing.T) {
		t.Parallel()

		a := MustParse[AggregateID]("01HK153X00WRE0FHNC52TH9Y1A")

		b := MustParse[AggregateID]("01HK153YYGPZ1D26JE8FR0H6AS")
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
		{
			name: "less than",
			aStr: "01HK153X00WRE0FHNC52TH9Y1A",
			bStr: "01HK153YYGPZ1D26JE8FR0H6AS",
			want: -1,
			desc: "first ULID should be less than second",
		},
		{
			name: "equal",
			aStr: testULID,
			bStr: testULID,
			want: 0,
			desc: "same IDs should compare equal",
		},
		{
			name: "greater than",
			aStr: "01HK153YYGPZ1D26JE8FR0H6AS",
			bStr: "01HK153X00WRE0FHNC52TH9Y1A",
			want: 1,
			desc: "second ULID should be greater than first",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := MustParse[AggregateID](tc.aStr)
			b := MustParse[AggregateID](tc.bStr)

			got := CompareIDs(a, b)

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

		id := MustParseAggregateID(testULID)

		fallback := MustParseAggregateID("01HK1542VGZX7VW38CS2WSRXBX")
		if result := id.Or(fallback); !result.Equal(id) {
			t.Error("non-zero ID should return self")
		}
	})

	t.Run("zero returns fallback", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		fallback := MustParseAggregateID("01HK1542VGZX7VW38CS2WSRXBX")
		if result := id.Or(fallback); !result.Equal(fallback) {
			t.Error("zero ID should return fallback")
		}
	})
}

func TestReset(t *testing.T) {
	t.Parallel()

	id := MustParse[AggregateID](testULID)
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

	id := MustParse[AggregateID](testULID)

	gs := id.GoString()
	if gs == "" {
		t.Error("GoString() should not be empty")
	}
}

func TestFormat(t *testing.T) {
	t.Parallel()

	id := MustParse[AggregateID](testULID)

	tests := []struct {
		format string
		want   string
	}{
		{"%v", testULID},
		{"%s", testULID},
		{"%q", `"` + testULID + `"`},
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

		id := MustParse[AggregateID](testULID)

		data, err := json.Marshal(id)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}

		if string(data) != `"`+testULID+`"` {
			t.Errorf("Marshal() = %s, want %q", data, `"`+testULID+`"`)
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := json.Unmarshal([]byte(`"`+testULID+`"`), &id)
		if err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if id.String() != testULID {
			t.Errorf("Unmarshal() = %q, want %q", id.String(), testULID)
		}
	})

	t.Run("marshal null", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		data, err := json.Marshal(id)
		if err != nil {
			t.Fatalf("Marshal() empty error = %v", err)
		}

		if string(data) != "null" {
			t.Errorf("Marshal() empty = %s, want null", data)
		}
	})

	t.Run("unmarshal null", func(t *testing.T) {
		t.Parallel()

		id := MustParse[AggregateID](testULID)

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
		unmarshal     func(*EventID, []byte) error
		marshalName   string
		unmarshalName string
	}{
		{
			name:      "binary",
			testValue: testULID,
			marshal: func(id string) ([]byte, error) {
				return MustParseEventID(id).MarshalBinary()
			},
			unmarshal: func(id *EventID, data []byte) error {
				return id.UnmarshalBinary(data)
			},
			marshalName:   "MarshalBinary",
			unmarshalName: "UnmarshalBinary",
		},
		{
			name:      "text",
			testValue: testULID,
			marshal: func(id string) ([]byte, error) {
				return MustParseEventID(id).MarshalText()
			},
			unmarshal: func(id *EventID, data []byte) error {
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

				if tc.name == "binary" {
					if len(data) != 16 {
						t.Errorf("%s() len = %d, want 16", tc.marshalName, len(data))
					}
				} else if string(data) != tc.testValue {
					t.Errorf("%s() = %q, want %q", tc.marshalName, string(data), tc.testValue)
				}
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()

				var id EventID

				var data []byte
				if tc.name == "binary" {
					data, _ = MustParseEventID(tc.testValue).MarshalBinary()
				} else {
					data = []byte(tc.testValue)
				}

				err := tc.unmarshal(&id, data)
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

		id := MustParse[AggregateID](testULID)

		val, err := id.Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}

		if val != testULID {
			t.Errorf("Value() = %v, want %q", val, testULID)
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

		err := id.Scan(testULID)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if id.String() != testULID {
			t.Errorf("Scan() = %q, want %q", id.String(), testULID)
		}
	})

	t.Run("scan bytes", func(t *testing.T) {
		t.Parallel()

		var id AggregateID

		err := id.Scan([]byte("01HK154ANGZHV2ZW0X3SKSNEN2"))
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if id.String() != "01HK154ANGZHV2ZW0X3SKSNEN2" {
			t.Errorf("Scan() = %q, want %q", id.String(), "01HK154ANGZHV2ZW0X3SKSNEN2")
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

	parsed, err := ParseAggregateID(testULID)
	if err != nil {
		t.Fatalf("ParseAggregateID() error = %v", err)
	}

	if parsed.String() != testULID {
		t.Errorf("ParseAggregateID() = %q, want %q", parsed.String(), testULID)
	}

	t.Run("accepts non-ULID strings", func(t *testing.T) {
		t.Parallel()

		domainID := "lock_user1_user2"
		parsed, err := ParseAggregateID(domainID)
		if err != nil {
			t.Fatalf("ParseAggregateID(%q) error = %v", domainID, err)
		}

		if parsed.String() != domainID {
			t.Errorf("ParseAggregateID(%q) = %q, want %q", domainID, parsed.String(), domainID)
		}

		if parsed.IsZero() {
			t.Error("parsed domain ID should not be zero")
		}
	})

	t.Run("rejects empty string", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAggregateID("")
		if err == nil {
			t.Error("ParseAggregateID() should error on empty string")
		}
	})
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

func TestClientID(t *testing.T) {
	t.Parallel()

	id := NewClientID()
	if id.IsZero() {
		t.Error("NewClientID() should not return zero ID")
	}

	parsed, err := ParseClientID(testULID)
	if err != nil {
		t.Fatalf("ParseClientID() error = %v", err)
	}

	if parsed.String() != testULID {
		t.Errorf("ParseClientID() = %q, want %q", parsed.String(), testULID)
	}
}

func TestMustParseID(t *testing.T) {
	t.Parallel()

	t.Run("AggregateID", func(t *testing.T) {
		t.Parallel()

		id := MustParseAggregateID(testULID)
		if id.String() != testULID {
			t.Errorf("got %q, want %q", id.String(), testULID)
		}
	})

	t.Run("EventID", func(t *testing.T) {
		t.Parallel()

		id := MustParseEventID("01HK1541W8PVV4E88DV993TP2A")
		if id.String() != "01HK1541W8PVV4E88DV993TP2A" {
			t.Errorf("got %q, want %q", id.String(), "01HK1541W8PVV4E88DV993TP2A")
		}
	})

	t.Run("UserID", func(t *testing.T) {
		t.Parallel()

		id := MustParseUserID("01HK1542VGZX7VW38CS2WSRXBX")
		if id.String() != "01HK1542VGZX7VW38CS2WSRXBX" {
			t.Errorf("got %q, want %q", id.String(), "01HK1542VGZX7VW38CS2WSRXBX")
		}
	})
}

func TestParse_InvalidULID(t *testing.T) {
	t.Parallel()

	_, err := Parse[AggregateID]("not-a-valid-ulid")
	if err == nil {
		t.Error("Parse() should error on invalid ULID string")
	}
}

func TestULID_Function(t *testing.T) {
	t.Parallel()

	id := New[struct{}]()

	ts, err := ULID(id)
	if err != nil {
		t.Fatalf("ULID() error = %v", err)
	}

	if ts.IsZero() {
		t.Error("ULID() should return non-zero time")
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	id := New[AggregateID]()

	ulidVal := id.Get()
	if ulidVal.String() != id.String() {
		t.Errorf("Get() = %q, want %q", ulidVal.String(), id.String())
	}
}

func TestParseConvenienceFuncs(t *testing.T) {
	t.Parallel()

	t.Run("CausationID", func(t *testing.T) {
		t.Parallel()

		parsed, err := ParseCausationID(testULID)
		if err != nil {
			t.Fatalf("ParseCausationID() error = %v", err)
		}

		if parsed.String() != testULID {
			t.Errorf("ParseCausationID() = %q, want %q", parsed.String(), testULID)
		}
	})

	t.Run("CorrelationID", func(t *testing.T) {
		t.Parallel()

		parsed, err := ParseCorrelationID(testULID)
		if err != nil {
			t.Fatalf("ParseCorrelationID() error = %v", err)
		}

		if parsed.String() != testULID {
			t.Errorf("ParseCorrelationID() = %q, want %q", parsed.String(), testULID)
		}
	})

	t.Run("EventID", func(t *testing.T) {
		t.Parallel()

		parsed, err := ParseEventID(testULID)
		if err != nil {
			t.Fatalf("ParseEventID() error = %v", err)
		}

		if parsed.String() != testULID {
			t.Errorf("ParseEventID() = %q, want %q", parsed.String(), testULID)
		}
	})

	t.Run("RequestID", func(t *testing.T) {
		t.Parallel()

		parsed, err := ParseRequestID(testULID)
		if err != nil {
			t.Fatalf("ParseRequestID() error = %v", err)
		}

		if parsed.String() != testULID {
			t.Errorf("ParseRequestID() = %q, want %q", parsed.String(), testULID)
		}
	})

	t.Run("UserID", func(t *testing.T) {
		t.Parallel()

		parsed, err := ParseUserID(testULID)
		if err != nil {
			t.Fatalf("ParseUserID() error = %v", err)
		}

		if parsed.String() != testULID {
			t.Errorf("ParseUserID() = %q, want %q", parsed.String(), testULID)
		}
	})
}

func TestMustParseConvenienceFuncs_Panic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func()
	}{
		{"CausationID", func() { _ = MustParseCausationID("") }},
		{"CorrelationID", func() { _ = MustParseCorrelationID("") }},
		{"RequestID", func() { _ = MustParseRequestID("") }},
		{"ClientID", func() { _ = MustParseClientID("") }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic")
				}
			}()

			tc.fn()
		})
	}
}

func TestUnmarshalJSON_InvalidData(t *testing.T) {
	t.Parallel()

	var id AggregateID

	err := json.Unmarshal([]byte("12345"), &id)
	if err == nil {
		t.Error("UnmarshalJSON() should error on non-string JSON")
	}
}

func TestUnmarshalJSON_InvalidULID(t *testing.T) {
	t.Parallel()

	var id EventID

	err := json.Unmarshal([]byte(`"not-a-ulid"`), &id)
	if err == nil {
		t.Error("UnmarshalJSON() should error on invalid ULID string")
	}
}

func TestScan_Nil(t *testing.T) {
	t.Parallel()

	id := MustParse[AggregateID](testULID)

	err := id.Scan(nil)
	if err != nil {
		t.Fatalf("Scan(nil) error = %v", err)
	}

	if !id.IsZero() {
		t.Error("Scan(nil) should reset ID to zero")
	}
}

func TestScan_InvalidString(t *testing.T) {
	t.Parallel()

	var id EventID

	err := id.Scan("not-a-ulid")
	if err == nil {
		t.Error("Scan() should error on invalid ULID string")
	}
}

func TestMarshalBinary_Zero(t *testing.T) {
	t.Parallel()

	var id AggregateID

	data, err := id.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	if data != nil {
		t.Errorf("MarshalBinary() zero = %v, want nil", data)
	}
}

func TestUnmarshalBinary_EmptyData(t *testing.T) {
	t.Parallel()

	id := MustParse[AggregateID](testULID)

	err := id.UnmarshalBinary([]byte{})
	if err != nil {
		t.Fatalf("UnmarshalBinary() error = %v", err)
	}

	if !id.IsZero() {
		t.Error("UnmarshalBinary(empty) should reset ID to zero")
	}
}

func TestUnmarshalBinary_InvalidSize(t *testing.T) {
	t.Parallel()

	var id EventID

	err := id.UnmarshalBinary([]byte{1, 2, 3})
	if err == nil {
		t.Error("UnmarshalBinary() should error on wrong size data")
	}
}

func TestMarshalText_Zero(t *testing.T) {
	t.Parallel()

	var id AggregateID

	data, err := id.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}

	if data != nil {
		t.Errorf("MarshalText() zero = %v, want nil", data)
	}
}

func TestUnmarshalText_InvalidData(t *testing.T) {
	t.Parallel()

	var id EventID

	err := id.UnmarshalText([]byte("not-a-ulid"))
	if err == nil {
		t.Error("UnmarshalText() should error on invalid ULID string")
	}
}

func TestUnmarshalText_EmptyData(t *testing.T) {
	t.Parallel()

	id := MustParse[AggregateID](testULID)

	err := id.UnmarshalText([]byte{})
	if err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}

	if !id.IsZero() {
		t.Error("UnmarshalText(empty) should reset ID to zero")
	}
}

func TestPtr(t *testing.T) {
	t.Parallel()

	id := NewAggregateID()
	p := id.Ptr()

	if p == nil {
		t.Fatal("Ptr() returned nil")
	}

	if *p != id {
		t.Errorf("Ptr() = %v, want %v", *p, id)
	}
}

func TestFromPtr_NonNil(t *testing.T) {
	t.Parallel()

	id := NewEventID()
	result := FromPtr(id.Ptr())

	if result != id {
		t.Errorf("FromPtr(non-nil) = %v, want %v", result, id)
	}
}

func TestFromPtr_Nil(t *testing.T) {
	t.Parallel()

	result := FromPtr[EventID](nil)

	if !result.IsZero() {
		t.Errorf("FromPtr(nil) = %v, want zero value", result)
	}
}
