package testutil

import (
	"time"

	"pgregory.net/rapid"
)

// EventType generates random event type strings matching the CQRS naming convention:
// starts with a letter, followed by letters, digits, dots, underscores, or hyphens.
func EventType() *rapid.Generator[string] {
	return rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]{2,63}$`)
}

// StreamType generates random stream type strings matching the same convention.
func StreamType() *rapid.Generator[string] {
	return rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]{2,63}$`)
}

// Version generates a rapid generator for positive event versions.
func Version() *rapid.Generator[int] {
	return rapid.IntRange(1, 10000) //nolint:mnd // test range
}

// NonEmptyString generates a non-empty string within a reasonable length.
func NonEmptyString() *rapid.Generator[string] {
	return rapid.StringN(1, 100, 200) //nolint:mnd // test range
}

// MetadataMap generates a rapid generator for event.Metadata maps with
// random string keys and values. Useful for property-based testing of
// serialization, signing, and encryption round-trips.
func MetadataMap() *rapid.Generator[map[string]string] {
	return rapid.MapOf(NonEmptyString(), NonEmptyString())
}

// Timestamp generates a rapid generator for timestamps within a reasonable
// range (year 2000 to year 2100).
func Timestamp() *rapid.Generator[time.Time] {
	return rapid.Map(
		rapid.Int64Range(
			946684800,  //nolint:mnd // 2000-01-01 UTC
			4102444800, //nolint:mnd // 2100-01-01 UTC
		),
		func(sec int64) time.Time { return time.Unix(sec, 0).UTC() },
	)
}
