package main

import (
	"fmt"
	"log"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	catalogadapters "github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type userCreatedEvent struct {
	*event.CatalogCore

	Email string `description:"The user's email address" json:"email"`
}

type userNameChangedEvent struct {
	*event.CatalogCore

	Name string `description:"The new display name" json:"name"`
}

func generateEventCatalog(outputDir string) error {
	builder := catalogadapters.NewBuilder("User Service", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "Manages user accounts")

	builder.AddEvent("user-svc", newUserCreatedEvent())
	builder.AddEvent("user-svc", newUserNameChangedEvent())

	builder.AddDomain(
		"identity",
		"Identity",
		"User identity and account management",
		[]string{"user-svc"},
	)

	cat := builder.Build()

	printCatalogSummary(cat)

	return builder.ExportEventCatalog(outputDir)
}

func newUserCreatedEvent() event.Catalogable {
	aggID := id.NewAggregateID()

	core, err := event.NewCatalogCore(
		"UserCreated",
		aggID,
		"User",
		1,
		nil,
		event.CatalogMeta{
			Name:          "User Created",
			Version:       "1.0.0",
			Summary:       "Fired when a new user account is created",
			AggregateType: "User",
		},
	)
	if err != nil {
		log.Fatalf("create UserCreated catalog core: %v", err)
	}

	return &userCreatedEvent{CatalogCore: core}
}

func newUserNameChangedEvent() event.Catalogable {
	aggID := id.NewAggregateID()

	core, err := event.NewCatalogCore(
		"UserNameChanged",
		aggID,
		"User",
		1,
		nil,
		event.CatalogMeta{
			Name:          "User Name Changed",
			Version:       "1.0.0",
			Summary:       "Fired when a user changes their display name",
			AggregateType: "User",
		},
	)
	if err != nil {
		log.Fatalf("create UserNameChanged catalog core: %v", err)
	}

	return &userNameChangedEvent{CatalogCore: core}
}

func printCatalogSummary(cat *catalog.Catalog) {
	fmt.Printf("Catalog: %s v%s\n", cat.Title, cat.Version)

	for _, svc := range cat.Services {
		fmt.Printf("  Service: %s (%s)\n", svc.Name, svc.ID)

		for _, evt := range svc.Events {
			schemaInfo := "(no schema)"
			if evt.Schema != nil {
				schemaInfo = fmt.Sprintf("%d properties", len(evt.Schema.Properties))
			}

			fmt.Printf("    Event: %s — %s %s\n", evt.Name, evt.Summary, schemaInfo)
		}
	}

	for _, domain := range cat.Domains {
		fmt.Printf("  Domain: %s (%s)\n", domain.Name, domain.ID)
	}
}
