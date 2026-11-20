package metadata

// BrandedString returns the string form of a branded ID, or "" if it is zero.
// It prevents zero-value ULIDs from leaking as "0000..." into Record metadata.
// This is the single shared implementation used by the event, command, and
// query as-record converters.
func BrandedString[T interface {
	String() string
	IsZero() bool
}](v T) string {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	if v.IsZero() {
		return ""
	}

	return v.String()
}

// ActorString resolves a Tracing's actor into the deprecated "kind:raw"
// WIRE form: the kind-discriminated ActorID when set, falling back to the
// bare UserID for records that predate ActorID.
//
// Deprecated: removed with the v4 train (T26). This is the wire-form view
// for the stringly ActorID twin; the canonical structural resolution is
// [RecordActor] (field wins, twin parsed at the edge, UserID fallback).
func ActorString(tracing Tracing) string {
	if !tracing.ActorID.IsZero() {
		return tracing.ActorID.PrefixedString()
	}

	return BrandedString(tracing.UserID)
}
