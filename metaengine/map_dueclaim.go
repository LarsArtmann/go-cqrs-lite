package metaengine

import (
	"cmp"
	"context"
	"fmt"
	"time"
)

// MapDueClaimer is the degraded [DueClaimer] runtime for map-shaped engines
// (memory, pebble, bbolt, badger): claims live as ordinary map entries in the
// caller's collection, gated by per-key [MapUpdater.MapUpdate]
// read-modify-writes. Under an engine's serialized writes each RMW is atomic,
// so two concurrent claimers never take the same item — the conformance
// contract holds; only the scan is O(collection size), which the engine
// profile must declare via DegradedADTs[ADTDueClaim] (ADR-0142 amendment).
//
// Engines embed it; they do not hand-write claims:
//
//	c := metaengine.NewMapDueClaimer(eng)
//	func (e *myEngine) ClaimDue(ctx context.Context, req metaengine.ClaimDueRequest) ([]metaengine.DueClaim, error) {
//		return c.ClaimDue(ctx, req) // + the other DueClaimer methods
//	}
//
// Stored records are this package's JSON shape; nothing else should read the
// collection. Deleted items become tombstones (deleted=true) so the
// epoch-guarded delete and a same-key re-insert cannot interleave.
type MapDueClaimer struct {
	maps MapBackend
	rmw  MapUpdater
	scan ScanBackend
}

// NewMapDueClaimer builds the runtime over an engine implementing
// MapBackend, MapUpdater, and ScanBackend. Panics never happen here; a
// missing capability is the caller's (engine author's) wiring bug and surfaces
// as this error at construction.
func NewMapDueClaimer(eng Engine) (*MapDueClaimer, error) {
	maps, ok := eng.(MapBackend)
	if !ok {
		return nil, fmt.Errorf(
			"metaengine.NewMapDueClaimer: engine %s lacks MapBackend",
			eng.Profile().Name,
		)
	}

	rmw, ok := eng.(MapUpdater)
	if !ok {
		return nil, fmt.Errorf(
			"metaengine.NewMapDueClaimer: engine %s lacks MapUpdater",
			eng.Profile().Name,
		)
	}

	scan, ok := eng.(ScanBackend)
	if !ok {
		return nil, fmt.Errorf(
			"metaengine.NewMapDueClaimer: engine %s lacks ScanBackend",
			eng.Profile().Name,
		)
	}

	return &MapDueClaimer{maps: maps, rmw: rmw, scan: scan}, nil
}

// mapClaimRecord is the stored claim shape. Key is duplicated inside the
// record because scans return values, not keys, and every mutation needs the
// key to address the row.
type mapClaimRecord struct {
	Key        string    `json:"key"`
	DueAt      time.Time `json:"dueAt"`
	LeaseUntil time.Time `json:"leaseUntil"`
	Owner      string    `json:"owner"`
	Payload    []byte    `json:"payload"`
	Deleted    bool      `json:"deleted,omitempty"`
}

// reifyClaim decodes a stored claim across engine value shapes (typed struct
// on memory, map[string]any or raw JSON on SQL/KV engines).
func reifyClaim(raw any) (mapClaimRecord, bool) {
	if raw == nil {
		return mapClaimRecord{}, false
	}

	rec, err := reify[mapClaimRecord](raw)
	if err != nil {
		return mapClaimRecord{}, false
	}

	return rec, true
}

// ClaimInsert implements [DueClaimer.ClaimInsert]: idempotent by key — an
// existing live record wins, a tombstone (or absence) is replaced.
func (m *MapDueClaimer) ClaimInsert(
	ctx context.Context,
	collection, key string,
	dueAt time.Time,
	payload []byte,
) error {
	err := m.rmw.MapUpdate(ctx, collection, key, func(prev any) any {
		if rec, ok := reifyClaim(prev); ok && !rec.Deleted {
			return prev
		}

		return mapClaimRecord{Key: key, DueAt: dueAt, Payload: payload}
	})

	return err //nolint:wrapcheck // engine pass-through; callers add context
}

