package metaengine

// Per-group projection caps (ADR-0164, issue #114): WithCap bounds a WHOLE
// collection; WithCapPerGroup bounds each key group inside it (per-owner
// quotas, per-tenant windows). Declaration rides on WithCap — its orderBy
// picks the survivors within each group, its capRows stays the global
// ceiling — so the two policies compose instead of competing.

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

var (
	// errCapGroupRequiresWithCap is returned when WithCapPerGroup is declared
	// without the WithCap declaration that supplies the ordering.
	errCapGroupRequiresWithCap = errors.New(
		"WithCapPerGroup requires a WithCap declaration (its orderBy orders survivors within each group)",
	)

	// errCapGroupColumnInvalid is returned when the group column (or the
	// order-by column of a grouped cap) is not a plain identifier — both flow
	// into engine SQL text, so anything else is a build-time error.
	errCapGroupColumnInvalid = errors.New(
		"WithCapPerGroup columns must be plain identifiers (letters, digits, underscore)",
	)

	// errCapGroupColumnMissing is returned when the result type carries no
	// field matching the declared group column (name or json tag,
	// case-insensitive).
	errCapGroupColumnMissing = errors.New(
		"WithCapPerGroup group column not found on the result type",
	)
)

// ErrCapGroupPushdownUnsupported is returned by a [CapGroupTrimmer] that
// cannot push a particular grouped trim down (cell versioning active, a
// planned table without the column, a non-identifier column name); the store
// then trims the collection through its portable Go path instead.
var ErrCapGroupPushdownUnsupported = errors.New(
	"engine cannot push this grouped cap trim down",
)

// CapGroupTrimmer is the optional engine capability behind grouped caps: one
// statement keeps the newest perGroupRows rows of EVERY group in a collection
// (window-function pushdown). Returns the deleted keys so the store can
// notify live subscribers; engines that must not bulk-delete (temporal
// engines) return [ErrCapGroupPushdownUnsupported] and stay correct via the
// store's Go path.
type CapGroupTrimmer interface {
	Engine

	// TrimGroupedCap deletes rows beyond the newest perGroupRows of each
	// distinct groupCol value, ordered by orderBy descending, and returns
	// the deleted keys (the engine's native key rendering).
	TrimGroupedCap(
		ctx context.Context,
		collection, groupCol, orderBy string,
		perGroupRows int,
	) ([]any, error)
}

// validateCapGroupedConfig rejects a malformed WithCapPerGroup declaration at
// plan/registration time. Called from validateCapConfig after the plain-cap
// checks, so a grouped cap also satisfies every WithCap rule.
func validateCapGroupedConfig(q queryMeta) error {
	cfg := q.QueryConfig()
	if cfg.capGroupRows <= 0 {
		return nil
	}

	if cfg.capRows <= 0 || cfg.capOrderBy == "" {
		return fmt.Errorf("query %q: %w", q.QueryName(), errCapGroupRequiresWithCap)
	}

	if !isPlainIdent(cfg.capGroupCol) || !isPlainIdent(cfg.capOrderBy) {
		return fmt.Errorf(
			"query %q: %w: group %q order %q",
			q.QueryName(), errCapGroupColumnInvalid, cfg.capGroupCol, cfg.capOrderBy,
		)
	}

	if _, err := capGroupExtractor(q); err != nil {
		return fmt.Errorf("query %q: %w", q.QueryName(), err)
	}

	return nil
}

// isPlainIdent reports whether name is a bare SQL-safe identifier. Group-cap
// column names flow into engine SQL text (quoted or inside a JSON path), so
// anything beyond [A-Za-z0-9_] is rejected at declaration time.
func isPlainIdent(name string) bool {
	if name == "" {
		return false
	}

	for i := range len(name) {
		c := name[i] //art-dupl:accept module-boundary twin (see sqliteengine/cap_grouped.go)

		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_':
		case c >= '0' && c <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}

	return true
}

// capGroupExtractor builds a value-to-group-key extractor over the query's
// RESULT type: the group column matches a struct field by json tag or field
// name (case-insensitive), exactly the matching [ExtractFields] applies, so
// the Go grouping path and SQL json_extract agree on which field groups a row.
func capGroupExtractor(q queryMeta) (func(any) any, error) {
	t := q.QueryResultType()
	if t == nil {
		return nil, fmt.Errorf("%w: no result type", errCapGroupColumnMissing)
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: result type %s is not a struct", errCapGroupColumnMissing, t)
	}

	col := q.QueryConfig().capGroupCol

	idx := -1

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		if strings.EqualFold(JSONFieldName(f), col) || strings.EqualFold(f.Name, col) {
			idx = i

			break
		}
	}

	if idx < 0 {
		return nil, fmt.Errorf("%w: %q on %s", errCapGroupColumnMissing, col, t)
	}

	jsonName := JSONFieldName(t.Field(idx))

	return func(value any) any {
		return fieldValueAt(value, idx, jsonName)
	}, nil
}

