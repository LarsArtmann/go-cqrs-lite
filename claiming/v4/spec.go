package claiming

// Spec names one claimable row set: the table, columns, and ordering a
// claim runs against, dialect-agnostic. Field values are SQL identifiers
// chosen by the CONSUMING STORE's author — compile-time constants, not
// user input — so the builders interpolate them into statements rather
// than binding them (identifiers cannot be bound in SQL). Never populate a
// Spec from runtime input.
type Spec struct {
	// Table is the claimable table name ("timers", "tasks").
	Table string

	// IDColumn uniquely identifies one claimable row.
	IDColumn string

	// DueColumn holds the moment a row becomes claimable ("fire_at",
	// "next_visible_at"). A row is due when DueColumn <= now.
	DueColumn string

	// LeaseColumn holds the claim fence. NULL (or <= now) means unclaimed
	// or the previous claim expired; a fresh future value fences the row
	// against every other claimer.
	LeaseColumn string

	// Returning lists the columns a claim hands back, unqualified and
	// ID-first by convention. The Postgres builder table-qualifies them
	// ("t.id, …"); SQLite and MySQL emit them verbatim.
	Returning []string

	// OrderBy optionally orders the claim ("fire_at ASC"). Honored by the
	// Postgres and MySQL statements. The SQLite builder ignores it:
	// SQLite UPDATE..RETURNING cannot order without the non-default
	// SQLITE_ENABLE_UPDATE_DELETE_LIMIT compile option, and a claim is a
	// SET of fenced rows, not a queue position — single-writer
	// serialization already guarantees exclusivity, only pick order is
	// advisory. Stores that need ordered claims on SQLite must rank
	// client-side after the claim or extend the builder deliberately.
	OrderBy string

	// OwnerColumn optionally names a column stamped with the claiming
	// worker's identity and re-checked on owner-fenced renewal
	// ([RenewOwnedStmt]). Zero value (empty) keeps the legacy owner-less
	// shape used by scheduling/sqlstore. Only the ClaimStmt family honors
	// it.
	OwnerColumn string

	// FilterColumn optionally names a column that ClaimStmt-family
	// statements constrain to ClaimParams.Filter (an equality predicate,
	// e.g. a collection column). Zero value (empty) = no filter — the
	// legacy whole-table shape.
	FilterColumn string
}

// orderExpr picks the claim order: the named [Spec.OrderBy] or the due
// column ascending by default.
func orderExpr(s Spec) string {
	if s.OrderBy != "" {
		return s.OrderBy
	}

	return s.DueColumn + " ASC"
}
