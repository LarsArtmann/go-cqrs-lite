package claiming

import "strings"

// PostgresClaimStmt builds the single-statement Postgres claim. A CTE
// fences every due row FOR UPDATE SKIP LOCKED — concurrent claimers skip
// each other's locked rows, so each row is claimed by exactly one worker —
// then the UPDATE stamps the lease and RETURNING hands back exactly the
// rows THIS claimer took. The lease predicate re-opens rows whose previous
// claim expired (crashed worker).
//
// now and leaseUntil are the dialect-formatted claim bounds (any values
// the caller's time encoding produces); they bind as $1 and $2.
func PostgresClaimStmt(s Spec, now, leaseUntil any) (string, []any) {
	due := "SELECT " + s.IDColumn + " FROM " + s.Table +
		"\nWHERE " + s.DueColumn + " <= $1 AND (" + s.LeaseColumn +
		" IS NULL OR " + s.LeaseColumn + " <= $1)" + andSuffix(
		s,
	) +
		"\nORDER BY " + orderExpr(
		s,
	) +
		"\nFOR UPDATE SKIP LOCKED"

	query := "WITH due AS (\n" + due + "\n)\n" +
		"UPDATE " + s.Table + " t SET " + s.LeaseColumn + " = $2 FROM due" +
		" WHERE t." + s.IDColumn + " = due." + s.IDColumn +
		"\nRETURNING " + qualified(s.Returning)

	return query, []any{now, leaseUntil}
}

// SQLiteClaimStmt builds the single-statement SQLite claim. SQLite has no
// SKIP LOCKED, but its single-writer model serializes claim transactions,
// which is equivalent for the no-double-processing guarantee: a plain
// UPDATE..RETURNING inside the caller's transaction is already atomic
// across claimers. [Spec.OrderBy] is ignored here (see its doc).
//
// now and leaseUntil bind as ?2 and ?1 respectively, matching the
// statement's placeholder numbering.
func SQLiteClaimStmt(s Spec, now, leaseUntil any) (string, []any) {
	query := "UPDATE " + s.Table + " SET " + s.LeaseColumn + " = ?1" +
		"\nWHERE " + s.DueColumn + " <= ?2 AND (" + s.LeaseColumn +
		" IS NULL OR " + s.LeaseColumn + " <= ?2)" + andSuffix(
		s,
	) +
		"\nRETURNING " + columns(
		s.Returning,
	)

	return query, []any{leaseUntil, now}
}

// MySQLClaimSelect builds the fencing SELECT of the two-statement
// MySQL/MariaDB claim: FOR UPDATE SKIP LOCKED locks every due row,
// excluding rows already locked by concurrent claimers. The caller scans
// the rows, then stamps the lease on exactly those IDs with
// [StampLeaseMySQL] inside the SAME transaction — the row locks keep the
// claim atomic until commit.
//
// SKIP LOCKED requires MySQL 8.0+ or MariaDB 10.6+; older servers fail
// this query loudly at the first claim — never silently.
func MySQLClaimSelect(s Spec, now any) (string, []any) {
	query := "SELECT " + columns(
		s.Returning,
	) + " FROM " + s.Table +
		"\nWHERE " + s.DueColumn + " <= ? AND (" + s.LeaseColumn +
		" IS NULL OR " + s.LeaseColumn + " <= ?)" + andSuffix(
		s,
	) +
		"\nORDER BY " + orderExpr(
		s,
	) +
		"\nFOR UPDATE SKIP LOCKED"

	return query, []any{now, now}
}

// RenewStmt builds the lease-extension UPDATE: the new deadline is granted
// only while the lease is still live ([LeaseColumn] > now). An expired
// claim cannot be resurrected — another worker may already be processing —
// so renewal after expiry affects zero rows and the caller reports
// [ErrLeaseNotHeld].
func RenewStmt(d Dialect, s Spec, newUntil, id, now any) (string, []any) {
	var query string

	switch d {
	case DialectPostgres:
		query = "UPDATE " + s.Table + " SET " + s.LeaseColumn +
			" = $1 WHERE " + s.IDColumn + " = $2 AND " + s.LeaseColumn + " > $3"
	case DialectMySQL:
		// MySQL has no ordinal ?N placeholders — plain ? only.
		query = "UPDATE " + s.Table + " SET " + s.LeaseColumn +
			" = ? WHERE " + s.IDColumn + " = ? AND " + s.LeaseColumn + " > ?"
	case DialectSQLite:
		query = "UPDATE " + s.Table + " SET " + s.LeaseColumn +
			" = ?1 WHERE " + s.IDColumn + " = ?2 AND " + s.LeaseColumn + " > ?3"
	default:
		// Unknown dialects degrade to the SQLite-compatible ordinal form —
		// renewal stays best-effort and the values still bind safely.
		query = "UPDATE " + s.Table + " SET " + s.LeaseColumn +
			" = ?1 WHERE " + s.IDColumn + " = ?2 AND " + s.LeaseColumn + " > ?3"
	}

	return query, []any{newUntil, id, now}
}

// columns joins unqualified column names for SELECT/RETURNING lists.
func columns(cols []string) string {
	return strings.Join(cols, ", ")
}

// qualified joins table-qualified column names ("t.id, t.fire_at") for the
// Postgres UPDATE..RETURNING list, which resolves against the aliased
// target table.
func qualified(cols []string) string {
	prefixed := make([]string, len(cols))
	for i, col := range cols {
		prefixed[i] = "t." + col
	}

	return strings.Join(prefixed, ", ")
}