// EnforceCapGrouped trims one grouped-capped collection: each distinct group
// column value keeps at most its declared perGroupRows newest rows (ordered
// by the WithCap order-by column, descending). Returns the number of rows
// removed; 0 means every group was already within its cap (or the query
// declares no grouped cap).
//
// Same contract as [Store.EnforceCap]: call it after a batch append or on a
// timer, it is safe to call concurrently with writes (the query's fold lock
// serializes trims, taken inside the engine's transaction — withCapTrimTx
// keeps the fold path's connection → lock order), and it is NOT part of
// replay — reset + replay restores full rows, so run it again after a rebuild.
func (s *Store) EnforceCapGrouped(ctx context.Context, queryName string) (int, error) {
	q, ok := s.lookupQuery(queryName)
	if !ok {
		return 0, fmt.Errorf("%w: %q", errNoQueryForName, queryName)
	}

	if q.QueryConfig().capGroupRows <= 0 {
		return 0, nil
	}

	eng := q.QueryEngine()
	if eng == nil {
		return 0, fmt.Errorf("metaengine.EnforceCapGrouped(%q): %w", queryName, errUnsupportedPattern)
	}

	removed := 0

	trimErr := s.withCapTrimTx(ctx, eng, queryName, func(ctx context.Context) error {
		n, trimErr := s.enforceGroupedTrim(ctx, q)
		removed = n

		return trimErr
	})

	return removed, trimErr
}

// enforceGroupedTrim is the grouped trim over an already-resolved query: the
// engine's [CapGroupTrimmer] pushdown when available (and applicable), the
// portable Go scan otherwise. The caller holds the query's fold lock.
func (s *Store) enforceGroupedTrim(ctx context.Context, q queryMeta) (int, error) {
	queryName := q.QueryName()
	cfg := q.QueryConfig()

	keyOf, err := capKeyExtractor(q, cfg.capGroupCol)
	if err != nil {
		return 0, fmt.Errorf("metaengine.EnforceCapGrouped(%q): %w", queryName, err)
	}

	eng := q.QueryEngine()
	if eng == nil {
		return 0, fmt.Errorf("metaengine.EnforceCapGrouped(%q): %w", queryName, errUnsupportedPattern)
	}

	if trimmer, ok := eng.(CapGroupTrimmer); ok {
		keys, trimErr := trimmer.TrimGroupedCap(
			ctx, queryName, cfg.capGroupCol, cfg.capOrderBy, cfg.capGroupRows,
		)

		if trimErr == nil {
			for _, key := range keys {
				s.notifyLive(q, queryName, key, nil)
			}

			return len(keys), nil
		}

		if !errors.Is(trimErr, ErrCapGroupPushdownUnsupported) {
			return 0, fmt.Errorf(
				"metaengine.EnforceCapGrouped(%q): engine trim: %w", queryName, trimErr,
			)
		}
		// Pushdown refused for this collection: fall through to the Go path.
	}

	groupOf, err := capGroupExtractor(q)
	if err != nil {
		return 0, fmt.Errorf("metaengine.EnforceCapGrouped(%q): %w", queryName, err)
	}

	// One newest-first scan of the whole collection: the first capGroupRows
	// rows seen per group survive, everything after is a trim victim.
	counts := make(map[any]int)
	removed := 0

	cursor := any(nil)

	for {
		items, hasMore, next, scanErr := capScanPage(
			ctx, eng, queryName, cfg.capOrderBy, capTrimBatchSize, cursor,
		)
		if scanErr != nil {
			return removed, fmt.Errorf(
				"metaengine.EnforceCapGrouped(%q): trim scan: %w", queryName, scanErr,
			)
		}

		for _, item := range items {
			group := groupOf(item)
			counts[group]++

			if counts[group] > cfg.capGroupRows {
				if delErr := s.capDelete(ctx, q, keyOf(item)); delErr != nil {
					return removed, fmt.Errorf(
						"metaengine.EnforceCapGrouped(%q): %w", queryName, delErr,
					)
				}

				removed++
			}
		}

		if !hasMore {
			return removed, nil
		}

		if next == nil {
			return removed, fmt.Errorf(
				"metaengine.EnforceCapGrouped(%q): %w", queryName, errCapCursorMissing,
			)
		}

		cursor = next
	}
}

// cappedQueryNames returns the sorted names of queries declaring WithCap
// (grouped declarations ride a WithCap base, so they are included and reach
// their grouped pass through EnforceCap's composition). Lives here, next to
// the grouped enforcement it also feeds, to keep cap.go inside the
// 350-line file ratchet.
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
