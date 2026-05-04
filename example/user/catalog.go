package main

import (
	"fmt"

	catalogadapters "github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type catalogableEvent struct {
	*event.CatalogCore
}

func generateEventCatalog(outputDir string) error {
	builder := catalogadapters.NewBuilder("User Service", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "Manages user accounts")

	builder.AddEvent("user-svc", mustNewCatalogEvent("UserCreated",
		"User Created", "Fired when a new user account is created"))
	builder.AddEvent("user-svc", mustNewCatalogEvent("UserNameChanged",
		"User Name Changed", "Fired when a user changes their display name"))

	builder.AddDomain("identity", "Identity",
		"User identity and account management", []string{"user-svc"})

	cat := builder.Build()

	fmt.Printf("  [catalog] %d services, %d domains\n",
		len(cat.Services), len(cat.Domains))

	for _, svc := range cat.Services {
		fmt.Printf("  [catalog] Service %q: %d events\n", svc.Name, len(svc.Events))
	}

	return builder.ExportEventCatalog(outputDir)
}

func mustNewCatalogEvent(eventType, name, summary string) event.Catalogable {
	aggID := id.NewAggregateID()

	core, err := event.NewCatalogCore(
		event.Type(eventType), aggID, "User", 1, nil,
		event.CatalogMeta{
			Name:          name,
			Version:       "1.0.0",
			Summary:       summary,
			AggregateType: "User",
		},
	)
	if err != nil {
		panic(err)
	}

	return &catalogableEvent{CatalogCore: core}
}
