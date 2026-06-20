package event

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestEncodingForCopy_ImmutableEvent(t *testing.T) {
	t.Parallel()

	evt, err := NewEvent("TestEvent", id.NewAggregateID(), "Test", 1, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	got := encodingForCopy(evt)
	if got != "" {
		t.Errorf("encodingForCopy of default event = %q, want empty string (raw field)", got)
	}

	if evt.Encoding() != codec.EncodingJSON {
		t.Errorf("Encoding() = %q, want %q (normalized)", evt.Encoding(), codec.EncodingJSON)
	}
}

func TestEncodingForCopy_WithExplicitEncoding(t *testing.T) {
	t.Parallel()

	evt, err := NewEvent(
		"TestEvent", id.NewAggregateID(), "Test", 1, []byte(`{}`),
		WithEncoding(codec.Encoding("protobuf")),
	)
	if err != nil {
		t.Fatal(err)
	}

	got := encodingForCopy(evt)
	if got != codec.Encoding("protobuf") {
		t.Errorf("encodingForCopy = %q, want %q", got, "protobuf")
	}
}
