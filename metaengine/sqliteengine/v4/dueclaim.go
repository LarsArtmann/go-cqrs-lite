package sqliteengine

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"
)

// wireClaimkit attaches the ONE shared SQL claim/dedup runtime (ADR-0142):
// the embedded *claimkit.Claims and *claimkit.Dedup promote their methods,
// so the sqlite engine satisfies metaengine.DueClaimer,
// metaengine.FactSink, and metaengine.DedupStore without a single
// engine-specific claim statement. Semantics are pinned by
// adttest.AssertDueClaimer / AssertDedupStore (see dueclaim_test.go).
func (e *sqliteEngine) wireClaimkit() error {
	ctx := context.Background()

	claims, err := claimkit.New(ctx, e.db, claiming.DialectSQLite)
	if err != nil {
		return fmt.Errorf("metaengine: sqlite claimkit claims: %w", err)
	}

	//art-dupl:accept engine wiring twin — each dep-isolated engine module wires the ONE claimkit runtime (AGENTS contract 19); adttest pins semantics
	dedup, err := claimkit.NewDedup(ctx, e.db, claiming.DialectSQLite)
	if err != nil {
		return fmt.Errorf("metaengine: sqlite claimkit dedup: %w", err)
	}

	e.Claims = claims
	e.Dedup = dedup

	return nil
}
