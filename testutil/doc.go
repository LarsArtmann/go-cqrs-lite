// Package testutil provides shared test helpers for go-cqrs-lite modules.
//
// # Rapid Property-Based Testing
//
// The rapidgen.go file provides generators and helpers for property-based
// testing with pgregory.net/rapid. These generators produce valid CQRS
// domain values (event types, aggregate types, versions, metadata maps,
// timestamps) for use in property tests.
//
// # Usage
//
//	func TestEventProperty(t *testing.T) {
//	    rapid.Check(t, func(rt *rapid.T) {
//	        eventType := testutil.EventType().Draw(rt, "eventType")
//	        aggType := testutil.StreamType().Draw(rt, "aggType")
//	        version := testutil.Version().Draw(rt, "version")
//
//	        evt, err := event.NewEvent(event.Type(eventType), id.NewStreamID(),
//	            id.StreamType(aggType), event.Version(version), nil)
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
// When a rapid test fails, it prints the failing seed. Set the standard
// `-rapid.seed=<seed>` flag to reproduce that exact failure:
//
//	go test -run TestEventProperty ./... -rapid.seed=12345
//
// # Available Generators
//
//   - EventType(): CQRS event type strings
//   - StreamType(): CQRS aggregate type strings
//   - Version(): positive version numbers [1, 10000]
//   - NonEmptyString(): non-empty strings up to 200 chars
//   - MetadataMap(): random string→string maps for metadata testing
//   - Timestamp(): UTC timestamps between 2000 and 2100
package testutil
