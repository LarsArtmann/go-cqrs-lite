package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/d2"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
)

func generateEventCatalog(outputDir string) error {
	builder := catalog.NewBuilder("User Service", "1.0.0")
	builder.AddService(
		"user-svc", "User Service", "1.0.0", "Manages user accounts",
		catalog.Command[UserCreatedPayload](
			catalog.MessageID(cmdCreateUser),
			catalog.Name("Create User"),
			catalog.Summary("Creates a new user account"),
		),
		catalog.Command[UserNameChangedPayload](
			catalog.MessageID(cmdChangeUserName),
			catalog.Name("Change User Name"),
			catalog.Summary("Changes a user's display name"),
		),
		catalog.Event[UserCreatedPayload](
			catalog.MessageID(eventUserCreated), catalog.Sends,
			catalog.Name("User Created"),
			catalog.Summary("Fired when a new user account is created"),
		),
		catalog.Event[UserNameChangedPayload](
			catalog.MessageID(eventUserNameChanged), catalog.Sends,
			catalog.Name("User Name Changed"),
			catalog.Summary("Fired when a user changes their display name"),
		),
	)

	builder.AddDomain("identity", "Identity", "1.0.0",
		"User identity and account management", "user-svc")

	builder.AddDataStore(catalog.DataStore{
		ID: "user-db", Name: "Users Database", Version: "1.0.0",
		ContainerType: "database", Technology: "postgres@16",
		Summary: "Primary user data store",
	})

	builder.AddChannel(catalog.Channel{
		ID: "user-events", Name: "User Events", Version: "1.0.0",
		Summary:   "All user-related domain events",
		Protocols: []string{"kafka"},
	})

	builder.AddTeam(catalog.Team{
		ID: "identity-team", Name: "Identity Team",
		Summary: "Team responsible for user identity",
		Members: []string{"alice"},
	})

	builder.AddUser(catalog.User{
		ID: "alice", Name: "Alice Smith", Role: "Senior Engineer",
	})

	cat := builder.Build()

	fmt.Printf("  [catalog] %d services, %d domains, %d channels, %d data stores\n",
		len(cat.Services), len(cat.Domains), len(cat.Channels), len(cat.DataStores))

	for _, svc := range cat.Services {
		fmt.Printf("  [catalog] Service %q: %d commands, %d events\n",
			svc.Name, len(svc.Commands), len(svc.Events))
	}

	if err := eventcatalog.NewExporter(outputDir).Export(cat); err != nil {
		return fmt.Errorf("export eventcatalog: %w", err)
	}

	d2Output := d2.NewExporter("User Service", "1.0.0").Export(cat)
	d2Path := filepath.Join(outputDir, "architecture.d2")
	if err := os.WriteFile(d2Path, []byte(d2Output), 0644); err != nil {
		return fmt.Errorf("write d2: %w", err)
	}

	asyncDoc := asyncapi.NewExporter("User Service", "1.0.0").Export(cat)
	asyncPath := filepath.Join(outputDir, "asyncapi.json")
	if err := os.WriteFile(asyncPath, mustMarshal(asyncDoc), 0644); err != nil {
		return fmt.Errorf("write asyncapi: %w", err)
	}

	fmt.Printf("  [catalog] Exported EventCatalog, D2, AsyncAPI to %s\n", outputDir)

	return nil
}
