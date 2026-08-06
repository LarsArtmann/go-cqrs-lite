package metaengine

// newMemoryEngineForTest wraps NewMemoryEngine to return (Engine, error) for
// drop-in compatibility with tests that previously called NewSQLiteEngine.
func newMemoryEngineForTest() (Engine, error) {
	return NewMemoryEngine(), nil
}
