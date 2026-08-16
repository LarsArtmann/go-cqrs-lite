package enginetest

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// runToken uniquely identifies one test-binary execution. Every collection
// the helpers in this package touch is tagged with it, so engines backed by
// a server that outlives a single test invocation (shared Postgres/MySQL/
// Dgraph instances) see fresh state under `go test -count>1` and `-race`
// re-runs instead of accumulating rows against absolute-count assertions.
//
// The token is process-unique (monotonic timestamp) and can be pinned from
// the outside via ENGINETEST_RUN_TOKEN, e.g. to correlate collections with a
// CI job or to force a known namespace when debugging a shared server.
var runToken = computeRunToken()

func computeRunToken() string {
	if pinned := os.Getenv("ENGINETEST_RUN_TOKEN"); pinned != "" {
		return pinned
	}

	return fmt.Sprintf("r%d", time.Now().UnixNano())
}

// collectionCounter guards against same-named helpers invoked twice within
// one binary run (e.g. two engines under one test) — each call gets a
// distinct scoped name.
var collectionCounter atomic.Uint64

// ScopedCollection returns name tagged with the per-run token and a
// per-call disambiguator, e.g. "events_r1755432100123456789_1".
//
// All enginetest helpers scope their collections through this function.
// Callers that need to reference a helper's collection from the outside
// (e.g. to apply a layout or inspect rows) should call ScopedCollection
// themselves — but note each call returns a DIFFERENT name, so capture the
// returned value once and share it.
//
// The result stays well under MySQL's VARCHAR(255) collection limit for any
// realistic input name.
func ScopedCollection(name string) string {
	return fmt.Sprintf("%s_%s_%d", name, runToken, collectionCounter.Add(1))
}
