//go:build cgo

package duckdbengine

import (
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// validateQueryIdents is the gate in front of every entry point that splices
// caller-supplied text into statement text: the filter operator (never a bind
// parameter), the filter and sort columns, and the value / group-by columns.
// The typed reader validates in front of the engine, but the engine's
// optional-reader interfaces are public too, so they must not trust input.
//
// art-dupl:accept cross-engine SQL-input gate — separate go.mod per engine
func validateQueryIdents(
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	columns ...string,
) error {
	if err := metaengine.ValidateFilterSpecs(filters); err != nil {
		return err
	}

	for _, f := range filters {
		if err := metaengine.ValidateIdentifier(f.Column); err != nil {
			return fmt.Errorf("filter column: %w", err)
		}
	}

	if sort != nil {
		columns = append(columns, sort.Column)
	}

	for _, c := range columns {
		if c == "" {
			continue
		}

		if err := metaengine.ValidateIdentifier(c); err != nil {
			return fmt.Errorf("column: %w", err)
		}
	}

	return nil
}

// specColumns lists the value columns of a multi-aggregate spec list so they
// can be checked by validateQueryIdents.
//
// art-dupl:accept cross-engine helper — separate go.mod per engine
func specColumns(specs []metaengine.AggregateSpec) []string {
	columns := make([]string, 0, len(specs))

	for _, s := range specs {
		columns = append(columns, s.Column)
	}

	return columns
}

// rejectedExplain is what an explain surface returns instead of SQL when its
// input fails validateQueryIdents; explains have no error return.
const rejectedExplain = "/* rejected: invalid filter or column */"
