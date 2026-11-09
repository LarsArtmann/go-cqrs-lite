package badgerengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestStreamLogBackend_BadgerContract(t *testing.T) {
	t.Parallel()

	eng := mustNewBadgerEngine(t)

	enginetest.RunStreamLogBackendTest(t, eng)
	enginetest.RunSeqSeekableStreamLogTest(t, eng)
}
