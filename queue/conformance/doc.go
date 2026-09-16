// Package conformance is the ONE shared suite that holds every queue
// engine (queue/sqlite, queue/postgres, …) to the contract's
// semantics. An engine registers the suite with a Harness; there is no
// second copy to drift:
//
//	func TestConformance(t *testing.T) {
//	    conformance.Run(t, conformance.Harness{
//	        NewStore: func(t *testing.T) queue.Store[conformance.Payload] {
//	            return mustOpen(t) // fresh, empty store
//	        },
//	        Backdate: func(t *testing.T, s queue.Store[conformance.Payload], id task.ID, d time.Duration) {
//	            // engine's white-box created_at rewind, e.g. direct UPDATE
//	        },
//	    })
//	}
//
// The suite is the upstreamed form of go-taskqueue's mirrored backend
// suites (its ADR-0007 pattern): the semantics were proven there over
// five weeks of production dogfood and transcribed into this contract;
// every pin here restates one of those proven behaviors. Engines do not
// get to disagree with each other — they run the same suite.
//
// Isolation model: Run opens a FRESH store per subtest via Harness.NewStore
// (the harness decides what fresh means — a new file, a truncated schema,
// a throwaway database). Subtests therefore cannot leak state into each
// other, and time-dependent pins sleep only bounded, subtest-local waits.
package conformance
