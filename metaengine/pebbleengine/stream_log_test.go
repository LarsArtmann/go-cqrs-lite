package pebbleengine

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestStreamLogBackend_PebbleRoundtrip(t *testing.T) {
	t.Parallel()

	eng, err := NewPebbleEngine("")
	if err != nil {
		t.Fatalf("NewPebbleEngine: %v", err)
	}

	defer func() { _ = eng.Close() }()

	enginetest.RunStreamLogBackendTest(t, eng)
}

func TestStreamLogBackend_PebbleAtomicAppender(t *testing.T) {
	t.Parallel()

	eng, err := NewPebbleEngine("")
	if err != nil {
		t.Fatalf("NewPebbleEngine: %v", err)
	}

	defer func() { _ = eng.Close() }()

	enginetest.RunAtomicAppenderTest(t, eng)
}