// ClaimDue implements [DueClaimer.ClaimDue]: scan the collection for due,
// unleased items (DueAt ascending, key tie-break), then per-key RMW-stamp the
// lease on the first limit claimable items; return exactly the ones THIS call
// won.
func (m *MapDueClaimer) ClaimDue(ctx context.Context, req ClaimDueRequest) ([]DueClaim, error) {
	now := req.resolvedNow()
	leaseUntil := now.Add(req.resolvedLease())
	limit := req.resolvedLimit()

	res, err := m.scan.MapScan(ctx, req.Collection, func(item any) bool {
		rec, ok := reifyClaim(item)

		return ok && !rec.Deleted && claimable(rec, now)
	}, func(a, b any) int {
		ra, _ := reifyClaim(a)
		rb, _ := reifyClaim(b)

		return compareClaims(ra, rb)
	}, nil, limit)
	if err != nil {
		return nil, fmt.Errorf("metaengine.MapDueClaimer.ClaimDue: scan: %w", err)
	}

	claimed := make([]DueClaim, 0, len(res.Items))

	for _, item := range res.Items {
		rec, ok := reifyClaim(item)
		if !ok {
			continue
		}

		won := false

		upd := m.rmw.MapUpdate(ctx, req.Collection, rec.Key, func(prev any) any {
			cur, ok := reifyClaim(prev)
			if !ok || cur.Deleted || !claimable(cur, now) {
				return prev // lost the race or not due: leave untouched
			}

			won = true

			cur.LeaseUntil = leaseUntil
			cur.Owner = req.Owner

			return cur
		})
		if err := upd; err != nil {
			return claimed, fmt.Errorf(
				"metaengine.MapDueClaimer.ClaimDue: stamp %q: %w",
				rec.Key,
				err,
			)
		}

		if won {
			claimed = append(claimed, DueClaim{
				Key:        rec.Key,
				DueAt:      rec.DueAt,
				Payload:    rec.Payload,
				LeaseUntil: leaseUntil,
			})
		}

		if len(claimed) >= limit {
			break
		}
	}

	return claimed, nil
}

// RenewLease implements [DueClaimer.RenewLease]: extend only while the claim
// is live and owned by owner; otherwise [ErrClaimLeaseNotHeld].
func (m *MapDueClaimer) RenewLease(
	ctx context.Context,
	collection, key, owner string,
	extend time.Duration,
	now time.Time,
) error {
	held := false

	err := m.rmw.MapUpdate(ctx, collection, key, func(prev any) any {
		cur, ok := reifyClaim(prev)
		if ok && !cur.Deleted && cur.Owner == owner && cur.LeaseUntil.After(now) {
			held = true
			cur.LeaseUntil = now.Add(extend)

			return cur
		}

		return prev
	})
	if err != nil {
		return fmt.Errorf("metaengine.MapDueClaimer.RenewLease: %w", err)
	}

	if !held {
		return fmt.Errorf("%w: key %q", ErrClaimLeaseNotHeld, key)
	}

	return nil
}

// ClaimDelete implements [DueClaimer.ClaimDelete]: unconditional, idempotent.
func (m *MapDueClaimer) ClaimDelete(ctx context.Context, collection, key string) error {
	err := m.maps.MapDelete(ctx, collection, key) //nolint:wrapcheck // engine pass-through
	if err != nil {
		return fmt.Errorf("metaengine.MapDueClaimer.ClaimDelete: %w", err)
	}

	return nil
}

// ClaimDeleteIfDue implements [DueClaimer.ClaimDeleteIfDue]: the item is
// tombstoned only when its DueAt still matches — a re-scheduled generation
// under the same key survives a stale finalizer (the MarkFired race fix).
func (m *MapDueClaimer) ClaimDeleteIfDue(
	ctx context.Context,
	collection, key string,
	dueAt time.Time,
) error {
	err := m.rmw.MapUpdate(ctx, collection, key, func(prev any) any {
		cur, ok := reifyClaim(prev)
		if !ok || cur.Deleted || !cur.DueAt.Equal(dueAt) {
			return prev
		}

		return mapClaimRecord{Key: key, Deleted: true}
	})
	if err != nil {
		return fmt.Errorf("metaengine.MapDueClaimer.ClaimDeleteIfDue: %w", err)
	}

	return nil
}

func claimable(rec mapClaimRecord, now time.Time) bool {
	return !rec.DueAt.After(now) && !rec.LeaseUntil.After(now)
}

func compareClaims(a, b mapClaimRecord) int {
	if c := a.DueAt.Compare(b.DueAt); c != 0 {
		return c
	}

	return cmp.Compare(a.Key, b.Key)
}
