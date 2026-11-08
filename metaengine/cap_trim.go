package metaengine

// Cap-trim execution mechanics: the engine-interaction half of bounded
// projection caps (ADR: EnforceCap). cap.go owns the declaration surface
// (validation, WithCap plumbing, the EnforceCap/EnforceCaps orchestration);
// this file owns what a trim physically does — paging scans, row deletes,
// and the lock/transaction ordering that keeps trims deadlock-free against
// concurrent fold batches (#123).

import (
	"context"
	"fmt"
	"time"
)

// withCapTrimTx runs a cap trim under the query's fold lock, acquiring the
// engine's transaction — and with it the pooled connection — BEFORE the lock
// whenever the engine is transactional. The fold path's order is fixed:
// dispatchFoldsCoreLocked opens the engine transaction (connection first,
// RunInTx) and takes each query's fold lock INSIDE it; a trim that instead
// held the fold lock across engine scans and deletes inverted that order and
// could deadlock against a concurrent fold batch whenever the connection pool
// was exhausted — the fold held a connection wanting the fold lock, the trim
// held the fold lock wanting a connection (observed as order-dependent 60-600s
// hangs in webphone's projection host over sqliteengine). Matching the fold
// path's order — transaction first, fold lock inside — closes the cycle for
// every transactional engine; non-transactional engines keep the plain fold
// lock, whose holders never wait on a pool.
func (s *Store) withCapTrimTx(
	ctx context.Context,
	eng Engine,
	queryName string,
	trim func(context.Context) error,
) error {
	lock := s.foldLocks.get(queryName)

	if tx, ok := eng.(Transactional); ok {
		return runInTxWithRetry(ctx, tx, func(ctx context.Context) error {
			lock.Lock()
			defer lock.Unlock()

			return trim(ctx)
		})
	}

	lock.Lock()
	defer lock.Unlock()

	return trim(ctx)
}

// capScanPage reads one descending page of a collection ordered by column.
// Descending order puts the newest rows first, so the keep page is a prefix and
// every later page is entirely overflow.
func capScanPage(
	ctx context.Context,
	eng Engine,
	collection, column string,
	limit int,
	cursor any,
) ([]any, bool, any, error) {
	sortSpec := &SortSpec{Column: column, Desc: true}

	if pushdown, ok := eng.(PushdownScan); ok {
		res, err := pushdown.PushdownMapScan(ctx, collection, nil, sortSpec, cursor, limit)
		if err != nil {
			return nil, false, nil, err //nolint:wrapcheck // caller wraps with query context
		}

		return res.Items, res.HasMore, res.NextCursor, nil
	}

	if scanBackend, ok := eng.(ScanBackend); ok {
		cfg := scanConfig{sort: sortSpec}

		res, err := scanBackend.MapScan(
			ctx, collection, nil, buildClosureSort(cfg), rawCursorValue(cursor), limit,
		)
		if err != nil {
			return nil, false, nil, err //nolint:wrapcheck // caller wraps with query context
		}

		return res.Items, res.HasMore, normalizeClosureCursor(res.NextCursor, cfg), nil
	}

	return nil, false, nil, fmt.Errorf("%w: %s", errCapUnsupported, eng.Profile().Name)
}

// capDelete removes one trimmed row, honoring temporal engines (a deletion is a
// timestamped write there, ADR-0141) and notifying live subscribers.
func (s *Store) capDelete(ctx context.Context, q queryMeta, key any) error {
	col := q.QueryName()
	eng := q.QueryEngine()

	if vw, ok := eng.(VersionedWriter); ok && EngineVersionsCells(eng) {
		if err := vw.MapDeleteAt(ctx, col, key, time.Now()); err != nil {
			return fmt.Errorf("map delete-at %s: %w", col, err)
		}

		s.notifyLive(q, col, key, nil)

		return nil
	}

	mb, ok := eng.(MapBackend)
	if !ok {
		return unsupportedEngine(errUnsupportedMapOps, eng.Profile().Name)
	}

	if err := mb.MapDelete(ctx, col, key); err != nil {
		return fmt.Errorf("map delete %s: %w", col, err)
	}

	s.notifyLive(q, col, key, nil)

	return nil
}
