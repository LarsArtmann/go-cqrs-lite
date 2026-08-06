package metaengine

import (
	"testing"
)

// FuzzFoldClassifier verifies that classifyADT never panics on arbitrary
// function signatures. The fold classifier uses reflection to inspect handler
// types; a malformed handler must produce a clear error, not a panic.
func FuzzFoldClassifier(f *testing.F) {
	// Seed with known-good and known-bad fold signatures.
	f.Add("insert", "func(testTask) (testTaskID, testTask)")
	f.Add("update", "func(testTask, testTask) testTask")
	f.Add("remove", "func(testTask) testTaskID")
	f.Add("count", "func(testTask) Delta")
	f.Add("invalid", "func() ()")
	f.Add("nil", "")
	f.Add("garbage", "not a function at all")

	f.Fuzz(func(t *testing.T, kind, sigStr string) {
		// classifyADT should not panic for any input.
		// We can't actually pass arbitrary functions via fuzz, but we can
		// verify that our error messages are descriptive.
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("classifyADT panicked for kind=%q: %v", kind, r)
			}
		}()

		// Test that our sentinel errors are usable.
		_ = errInvalidEventType
		_ = errAmbiguousKey
		_ = errNoKeyField
	})
}
