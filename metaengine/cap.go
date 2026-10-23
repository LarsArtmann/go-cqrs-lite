package metaengine

// Bounded projection caps: "keep the newest N rows" declared once on the query
// and enforced by the store, instead of every consumer hand-rolling a
// check-then-expunge converging loop.
//
// The declaration lives on the query ([WithCap]); enforcement is an explicit
// store maintenance call ([Store.EnforceCap] / [Store.EnforceCaps]). Keeping
// enforcement out of the fold hot path avoids paying a scan on every insert:
// a capped collection is trimmed when the caller says so (after a batch append,
// on a timer), which is the cadence bounded read models actually need.
//
// Replay semantics: trimming removes projection rows directly, it does not
// append a domain event. Rebuilding the projection (reset + replay) restores
// the full, uncapped row set, so a replay must be followed by another
// EnforceCap pass. This mirrors compaction: the trim is derived-state
// maintenance, not a fact in the event log (ADR-0136's invertibility ladder).

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"time"
)

var (
	// errCapRequiresMap is returned at plan/registration time when WithCap is
	// declared on a non-Map projection.
	errCapRequiresMap = errors.New(
		"WithCap requires a Map projection (insert fold)",
	)

	// errCapOrderByRequired is returned when WithCap has no order-by column.
	errCapOrderByRequired = errors.New("WithCap requires a non-empty orderBy column")

	// errCapKeyUnavailable is returned when the projection's result type does
	// not carry the query's key type, so trimmed rows cannot be addressed.
	errCapKeyUnavailable = errors.New(
		"WithCap requires the result type to carry the query's key type",
	)

	// errCapUnsupported is returned when the engine cannot scan or delete.
	errCapUnsupported = errors.New("engine does not support cap enforcement")

	// errCapCursorMissing is returned when an engine reports more rows but no
	// continuation cursor, which would make trimming loop forever.
	errCapCursorMissing = errors.New("engine reported more rows without a cursor")
)

// ErrCapRequiresMap is returned at Plan/RegisterQuery time when WithCap is
// declared on a query that is not a Map projection.
var ErrCapRequiresMap = errCapRequiresMap

// ErrCapUnsupported is returned by EnforceCap when the query's engine lacks the
// scan or delete capability needed to trim rows.
var ErrCapUnsupported = errCapUnsupported

// capTrimBatchSize bounds how many overflow rows one trim page deletes.
const capTrimBatchSize = 500

// validateCapConfig rejects a malformed WithCap declaration before the store
// is built: an empty order-by column has no deterministic victim order, a
// non-Map ADT has no rows to keep, and a result type without the key field
// cannot be trimmed at all.
func validateCapConfig(q queryMeta) error {
	cfg := q.QueryConfig()
	if cfg.capRows <= 0 {
		return nil
	}

	if cfg.capOrderBy == "" {
		return fmt.Errorf("query %q: %w", q.QueryName(), errCapOrderByRequired)
	}

	if q.QueryADT() != ADTMap {
		return fmt.Errorf("query %q: %w (ADT %s)", q.QueryName(), errCapRequiresMap, q.QueryADT())
	}

	if _, err := capKeyExtractor(q); err != nil {
		return fmt.Errorf("query %q: %w: %w", q.QueryName(), errCapKeyUnavailable, err)
	}

	return nil
}

// capKeyExtractor builds a value-to-key extractor from the query's insert fold:
// the fold declares both the key type and the result value type, and the
// trimmed rows arrive as values (never as event payloads).
func capKeyExtractor(q queryMeta) (func(any) any, error) {
	var ins *insertFold

	for _, fold := range q.QueryFolds() {
		if candidate, ok := fold.(*insertFold); ok {
			ins = candidate

			break
		}
	}

	if ins == nil {
		return nil, fmt.Errorf("%w: no insert fold", errCapRequiresMap)
	}

	if ins.keyType == nil || ins.valueType == nil {
		return nil, fmt.Errorf("%w: fold has no key/value type", errCapKeyUnavailable)
	}

	valueType := ins.valueType
	if valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}

	if valueType.Kind() != reflect.Struct {
		return nil, fmt.Errorf(
			"%w: result type %s is not a struct",
			errCapKeyUnavailable,
			valueType,
		)
	}

	sample := reflect.New(valueType).Interface()

	extractor, err := buildKeyExtractor(sample, ins.keyType)
	if err != nil {
		return nil, err
	}

	return extractor.(func(any) any), nil //nolint:forcetypeassert // buildKeyExtractor returns this exact signature
}

