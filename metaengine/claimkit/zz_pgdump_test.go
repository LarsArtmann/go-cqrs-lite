package claimkit_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
)

func TestDumpPGClaimStmt(t *testing.T) {
	spec := claiming.Spec{Table: "meta_due_claims", IDColumn: "key", DueColumn: "due_at",
		LeaseColumn: "lease_until", OwnerColumn: "owner", FilterColumn: "collection",
		Returning: []string{"key", "due_at", "lease_until", "payload"}, OrderBy: "due_at ASC, key ASC"}
	q, args := claiming.ClaimStmt(claiming.DialectPostgres, spec, claiming.ClaimParams{
		Now: "NOW", LeaseUntil: "UNTIL", Owner: "w", Filter: "COL", Limit: 10,
	})
	t.Logf("QUERY:\n%s\nARGS: %v", q, args)
}
