package claiming

import "strings"

// ClaimParams carries the runtime values of one ClaimStmt-family claim:
// everything a due-claim needs beyond the static [Spec]. Zero-value fields
// degrade to the legacy whole-table behavior (no owner stamping, no filter,
// unlimited).
type ClaimParams struct {
	// Now is the claim's reference time (due gating + lease-expiry
	// reclaim). Required.
	Now any

	// LeaseUntil is the new lease deadline stamped on claimed rows
	// (typically Now + lease). Required.
	LeaseUntil any

	// Owner, when non-nil, is stamped into Spec.OwnerColumn (which must be
	// set). nil = no owner stamping.
	Owner any

	// Filter, when non-nil, is the equality value for Spec.FilterColumn
	// (which must be set) — e.g. the collection name scoping one claim
	// table to many keyspaces. nil = no filter.
	Filter any

	// Limit caps claimed rows. <=0 = unlimited.
	Limit int
}

// ClaimStmt builds the full-capability claim statement for the dialect:
// due-gated, lease-fenced, optionally owner-stamped, collection-filtered,
// limited, and due-ordered — the single statement behind
// metaengine.DueClaimer (ADR-0142).
//
//   - Postgres: a CTE fences candidate rows FOR UPDATE SKIP LOCKED (LIMIT
//     inside the CTE), then UPDATE stamps lease (and owner) and RETURNING
//     hands back exactly the claimed rows.
//   - SQLite: UPDATE..RETURNING constrained to keys selected by an ordered,
//     limited IN-subquery — core SQL, no SQLITE_ENABLE_UPDATE_DELETE_LIMIT
//     needed; the single-writer model serializes claimers.
//   - MySQL: the two-statement shape — this returns the fencing SELECT (FOR
//     UPDATE SKIP LOCKED, ordered, limited); stamp the lease on exactly
//     those IDs with [StampLeaseMySQLStmt] inside the SAME transaction.
func ClaimStmt(d Dialect, s Spec, p ClaimParams) (string, []any) {
	switch d {
	case DialectPostgres:
		return postgresClaimStmt(s, p)
	case DialectSQLite:
		return sqliteClaimStmt(s, p)
	case DialectMySQL:
		return mySQLClaimSelectFull(s, p)
	default:
		return "", nil
	}
}

func claimPredicates(s Spec, p ClaimParams, arg func(v any) string) string {
	var b strings.Builder

	b.WriteString(s.DueColumn + " <= " + arg(p.Now))
	b.WriteString(" AND (" + s.LeaseColumn + " IS NULL OR " + s.LeaseColumn + " <= " + arg(p.Now) + ")")

	if s.FilterColumn != "" && p.Filter != nil {
		b.WriteString(" AND " + s.FilterColumn + " = " + arg(p.Filter))
	}

	return b.String()
}

func postgresClaimStmt(s Spec, p ClaimParams) (string, []any) {
	// Placeholder allocation order: $1 now, $2 lease_until, then owner,
	// filter, limit as present.
	args := []any{p.Now, p.LeaseUntil}

	n := 2

	ownerSet := ""
	if s.OwnerColumn != "" && p.Owner != nil {
		n++
		ownerSet = ", " + s.OwnerColumn + " = $" + itoa(n)
		args = append(args, p.Owner)
	}

	filterAnd := ""
	if s.FilterColumn != "" && p.Filter != nil {
		n++
		filterAnd = " AND " + s.FilterColumn + " = $" + itoa(n)
		args = append(args, p.Filter)
	}

	limit := ""
	if p.Limit > 0 {
		n++
		limit = " LIMIT $" + itoa(n)
		args = append(args, p.Limit)
	}

	due := "SELECT " + s.IDColumn + " FROM " + s.Table +
		"\nWHERE " + claimPredicates(s, p, func(any) string { return "$1" }) + filterAnd +
		"\nORDER BY " + orderExpr(s) + limit +
		"\nFOR UPDATE SKIP LOCKED"

	query := "WITH due AS (\n" + due + "\n)\n" +
		"UPDATE " + s.Table + " t SET " + s.LeaseColumn + " = $2" + ownerSet + " FROM due" +
		" WHERE t." + s.IDColumn + " = due." + s.IDColumn +
		"\nRETURNING " + qualified(s.Returning)

	return query, args
}

