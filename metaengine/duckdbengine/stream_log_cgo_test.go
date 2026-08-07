//go:build cgo

package duckdbengine_test

import (
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestStreamLogBackend_DuckDBRoundtrip(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Fatalf("duckdbengine.New: %v", err)
	}

	defer func() { _ = eng.Close() }()

	enginetest.RunStreamLogBackendTest(t, eng)
}

func TestStreamLogBackend_DuckDBAtomicAppender(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Fatalf("duckdbengine.New: %v", err)
	}

	defer func() { _ = eng.Close() }()

	enginetest.RunAtomicAppenderTest(t, eng)
}

func TestStreamLogBackend_DuckDBTransactional(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Fatalf("duckdbengine.New: %v", err)
	}

	defer func() { _ = eng.Close() }()

	enginetest.RunTransactionalTest(t, eng)
}

func TestStreamLogBackend_DuckDBConcurrentTx(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Fatalf("duckdbengine.New: %v", err)
	}

	defer func() { _ = eng.Close() }()

	enginetest.RunConcurrentTxTest(t, eng)
}
