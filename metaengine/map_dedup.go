package metaengine

import (
	"context"
	"fmt"
	"time"
)

// MapDedupStore is the degraded [DedupStore] runtime for map-shaped engines
// (memory, pebble, bbolt, badger): dedup keys live as ordinary map entries
// whose value is an expiry timestamp, and the check-and-set window is a
// per-key [MapUpdater.MapUpdate] read-modify-write — atomic under the
// engine's serialized writes, so two concurrent DedupCheckAndRecord calls for
// one key see exactly one true (the conformance contract). Engines embed it
// (ADR-0142 amendment); the engine profile declares the degraded scan via
// DegradedADTs[ADTDedup].
type MapDedupStore struct {
	maps MapBackend
	rmw  MapUpdater
	scan ScanBackend
}

// NewMapDedupStore builds the runtime over an engine implementing MapBackend,
// MapUpdater, and ScanBackend.
func NewMapDedupStore(eng Engine) (*MapDedupStore, error) {
	maps, ok := eng.(MapBackend)
	if !ok {
		return nil, fmt.Errorf("metaengine.NewMapDedupStore: engine %s lacks MapBackend", eng.Profile().Name)
	}

	rmw, ok := eng.(MapUpdater)
	if !ok {
		return nil, fmt.Errorf("metaengine.NewMapDedupStore: engine %s lacks MapUpdater", eng.Profile().Name)
	}

	scan, ok := eng.(ScanBackend)
	if !ok {
		return nil, fmt.Errorf("metaengine.NewMapDedupStore: engine %s lacks ScanBackend", eng.Profile().Name)
	}

	return &MapDedupStore{maps: maps, rmw: rmw, scan: scan}, nil
}

// mapDedupRecord is the stored window shape. Key is duplicated inside the
// record because sweeps scan values, not keys.
type mapDedupRecord struct {
	Key       string    `json:"key"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func reifyDedup(raw any) (mapDedupRecord, bool) {
	if raw == nil {
		return mapDedupRecord{}, false
	}

	rec, err := reify[mapDedupRecord](raw)
	if err != nil {
		return mapDedupRecord{}, false
	}

	return rec, true
}

// DedupCheckAndRecord implements [DedupStore.DedupCheckAndRecord]: the atomic
// CAS window. A live unexpired record is kept and reported as seen; an
// expired or absent key is (re-)recorded with a fresh window and reported
// unseen.
func (m *MapDedupStore) DedupCheckAndRecord(
	ctx context.Context,
	collection, key string,
	ttl time.Duration,
	now time.Time,
) (bool, error) {
	seen := false

	err := m.rmw.MapUpdate(ctx, collection, key, func(prev any) any {
		cur, ok := reifyDedup(prev)
		if ok && cur.ExpiresAt.After(now) {
			seen = true

			return prev
		}

		return mapDedupRecord{Key: key, ExpiresAt: now.Add(ttl)}
	})
	if err != nil {
		return false, fmt.Errorf("metaengine.MapDedupStore.DedupCheckAndRecord: %w", err)
	}

	return seen, nil
}

// DedupSeen implements [DedupStore.DedupSeen]: a read with lazy expiry — a
// lapsed record reads as unseen and is deleted opportunistically.
func (m *MapDedupStore) DedupSeen(
	ctx context.Context,
	collection, key string,
	now time.Time,
) (bool, error) {
	raw, ok, err := m.maps.MapGet(ctx, collection, key)
	if err != nil {
		return false, fmt.Errorf("metaengine.MapDedupStore.DedupSeen: %w", err)
	}

	if !ok {
		return false, nil
	}

	rec, ok := reifyDedup(raw)
	if !ok {
		return false, nil
	}

	if rec.ExpiresAt.After(now) {
		return true, nil
	}

	_ = m.maps.MapDelete(ctx, collection, key)

	return false, nil
}

// DedupSweep implements [DedupStore.DedupSweep]: delete every expired record,
// return the count.
func (m *MapDedupStore) DedupSweep(ctx context.Context, collection string, now time.Time) (int, error) {
	res, err := m.scan.MapScan(ctx, collection, func(item any) bool {
		rec, ok := reifyDedup(item)

		return ok && !rec.ExpiresAt.After(now)
	}, nil, nil, DefaultClaimLimit)
	if err != nil {
		return 0, fmt.Errorf("metaengine.MapDedupStore.DedupSweep: %w", err)
	}

	removed := 0

	for _, item := range res.Items {
		rec, ok := reifyDedup(item)
		if !ok {
			continue
		}

		if err := m.maps.MapDelete(ctx, collection, rec.Key); err == nil {
			removed++
		}
	}

	return removed, nil
}
