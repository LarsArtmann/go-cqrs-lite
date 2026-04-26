package id

import (
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add("01H4S2Z4QX8N1P5K3M7R9T0V2W")
	f.Add("")
	f.Add("01HK1549P84T9XF8R94E960633")

	f.Fuzz(func(t *testing.T, input string) {
		parsed, err := Parse[AggregateID](input)

		if input == "" {
			if err == nil {
				t.Error("expected error for empty input")
			}

			return
		}

		// Invalid inputs (not valid ULID) should error
		if err != nil {
			return
		}

		if parsed.String() != input {
			t.Errorf("roundtrip mismatch: got %q, want %q", parsed.String(), input)
		}
	})
}
