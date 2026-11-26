//go:build cgo

package duckdbengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// specColumns lists the value columns of a multi-aggregate spec list so they
// can be checked by [metaengine.ValidateQueryIdents].
//
// art-dupl:accept cross-engine helper — separate go.mod per engine
func specColumns(specs []metaengine.AggregateSpec) []string {
	columns := make([]string, 0, len(specs))

	for _, s := range specs {
		columns = append(columns, s.Column)
	}

	return columns
}

// emptyInPredicate is the false predicate rendered for a FilterIn whose list
// is empty or not a list (ruling S041-02: such a filter matches no rows).
const emptyInPredicate = "1 = 0"

// rejectedExplain is what an explain surface returns instead of SQL when its
// input fails [metaengine.ValidateQueryIdents]; explains have no error return.
const rejectedExplain = "/* rejected: invalid filter or column */"
