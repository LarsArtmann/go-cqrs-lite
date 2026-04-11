package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

const (
	CmdCreateUser      command.Type = "user.create"
	CmdChangeUserEmail command.Type = "user.change_email"
)

type CreateUser struct {
	*command.CatalogCore

	Name  string `json:"name" doc:"Full name of the user"`
	Email string `json:"email" doc:"Email address of the user"`
}

type ChangeUserEmail struct {
	*command.CatalogCore

	NewEmail string `json:"newEmail" doc:"New email address for the user"`
}

func main() {
	outputDir := "output"
	if len(os.Args) > 1 {
		outputDir = os.Args[1]
	}

	aggID := id.NewAggregateID()

	builder := adapters.NewBuilder("User Service API", "1.0.0")
	builder.AddService("user-service", "User Service", "1.0.0", "Manages user accounts")
	builder.AddDomain("identity", "Identity", "User identity management", []string{"user-service"})

	builder.AddCommand("user-service", &CreateUser{
		CatalogCore: command.NewCatalogCore(CmdCreateUser, aggID, command.CatalogMeta{
			Name: "CreateUser", Version: "1.0.0", Summary: "Creates a new user account",
		}),
	})
	builder.AddCommand("user-service", &ChangeUserEmail{
		CatalogCore: command.NewCatalogCore(CmdChangeUserEmail, aggID, command.CatalogMeta{
			Name: "ChangeUserEmail", Version: "1.0.0", Summary: "Updates a user's email address",
		}),
	})

	ecDir := filepath.Join(outputDir, "eventcatalog")
	if err := builder.ExportEventCatalog(ecDir); err != nil {
		log.Fatalf("export eventcatalog: %v", err)
	}
	fmt.Printf("EventCatalog exported to %s\n", ecDir)

	doc, err := builder.ExportAsyncAPI("User Service API", "1.0.0",
		asyncapi.WithServer("production", "kafka:9092", "kafka"),
	)
	if err != nil {
		log.Fatalf("export asyncapi: %v", err)
	}

	yamlBytes, err := doc.MarshalYAML()
	if err != nil {
		log.Fatalf("marshal yaml: %v", err)
	}

	asyncPath := filepath.Join(outputDir, "asyncapi.yaml")
	if err := os.WriteFile(asyncPath, yamlBytes, 0o644); err != nil {
		log.Fatalf("write asyncapi: %v", err)
	}
	fmt.Printf("AsyncAPI exported to %s\n", asyncPath)

	jsonBytes, err := doc.MarshalJSON()
	if err != nil {
		log.Fatalf("marshal json: %v", err)
	}

	jsonPath := filepath.Join(outputDir, "asyncapi.json")
	if err := os.WriteFile(jsonPath, jsonBytes, 0o644); err != nil {
		log.Fatalf("write asyncapi json: %v", err)
	}
	fmt.Printf("AsyncAPI JSON exported to %s\n", jsonPath)

	fmt.Println("\nDone! Catalog generated from Go types with zero manual config.")
}
