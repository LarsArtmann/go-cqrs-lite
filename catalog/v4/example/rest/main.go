// Package main demonstrates registering a REST-style catalog with explicit
// HTTP operations, typed responses, and security schemes, then exporting it
// to OpenAPI.
//
// Run:
//
//	go run ./example/rest
package main

import (
	"fmt"
	"os"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/openapi"
)

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type userDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type listUsersQuery struct {
	Limit int `json:"limit,omitempty"`
}

func main() {
	builder := catalog.NewBuilder("User API", "1.0.0")

	builder.AddService(
		"user-svc", "User Service", "1.0.0", "Manages user accounts",
		catalog.Command[createUserRequest](
			"user.create",
			catalog.MsgOperation("POST", "/api/users", "201", "400"),
			catalog.Response[userDTO]("201", "User created"),
			catalog.MsgSecurity("bearerAuth"),
		),
		catalog.Query[listUsersQuery](
			"user.list",
			catalog.MsgOperation("GET", "/api/users"),
			catalog.Response[userDTO]("200", "User details"),
		),
		catalog.Command[struct{}](
			"user.delete",
			catalog.MsgOperation("DELETE", "/api/users/{id}"),
			catalog.MsgSecurity("bearerAuth"),
		),
	)

	cat := builder.Build()

	if violations := cat.Validate(); len(violations) > 0 {
		fmt.Fprintf(os.Stderr, "catalog validation violations: %v\n", violations)
		os.Exit(1)
	}

	doc := openapi.NewExporter(
		"User API", "1.0.0",
		openapi.WithDescription("Example REST catalog with typed operations"),
	).Export(cat)

	data, err := doc.MarshalJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}
