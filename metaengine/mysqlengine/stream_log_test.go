package mysqlengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestStreamLogBackend_MySQLRoundtrip(t *testing.T) {
	t.Parallel()
	eng := mustNewMySQLEngine(t)
	enginetest.RunStreamLogBackendTest(t, eng)
}

func TestStreamLogBackend_MySQLAtomicAppender(t *testing.T) {
	t.Parallel()
	eng := mustNewMySQLEngine(t)
	enginetest.RunAtomicAppenderTest(t, eng)
}
