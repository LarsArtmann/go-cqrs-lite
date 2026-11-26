package metaengine

import (
	"fmt"
	"reflect"
)

// applyScanOptions builds the config of one Scan/ScanPage/aggregate call: the
// operator's [WithDefaultLimit] when one is pinned, else the built-in 100,
// then the caller's options. Every entry point that needs the resolved config
// goes through here, because a second hand-rolled application of the options
// diverged from the scan (a wrong page cursor under a non-default ceiling).
func (r *TypedReader[V]) applyScanOptions(opts []ScanOption) scanConfig {
	cfg := scanConfig{limit: 100}
	if dl := r.store.defaultLimit; dl > 0 {
		cfg.limit = dl
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// expandScanFilters rewrites the declarative range and IN specs into plain
// FilterSpecs so pushdown, raw and aggregate paths all see one filter list.
func expandScanFilters(cfg scanConfig) scanConfig {
	for _, rg := range cfg.ranges {
		cfg.filters = append(
			cfg.filters,
			FilterSpec{Column: rg.Column, Op: FilterGe, Value: rg.Low},
			FilterSpec{Column: rg.Column, Op: FilterLe, Value: rg.High},
		)
	}

	for _, in := range cfg.inSpecs {
		cfg.filters = append(cfg.filters, FilterSpec{
			Column: in.Column, Op: FilterIn, Value: in.Values,
		})
	}

	// Merged into filters — clearing keeps the closure fallback from
	// evaluating them twice.
	cfg.inSpecs = nil
	coerceInLists(cfg.filters)

	return cfg
}

// ValidateQueryIdents is the gate in front of every query surface that
// turns caller-supplied filters into statement text: the filter operator, the
// filter and sort columns and the extra columns (value / group-by / distinct)
// are spliced into the SQL text, never bound, so each is checked before a SQL
// builder sees it. A FilterIn value must be a []any list, the only form the
// engines render (the typed reader coerces typed slices into it).
//
// The SQL engines call it from their optional-reader interfaces too: those
// are public, so a consumer reaching an engine directly bypasses the typed
// reader's own validation.
func ValidateQueryIdents(filters []FilterSpec, sort *SortSpec, columns ...string) error {
	if err := ValidateFilterSpecs(filters); err != nil {
		return err
	}

	for _, f := range filters {
		if err := ValidateIdentifier(f.Column); err != nil {
			return fmt.Errorf("filter column: %w", err)
		}

		if err := validateInList(f); err != nil {
			return err
		}
	}

	names := make([]string, 0, len(columns)+1)
	if sort != nil {
		names = append(names, sort.Column)
	}

	for _, c := range append(names, columns...) {
		if c == "" {
			continue
		}

		if err := ValidateIdentifier(c); err != nil {
			return fmt.Errorf("column: %w", err)
		}
	}

	return nil
}

// validateScanFilters is the typed reader's use of ValidateQueryIdents; the
// sort column is checked separately so its error names it.
func validateScanFilters(filters []FilterSpec) error {
	return ValidateQueryIdents(filters, nil)
}

// aggregateFilters resolves the options of an aggregate-style call (Count,
// Sum, Distinct, Grouped*, Multi*) into the filter list the engine receives.
// The aggregate entry points bypass scanItemsConfigured, so without this the
// raw operator and column reached SQL text unchecked.
func (r *TypedReader[V]) aggregateFilters(opts ...ScanOption) ([]FilterSpec, error) {
	cfg := r.applyScanOptions(opts)
	cfg = expandScanFilters(cfg)
	if err := validateScanFilters(cfg.filters); err != nil {
		return nil, err
	}

	return cfg.filters, nil
}

// validateIdentifiers checks the aggregate argument columns (value column,
// group-by column) that engines interpolate into SQL.
func validateIdentifiers(what string, columns ...string) error {
	for _, c := range columns {
		if c == "" {
			continue
		}

		if err := ValidateIdentifier(c); err != nil {
			return fmt.Errorf("%s column: %w", what, err)
		}
	}

	return nil
}

// validateInList requires the value of a FilterIn spec to be a []any list
// (or nil). No engine renders any other form: the typed reader coerces typed
// slices into []any, so anything else reaching here was passed around it.
func validateInList(f FilterSpec) error {
	if f.Op != FilterIn || f.Value == nil {
		return nil
	}

	if _, ok := f.Value.([]any); ok {
		return nil
	}

	return fmt.Errorf(
		"metaengine: FilterIn on column %q needs a []any list of values, got %T",
		f.Column, f.Value,
	)
}

// coerceInLists rewrites FilterIn values held in a typed slice ([]string,
// []int, ...) into []any in place. The engines and evalFilterOp only accept
// []any; any other slice type made the SQL builders drop the clause.
func coerceInLists(filters []FilterSpec) {
	for i, f := range filters {
		if f.Op != FilterIn {
			continue
		}

		filters[i].Value = inListAsAny(f.Value)
	}
}

func inListAsAny(v any) any {
	if list, ok := v.([]any); ok {
		return list
	}

	if b, ok := v.([]byte); ok {
		return b
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return v
	}

	list := make([]any, rv.Len())
	for i := range list {
		list[i] = rv.Index(i).Interface()
	}

	return list
}

// validateMultiAggregate checks the value columns of a multi-aggregate spec
// list (and the group-by column) before they reach an engine.
func validateMultiAggregate(specs []AggregateSpec, groupBy string) error {
	for _, s := range specs {
		if err := validateIdentifiers("aggregate", s.Column); err != nil {
			return err
		}
	}

	return validateIdentifiers("group by", groupBy)
}
