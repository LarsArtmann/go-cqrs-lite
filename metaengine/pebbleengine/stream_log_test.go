package pebbleengine

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestStreamLogBackend_PebbleRoundtrip(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngineInternal(t)

	enginetest.RunStreamLogBackendTest(t, eng)
	enginetest.RunSeqSeekableStreamLogTest(t, eng)
}

func TestStreamLogBackend_PebbleAtomicAppender(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngineInternal(t)

	enginetest.RunAtomicAppenderTest(t, eng)
}