// EnforceCap trims one capped collection down to its declared maximum, keeping
// the newest rows as ordered by the WithCap order-by column (descending).
// Returns the number of rows removed; 0 means the collection was already within
// its cap (or the query declares no cap).
//
// Call it after a batch append or on a timer. It is safe to call concurrently
// with writes: the trim runs under the query's fold lock, and a write that
// lands after the trim simply makes the next pass remove one more row. It is
// NOT part of event replay: rebuilding the projection restores every row, so
// run EnforceCap again after a reset + replay.
func (s *Store) EnforceCap(ctx context.Context, queryName string) (int, error) {
	q, ok := s.lookupQuery(queryName)
	if !ok {
		return 0, fmt.Errorf("%w: %q", errNoQueryForName, queryName)
	}

	cfg := q.QueryConfig()
	if cfg.capRows <= 0 {
		return 0, nil
	}

	keyOf, err := capKeyExtractor(q)
	if err != nil {
		return 0, fmt.Errorf("metaengine.EnforceCap(%q): %w", queryName, err)
	}

	lock := s.foldLocks.get(q.QueryName())
	lock.Lock()
	defer lock.Unlock()

	eng := q.QueryEngine()
	if eng == nil {
		return 0, fmt.Errorf("metaengine.EnforceCap(%q): %w", queryName, errUnsupportedPattern)
	}

	// Keep page: the newest capRows rows. hasMore means older rows exist and
	// every row after this cursor is a trim victim.
	_, hasMore, cursor, err := capScanPage(
		ctx,
		eng,
		q.QueryName(),
		cfg.capOrderBy,
		cfg.capRows,
		nil,
	)
	if err != nil {
		return 0, fmt.Errorf("metaengine.EnforceCap(%q): keep scan: %w", queryName, err)
	}

	if !hasMore {
		return 0, nil
	}

	removed := 0

	for {
		items, more, next, scanErr := capScanPage(
			ctx, eng, q.QueryName(), cfg.capOrderBy, capTrimBatchSize, cursor,
		)
		if scanErr != nil {
			return removed, fmt.Errorf(
				"metaengine.EnforceCap(%q): trim scan: %w",
				queryName,
				scanErr,
			)
		}

		for _, item := range items {
			if deleteErr := s.capDelete(ctx, q, keyOf(item)); deleteErr != nil {
				return removed, fmt.Errorf("metaengine.EnforceCap(%q): %w", queryName, deleteErr)
			}

			removed++
		}

		if !more {
			return removed, nil
		}

		if next == nil {
			return removed, fmt.Errorf(
				"metaengine.EnforceCap(%q): %w",
				queryName,
				errCapCursorMissing,
			)
		}

		cursor = next
	}
}

// EnforceCaps trims every collection that declares a WithCap policy, in
// deterministic query-name order. Returns the total rows removed and the
// joined per-query failures (successful trims still count).
func (s *Store) EnforceCaps(ctx context.Context) (int, error) {
	names := s.cappedQueryNames()

	total := 0

	var errs []error

	for _, name := range names {
		removed, err := s.EnforceCap(ctx, name)
		total += removed

		if err != nil {
			errs = append(errs, err)
		}
	}

	return total, errors.Join(errs...)
}

// cappedQueryNames returns the sorted names of queries declaring WithCap.
func (s *Store) cappedQueryNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	capped := make(map[string]struct{}, len(s.queries))

	for name, q := range s.queries {
		if q.QueryConfig().capRows > 0 {
			capped[name] = struct{}{}
		}
	}

	return slices.Sorted(maps.Keys(capped))
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
