// Conformance suite for the ADR-0142 [FactSink] capability: the
// journal-never-disagrees invariant (T14c).

package adttest

import (
	"context"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// factLister is the read side of the fact journal — claimkit's exported
// ClaimFactsList, promoted through the engine embedding. AssertFactSink uses
// it to observe what FactSink wrote, so the invariant is verified black-box.
type factLister interface {
	ClaimFactsList(ctx context.Context, collection, key string) ([]metaengine.ClaimFact, error)
}

// AssertFactSink runs the FactSink conformance suite against every factory:
// a claim transition and its facts are ONE transaction, so a state change
// without its fact — and a fact without its state change — must both be
// unobservable (the queue contract's invariant #1, ADR-0142).
//
// Engines without same-tx facts (map-shaped runtimes) do not implement
// metaengine.FactSink and simply do not call this suite.
func AssertFactSink(t *testing.T, factories []Factory) {
	t.Helper()

	if len(factories) == 0 {
		t.Fatal("AssertFactSink requires at least 1 factory")
	}

	for _, factory := range factories {
		t.Run(factory.Name, func(t *testing.T) {
			eng := factory.Create(t)
			defer metaengine.DeferClose(eng)

			sink, ok := eng.(metaengine.FactSink)
			if !ok {
				t.Fatalf("%s does not implement metaengine.FactSink", factory.Name)
			}

			claimer, ok := eng.(metaengine.DueClaimer)
			if !ok {
				t.Fatalf("%s implements FactSink but not DueClaimer", factory.Name)
			}

			lister, ok := eng.(factLister)
			if !ok {
				t.Fatalf(
					"%s implements FactSink but not the fact reader (ClaimFactsList)"+
						" — the invariant cannot be verified",
					factory.Name,
				)
			}

			ctx := context.Background()

			t.Run("FactsRideTheClaimTx", func(t *testing.T) {
				col := claimConformanceCollection("factsink")
				now := ms(1000)

				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "due-a", now.Add(-time.Second), []byte("a")),
				)
				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "due-b", now.Add(-time.Second), []byte("b")),
				)
				// NotBefore-gated: never claimed, so a fact for it must never exist.
				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "future", now.Add(time.Hour), []byte("f")),
				)

				claims, err := sink.ClaimDueFacts(ctx, metaengine.ClaimDueRequest{
					Collection: col, Owner: "w1", Now: now,
				}, func(cl metaengine.DueClaim) []metaengine.ClaimFact {
					return []metaengine.ClaimFact{{Type: "claimed", Payload: cl.Payload}}
				})
				if err != nil {
					t.Fatalf("ClaimDueFacts: %v", err)
				}

				if len(claims) != 2 {
					t.Fatalf("got %d claims, want 2 (future must stay gated)", len(claims))
				}

				for _, key := range []string{"due-a", "due-b"} {
					facts := listFactsT(t, ctx, lister, col, key)

					if len(facts) != 1 || facts[0].Type != "claimed" ||
						string(facts[0].Payload) != key[len("due-"):] {
						t.Fatalf(
							"%s facts = %+v, want exactly one claimed fact with the payload",
							key,
							facts,
						)
					}
				}

				if facts := listFactsT(t, ctx, lister, col, "future"); len(facts) != 0 {
					t.Fatalf(
						"unclaimed item recorded facts: %+v — a fact without its state change",
						facts,
					)
				}
			})

			t.Run("DeleteFactsAtomicBothDirections", func(t *testing.T) {
				col := claimConformanceCollection("delfacts")
				now := ms(1000)

				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "t", now.Add(-time.Minute), []byte("gen1")),
				)

				claims := claimDueT(t, ctx, claimer, col, now)
				if len(claims) != 1 {
					t.Fatalf("claim: %d, want 1", len(claims))
				}

				// Matching epoch: the delete AND its fact land together.
				matched, err := sink.ClaimDeleteFacts(ctx, col, "t", claims[0].DueAt,
					metaengine.ClaimFact{Type: "completed", Payload: []byte{0x00, 0xff}})
				if err != nil || !matched {
					t.Fatalf("ClaimDeleteFacts matched=%v err=%v, want true/nil", matched, err)
				}

				facts := listFactsT(t, ctx, lister, col, "t")
				if len(facts) != 1 || facts[0].Type != "completed" ||
					string(facts[0].Payload) != "\x00\xff" {
					t.Fatalf(
						"facts = %+v, want exactly the completed fact with binary payload intact",
						facts,
					)
				}

				// Stale epoch (re-scheduled under the same key since): the
				// delete is a no-op AND records nothing — a fact without its
				// state change must be unobservable.
				mustClaimT(
					t,
					claimer.ClaimInsert(ctx, col, "t", now.Add(time.Hour), []byte("gen2")),
				)

				stale, err := sink.ClaimDeleteFacts(ctx, col, "t", claims[0].DueAt,
					metaengine.ClaimFact{Type: "completed"})
				if err != nil || stale {
					t.Fatalf("stale ClaimDeleteFacts matched=%v err=%v, want false/nil", stale, err)
				}

				if facts = listFactsT(t, ctx, lister, col, "t"); len(facts) != 1 {
					t.Fatalf("stale delete recorded facts: %+v — fact without state change", facts)
				}
			})
		})
	}
}

func listFactsT(
	t *testing.T,
	ctx context.Context,
	lister factLister,
	collection, key string,
) []metaengine.ClaimFact {
	t.Helper()

	facts, err := lister.ClaimFactsList(ctx, collection, key)
	if err != nil {
		t.Fatalf("ClaimFactsList %s/%s: %v", collection, key, err)
	}

	return facts
}
