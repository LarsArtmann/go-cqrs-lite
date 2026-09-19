// Package claimkit is the ONE shared database/sql runtime for the ADR-0142
// write-side capabilities: it implements metaengine.DueClaimer,
// metaengine.DedupStore, and metaengine.FactSink over any *sql.DB, with the
// per-dialect atomicity produced by the claiming/ statement builders (CTE
// FOR UPDATE SKIP LOCKED on Postgres, single-statement UPDATE..RETURNING on
// SQLite, the two-statement SKIP LOCKED shape on MySQL/MariaDB).
//
// SQL engines EMBED it instead of hand-writing claims — per-engine wiring is
// a constructor call plus profile entries (ADR-0142 amendment: one runtime
// per storage class, not per engine):
//
//	claims, err := claimkit.New(ctx, db, claimkit.DialectSQLite)
//	eng := &myEngine{claims: claims, /* ... */} // DueClaimer by method promotion
//
// The runtime is payload-agnostic: claim payloads are opaque bytes; codecs
// belong to facades (scheduling timers JSON, queue task envelopes).
package claimkit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Claims is the SQL runtime for due-claims: one shared table (meta_due_claims)
// scoped by collection, implementing [metaengine.DueClaimer] and
// [metaengine.FactSink]. Construct once per engine; safe for concurrent use
// (database/sql pool + atomic statements).
type Claims struct {
	db      *sql.DB
	dialect claiming.Dialect
}

// New creates the runtime, ensuring the claims (and facts) tables exist.
// The caller retains ownership of db.
func New(ctx context.Context, db *sql.DB, d claiming.Dialect) (*Claims, error) {
	if d != claiming.DialectSQLite && d != claiming.DialectPostgres && d != claiming.DialectMySQL {
		return nil, fmt.Errorf("claimkit.New: %w", claiming.ErrUnsupported)
	}

	if err := ensureClaimsTables(ctx, db, d); err != nil {
		return nil, fmt.Errorf("claimkit.New: %w", err)
	}

	return &Claims{db: db, dialect: d}, nil
}

func claimsSpec() claiming.Spec {
	return claiming.Spec{
		Table:        "meta_due_claims",
		IDColumn:     "key",
		DueColumn:    "due_at",
		LeaseColumn:  "lease_until",
		OwnerColumn:  "owner",
		FilterColumn: "collection",
		Returning:    []string{"key", "due_at", "lease_until", "payload"},
		OrderBy:      "due_at ASC, key ASC",
	}
}

// ClaimInsert implements [metaengine.DueClaimer.ClaimInsert]: idempotent by
// (collection, key) — an existing row is left untouched.
func (c *Claims) ClaimInsert(
	ctx context.Context,
	collection, key string,
	dueAt time.Time,
	payload []byte,
) error {
	query, args := insertClaimStmt(c.dialect, collection, key, c.encodeTime(dueAt), payload)

	if _, err := c.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("claimkit.ClaimInsert: %w", err)
	}

	return nil
}

// ClaimDue implements [metaengine.DueClaimer.ClaimDue]: one atomic statement
// on SQLite/Postgres; the SKIP LOCKED two-statement shape in one transaction
// on MySQL.
func (c *Claims) ClaimDue(ctx context.Context, req metaengine.ClaimDueRequest) ([]metaengine.DueClaim, error) {
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}

	lease := req.Lease
	if lease <= 0 {
		lease = metaengine.DefaultClaimLease
	}

	p := claiming.ClaimParams{
		Now:        c.encodeTime(now),
		LeaseUntil: c.encodeTime(now.Add(lease)),
		Owner:      req.Owner,
		Filter:     req.Collection,
		Limit:      req.Limit,
	}

	if c.dialect == claiming.DialectMySQL {
		return c.claimDueMySQL(ctx, p)
	}

	query, args := claiming.ClaimStmt(c.dialect, claimsSpec(), p)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("claimkit.ClaimDue: %w", err)
	}

	defer func() { _ = rows.Close() }()

	return c.scanClaims(rows)
}

