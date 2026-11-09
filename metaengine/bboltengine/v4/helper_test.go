// helper_test.go provides shared test constructors for the bbolt engine.
//
// Parity with pebbleengine: the following pebble test files have intentional
// equivalents here:
//
//   - stream_log_test.go  — bbolt implements StreamLogBackend + AtomicAppender
//   - watcher_test.go     — engine-agnostic (Store-level watcher via metaengine.Plan)
//
// The following pebble test files are NOT ported because they exercise
// LayoutPlanner / RawScanReader — pebble-specific optimization layers that
// bbolt does not implement (single-writer bucket model, no secondary indexes):
//
//   - edge_cases_test.go  — tests ApplyLayout + ScanRawValues edge cases
//   - fuzz_test.go        — fuzzes ScanRawValues filter index
//   - scan_bench_test.go  — benchmarks ScanRawValues at 100/1K/10K/100K
//
// bbolt's scan path uses MapScan (ScanBackend), which is covered by adt_matrix_test.go.
package bboltengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func mustNewBboltEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := bboltengine.NewBboltEngine("")
	if err != nil {
		tb.Skipf("bbolt not available: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}

func newBboltEngineOrSkip(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := bboltengine.NewBboltEngine("")
	if err != nil {
		tb.Skipf("bbolt not available: %v", err)
	}

	return eng
}
