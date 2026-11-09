package compat

// Class is the compatibility verdict for one schema change. The values are
// ordered from least to most severe so verdicts combine with [Class.Worst].
type Class int

const (
	// ClassUnchanged means both sides declare the same shape.
	ClassUnchanged Class = iota

	// ClassAdditive means every instance of the old shape is still valid
	// under the new one, so stored events keep decoding with zero-value
	// fills. The policy's "safe without a version bump" rows land here.
	ClassAdditive

	// ClassNeedsVersionBump means at least one old instance is no longer
	// valid under the new shape, so an upcast op must be declared and the
	// declaration's current version must move. The policy's "version-bump
	// triggers" rows land here.
	ClassNeedsVersionBump

	// ClassIncomparable means the two shapes cannot be related at all, for
	// example because a schema is missing or declares no type. Never treat
	// it as safe.
	ClassIncomparable
)

// String returns the stable wire name of the class.
func (c Class) String() string {
	switch c {
	case ClassUnchanged:
		return "unchanged"
	case ClassAdditive:
		return "additive"
	case ClassNeedsVersionBump:
		return "needs-version-bump"
	case ClassIncomparable:
		return "incomparable"
	default:
		return "unknown"
	}
}

// IsBreaking reports whether the change requires an upcast op. Both
// [ClassNeedsVersionBump] and [ClassIncomparable] do.
func (c Class) IsBreaking() bool {
	return c >= ClassNeedsVersionBump
}

// Worst returns the more severe of c and other.
func (c Class) Worst(other Class) Class {
	if other > c {
		return other
	}

	return c
}
