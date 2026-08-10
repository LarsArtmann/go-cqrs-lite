package bboltengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestBbolt_RecordStamping(t *testing.T) {
	t.Parallel()

	eng := mustNewBboltEngine(t)

	enginetest.RunRecordStampTest(t, eng)
}
