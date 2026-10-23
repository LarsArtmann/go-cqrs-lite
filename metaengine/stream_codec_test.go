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
