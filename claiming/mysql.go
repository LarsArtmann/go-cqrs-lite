package claiming

import (
	"context"
	"database/sql"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// StampLeaseMySQL stamps the lease on the claimed rows inside the claim
// transaction — the second half of the MySQL claim after
// [MySQLClaimSelect]. Only claimed rows are updated: SKIP LOCKED excluded
// rows already locked by others, and our own locks keep everyone else out
// until commit, so exactly the claimed rows are stamped.
//
// A nil/empty ids slice is a no-op (a poll that found nothing due stamps
// nothing).
func StampLeaseMySQL(ctx context.Context, tx *sql.Tx, s Spec, leaseUntil any, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	args := make([]any, 0, len(ids)+1)
	args = append(args, leaseUntil)

	for _, id := range ids {
		args = append(args, id)
	}

	// Concatenation builds ONLY "?" placeholders; ids are bound args.
	query := "UPDATE " + s.Table + " SET " + s.LeaseColumn + " = ? WHERE " + s.IDColumn + //nolint:gosec // placeholders only, ids bound
		" IN (" + strings.TrimSuffix(
		strings.Repeat("?,", len(ids)),
		",",
	) + ")"

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errorfamily.WrapInfrastructure(
			err, "claiming.stamp_lease", "stamp claimed lease")
	}

	return nil
}
