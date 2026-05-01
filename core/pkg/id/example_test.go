package id_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type orderMarker struct{}

type OrderID = id.Of[orderMarker]

func ExampleOf() {
	orderID := id.New[OrderID]()
	fmt.Println(orderID.IsZero())

	parsed, err := id.Parse[OrderID](orderID.String())
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(orderID.Equal(parsed))

	// Output:
	// false
	// true
}
