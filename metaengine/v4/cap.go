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
	"reflect"
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

	// Grouped validation runs FIRST: a WithCapPerGroup without its WithCap
	// base must be rejected even though the plain-cap checks below return
	// early for capRows <= 0.
	if err := validateCapGroupedConfig(q); err != nil {
		return err
	}

	if cfg.capRows <= 0 {
		return nil
	}

	if cfg.capOrderBy == "" {
		return fmt.Errorf("query %q: %w", q.QueryName(), errCapOrderByRequired)
	}

	if q.QueryADT() != ADTMap {
		return fmt.Errorf("query %q: %w (ADT %s)", q.QueryName(), errCapRequiresMap, q.QueryADT())
	}

	if _, err := capKeyExtractor(q, cfg.capGroupCol); err != nil {
		return fmt.Errorf("query %q: %w: %w", q.QueryName(), errCapKeyUnavailable, err)
	}

	return nil
}

// capKeyExtractor builds a value-to-key extractor from the query's insert fold:
// the fold declares both the key type and the result value type, and the
// trimmed rows arrive as values (never as event payloads). excludeGroupCol
// ("" for plain caps) skips the group column of a per-group cap: grouping
// by the key field itself is degenerate, and the exclusion resolves
// otherwise type-ambiguous result types (string ID + string Owner, ADR-0164).
func capKeyExtractor(q queryMeta, excludeGroupCol string) (func(any) any, error) {
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

	extractor, err := buildKeyExtractor(sample, ins.keyType, excludeGroupCol)
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
// with writes: the trim runs under the query's fold lock, inside the engine's
// transaction when the engine is transactional — the same connection → fold
// lock order the fold path holds (see withCapTrimTx), so a trim can never
// deadlock against a concurrent fold batch on a bounded pool. A write that
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

	keyOf, err := capKeyExtractor(q, cfg.capGroupCol)
	if err != nil {
		return 0, fmt.Errorf("metaengine.EnforceCap(%q): %w", queryName, err)
	}

	eng := q.QueryEngine()
	if eng == nil {
		return 0, fmt.Errorf("metaengine.EnforceCap(%q): %w", queryName, errUnsupportedPattern)
	}

	removed := 0

	trimErr := s.withCapTrimTx(ctx, eng, queryName, func(ctx context.Context) error {
		// Grouped pre-pass (ADR-0164): trim each group to its per-group bound
		// first; the keep-page scan below then enforces the global ceiling on
		// what survived. Both run under the fold lock this closure already
		// holds — enforceGroupedTrim never locks.
		if cfg.capGroupRows > 0 {
			grouped, groupErr := s.enforceGroupedTrim(ctx, q)
			removed += grouped
			if groupErr != nil {
				return groupErr
			}
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
			return fmt.Errorf("metaengine.EnforceCap(%q): keep scan: %w", queryName, err)
		}

		if !hasMore {
			return nil
		}

		for {
			items, more, next, scanErr := capScanPage(
				ctx, eng, q.QueryName(), cfg.capOrderBy, capTrimBatchSize, cursor,
			)
			if scanErr != nil {
				return fmt.Errorf(
					"metaengine.EnforceCap(%q): trim scan: %w",
					queryName,
					scanErr,
				)
			}

			for _, item := range items {
				if deleteErr := s.capDelete(ctx, q, keyOf(item)); deleteErr != nil {
					return fmt.Errorf("metaengine.EnforceCap(%q): %w", queryName, deleteErr)
				}

				removed++
			}

			if !more {
				return nil
			}

			if next == nil {
				return fmt.Errorf(
					"metaengine.EnforceCap(%q): %w",
					queryName,
					errCapCursorMissing,
				)
			}

			cursor = next
		}
	})
	if trimErr != nil {
		return removed, trimErr
	}

	return removed, nil
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
