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

		if err != nil {
			return
		}

		roundtrip, err := Parse[AggregateID](parsed.String())
		if err != nil {
			t.Fatalf("failed to parse canonical output %q: %v", parsed.String(), err)
		}

		if !roundtrip.Equal(parsed) {
			t.Errorf("roundtrip mismatch: Parse(String()) != parsed (input was %q)", input)
		}
	})
}
