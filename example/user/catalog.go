package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/d2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/eventcatalog"
)

const outputFilePerm = 0o600

func generateEventCatalog(outputDir string) error {
	builder := catalog.NewBuilder("User Service", "1.0.0")
	builder.AddService(
		"user-svc", "User Service", "1.0.0", "Manages user accounts",
		catalog.Command[CreateUserPayload](
			catalog.MessageID(cmdCreateUser),
			catalog.WithName("Create User"),
			catalog.WithSummary("Creates a new user account"),
		),
		catalog.Command[ChangeUserNamePayload](
			catalog.MessageID(cmdChangeUserName),
			catalog.WithName("Change User Name"),
			catalog.WithSummary("Changes a user's display name"),
		),
		catalog.Event[UserCreatedPayload](
			catalog.MessageID(eventUserCreated), catalog.Sends,
			catalog.WithName("User Created"),
			catalog.WithSummary("Fired when a new user account is created"),
		),
		catalog.Event[UserNameChangedPayload](
			catalog.MessageID(eventUserNameChanged), catalog.Sends,
			catalog.WithName("User Name Changed"),
			catalog.WithSummary("Fired when a user changes their display name"),
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
		Protocols: []catalog.Protocol{"kafka"},
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
		return fmt.Errorf("export eventcatalog to %s: %w", outputDir, err)
	}

	d2Output := d2.NewExporter("User Service", "1.0.0").Export(cat)

	d2Path := filepath.Join(outputDir, "architecture.d2")
	if err := os.WriteFile(d2Path, []byte(d2Output), outputFilePerm); err != nil {
		return fmt.Errorf("write d2 to %s: %w", d2Path, err)
	}

	asyncDoc := asyncapi.NewExporter("User Service", "1.0.0").Export(cat)

	asyncPath := filepath.Join(outputDir, "asyncapi.json")
	if err := os.WriteFile(asyncPath, mustMarshal(asyncDoc), outputFilePerm); err != nil {
		return fmt.Errorf("write asyncapi to %s: %w", asyncPath, err)
	}

	fmt.Printf("  [catalog] Exported EventCatalog, D2, AsyncAPI to %s\n", outputDir)

	return nil
}
