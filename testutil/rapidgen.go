package testutil

import (
	"pgregory.net/rapid"
)

// EventType generates random event type strings matching the CQRS naming convention:
// starts with a letter, followed by letters, digits, dots, underscores, or hyphens.
func EventType() *rapid.Generator[string] {
	return rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]{2,63}$`)
}

// AggregateType generates random aggregate type strings matching the same convention.
func AggregateType() *rapid.Generator[string] {
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
