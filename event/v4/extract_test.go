package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestExtractCustomBytes(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	key := event.MetadataKey("test.key")

	t.Run("valid_base64_value", func(t *testing.T) {
		t.Parallel()
		evt, err := event.NewEvent("test.created", streamID, "Test", 1, []byte(`{}`),
			event.WithCustom(key, "aGVsbG8=")) // "hello" in base64
		if err != nil {
			t.Fatal(err)
		}

		decoded, found, err := event.ExtractCustomBytes(evt, key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}
		if string(decoded) != "hello" {
			t.Fatalf("expected 'hello', got %q", string(decoded))
		}
	})

	t.Run("missing_key", func(t *testing.T) {
		t.Parallel()
		evt, err := event.NewEvent("test.created", streamID, "Test", 1, []byte(`{}`))
		if err != nil {
			t.Fatal(err)
		}

		_, found, err := event.ExtractCustomBytes(evt, key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found {
			t.Fatal("expected found=false for missing key")
		}
	})

	t.Run("empty_value", func(t *testing.T) {
		t.Parallel()
		evt, err := event.NewEvent("test.created", streamID, "Test", 1, []byte(`{}`),
			event.WithCustom(key, ""))
		if err != nil {
			t.Fatal(err)
		}

		_, found, err := event.ExtractCustomBytes(evt, key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found {
			t.Fatal("expected found=false for empty value")
		}
	})

	t.Run("corrupt_base64", func(t *testing.T) {
		t.Parallel()
		evt, err := event.NewEvent("test.created", streamID, "Test", 1, []byte(`{}`),
			event.WithCustom(key, "!!!not-base64!!!"))
		if err != nil {
			t.Fatal(err)
		}

		_, _, err = event.ExtractCustomBytes(evt, key)
		if err == nil {
			t.Fatal("expected error for corrupt base64")
		}
	})
}
