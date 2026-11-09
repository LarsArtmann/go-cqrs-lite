package metadata

// BrandedString returns the string form of a branded ID, or "" if it is zero.
// It prevents zero-value ULIDs from leaking as "0000..." into Record metadata.
// This is the single shared implementation used by the event, command, and
// query as-record converters.
func BrandedString[T interface {
	String() string
	IsZero() bool
}](v T) string {
	if v.IsZero() {
		return ""
	}

	return v.String()
}

// ActorString resolves a Tracing's actor: the kind-discriminated ActorID in
// its self-describing "kind:raw" form when set, falling back to the bare
// UserID for records that predate ActorID.
func ActorString(tracing Tracing) string {
	if !tracing.ActorID.IsZero() {
		return tracing.ActorID.PrefixedString()
	}

	return BrandedString(tracing.UserID)
}
