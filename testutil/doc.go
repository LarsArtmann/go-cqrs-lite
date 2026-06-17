// Package testutil provides shared test helpers for go-cqrs-lite modules.
//
// # Rapid Property-Based Testing
//
// The rapidgen.go file provides generators and helpers for property-based
// testing with pgregory.net/rapid. These generators produce valid CQRS
// domain values (event types, aggregate types, versions, metadata maps,
// timestamps, event slices) for use in property tests.
//
// # Usage
//
//	func TestEventProperty(t *testing.T) {
//	    rapid.Check(t, func(rt *rapid.T) {
//	        eventType := testutil.EventType().Draw(rt, "eventType")
//	        aggType := testutil.AggregateType().Draw(rt, "aggType")
//	        version := testutil.Version().Draw(rt, "version")
//
//	        evt, err := event.NewEvent(event.Type(eventType), id.NewAggregateID(),
//	            event.AggregateType(aggType), event.Version(version), nil)
//	        if err != nil {
//	            return // skip invalid combinations
//	        }
//
//	        // Verify invariant: round-trip through clone preserves identity
//	        clone := evt.Clone()
//	        if clone.ID() != evt.ID() {
//	            rt.Fatalf("clone ID mismatch")
//	        }
//	    })
//	}
//
// # Reproducible Failures
//
// When a rapid test fails, it prints the failing seed. Set RAPID_SEED=<seed>
// to reproduce that exact failure:
//
//	RAPID_SEED=12345 go test -run TestEventProperty ./...
//
// Use SeedFromEnv to log the seed in test output:
//
//	if seed, ok := testutil.SeedFromEnv(); ok {
//	    t.Logf("reproducing rapid failure with seed %d", seed)
//	}
//
// # Available Generators
//
//   - EventType(): CQRS event type strings
//   - AggregateType(): CQRS aggregate type strings
//   - Version(): positive version numbers [1, 10000]
//   - NonEmptyString(): non-empty strings up to 200 chars
//   - MetadataMap(): random string→string maps for metadata testing
//   - Timestamp(): UTC timestamps between 2000 and 2100
//   - EventSlice[T](gen, min, max): slices of varying length
package testutil
