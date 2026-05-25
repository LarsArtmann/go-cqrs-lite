package main

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
)

func generateEventCatalog(outputDir string) error {
	builder := catalog.NewBuilder("User Service", "1.0.0")
	builder.AddService(
		"user-svc", "User Service", "1.0.0", "Manages user accounts",
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
		Summary:       "Primary user data store",
	})

	builder.AddChannel(catalog.Channel{
		ID: "user-events", Name: "User Events", Version: "1.0.0",
		Summary: "All user-related domain events",
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
		fmt.Printf("  [catalog] Service %q: %d events\n", svc.Name, len(svc.Events))
	}

	return eventcatalog.NewExporter(outputDir).Export(cat)
}
