package compat

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func validCatalog() *catalog.Catalog {
	return &catalog.Catalog{
		Title:   "orders",
		Version: "1.0.0",
		Channels: []catalog.Channel{
			{ID: "events", Name: "events", Messages: []catalog.MessageID{"order_placed"}},
		},
		DataStores: []catalog.DataStore{{ID: "orders_db", Name: "orders_db"}},
		Services: []catalog.Service{{
			ID:       "orders",
			Name:     "Orders",
			Version:  "1.0.0",
			WritesTo: []catalog.DataStoreID{"orders_db"},
			Events: []catalog.Message{{
				Kind:      catalog.EventMessage,
				ID:        "order_placed",
				Name:      "order_placed",
				Version:   "1.0.0",
				Direction: catalog.Sends,
				Channels:  []catalog.ChannelID{"events"},
			}},
			Commands: []catalog.Message{{
				Kind:      catalog.CommandMessage,
				ID:        "place_order",
				Name:      "place_order",
				Version:   "1.0.0",
				Direction: catalog.Receives,
			}},
		}},
	}
}

func TestValidateCleanCatalog(t *testing.T) {
	t.Parallel()

	if violations := Validate(validCatalog()); len(violations) != 0 {
		t.Fatalf("a valid catalog must produce no violations, got %v", violations)
	}

	if violations := Validate(nil); len(violations) != 0 {
		t.Fatalf("nil catalog must produce no violations, got %v", violations)
	}
}

func TestValidateDirectionLegality(t *testing.T) {
	t.Parallel()

	cat := validCatalog()
	cat.Services[0].Commands[0].Direction = catalog.Sends

	violations := Validate(cat)
	if len(violations) != 1 {
		t.Fatalf("want one violation, got %v", violations)
	}

	if got := violations[0].Path; got != "services[orders].command[place_order].direction" {
		t.Fatalf("path = %q", got)
	}
}

func TestValidateEventDirectionRequired(t *testing.T) {
	t.Parallel()

	cat := validCatalog()
	cat.Services[0].Events[0].Direction = ""

	violations := Validate(cat)

	found := false

	for _, violation := range violations {
		if violation.Path == "services[orders].event[order_placed].direction" {
			found = true
		}
	}

	if !found {
		t.Fatalf("an event without a direction must be flagged, got %v", violations)
	}
}

func TestValidateChannelReferenceResolution(t *testing.T) {
	t.Parallel()

	cat := validCatalog()
	cat.Channels[0].Messages = append(cat.Channels[0].Messages, "ghost_event")

	violations := Validate(cat)
	if len(violations) != 1 {
		t.Fatalf("want one violation for the channel side, got %v", violations)
	}

	cat = validCatalog()
	cat.Services[0].Events[0].Channels = []catalog.ChannelID{"ghost_channel"}

	violations = Validate(cat)
	if len(violations) != 1 {
		t.Fatalf("want one violation for the message side, got %v", violations)
	}

	if got := violations[0].Path; got != "services[orders].event[order_placed].channels" {
		t.Fatalf("path = %q", got)
	}
}

func TestValidateDataStoreReferenceResolution(t *testing.T) {
	t.Parallel()

	cat := validCatalog()
	cat.Services[0].ReadsFrom = []catalog.DataStoreID{"ghost_db"}

	violations := Validate(cat)
	if len(violations) != 1 {
		t.Fatalf("want one violation, got %v", violations)
	}

	if got := violations[0].Path; got != "services[orders].readsFrom" {
		t.Fatalf("path = %q", got)
	}
}

// TestValidateDelegatesCoeffects is the anti-split-brain pin: the coeffect rule
// must come from Catalog.ValidateCoeffects verbatim, so the two can never
// disagree about the same catalog.
func TestValidateDelegatesCoeffects(t *testing.T) {
	t.Parallel()

	cat := validCatalog()
	cat.Services[0].Events = append(cat.Services[0].Events, catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        "order_placed_elsewhere",
		Name:      "order_placed_elsewhere",
		Version:   "1.0.0",
		Direction: catalog.Receives,
	})

	own := cat.ValidateCoeffects()
	if len(own) != 1 {
		t.Fatalf("fixture must produce exactly one coeffect violation, got %v", own)
	}

	composed := Validate(cat)

	found := false

	for _, v := range composed {
		if v.Path == own[0].Path && v.Message == own[0].Message {
			found = true
		}
	}

	if !found {
		t.Fatalf("Validate must reproduce ValidateCoeffects verbatim: want %v in %v", own, composed)
	}
}

// TestValidateDoesNotDuplicateCatalogValidate pins the additive contract: a
// catalog that Catalog.Validate flags must produce ZERO compat violations, so
// the two gates stay disjoint.
func TestValidateDoesNotDuplicateCatalogValidate(t *testing.T) {
	t.Parallel()

	cat := validCatalog()
	cat.Title = ""
	cat.Version = ""

	if own := cat.Validate(); len(own) == 0 {
		t.Fatal("fixture must be invalid for Catalog.Validate")
	}

	if violations := Validate(cat); len(violations) != 0 {
		t.Fatalf("compat.Validate must not restate Catalog.Validate's checks, got %v", violations)
	}
}
