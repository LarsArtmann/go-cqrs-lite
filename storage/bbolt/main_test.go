package bbolt

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain turns teardown completeness into a CI failure: `Backend.Close`
// hands the bolt close off to a goroutine (backend.go), so an unclosed backend
// in any test leaves a goroutine behind and the suite would previously pass.
// Temporal composability (ADR-0136) makes "the effect stopped" a contract, and
// this is the cheapest place to enforce it for this engine.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
