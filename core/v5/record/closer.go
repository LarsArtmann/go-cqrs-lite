package record

import "io"

// DeferClose calls c.Close(), discarding the error. Intended for use in defer
// statements where the close error is not actionable (rows, iterators, batches
// in defer scope). Replaces the verbose `defer func() { _ = x.Close() }()`
// idiom at Tier 0 so every module — including leaf storage modules that cannot
// reach the engine substrate — shares one address for the idiom (ADR-0144).
//
// Scope guard: this is for resources whose close error is genuinely
// discardable. Bare `defer x.Close()` and actionable `if err := x.Close()`
// remain valid; explicit rollback-defers (`_ = tx.Rollback()`) are a different
// intent and must not be converted.
func DeferClose(c io.Closer) {
	_ = c.Close()
}
