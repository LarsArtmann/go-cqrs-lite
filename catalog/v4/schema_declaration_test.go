package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
)

type wireBalanceUpdated struct {
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
}

func TestFromTypedSchema_RendersDeclarationAsCatalogEvent(t *testing.T) {
	t.Parallel()

	decl := schema.EventOf[wireBalanceUpdated]("balance.updated", 3,
		schema.RenameField("balance.updated", 2, "amount", "amountCents"),
	)

	builder := catalog.NewBuilder("Ledger", "1.0.0")
	builder.AddService("ledger", "Ledger", "1.0.0", "The ledger service",
		catalog.FromTypedSchema(decl, catalog.Sends),
	)

	cat := builder.Build()
	svc := cat.Services[0]

	events := svc.Events
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}

	rendered := events[0]

	if rendered.ID != "balance.updated" {
		t.Errorf("ID = %q, want the wire name %q", rendered.ID, "balance.updated")
	}

	if string(rendered.Version) != "3.0.0" {
		t.Errorf("Version = %q, want derived %q", rendered.Version, "3.0.0")
	}

	if rendered.Schema == nil {
		t.Fatal("payload schema not derived from the bound Go type")
	}

	if _, ok := rendered.Schema.Properties["amountCents"]; !ok {
		t.Error("expected schema to carry the payload type's fields")
	}
}

func TestFromTypedSchema_WithVersionOverridesDerivedString(t *testing.T) {
	t.Parallel()

	decl := schema.EventOf[wireBalanceUpdated]("balance.updated", 3)

	builder := catalog.NewBuilder("Ledger", "1.0.0")
	builder.AddService("ledger", "Ledger", "1.0.0", "The ledger service",
		catalog.FromTypedSchema(decl, catalog.Sends, catalog.WithVersion("3.1.0")),
	)

	events := builder.Build().Services[0].Events
	if len(events) != 1 || string(events[0].Version) != "3.1.0" {
		t.Fatalf("explicit WithVersion must win; got %+v", events)
	}
}

func TestSemverFromWire(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wire int
		want string
	}{
		{1, "1.0.0"},
		{2, "2.0.0"},
		{12, "12.0.0"},
	}

	for _, tt := range tests {
		if got := catalog.SemverFromWire(tt.wire); got != tt.want {
			t.Errorf("SemverFromWire(%d) = %q, want %q", tt.wire, got, tt.want)
		}
	}
}
