//go:build cgo

package duckdbengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestStreamLogBackend_DuckDBRoundtrip(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)

	enginetest.RunStreamLogBackendTest(t, eng)
}

func TestStreamLogBackend_DuckDBAtomicAppender(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)

	enginetest.RunAtomicAppenderTest(t, eng)
}

func TestStreamLogBackend_DuckDBTransactional(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)

	enginetest.RunTransactionalTest(t, eng)
}

func TestStreamLogBackend_DuckDBConcurrentTx(t *testing.T) {
	t.Parallel()

	eng := mustNewDuckEngine(t)

	enginetest.RunConcurrentTxTest(t, eng)
}
