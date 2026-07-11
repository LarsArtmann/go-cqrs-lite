package catalog_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func ExampleRegistry() {
	r := catalog.NewRegistry("My Platform", "1.0.0")

	r.AddCommand("user-service", catalog.Message{
		Name:    "CreateUser",
		Summary: "Create a new user account",
	})

	r.AddEvent("user-service", catalog.Message{
		Name:    "UserCreated",
		Summary: "A new user was created",
	})

	c := r.Build()

	fmt.Println(c.Title)
	fmt.Println(len(c.Services))

	// Output:
	// My Platform
	// 1
}
