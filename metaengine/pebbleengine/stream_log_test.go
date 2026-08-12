package pebbleengine

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestStreamLogBackend_PebbleRoundtrip(t *testing.T) {
	t.Parallel()

	eng, err := NewPebbleEngine("")
	if err != nil {
		t.Fatalf("NewPebbleEngine: %v", err)
	}

	defer metaengine.DeferClose(eng)

	enginetest.RunStreamLogBackendTest(t, eng)
}

func TestStreamLogBackend_PebbleAtomicAppender(t *testing.T) {
	t.Parallel()

	eng, err := NewPebbleEngine("")
	if err != nil {
		t.Fatalf("NewPebbleEngine: %v", err)
	}

	defer metaengine.DeferClose(eng)

	enginetest.RunAtomicAppenderTest(t, eng)
}
