package otel_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/otel/v3"
)

type stringer string

func (s stringer) String() string { return string(s) }

func ExampleAggregateAttrs() {
	attrs := otel.AggregateAttrs(stringer("User"), stringer("01HXYZ"))

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
