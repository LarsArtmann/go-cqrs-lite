package benchkit

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// journeyEventType is a dedicated event type for the journey phase so the
// journey projection does not replay the entire write-phase journal — the
// projectionhost still scans the journal but only HANDLES journey events,
// making initial catch-up fast regardless of profile size.
const journeyEventType event.Type = "bench.journey"

// Query type identifiers used by the query and journey phases.
const (
	getCountQueryType   query.Type = "bench.get-count"
	listCountsQueryType query.Type = "bench.list-counts"
	missQueryType       query.Type = "bench.miss"
)

// CounterState is the decider state for the snapshot/cache phase: a running
// total of the BenchPayload.Items field across all folded events.
type CounterState struct {
	TotalItems int
}

// counterDecider folds BenchPayload events into CounterState by summing the
// Items field. Used by the snapshot/cache phase (M16).
func counterDecider() decider.Decider[CounterState] {
	return decider.Decider[CounterState]{
		Initial: CounterState{},
		Apply: func(state CounterState, evt event.Event) (CounterState, error) {
			p, err := event.DecodePayloadAuto[BenchPayload](evt)
			if err != nil {
				return state, fmt.Errorf("decode bench payload: %w", err)
			}

			state.TotalItems += p.Items

			return state, nil
		},
	}
}

// ── Query types ──

// getCountQuery reads a single stream's materialized counter from the read
// model. Implements query.Query via the value-receiver Type() method.
type getCountQuery struct {
	streamID string
}

func (q getCountQuery) Type() query.Type { return getCountQueryType }

// listCountsQuery reads a page of counters from the read model.
type listCountsQuery struct {
	page     uint
	pageSize uint
}

func (q listCountsQuery) Type() query.Type { return listCountsQueryType }

// missQuery is dispatched against an unregistered type to measure the miss
// (handler-not-found) path through the dispatcher.
type missQuery struct{}

func (missQuery) Type() query.Type { return missQueryType }

// CountResult is the typed result of getCountQuery.
type CountResult struct {
	StreamID string
	Count    uint64
}

// countKey builds the read-model key for a stream's materialized counter.
// Uses a dedicated prefix (bench:jcount:) so it never collides with the
// projection phase (bench:count:) or the read-model phase (raw stream ID).
func countKey(streamID string) []byte {
	return []byte("bench:jcount:" + streamID)
}

// readCount reads and decodes a uint64 counter from the kv.Store.
// Returns 0, false when the key does not exist yet.
func readCount(ctx context.Context, store kv.Store, key []byte) (uint64, bool, error) {
	val, err := store.Get(ctx, key)
	if err != nil {
		if errors.Is(err, kv.ErrNotFound) {
			return 0, false, nil
		}

		return 0, false, err
	}

	if len(val) == 0 {
		return 0, false, nil
	}

	return binary.BigEndian.Uint64(val), true, nil
}

// writeCount encodes and writes a uint64 counter to the kv.Store.
func writeCount(ctx context.Context, store kv.Store, key []byte, n uint64) error {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], n)

	return store.Set(ctx, key, buf[:])
}

// newJourneyProjection creates a projection that increments a per-stream
// counter in the kv.Store for each journey event. Each event triggers a
// Get (current count) + Set (incremented count), measuring real projection
// write amplification on the journey path.
func newJourneyProjection(store kv.Store) projection.Projection {
	return projection.NewProjection(
		"bench-journey",
		func(ctx context.Context, evt event.Event) error {
			key := countKey(evt.StreamID().String())

			current, _, err := readCount(ctx, store, key)
			if err != nil {
				return fmt.Errorf("journey projection read: %w", err)
			}

			if err := writeCount(ctx, store, key, current+1); err != nil {
				return fmt.Errorf("journey projection write: %w", err)
			}

			return nil
		},
		[]event.Type{journeyEventType},
	)
}

// newBenchQueryDispatcher creates a query.Dispatcher with typed handlers
// backed by the given kv.Store. Registers:
//   - getCountQueryType: reads a single stream counter (hit path)
//   - listCountsQueryType: reads a page of counters (paginated path)
//
// streamKeys is the full ordered list of count keys, used by the paginated
// handler to slice pages without scanning.
func newBenchQueryDispatcher(
	store kv.Store,
	streamIDs []id.StreamID,
) *query.Dispatcher {
	disp := query.NewDispatcher()

	_ = query.RegisterTyped[getCountQuery, CountResult](
		disp, getCountQueryType,
		func(ctx context.Context, q getCountQuery) (CountResult, error) {
			n, ok, err := readCount(ctx, store, countKey(q.streamID))
			if err != nil {
				return CountResult{}, err
			}

			if !ok {
				return CountResult{StreamID: q.streamID, Count: 0}, nil
			}

			return CountResult{StreamID: q.streamID, Count: n}, nil
		},
	)

	_ = query.RegisterTyped[listCountsQuery, query.PaginatedResult[CountResult]](
		disp, listCountsQueryType,
		func(ctx context.Context, q listCountsQuery) (query.PaginatedResult[CountResult], error) {
			page := query.NewPagination(q.page, q.pageSize)

			offset := min(page.Offset(), len(streamIDs))
			end := min(offset+int(q.pageSize), len(streamIDs))

			data := make([]CountResult, 0, end-offset)

			for i := offset; i < end; i++ {
				sid := streamIDs[i].String()

				n, _, err := readCount(ctx, store, countKey(sid))
				if err != nil {
					return query.PaginatedResult[CountResult]{}, err
				}

				data = append(data, CountResult{StreamID: sid, Count: n})
			}

			return query.NewPaginatedResult(data, uint(len(streamIDs)), page), nil
		},
	)

	return disp
}
