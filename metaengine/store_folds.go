package metaengine

// Fold application: the Store's write path. applyFold dispatches a declared
// fold onto the query engine's backend capabilities; the temporal branches
// (ADR-0141) stamp every write with the event's time on versioned engines so
// replays rebuild the true temporal order.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func (s *Store) applyFold(
	ctx context.Context,
	q queryMeta,
	fold Fold,
	rec record.Record,
	payload any,
) (err error) {
	start := time.Now()

	// Wrap errors with structured context for debugging. Registered first so
	// it runs last (LIFO) — hooks and panic recovery see the raw error.
	defer func() {
		if err != nil {
			err = &ApplyError{
				Query:     q.QueryName(),
				EventType: fold.EventType(),
				FoldKind:  fold.Kind(),
				Cause:     err,
			}
		}
	}()

	defer func() {
		if s.hooks != nil && s.hooks.OnFold != nil {
			s.hooks.OnFold(q.QueryName(), fold.EventType(), fold.Kind(), time.Since(start), err)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			poisonErr := fmt.Errorf("%w: collection %q, panic: %v", ErrPoisoned, q.QueryName(), r)
			s.poison.Poison(q.QueryName(), poisonErr)
			err = poisonErr
		}
	}()

	// Raw-JSON payloads (ApplyEncoded/ApplyEncodedRecord and their EventLog
	// replays) decode into this fold's expected event type here — the single
	// funnel every dispatch path (primary folds, shadows, replays) shares.
	payload, decodeErr := decodeRawFoldPayload(fold, payload)
	if decodeErr != nil {
		return decodeErr
	}

	switch f := fold.(type) {
	case *insertFold:
		return s.applyFoldInsert(ctx, q, f, rec, payload)
	case *updateFold:
		return s.applyFoldUpdate(ctx, q, f, rec, payload)
	case *removeFold:
		return s.applyFoldRemove(ctx, q, f, rec, payload)
	case *countFold:
		return s.applyFoldCount(ctx, q, f, rec, payload)
	case *edgeFold:
		return s.applyFoldEdge(ctx, q, f, rec, payload)
	case *edgeRemoveFold:
		return s.applyFoldEdgeRemove(ctx, q, f, rec, payload)
	case *setFold:
		return s.applyFoldSet(ctx, q, f, rec, payload)
	case *skipFold:
		return nil
	case *multiInsertFold:
		return s.applyFoldMultiInsert(ctx, q, f, rec, payload)
	case *appendFold:
		return s.applyFoldAppend(ctx, q, f, rec, payload)
	case *vectorFold:
		return s.applyFoldVector(ctx, q, f, rec, payload)
	case *searchFold:
		return s.applyFoldSearch(ctx, q, f, rec, payload)
	case *spatialFold:
		return s.applyFoldSpatial(ctx, q, f, rec, payload)
	default:
		return fmt.Errorf("%w: %T", errUnknownFoldKind, fold)
	}
}

func (s *Store) applyFoldInsert(
	ctx context.Context,
	q queryMeta,
	fold *insertFold,
	rec record.Record,
	payload any,
) error {
	key, value := fold.invoke(rec, payload)
	col := q.QueryName()

	// Temporal engines (ADR-0141 §3): stamp the cell with the event's time so
	// replays rebuild the true temporal order. Plain engines take the
	// untimestamped path.
	if vw, ok := q.QueryEngine().(VersionedWriter); ok && EngineVersionsCells(q.QueryEngine()) {
		if err := vw.MapSetAt(ctx, col, key, value, CellTimestamp(rec)); err != nil {
			return fmt.Errorf("map set-at %s: %w", col, err)
		}

		s.notifyLive(q, col, key, value)

		return nil
	}

	if mb, ok := q.QueryEngine().(MapBackend); ok {
		if err := mb.MapSet(ctx, col, key, value); err != nil {
			return fmt.Errorf("map set %s: %w", col, err)
		}

		s.notifyLive(q, col, key, value)

		return nil
	}

	return unsupportedEngine(errUnsupportedMapOps, q.QueryEngine().Profile().Name)
}

