package scheduling_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain makes ticker/deadline teardown a gate: a scheduler whose
// Close/Stop was skipped leaves its poll loop running, and a timer created
// without a cancel path does the same. That class was previously invisible —
// the suite asserted deadlines but never that the machinery had stopped.
// goleak reports the creation stack, so the fix is named by the failure.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
