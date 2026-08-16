package mysqlengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// The MySQL server is shared across the whole suite: each test uses a unique
// collection so parallel tests (and the adttest matrix, which owns "events")
// never write to the same stream.
func TestStreamLogBackend_MySQLRoundtrip(t *testing.T) {
	t.Parallel()
	eng := mustNewMySQLEngine(t)
	enginetest.RunStreamLogBackendTestIn(t, eng, "events_mysql_roundtrip")
	enginetest.RunSeqSeekableStreamLogTestIn(t, eng, "events_mysql_seqseek")
}

func TestStreamLogBackend_MySQLAtomicAppender(t *testing.T) {
	t.Parallel()
	eng := mustNewMySQLEngine(t)
	enginetest.RunAtomicAppenderTestIn(t, eng, "events_mysql_atomic")
}
