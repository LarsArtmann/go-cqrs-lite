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

func ExampleBuilder_withOperation() {
	type createUserCmd struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	type userResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	builder := catalog.NewBuilder("REST API", "1.0.0")
	builder.AddService(
		"user-svc", "User Service", "1.0.0", "Manages users",
		catalog.Command[createUserCmd](
			"user.create",
			catalog.MsgOperation("POST", "/api/users", "201", "400"),
			catalog.Response[userResponse]("201", "User created"),
		),
		catalog.Query[struct{}](
			"user.list",
			catalog.MsgOperation("GET", "/api/users"),
		),
	)

	cat := builder.Build()
	svc := cat.Services[0]

	fmt.Println(len(svc.Commands), len(svc.Queries))
	fmt.Println(svc.Commands[0].Operation.Method, svc.Commands[0].Operation.Path)

	// Output:
	// 1 1
	// POST /api/users
}

func ExampleMsgOperation() {
	builder := catalog.NewBuilder("API", "1.0.0")
	builder.AddService(
		"order-svc", "Order Service", "1.0.0", "Orders",
		catalog.Command[struct{}](
			"order.create",
			catalog.MsgOperation("POST", "/api/orders"),
			catalog.Response[struct{}]("201", "Created"),
		),
	)

	cat := builder.Build()
	violations := cat.Validate()

	fmt.Println(len(violations))

	// Output:
	// 0
}
