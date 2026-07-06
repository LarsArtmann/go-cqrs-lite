package storage

import (
	"regexp"

	errorfamily "github.com/larsartmann/go-error-family"
)

var validListingTablePrefix = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func validateListingTablePrefix(prefix string) error {
	if !validListingTablePrefix.MatchString(prefix) {
		return errorfamily.NewRejection(
			"listing.invalid_table_prefix",
			"invalid table prefix: must match ^[a-z_][a-z0-9_]*$",
		)
	}

	return nil
}

// listingTable centralizes the listing aggregates table name derivation and
// prefix validation shared by AggregateProjection (write side) and
// SQLAggregateReader (read side). Keeping both in one constructor prevents the
// two stores from drifting on the table name or the prefix rule.
type listingTable struct {
	name string
}

func newListingTable(prefix string) (listingTable, error) {
	if err := validateListingTablePrefix(prefix); err != nil {
		return listingTable{}, errorfamily.Wrapf(
			err,
			errorfamily.Rejection,
			"storage.listing_table_prefix",
			"prefix=%v",
			prefix,
		)
	}

	return listingTable{name: prefix + "listing_aggregates"}, nil
}
