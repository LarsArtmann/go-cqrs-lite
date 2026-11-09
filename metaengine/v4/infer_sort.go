package metaengine

import "reflect"

// sortableFieldNames lists result-type field names that conventionally imply
// temporal ordering. When a collection/paginated query's result type contains
// one of these fields and no explicit sort has been declared, autoInferSort
// generates a SortOnField declaration (descending for timestamps — newest
// first, the most common access pattern for collection queries).
var sortableFieldNames = []string{
	"CreatedAt",
	"CreatedAtTs",
	"UpdatedTs",
	"UpdatedAt",
	"Timestamp",
	"Date",
}

// autoInferSort detects temporal sort fields on the result type and generates
// a SortOnField declaration when no explicit sort accessor has been set.
// Only fires for collection result types (struct with Items []T) — scalar
// point-lookup results never need sorting.
//
// The first matching field from sortableFieldNames wins. Sort is descending
// (newest first) to match the dominant access pattern for time-ordered
// collections.
func autoInferSort(_, resultType reflect.Type, cfg QueryConfig) QueryConfig {
	if cfg.sortAccessor.spec != nil || cfg.sortAccessor.closure != nil {
		return cfg
	}

	elemType := collectionElementType(resultType)
	if elemType == resultType {
		return cfg
	}

	resultFields := buildFieldIndex(elemType)

	for _, name := range sortableFieldNames {
		if _, ok := resultFields[name]; ok {
			cfg.sortAccessor = sortAccessor{
				spec: &SortSpec{Column: name, Desc: true},
			}

			break
		}
	}

	return cfg
}
