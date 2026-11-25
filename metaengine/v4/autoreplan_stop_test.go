package metaengine_test

import (
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// profileCountingEngine counts how often the store read the engine's profile:
// the auto-replan loop does that on every routing check, so the counter is the
// loop's heartbeat.
type profileCountingEngine struct {
	*fakeRemoteEngine

	profileCalls atomic.Int64
}

func (e *profileCountingEngine) Profile() metaengine.EngineProfile {
	e.profileCalls.Add(1)

	return e.fakeRemoteEngine.Profile()
}

// awaitProfileCalls polls until the profile was read at least want times.
func awaitProfileCalls(eng *profileCountingEngine, want int64, within time.Duration) bool {
	deadline := time.NewTimer(within)
	defer deadline.Stop()

	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()

	for {
		if eng.profileCalls.Load() >= want {
			return true
		}

		select {
		case <-deadline.C:
			return false
		case <-tick.C:
		}
	}
}

// awaitProfileCallsStable reports whether the profile-read counter stayed flat
// for the whole window — many loop intervals wide, so a loop that is still
// scheduled would be seen.
func awaitProfileCallsStable(eng *profileCountingEngine, window time.Duration) bool {
	deadline := time.NewTimer(window)
	defer deadline.Stop()

	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()

	last := eng.profileCalls.Load()

	for {
		select {
		case <-deadline.C:
			return true
		case <-tick.C:
			if eng.profileCalls.Load() != last {
				return false
			}
		}
	}
}
