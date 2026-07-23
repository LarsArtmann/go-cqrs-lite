package id

import (
	"fmt"
	"testing"

	"github.com/oklog/ulid/v2"
)

const testULID = "01HK1549P84T9XF8R94E960633"

func parseID[T any](tb testing.TB, s string) Of[T] {
	tb.Helper()

	v, err := Parse[T](s)
	if err != nil {
		tb.Fatalf("parse ID %q: %v", s, err)
	}

	return v
}

func parseAggID(tb testing.TB, s string) StreamID {
	tb.Helper()

	v, err := ParseAggregateID(s)
	if err != nil {
		tb.Fatalf("parse aggID %q: %v", s, err)
	}

	return v
}

func TestNew(t *testing.T) {
	t.Parallel()

	id := New[StreamID]()

	if id.IsZero() {
		t.Error("New() should not return zero ID")
	}

	if len(id.String()) != ulid.EncodedSize {
		t.Errorf("ID string length = %d, want %d", len(id.String()), ulid.EncodedSize)
	}
}

func lower(s string) string {
	result := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}

	return string(result)
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("valid string", func(t *testing.T) {
		t.Parallel()

		id, err := Parse[StreamID](testULID)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if id.String() != testULID {
			t.Errorf("Parse() = %q, want %q", id.String(), testULID)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()

		_, err := Parse[StreamID]("")
		if err == nil {
			t.Error("Parse() should error on empty string")
		}
	})

	t.Run("lowercase input normalizes to uppercase", func(t *testing.T) {
		t.Parallel()

		id, err := Parse[StreamID](lower(testULID))
		if err != nil {
			t.Fatalf("Parse(lowercase) error = %v", err)
		}

		if id.String() != testULID {
			t.Errorf("Parse(lowercase) = %q, want canonical uppercase %q", id.String(), testULID)
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

			id := parseID[StreamID](t, tc.input)
			if id.String() != tc.expected {
				t.Errorf("MustParse() = %q, want %q", id.String(), tc.expected)
			}
		})
	}

	t.Run("empty string returns error", func(t *testing.T) {
		t.Parallel()

		_, err := Parse[StreamID]("")
		if err == nil {
			t.Error("Parse() should return error on empty string")
		}
	})
}

func TestIsZero(t *testing.T) {
	t.Parallel()

	t.Run("zero ID", func(t *testing.T) {
		t.Parallel()

		var id StreamID
		if !id.IsZero() {
			t.Error("zero value ID should be zero")
		}
	})

	t.Run("non-zero ID", func(t *testing.T) {
		t.Parallel()

		id := parseID[StreamID](t, testULID)
		if id.IsZero() {
			t.Error("parsed ID should not be zero")
		}
	})
}

func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("equal IDs", func(t *testing.T) {
		t.Parallel()

		a := parseID[StreamID](t, testULID)

		b := parseID[StreamID](t, testULID)
		if !a.Equal(b) {
			t.Error("equal IDs should be equal")
		}
	})

	t.Run("different IDs", func(t *testing.T) {
		t.Parallel()

		a := parseID[StreamID](t, "01HK153X00WRE0FHNC52TH9Y1A")

		b := parseID[StreamID](t, "01HK153YYGPZ1D26JE8FR0H6AS")
		if a.Equal(b) {
			t.Error("different IDs should not be equal")
		}
	})

	t.Run("zero IDs", func(t *testing.T) {
		t.Parallel()

		var a, b StreamID
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

			a := parseID[StreamID](t, tc.aStr)
			b := parseID[StreamID](t, tc.bStr)

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

		id := parseAggID(t, testULID)

		fallback := parseAggID(t, "01HK1542VGZX7VW38CS2WSRXBX")
		if result := id.Or(fallback); !result.Equal(id) {
			t.Error("non-zero ID should return self")
		}
	})

	t.Run("zero returns fallback", func(t *testing.T) {
		t.Parallel()

		var id StreamID

		fallback := parseAggID(t, "01HK1542VGZX7VW38CS2WSRXBX")
		if result := id.Or(fallback); !result.Equal(fallback) {
			t.Error("zero ID should return fallback")
		}
	})
}

func TestReset(t *testing.T) {
	t.Parallel()

	id := parseID[StreamID](t, testULID)
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

	id := parseID[StreamID](t, testULID)

	gs := id.GoString()
	if gs == "" {
		t.Error("GoString() should not be empty")
	}
}

func TestFormat(t *testing.T) {
	t.Parallel()

	id := parseID[StreamID](t, testULID)

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
