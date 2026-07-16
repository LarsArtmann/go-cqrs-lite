package encryption

import (
	"encoding/json/v2"
	"testing"
)

func TestCiphertext_GoldenJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ct   Ciphertext
	}{
		{"empty", Ciphertext{}},
		{"single_byte", Ciphertext{0x01}},
		{"hello", Ciphertext("hello")},
		{"binary", Ciphertext{0x00, 0x01, 0x02, 0xfe, 0xff}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tc.ct)
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}

			var decoded Ciphertext
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}

			if !decoded.Equal(tc.ct) {
				t.Fatalf("roundtrip failed: got %v, want %v", decoded, tc.ct)
			}

			if tc.ct.IsZero() {
				if string(encoded) != `""` {
					t.Fatalf("empty ciphertext should marshal as empty string, got %s", encoded)
				}
			}
		})
	}
}

func TestCiphertext_GoldenValues(t *testing.T) {
	t.Parallel()

	ct := Ciphertext{0x01, 0x02, 0x03}

	expected := `"AQID"`
	encoded, err := json.Marshal(ct)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	if string(encoded) != expected {
		t.Fatalf("expected %s, got %s", expected, encoded)
	}
}
