package bboltengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestStreamLogBackend_BboltRoundtrip(t *testing.T) {
	t.Parallel()

	eng := mustNewBboltEngine(t)

	enginetest.RunStreamLogBackendTest(t, eng)
	enginetest.RunSeqSeekableStreamLogTest(t, eng)
}

func TestStreamLogBackend_BboltAtomicAppender(t *testing.T) {
	t.Parallel()

	eng := mustNewBboltEngine(t)

	enginetest.RunAtomicAppenderTest(t, eng)
}
