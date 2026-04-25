package id

import (
	"strings"
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add("01H4S2Z4QX8N1P5K3M7R9T0V2W")
	f.Add("")
	f.Add("not-a-uuid")
	f.Add(strings.Repeat("x", 256))

	f.Fuzz(func(t *testing.T, input string) {
		parsed, err := Parse[AggregateID](input)

		if input == "" {
			if err == nil {
				t.Error("expected error for empty input")
			}

			return
		}

		if err != nil {
			t.Errorf("unexpected error for %q: %v", input, err)
		}

		if parsed.String() != input {
			t.Errorf("roundtrip mismatch: got %q, want %q", parsed.String(), input)
		}
	})
}