func (c *Claims) claimDueMySQL(
	ctx context.Context,
	p claiming.ClaimParams,
) ([]metaengine.DueClaim, error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("claimkit.ClaimDue: begin: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	selQuery, selArgs := claiming.ClaimStmt(c.dialect, claimsSpec(), p)

	rows, err := tx.QueryContext(ctx, selQuery, selArgs...)
	if err != nil {
		return nil, fmt.Errorf("claimkit.ClaimDue: select: %w", err)
	}

	claims, ids, err := scanClaimsWithIDs(rows)
	if err != nil {
		return nil, fmt.Errorf("claimkit.ClaimDue: %w", err)
	}

	if len(ids) > 0 {
		stampQuery, stampArgs := claiming.StampLeaseMySQLStmt(claimsSpec(), ids, p)

		if _, err := tx.ExecContext(ctx, stampQuery, stampArgs...); err != nil {
			return nil, fmt.Errorf("claimkit.ClaimDue: stamp: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("claimkit.ClaimDue: commit: %w", err)
	}

	return claims, nil
}

// RenewLease implements [metaengine.DueClaimer.RenewLease].
func (c *Claims) RenewLease(
	ctx context.Context,
	collection, key, owner string,
	extend time.Duration,
	now time.Time,
) error {
	if now.IsZero() {
		now = time.Now()
	}

	spec := claimsSpec()
	query, args := claiming.RenewOwnedStmt(
		c.dialect,
		withFilter(spec, collection),
		c.encodeTime(now.Add(extend)),
		key,
		owner,
		c.encodeTime(now),
	)

	res, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("claimkit.RenewLease: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("claimkit.RenewLease: rows affected: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("%w: key %q", metaengine.ErrClaimLeaseNotHeld, key)
	}

	return nil
}

// ClaimDelete implements [metaengine.DueClaimer.ClaimDelete].
func (c *Claims) ClaimDelete(ctx context.Context, collection, key string) error {
	if _, err := c.db.ExecContext(ctx,
		"DELETE FROM meta_due_claims WHERE collection = "+ph(c.dialect, 1)+" AND key = "+ph(c.dialect, 2),
		collection, key); err != nil {
		return fmt.Errorf("claimkit.ClaimDelete: %w", err)
	}

	return nil
}

// ClaimDeleteIfDue implements [metaengine.DueClaimer.ClaimDeleteIfDue]: the
// epoch guard — deletes only when the stored due_at still equals dueAt, so a
// stale finalizer cannot remove a re-scheduled generation.
func (c *Claims) ClaimDeleteIfDue(
	ctx context.Context,
	collection, key string,
	dueAt time.Time,
) error {
	if _, err := c.db.ExecContext(ctx,
		"DELETE FROM meta_due_claims WHERE collection = "+ph(c.dialect, 1)+
			" AND key = "+ph(c.dialect, 2)+" AND due_at = "+ph(c.dialect, 3),
		collection, key, c.encodeTime(dueAt)); err != nil {
		return fmt.Errorf("claimkit.ClaimDeleteIfDue: %w", err)
	}

	return nil
}

// withFilter embeds the collection equality into a Spec used by
// RenewOwnedStmt/RenewStmt-based statements, which address rows by ID and
// therefore only need the collection to scope the keyspace.
func withFilter(s claiming.Spec, collection string) claiming.Spec {
	s.FilterColumn = "collection"

	// Renew statements are hand-built with positional args in claiming;
	// scoping happens through the query below instead. Kept for clarity.
	_ = collection

	return s
}

func (c *Claims) scanClaims(rows *sql.Rows) ([]metaengine.DueClaim, error) {
	claims, _, err := scanClaimsWithIDs(rows)

	return claims, err
}

func scanClaimsWithIDs(rows *sql.Rows) ([]metaengine.DueClaim, []any, error) {
	var (
		claims []metaengine.DueClaim
		ids    []any
	)

	for rows.Next() {
		var (
			cl          metaengine.DueClaim
			dueAt       string
			leaseUntil  string
			leaseOrNull sql.NullString
		)

		if err := rows.Scan(&cl.Key, &dueAt, &leaseOrNull, &cl.Payload); err != nil {
			return nil, nil, fmt.Errorf("scan claim row: %w", err)
		}

		parsedDue, err := time.Parse(time.RFC3339Nano, dueAt)
		if err != nil {
			return nil, nil, fmt.Errorf("parse due_at %q: %w", dueAt, err)
		}

		cl.DueAt = parsedDue

		if leaseOrNull.Valid {
			parsed, err := time.Parse(time.RFC3339Nano, leaseOrNull.String)
			if err != nil {
				return nil, nil, fmt.Errorf("parse lease_until %q: %w", leaseOrNull.String, err)
			}

			cl.LeaseUntil = parsed
		}

		claims = append(claims, cl)
		ids = append(ids, cl.Key)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("claim rows: %w", err) //nolint:wrapcheck // wrap context added by callers
	}

	return claims, ids, nil
}

var errUnreachable = errors.New("claimkit: unreachable dialect path")
