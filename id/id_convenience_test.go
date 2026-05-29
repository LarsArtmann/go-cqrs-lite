package id

import (
	"testing"

	"github.com/oklog/ulid/v2"
)

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
		{"AggregateID", func() { _ = MustParseAggregateID("") }},
		{"EventID", func() { _ = MustParseEventID("") }},
		{"UserID", func() { _ = MustParseUserID("") }},
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

func TestULIDPackage(t *testing.T) {
	t.Parallel()

	_ = ulid.EncodedSize
}
