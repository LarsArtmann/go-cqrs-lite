package system_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain guards teardown completeness: any goroutine still running after
// the suite ends (a background replan loop that survived Shutdown, an engine
// closer that missed a worker) is reported with its creation stack, so
// temporal-composability regressions (revert leaves something running)
// surface here first.
//
// The memory driver ships inside metaengine core and needs no blank imports:
// this suite exercises system with `Driver: "memory"` and direct engines
// only. The REAL-engine suites (sqlite/pebble/badger/postgres driver
// registration + their behaviors) live in the sibling `systemtest/` module —
// the Feedback-#4 split that keeps engine implementations out of
// system/go.mod, so consumers importing system/v4 pull no engine they do
// not use.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
