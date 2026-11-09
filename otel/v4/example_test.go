package otel_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/otel/v4"
)

type stringer string

func (s stringer) String() string { return string(s) }

func ExampleStreamAttrs() {
	attrs := otel.StreamAttrs(stringer("User"), stringer("01HXYZ"))

	fmt.Println(len(attrs) > 0)

	// Output:
	// true
}

func ExampleEventAttrs() {
	attrs := otel.EventAttrs("user.created", stringer("01HXYZ"), "User")

	fmt.Println(len(attrs) > 0)

	// Output:
	// true
}

func ExampleCommandAttrs() {
	attrs := otel.CommandAttrs("user.create", stringer("01HXYZ"))

	fmt.Println(len(attrs) > 0)

	// Output:
	// true
}
