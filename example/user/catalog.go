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
			string(eventUserCreated), catalog.Sends,
			catalog.Name("User Created"),
			catalog.Summary("Fired when a new user account is created"),
		),
		catalog.Event[UserNameChangedPayload](
			string(eventUserNameChanged), catalog.Sends,
			catalog.Name("User Name Changed"),
			catalog.Summary("Fired when a user changes their display name"),
		),
	)

	builder.AddDomain("identity", "Identity", "1.0.0",
		"User identity and account management", "user-svc")

	cat := builder.Build()

	fmt.Printf("  [catalog] %d services, %d domains\n",
		len(cat.Services), len(cat.Domains))

	for _, svc := range cat.Services {
		fmt.Printf("  [catalog] Service %q: %d events\n", svc.Name, len(svc.Events))
	}

	return eventcatalog.NewExporter(outputDir).Export(cat)
}
