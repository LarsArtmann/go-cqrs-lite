package event

import (
	"context"

	ro "github.com/samber/ro"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// DistinctByEventIDBounded returns an operator like DistinctByEventID but with
// a bounded seen-set that evicts the oldest IDs (FIFO) once capacity is reached.
// This trades exact dedup for bounded memory — an event ID evicted from the
// ring may be processed again if it reappears after `cap` unique events.
//
// Use this for long-running live streams (24/7 projections) where
// DistinctByEventIDWith would grow unbounded.
func DistinctByEventIDBounded(
	capacity int,
) func(ro.Observable[Event]) ro.Observable[Event] {
	return DistinctByEventIDBoundedWith(capacity, nil)
}

// DistinctByEventIDBoundedWith is the seeded variant of DistinctByEventIDBounded.
// It pre-populates the ring with already-seen IDs (e.g. from journal replay),
// then continues with FIFO eviction once the ring fills.
func DistinctByEventIDBoundedWith(
	capacity int,
	seen map[id.EventID]struct{},
) func(ro.Observable[Event]) ro.Observable[Event] {
	return func(source ro.Observable[Event]) ro.Observable[Event] {
		return ro.NewUnsafeObservableWithContext(
			func(subscriberCtx context.Context, destination ro.Observer[Event]) ro.Teardown {
				ring := newBoundedSeenSet(capacity, seen)

				sub := source.SubscribeWithContext(
					subscriberCtx,
					ro.NewObserverWithContext(
						func(ctx context.Context, evt Event) {
							eid := evt.ID()

							if ring.has(eid) {
								return
							}

							ring.add(eid)
							destination.NextWithContext(ctx, evt)
						},
						destination.ErrorWithContext,
						destination.CompleteWithContext,
					),
				)

				return sub.Unsubscribe
			},
		)
	}
}

// boundedSeenSet is a FIFO ring buffer that tracks seen IDs with O(1) lookup
// and bounded memory. When capacity is reached, the oldest ID is evicted.
type boundedSeenSet struct {
	seen map[id.EventID]struct{}
	ring []id.EventID
	head int
	cap  int
}

func newBoundedSeenSet(capacity int, seed map[id.EventID]struct{}) *boundedSeenSet {
	if capacity < 1 {
		capacity = 1
	}

	s := &boundedSeenSet{
		seen: make(map[id.EventID]struct{}, capacity),
		ring: make([]id.EventID, capacity),
		cap:  capacity,
	}

	for eid := range seed {
		s.add(eid)
	}

	return s
}

func (s *boundedSeenSet) has(eid id.EventID) bool {
	_, ok := s.seen[eid]
	return ok
}

func (s *boundedSeenSet) add(eid id.EventID) {
	if _, ok := s.seen[eid]; ok {
		return
	}

	if len(s.seen) >= s.cap {
		delete(s.seen, s.ring[s.head])
	}

	s.seen[eid] = struct{}{}
	s.ring[s.head] = eid
	s.head = (s.head + 1) % s.cap
}
