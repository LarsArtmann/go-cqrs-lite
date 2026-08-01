package metaengine

// ── Enum validation ──
// Each enum family gets a Valid() method and a package-level registry slice.
// The planner calls Valid() at Plan() time to catch typos early.

// AllADTs returns every registered ADT value.
func AllADTs() []ADT {
	return []ADT{
		ADTMap, ADTSet, ADTCounter, ADTGraph, ADTLog,
		ADTSortedMap, ADTMultimap, ADTVector, ADTSearch, ADTSpatial,
	}
}

func (a ADT) Valid() bool {
	for _, v := range AllADTs() {
		if a == v {
			return true
		}
	}

	return false
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
	for _, v := range AllReadPatterns() {
		if r == v {
			return true
		}
	}

	return false
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
	for _, v := range AllFoldKinds() {
		if k == v {
			return true
		}
	}

	return false
}

// AllComplexities returns every registered Complexity value.
func AllComplexities() []Complexity {
	return []Complexity{
		ComplexityO1, ComplexityOLogN, ComplexityON,
		ComplexityONLogN, ComplexityODegree,
	}
}

func (c Complexity) Valid() bool {
	for _, v := range AllComplexities() {
		if c == v {
			return true
		}
	}

	return false
}

// AllStorageLayouts returns every registered StorageLayout value.
func AllStorageLayouts() []StorageLayout {
	return []StorageLayout{
		LayoutRow, LayoutColumnar, LayoutLSM, LayoutKV,
	}
}

func (l StorageLayout) Valid() bool {
	for _, v := range AllStorageLayouts() {
		if l == v {
			return true
		}
	}

	return false
}

// AllFilterOps returns every registered FilterOp value.
func AllFilterOps() []FilterOp {
	return []FilterOp{
		FilterEq, FilterNe, FilterLt, FilterLe, FilterGt, FilterGe, FilterIn,
	}
}

func (o FilterOp) Valid() bool {
	for _, v := range AllFilterOps() {
		if o == v {
			return true
		}
	}

	return false
}
