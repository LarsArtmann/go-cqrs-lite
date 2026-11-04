package asyncapi

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func TestExporter_Export_NoFabricatedServer(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry(
		catalog.Service{ID: "svc", Name: "Svc", Version: "1.0.0"},
	)
	reg.AddCommand("svc", basicCommand("DoThing"))
	cat := reg.Build()

	doc := NewExporter("Svc", "1.0.0").Export(cat)

	if len(doc.Servers) != 0 {
		t.Errorf(
			"no server declared: expected no fabricated broker, got %v",
			doc.Servers,
		)
	}
}

func TestExporter_Export_DeclaredChannelWins(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry(
		catalog.Service{ID: "crm", Name: "CRM", Version: "1.0.0"},
	)
	reg.AddCommand("crm", basicCommand("contact.create"))
	reg.AddCommand("crm", basicCommand("unlinked.thing"))
	reg.AddChannel(catalog.Channel{
		ID:                "crm.journal",
		Name:              "CRM Journal",
		Summary:           "The journal of record",
		Address:           "ledger",
		DeliveryGuarantee: "at-least-once",
		Messages:          []catalog.MessageID{"contact.create"},
	})
	cat := reg.Build()

	doc := NewExporter("CRM", "1.0.0").Export(cat)

	linked, ok := doc.Channels["crm.journal"]
	if !ok {
		t.Fatalf("declared channel crm.journal missing from %v", keys(doc.Channels))
	}

	if linked.Address != "ledger" {
		t.Errorf("declared channel address = %q, want %q", linked.Address, "ledger")
	}

	if linked.Title != "CRM Journal" {
		t.Errorf("declared channel title = %q, want %q", linked.Title, "CRM Journal")
	}

	if _, ok := linked.Messages["command.contact.create"]; !ok {
		t.Errorf(
			"declared channel must reference its message, got messages %v",
			linked.Messages,
		)
	}

	if _, ok := doc.Channels["commands.unlinked.thing"]; !ok {
		t.Fatal("unlinked message must keep its synthetic channel")
	}

	if _, ok := doc.Channels["commands.contact.create"]; ok {
		t.Error("linked message must not ALSO emit a synthetic channel")
	}

	op, ok := doc.Operations["receivecontact.create"]
	if !ok {
		t.Fatal("missing receivecontact.create operation")
	}

	if op.Channel.Ref != "#/channels/crm.journal" {
		t.Errorf("operation must point at the declared channel, got %q", op.Channel.Ref)
	}
}

func TestExporter_Export_MessageChannelLink(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry(
		catalog.Service{ID: "crm", Name: "CRM", Version: "1.0.0"},
	)
	reg.AddCommand("crm", basicCommand("contact.update"))
	reg.AddChannel(catalog.Channel{
		ID:      "crm.http",
		Name:    "CRM HTTP",
		Address: "api",
	})

	b := cattest.NewTestRegistry(
		catalog.Service{ID: "crm", Name: "CRM", Version: "1.0.0"},
	)
	b.AddCommand("crm", catalog.Message{
		Kind:     catalog.CommandMessage,
		ID:       "contact.update",
		Name:     "contact.update",
		Version:  "1.0.0",
		Channels: []catalog.ChannelID{"crm.http"},
	})
	catMsgLinked := b.Build()

	doc := NewExporter("CRM", "1.0.0").Export(cat)
	docMsgLinked := NewExporter("CRM", "1.0.0").Export(catMsgLinked)

	for name, d := range map[string]*Document{"channel-lists-message": doc, "message-references-channel": docMsgLinked} {
		ch, ok := d.Channels["crm.http"]
		if !ok {
			t.Fatalf("%s: declared channel crm.http missing from %v", name, keys(d.Channels))
		}

		if len(ch.Messages) != 1 {
			t.Errorf("%s: expected 1 message on declared channel, got %d", name, len(ch.Messages))
		}
	}
}

func keys(m map[string]Channel) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}
