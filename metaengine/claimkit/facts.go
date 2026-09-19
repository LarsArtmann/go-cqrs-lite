package claimkit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ClaimDueFacts implements [metaengine.FactSink.ClaimDueFacts]: the claim and
// its per-item facts commit in ONE transaction — a claimed item without its
// fact did not happen (queue contract invariant #1, ADR-0142).
func (c *Claims) ClaimDueFacts(
	ctx context.Context,
	req metaengine.ClaimDueRequest,
	factFor func(metaengine.DueClaim) []metaengine.ClaimFact,
) ([]metaengine.DueClaim, error) {
	defer c.lockWriter()()

	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}

	lease := req.Lease
	if lease <= 0 {
		lease = metaengine.DefaultClaimLease
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("claimkit.ClaimDueFacts: begin: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	claims, err := c.claimDueTx(ctx, tx, req, now, lease)
	if err != nil {
		return nil, err
	}

	for _, cl := range claims {
		if factFor == nil {
			break
		}

		for _, fact := range factFor(cl) {
			if err := insertFact(ctx, c.dialect, tx, req.Collection, cl.Key, fact); err != nil {
				return nil, fmt.Errorf("claimkit.ClaimDueFacts: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("claimkit.ClaimDueFacts: commit: %w", err)
	}

	return claims, nil
}

// ClaimDeleteFacts implements [metaengine.FactSink.ClaimDeleteFacts]: the
// epoch-guarded delete and its facts commit in ONE transaction. Returns
// whether the delete matched (false = wrong epoch or absent; NOT an error).
func (c *Claims) ClaimDeleteFacts(
	ctx context.Context,
	collection, key string,
	dueAt time.Time,
	facts ...metaengine.ClaimFact,
) (bool, error) {
	defer c.lockWriter()()

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("claimkit.ClaimDeleteFacts: begin: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		"DELETE FROM meta_due_claims WHERE collection = "+ph(c.dialect, 1)+
			" AND "+idColumn(c.dialect)+" = "+ph(c.dialect, 2)+
			" AND due_at = "+ph(c.dialect, 3),
		collection, key, c.encodeTime(dueAt))
	if err != nil {
		return false, fmt.Errorf("claimkit.ClaimDeleteFacts: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("claimkit.ClaimDeleteFacts: rows affected: %w", err)
	}

	if affected == 0 {
		return false, nil // wrong epoch or absent: nothing happened, no facts
	}

	for _, fact := range facts {
		if err := insertFact(ctx, c.dialect, tx, collection, key, fact); err != nil {
			return false, fmt.Errorf("claimkit.ClaimDeleteFacts: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("claimkit.ClaimDeleteFacts: commit: %w", err)
	}

	return true, nil
}

// claimDueTx runs the dialect-dispatched claim inside the caller's
// transaction (facts ride the same tx).
func (c *Claims) claimDueTx(
	ctx context.Context,
	tx *sql.Tx,
	req metaengine.ClaimDueRequest,
	now time.Time,
	lease time.Duration,
) ([]metaengine.DueClaim, error) {
	p := claiming.ClaimParams{
		Now:        c.encodeTime(now),
		LeaseUntil: c.encodeTime(now.Add(lease)),
		Owner:      req.Owner,
		Filter:     req.Collection,
		Limit:      req.Limit,
	}

	if c.dialect == claiming.DialectMySQL {
		selQuery, selArgs := claiming.ClaimStmt(c.dialect, claimsSpec(c.dialect), p)

		rows, err := tx.QueryContext(ctx, selQuery, selArgs...)
		if err != nil {
			return nil, fmt.Errorf("claimkit.ClaimDueFacts: select: %w", err)
		}

		claims, ids, err := scanClaimsWithIDs(rows)
		if err != nil {
			return nil, fmt.Errorf("claimkit.ClaimDueFacts: %w", err)
		}

		if len(ids) > 0 {
			stampQuery, stampArgs := claiming.StampLeaseMySQLStmt(claimsSpec(c.dialect), ids, p)

			if _, err := tx.ExecContext(ctx, stampQuery, stampArgs...); err != nil {
				return nil, fmt.Errorf("claimkit.ClaimDueFacts: stamp: %w", err)
			}
		}

		return claims, nil
	}

	query, args := claiming.ClaimStmt(c.dialect, claimsSpec(c.dialect), p)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("claimkit.ClaimDueFacts: claim: %w", err)
	}

	claims, _, err := scanClaimsWithIDs(rows)
	if err != nil {
		return nil, fmt.Errorf("claimkit.ClaimDueFacts: %w", err)
	}

	return claims, nil
}

func insertFact(
	ctx context.Context,
	dialect claiming.Dialect,
	tx *sql.Tx,
	collection, key string,
	fact metaengine.ClaimFact,
) error {
	query := "INSERT INTO meta_claim_facts (collection, " + idColumn(dialect) + ", type, payload) VALUES (" +
		ph(dialect, 1) + ", " + ph(dialect, 2) + ", " + ph(dialect, 3) + ", " + ph(dialect, 4) + ")"

	if _, err := tx.ExecContext(ctx, query, collection, key, fact.Type, fact.Payload); err != nil {
		return fmt.Errorf("insert fact %q: %w", fact.Type, err)
	}

	return nil
}
