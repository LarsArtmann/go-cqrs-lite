package id

import (
	"encoding"
	"encoding/json"
	"testing"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	t.Run("marshal", func(t *testing.T) {
		t.Parallel()

		id := parseID[AggregateID](t, testULID)

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

		id := parseID[AggregateID](t, testULID)

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
				return parseID[EventMarker](t, id).MarshalBinary()
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
				return parseID[EventMarker](t, id).MarshalText()
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
					data, _ = parseID[EventMarker](t, tc.testValue).MarshalBinary()
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

		id := parseID[AggregateID](t, testULID)

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

	id := parseID[AggregateID](t, testULID)

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

	id := parseID[AggregateID](t, testULID)

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

	id := parseID[AggregateID](t, testULID)

	err := id.UnmarshalText([]byte{})
	if err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}

	if !id.IsZero() {
		t.Error("UnmarshalText(empty) should reset ID to zero")
	}
}
