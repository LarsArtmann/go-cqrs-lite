package deriver

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain catches a deriver goroutine that outlives its dispatch. The
// async-dispatch path (deriver.go) spawns a goroutine per event when
// WithAsyncDispatch is set, so a missed wait shows up as a leaked goroutine
// instead of a silently truncated saga. VerifyTestMain reports the creation
// stack, which is what makes the leak actionable.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
