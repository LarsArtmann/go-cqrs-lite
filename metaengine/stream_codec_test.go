package metaengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestEncodeStreamValueErr(t *testing.T) {
	t.Parallel()

	t.Run("strings are stored as-is", func(t *testing.T) {
		t.Parallel()

		got, err := metaengine.EncodeStreamValueErr("plain")
		if err != nil {
			t.Fatalf("EncodeStreamValueErr: %v", err)
		}

		if got != "plain" {
			t.Fatalf("encoded = %q, want %q", got, "plain")
		}
	})

	t.Run("values are JSON encoded", func(t *testing.T) {
		t.Parallel()

		got, err := metaengine.EncodeStreamValueErr(map[string]int{"a": 1})
		if err != nil {
			t.Fatalf("EncodeStreamValueErr: %v", err)
		}

		if got != `{"a":1}` {
			t.Fatalf("encoded = %q, want %q", got, `{"a":1}`)
		}
	})

	t.Run("unrepresentable values fail instead of storing empty", func(t *testing.T) {
		t.Parallel()

		if _, err := metaengine.EncodeStreamValueErr(make(chan int)); err == nil {
			t.Fatal("EncodeStreamValueErr(chan) = nil error, want marshal failure")
		}
	})
}

func TestDecodeStreamValueErr(t *testing.T) {
	t.Parallel()

	t.Run("round-trips a JSON value", func(t *testing.T) {
		t.Parallel()

		encoded, err := metaengine.EncodeStreamValueErr([]int{1, 2, 3})
		if err != nil {
			t.Fatalf("EncodeStreamValueErr: %v", err)
		}

		decoded, err := metaengine.DecodeStreamValueErr(encoded)
		if err != nil {
			t.Fatalf("DecodeStreamValueErr: %v", err)
		}

		items, ok := decoded.([]any)
		if !ok || len(items) != 3 {
			t.Fatalf("decoded = %#v, want a 3-element array", decoded)
		}
	})

	t.Run("corrupt stored values surface an error", func(t *testing.T) {
		t.Parallel()

		if _, err := metaengine.DecodeStreamValueErr("not json"); err == nil {
			t.Fatal("DecodeStreamValueErr(corrupt) = nil error, want decode failure")
		}
	})
}

func TestDecodeStreamValue_LegacyFallback(t *testing.T) {
	t.Parallel()

	// Plain strings are stored unquoted, so the legacy read path must return
	// them unchanged — that fallback is the contract read sites rely on.
	if got := metaengine.DecodeStreamValue("plain"); got != "plain" {
		t.Fatalf("DecodeStreamValue(plain) = %#v, want %q", got, "plain")
	}

	if got := metaengine.DecodeStreamValue(""); got != "" {
		t.Fatalf("DecodeStreamValue(\"\") = %#v, want \"\"", got)
	}

	if got := metaengine.DecodeStreamValue(`{"a":1}`); got == nil {
		t.Fatal("DecodeStreamValue(JSON) = nil, want decoded map")
	}
}

// TestDecodeStreamValue_JSONShapedStringsAreAmbiguous pins the KNOWN
// limitation of the untagged stored form: a string value that is itself valid
// JSON reads back as the parsed value, not as a string. The pin exists so the
// behavior cannot change silently in either direction; fixing it needs the
// stamped engine-value envelope (ADR-0066 reversal, plan T16), not a read-side
// change.
func TestDecodeStreamValue_JSONShapedStringsAreAmbiguous(t *testing.T) {
	t.Parallel()

	cases := []struct {
		stored string
		check  func(t *testing.T, got any)
	}{
		{`{"user":"42"}`, func(t *testing.T, got any) {
			t.Helper()

			if _, ok := got.(map[string]any); !ok {
				t.Fatalf("object-shaped string decoded to %T, want map[string]any", got)
			}
		}},
		{`[1,2]`, func(t *testing.T, got any) {
			t.Helper()

			if _, ok := got.([]any); !ok {
				t.Fatalf("array-shaped string decoded to %T, want []any", got)
			}
		}},
		{`123`, func(t *testing.T, got any) {
			t.Helper()

			if got != float64(123) {
				t.Fatalf("numeric string decoded to %#v, want float64(123)", got)
			}
		}},
		{`true`, func(t *testing.T, got any) {
			t.Helper()

			if got != true {
				t.Fatalf("boolean string decoded to %#v, want true", got)
			}
		}},
		{`null`, func(t *testing.T, got any) {
			t.Helper()

			if got != nil {
				t.Fatalf("null string decoded to %#v, want nil", got)
			}
		}},
	}

	for _, tc := range cases {
		encoded, err := metaengine.EncodeStreamValueErr(tc.stored)
		if err != nil {
			t.Fatalf("EncodeStreamValueErr(%q): %v", tc.stored, err)
		}

		if encoded != tc.stored {
			t.Fatalf("string %q stored as %q, want verbatim", tc.stored, encoded)
		}

		tc.check(t, metaengine.DecodeStreamValue(encoded))
	}
}

// TestDecodeStreamValue_PlainStringsRoundTrip guards the other half of the
// contract: ordinary strings, including anything JSON rejects, survive.
func TestDecodeStreamValue_PlainStringsRoundTrip(t *testing.T) {
	t.Parallel()

	for _, want := range []string{"plain", "with space", "tab\there", "leading {", "0x10", ""} {
		encoded, err := metaengine.EncodeStreamValueErr(want)
		if err != nil {
			t.Fatalf("EncodeStreamValueErr(%q): %v", want, err)
		}

		got := metaengine.DecodeStreamValue(encoded)
		if got != want {
			t.Fatalf("round trip of %q = %#v", want, got)
		}
	}
}

func TestEncodeStreamValuesErr(t *testing.T) {
	t.Parallel()

	t.Run("encodes every value in order", func(t *testing.T) {
		t.Parallel()

		got, err := metaengine.EncodeStreamValuesErr([]any{"plain", 7, map[string]int{"a": 1}})
		if err != nil {
			t.Fatalf("EncodeStreamValuesErr: %v", err)
		}

		want := []string{"plain", "7", `{"a":1}`}

		if len(got) != len(want) {
			t.Fatalf("encoded %d values, want %d", len(got), len(want))
		}

		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("encoded[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("one unrepresentable value rejects the batch", func(t *testing.T) {
		t.Parallel()

		got, err := metaengine.EncodeStreamValuesErr([]any{"ok", make(chan int)})
		if err == nil {
			t.Fatal("EncodeStreamValuesErr(chan) = nil error")
		}

		if got != nil {
			t.Fatalf("encoded values = %v, want nil on error", got)
		}
	})
}