func sqliteClaimStmt(s Spec, p ClaimParams) (string, []any) {
	// Placeholder order: ?1 lease_until, ?2 owner?, ?3 filter?, ?4 now,
	// ?5.. filter/limit repeats. SQLite reuses ?N ordinals, so repeated
	// references are free.
	args := []any{p.LeaseUntil}

	set := s.LeaseColumn + " = ?1"

	n := 1

	if s.OwnerColumn != "" && p.Owner != nil {
		n++
		set += ", " + s.OwnerColumn + " = ?" + itoa(n)
		args = append(args, p.Owner)
	}

	n++

	nowPh := "?" + itoa(n)

	args = append(args, p.Now)

	filterEq := ""
	filterPh := ""

	if s.FilterColumn != "" && p.Filter != nil {
		n++
		filterPh = "?" + itoa(n)
		filterEq = " AND " + s.FilterColumn + " = " + filterPh
		args = append(args, p.Filter)
	}

	limit := ""

	if p.Limit > 0 {
		n++
		limit = " LIMIT ?" + itoa(n)
		args = append(args, p.Limit)
	}

	pred := claimPredicates(s, p, func(any) string { return nowPh }) + filterEq

	// The IN-subquery makes the claim ordered+limited WITHOUT
	// UPDATE..LIMIT (which stock SQLite lacks): pick the candidate keys
	// first, then stamp exactly those.
	query := "UPDATE " + s.Table + " SET " + set +
		"\nWHERE " + s.IDColumn + " IN (\n" +
		"  SELECT " + s.IDColumn + " FROM " + s.Table +
		"\n  WHERE " + pred + "\n  ORDER BY " + orderExpr(s) + limit + "\n)" +
		"\nAND " + pred +
		"\nRETURNING " + columns(s.Returning)

	return query, args
}

func mySQLClaimSelectFull(s Spec, p ClaimParams) (string, []any) {
	var args []any

	pred := claimPredicates(s, p, func(any) string {
		args = append(args, p.Now)

		return "?"
	})

	if s.FilterColumn != "" && p.Filter != nil {
		pred += " AND " + s.FilterColumn + " = ?"
		args = append(args, p.Filter)
	}

	limit := ""

	if p.Limit > 0 {
		limit = " LIMIT ?"
		args = append(args, p.Limit)
	}

	query := "SELECT " + columns(s.Returning) + " FROM " + s.Table +
		"\nWHERE " + pred +
		"\nORDER BY " + orderExpr(s) + limit +
		"\nFOR UPDATE SKIP LOCKED"

	return query, args
}

// StampLeaseMySQLStmt builds the lease-stamping UPDATE of the MySQL
// two-statement claim for ClaimStmt-selected keys: it stamps lease (and
// owner when configured) on exactly the listed IDs inside the caller's
// transaction. The row locks from the SELECT keep the claim atomic until
// commit.
func StampLeaseMySQLStmt(s Spec, ids []any, p ClaimParams) (string, []any) {
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+4)

	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	set := s.LeaseColumn + " = ?"
	args = append(args, p.LeaseUntil)

	if s.OwnerColumn != "" && p.Owner != nil {
		set += ", " + s.OwnerColumn + " = ?"
		args = append(args, p.Owner)
	}

	query := "UPDATE " + s.Table + " SET " + set +
		" WHERE " + s.IDColumn + " IN (" + strings.Join(placeholders, ", ") + ")"

	return query, args
}

// RenewOwnedStmt builds the owner-fenced lease extension: the new deadline
// is granted only while the lease is live AND the row is still owned by the
// caller. Zero rows affected means [ErrLeaseNotHeld] for the caller.
func RenewOwnedStmt(d Dialect, s Spec, newUntil, id, owner, now any) (string, []any) {
	if s.OwnerColumn == "" {
		// No owner tracking configured: fall back to the fence-only form.
		return RenewStmt(d, s, newUntil, id, now)
	}

	switch d {
	case DialectPostgres:
		return "UPDATE " + s.Table + " SET " + s.LeaseColumn +
			" = $1 WHERE " + s.IDColumn + " = $2 AND " + s.OwnerColumn +
			" = $3 AND " + s.LeaseColumn + " > $4", []any{newUntil, id, owner, now}
	case DialectMySQL:
		return "UPDATE " + s.Table + " SET " + s.LeaseColumn +
			" = ? WHERE " + s.IDColumn + " = ? AND " + s.OwnerColumn +
			" = ? AND " + s.LeaseColumn + " > ?", []any{newUntil, id, owner, now}
	case DialectSQLite:
		return "UPDATE " + s.Table + " SET " + s.LeaseColumn +
			" = ?1 WHERE " + s.IDColumn + " = ?2 AND " + s.OwnerColumn +
			" = ?3 AND " + s.LeaseColumn + " > ?4", []any{newUntil, id, owner, now}
	default:
		return "UPDATE " + s.Table + " SET " + s.LeaseColumn +
			" = ?1 WHERE " + s.IDColumn + " = ?2 AND " + s.OwnerColumn +
			" = ?3 AND " + s.LeaseColumn + " > ?4", []any{newUntil, id, owner, now}
	}
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}

	return itoa(n/10) + string(rune('0'+n%10))
}