func (s *Store) applyFoldUpdate(
	ctx context.Context,
	q queryMeta,
	fold *updateFold,
	rec record.Record,
	payload any,
) error {
	key := fold.keyExtractor(payload)
	col := q.QueryName()
	ts := CellTimestamp(rec)

	// Temporal engines (ADR-0141 §3): event-time-stamped updates. The atomic
	// VersionedUpdater keeps the fold a single write-shaped engine call; the
	// VersionedWriter fallback reads latest under the dispatch path's per-query
	// fold locks (runtime_backend.go, replicator.go).
	if EngineVersionsCells(q.QueryEngine()) {
		if vu, ok := q.QueryEngine().(VersionedUpdater); ok {
			var updatedVal any

			if err := vu.MapUpdateAt(ctx, col, key, func(prev any) any {
				updatedVal = fold.invoke(rec, payload, prev)

				return updatedVal
			}, ts); err != nil {
				return fmt.Errorf("map update-at %s: %w", col, err)
			}

			s.notifyLive(q, col, key, updatedVal)

			return nil
		}

		if vw, ok := q.QueryEngine().(VersionedWriter); ok {
			return s.applyFoldUpdateVersioned(ctx, q, vw, col, key, fold, rec, payload)
		}
	}

	if mu, ok := q.QueryEngine().(MapUpdater); ok {
		var updatedVal any

		if err := mu.MapUpdate(ctx, col, key, func(prev any) any {
			updatedVal = fold.invoke(rec, payload, prev)

			return updatedVal
		}); err != nil {
			return fmt.Errorf("map update %s: %w", col, err)
		}

		s.notifyLive(q, col, key, updatedVal)

		return nil
	}

	if mapBackend, ok := q.QueryEngine().(MapBackend); ok {
		prev, exists, err := mapBackend.MapGet(ctx, col, key)
		if err != nil {
			return fmt.Errorf("map get %s: %w", col, err)
		}

		var prevVal any
		if exists {
			prevVal = prev
		}

		updated := fold.invoke(rec, payload, prevVal)

		if err := mapBackend.MapSet(ctx, col, key, updated); err != nil {
			return fmt.Errorf("map set %s: %w", col, err)
		}

		s.notifyLive(q, col, key, updated)

		return nil
	}

	return unsupportedEngine(errUnsupportedMapOps, q.QueryEngine().Profile().Name)
}

// applyFoldUpdateVersioned is the VersionedWriter branch of update folds:
// resolve the current latest value, fold, and write the result stamped with
// the event's time so as-of reads and replays stay temporally truthful.
func (s *Store) applyFoldUpdateVersioned(
	ctx context.Context,
	q queryMeta,
	vw VersionedWriter,
	col string,
	key any,
	fold *updateFold,
	rec record.Record,
	payload any,
) error {
	prev, found, err := s.latestForVersioned(ctx, q, key)
	if err != nil {
		return err
	}

	var prevVal any
	if found {
		prevVal = prev
	}

	updated := fold.invoke(rec, payload, prevVal)

	if err := vw.MapSetAt(ctx, col, key, updated, CellTimestamp(rec)); err != nil {
		return fmt.Errorf("map set-at %s: %w", col, err)
	}

	s.notifyLive(q, col, key, updated)

	return nil
}

// latestForVersioned resolves the current latest value for a key on a
// VersionedWriter engine: the plain MapBackend read when available (the hot
// path — temporal engines keep the latest cell), otherwise an as-of-now read
// through VersionedStorage.
func (s *Store) latestForVersioned(
	ctx context.Context,
	q queryMeta,
	key any,
) (any, bool, error) {
	if mb, ok := q.QueryEngine().(MapBackend); ok {
		prev, found, err := mb.MapGet(ctx, q.QueryName(), key)
		if err != nil {
			return nil, false, fmt.Errorf("map get %s: %w", q.QueryName(), err)
		}

		return prev, found, nil
	}

	if vs, ok := q.QueryEngine().(VersionedStorage); ok {
		prev, err := vs.MapGetAsOf(ctx, q.QueryName(), fmt.Sprint(key), time.Now())
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, false, nil
			}

			return nil, false, fmt.Errorf("map get-as-of %s: %w", q.QueryName(), err)
		}

		return prev, true, nil
	}

	return nil, false, unsupportedEngine(errUnsupportedMapOps, q.QueryEngine().Profile().Name)
}

func (s *Store) applyFoldRemove(
	ctx context.Context,
	q queryMeta,
	fold *removeFold,
	rec record.Record,
	payload any,
) error {
	key := fold.keyExtractor(payload)
	col := q.QueryName()

	// Temporal engines tombstone at the event's time (ADR-0141 §1): deletion
	// is a timestamped write, never a hard erase.
	if vw, ok := q.QueryEngine().(VersionedWriter); ok && EngineVersionsCells(q.QueryEngine()) {
		if err := vw.MapDeleteAt(ctx, col, key, CellTimestamp(rec)); err != nil {
			return fmt.Errorf("map delete-at %s: %w", col, err)
		}

		s.notifyLive(q, col, key, nil)

		return nil
	}

	if mb, ok := q.QueryEngine().(MapBackend); ok {
		if err := mb.MapDelete(ctx, col, key); err != nil {
			return fmt.Errorf("map delete %s: %w", col, err)
		}

		s.notifyLive(q, col, key, nil)

		return nil
	}

	return unsupportedEngine(errUnsupportedMapOps, q.QueryEngine().Profile().Name)
}
