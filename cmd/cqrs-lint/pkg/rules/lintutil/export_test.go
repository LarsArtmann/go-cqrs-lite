package lintutil

// LastSegmentForTest exposes the unexported lastSegment to the external
// test package so its version-suffix stripping can be tested directly.
func LastSegmentForTest(path string) string { return lastSegment(path) }
