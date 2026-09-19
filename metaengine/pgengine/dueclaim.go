package pgengine

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"
)

// wireClaimkit attaches the ONE shared SQL claim/dedup runtime (ADR-0142):
// the embedded *claimkit.Claims and *claimkit.Dedup promote their methods,
// so the Postgres engine satisfies metaengine.DueClaimer,
// metaengine.FactSink, and metaengine.DedupStore without a single
// engine-specific claim statement — the CTE FOR UPDATE SKIP LOCKED claim,
// owner-fenced renewal, epoch delete, same-tx facts, and the atomic dedup
// upsert all come from claimkit. Semantics are pinned by
// adttest.AssertDueClaimer / AssertDedupStore (dueclaim_test.go).
func (e *pgEngine) wireClaimkit() error {
	ctx := context.Background()

	claims, err := claimkit.New(ctx, e.db, claiming.DialectPostgres)
	if err != nil {
		return fmt.Errorf("pgengine: claimkit claims: %w", err)
	}

	dedup, err := claimkit.NewDedup(ctx, e.db, claiming.DialectPostgres)
	if err != nil {
		return fmt.Errorf("pgengine: claimkit dedup: %w", err)
	}

	e.Claims = claims
	e.Dedup = dedup

	return nil
}
