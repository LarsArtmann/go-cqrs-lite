package metaengine

import (
	"context"
	"time"
)

// ADR-0142 wiring: the memory engine satisfies DueClaimer and DedupStore by
// delegating to the shared Map runtimes — the degraded reference every
// map-shaped engine embeds. The runtimes are stateless (all state lives in
// the engine's maps), so construction is free and needs no initialization
// ordering. Defined here, not in memory_engine.go, because that file sits on
// the file-size ratchet baseline (methods may live in any file of a package).

var (
	_ DueClaimer = (*memoryEngine)(nil)
	_ DedupStore = (*memoryEngine)(nil)
)

func (m *memoryEngine) claimer() *MapDueClaimer {
	return &MapDueClaimer{maps: m, rmw: m, scan: m}
}

func (m *memoryEngine) deduper() *MapDedupStore {
	return &MapDedupStore{maps: m, rmw: m, scan: m}
}

func (m *memoryEngine) ClaimInsert(
	ctx context.Context, collection, key string, dueAt time.Time, payload []byte,
) error {
	return m.claimer().ClaimInsert(ctx, collection, key, dueAt, payload)
}

func (m *memoryEngine) ClaimDue(ctx context.Context, req ClaimDueRequest) ([]DueClaim, error) {
	return m.claimer().ClaimDue(ctx, req)
}

func (m *memoryEngine) RenewLease(
	ctx context.Context, collection, key, owner string, extend time.Duration, now time.Time,
) error {
	return m.claimer().RenewLease(ctx, collection, key, owner, extend, now)
}

func (m *memoryEngine) ClaimDelete(ctx context.Context, collection, key string) error {
	return m.claimer().ClaimDelete(ctx, collection, key)
}

func (m *memoryEngine) ClaimDeleteIfDue(
	ctx context.Context, collection, key string, dueAt time.Time,
) error {
	return m.claimer().ClaimDeleteIfDue(ctx, collection, key, dueAt)
}

func (m *memoryEngine) DedupCheckAndRecord(
	ctx context.Context, collection, key string, ttl time.Duration, now time.Time,
) (bool, error) {
	return m.deduper().DedupCheckAndRecord(ctx, collection, key, ttl, now)
}

func (m *memoryEngine) DedupSeen(
	ctx context.Context, collection, key string, now time.Time,
) (bool, error) {
	return m.deduper().DedupSeen(ctx, collection, key, now)
}

func (m *memoryEngine) DedupSweep(
	ctx context.Context, collection string, now time.Time,
) (int, error) {
	return m.deduper().DedupSweep(ctx, collection, now)
}
