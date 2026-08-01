package metaengine

// ── Enum validation ──
// Each enum family gets a Valid() method and a package-level registry slice.
// The planner calls Valid() at Plan() time to catch typos early.

import "slices"

// AllADTs returns every registered ADT value.
func AllADTs() []ADT {
	return []ADT{
		ADTMap, ADTSet, ADTCounter, ADTGraph, ADTLog,
		ADTSortedMap, ADTMultimap, ADTVector, ADTSearch, ADTSpatial,
	}
}

func (a ADT) Valid() bool {
	return slices.Contains(AllADTs(), a)
}

// AllReadPatterns returns every registered ReadPattern value.
func AllReadPatterns() []ReadPattern {
	return []ReadPattern{
		ReadPointLookup, ReadMembership, ReadFilteredScan, ReadAggregate,
		ReadTraversal, ReadScan, ReadMultiLookup, ReadLogTail,
		ReadVectorSearch, ReadFullTextSearch, ReadSpatialRange,
	}
}

func (r ReadPattern) Valid() bool {
	return slices.Contains(AllReadPatterns(), r)
}

// AllFoldKinds returns every registered FoldKind value.
func AllFoldKinds() []FoldKind {
	return []FoldKind{
		FoldInsert, FoldUpdate, FoldRemove, FoldCount, FoldEdge,
		FoldSet, FoldSkip, FoldMultiInsert, FoldAppend,
		FoldVector, FoldSearch, FoldSpatial,
	}
}

func (k FoldKind) Valid() bool {
	return slices.Contains(AllFoldKinds(), k)
}

// AllComplexities returns every registered Complexity value.
func AllComplexities() []Complexity {
	return []Complexity{
		ComplexityO1, ComplexityOLogN, ComplexityON,
		ComplexityONLogN, ComplexityODegree,
	}
}

func (c Complexity) Valid() bool {
	return slices.Contains(AllComplexities(), c)
}

// AllStorageLayouts returns every registered StorageLayout value.
func AllStorageLayouts() []StorageLayout {
	return []StorageLayout{
		LayoutRow, LayoutColumnar, LayoutLSM, LayoutKV,
	}
}

func (l StorageLayout) Valid() bool {
	return slices.Contains(AllStorageLayouts(), l)
}

// AllFilterOps returns every registered FilterOp value.
func AllFilterOps() []FilterOp {
	return []FilterOp{
		FilterEq, FilterNe, FilterLt, FilterLe, FilterGt, FilterGe, FilterIn,
	}
}

func (o FilterOp) Valid() bool {
	return slices.Contains(AllFilterOps(), o)
}
