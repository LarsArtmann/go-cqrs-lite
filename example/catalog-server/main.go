package main

import (
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/docserver"
)

func main() {
	provider := func() *catalog.Catalog {
		return buildCatalog()
	}

	srv := docserver.NewDocsServer(provider, docserver.Config{
		ServiceName: "User Service",
		Version:     "1.0.0",
		Description: "Example CQRS user service with auto-generated docs",
	})

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	fmt.Println("=== go-cqrs-lite: Catalog Docs Server ===")
	fmt.Println()
	fmt.Println("  http://localhost:8080/             — OpenAPI UI (Scalar)")
	fmt.Println("  http://localhost:8080/api/spec     — OpenAPI JSON")
	fmt.Println("  http://localhost:8080/api/spec.yaml — OpenAPI YAML")
	fmt.Println("  http://localhost:8080/asyncapi/     — AsyncAPI UI")
	fmt.Println("  http://localhost:8080/asyncapi/spec — AsyncAPI JSON")
	fmt.Println("  http://localhost:8080/catalog       — Full Catalog JSON")
	fmt.Println()

	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Printf("server error: %v\n", err)
	}
}

type CreateUserPayload struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func addEvent(
	reg *catalog.Registry,
	msgID catalog.MessageID,
	name, summary string,
	direction catalog.Direction,
	props ...string,
) {
	schema := &catalog.Schema{Type: catalog.TypeObject, Properties: map[string]catalog.Property{}}
	for _, p := range props {
		schema.Properties[p] = catalog.Property{Type: catalog.TypeString}
	}

	reg.AddEvent("user-service", catalog.Message{
		Kind: catalog.EventMessage, ID: msgID, Name: name,
		Version: "1.0.0", Summary: summary, Direction: direction,
		Schema: schema,
	})
}

func buildCatalog() *catalog.Catalog {
	reg := catalog.NewRegistry("User Service", "1.0.0")

	reg.AddService(catalog.Service{
		ID:      "user-service",
		Name:    "User Service",
		Version: "1.0.0",
		Summary: "Manages user accounts",
	})

	reg.AddCommand("user-service", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateUser", Name: "Create User",
		Version: "1.0.0", Summary: "Create a new user account", Direction: catalog.Receives,
		Schema: &catalog.Schema{Type: catalog.TypeObject, Properties: map[string]catalog.Property{
			"email": {Type: catalog.TypeString}, "name": {Type: catalog.TypeString},
		}},
	})

	reg.AddCommand("user-service", catalog.Message{
		Kind: catalog.CommandMessage, ID: "ChangeUserName", Name: "Change User Name",
		Version: "1.0.0", Summary: "Change a user's display name", Direction: catalog.Receives,
		Schema: &catalog.Schema{Type: catalog.TypeObject, Properties: map[string]catalog.Property{
			"name": {Type: catalog.TypeString},
		}},
	})

	addEvent(reg, "UserCreated", "User Created",
		"A new user account was created", catalog.Sends, "email", "name")

	addEvent(reg, "UserNameChanged", "User Name Changed",
		"A user's name was changed", catalog.Sends, "oldName", "newName")

	reg.AddQuery("user-service", catalog.Message{
		Kind: catalog.QueryMessage, ID: "GetUser", Name: "Get User",
		Version: "1.0.0", Summary: "Get a user by ID", Direction: catalog.Receives,
		Schema: &catalog.Schema{Type: catalog.TypeObject, Properties: map[string]catalog.Property{
			"id": {Type: catalog.TypeString},
		}},
	})

	return reg.Build()
}
