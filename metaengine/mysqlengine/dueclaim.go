package mysqlengine

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"
)

// wireClaimkit attaches the ONE shared SQL claim/dedup runtime (ADR-0142):
// the embedded *claimkit.Claims and *claimkit.Dedup promote their methods,
// so the MySQL engine satisfies metaengine.DueClaimer,
// metaengine.FactSink, and metaengine.DedupStore without a single
// engine-specific claim statement. The MySQL dialect claims with the
// two-statement SKIP LOCKED shape in one transaction. Semantics are pinned
// by adttest.AssertDueClaimer / AssertDedupStore (see dueclaim_test.go,
// live-gated on MYSQL_TEST_DSN).
func (e *mysqlEngine) wireClaimkit() error {
	ctx := context.Background()

	claims, err := claimkit.New(ctx, e.db, claiming.DialectMySQL)
	if err != nil {
		return fmt.Errorf("metaengine: mysql claimkit claims: %w", err)
	}

	dedup, err := claimkit.NewDedup(ctx, e.db, claiming.DialectMySQL)
	if err != nil {
		return fmt.Errorf("metaengine: mysql claimkit dedup: %w", err)
	}

	e.Claims = claims
	e.Dedup = dedup

	return nil
}
