package duckdbengine

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"
)

// wireClaimkit attaches the ONE shared SQL claim/dedup runtime (ADR-0142):
// the embedded *claimkit.Claims and *claimkit.Dedup promote their methods,
// so the DuckDB engine satisfies metaengine.DueClaimer,
// metaengine.FactSink, and metaengine.DedupStore without a single
// engine-specific claim statement. The DuckDB dialect rides the numbered
// IN-subquery UPDATE..RETURNING claim shape (DuckDB supports RETURNING on
// UPDATE); DuckDB's single-process single-writer serialization replaces row
// locks. Semantics are pinned by adttest.AssertDueClaimer / AssertDedupStore
// (see dueclaim_cgo_test.go).
func (e *duckdbEngine) wireClaimkit() error {
	ctx := context.Background()

	claims, err := claimkit.New(ctx, e.db, claiming.DialectDuckDB)
	if err != nil {
		return fmt.Errorf("metaengine: duckdb claimkit claims: %w", err)
	}

	dedup, err := claimkit.NewDedup(ctx, e.db, claiming.DialectDuckDB)
	if err != nil {
		return fmt.Errorf("metaengine: duckdb claimkit dedup: %w", err)
	}

	e.Claims = claims
	e.Dedup = dedup

	return nil
}
