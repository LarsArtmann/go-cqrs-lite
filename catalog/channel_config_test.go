package catalog_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func newConfiguredChannel(t *testing.T, opts ...catalog.ChannelOption) catalog.Channel {
	t.Helper()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.AddChannel(catalog.Channel{ID: "ch", Name: "Ch", Version: "1.0.0"})
	b.ConfigureChannel("ch", opts...)

	cat := b.Build()

	return cat.Channels[0]
}

func TestBuilder_ConfigureChannel_Address(t *testing.T) {
	t.Parallel()

	ch := newConfiguredChannel(t, catalog.ChannelAddress("order.events"))

	if ch.Address != "order.events" {
		t.Errorf("expected order.events, got %s", ch.Address)
	}
}

func TestBuilder_ConfigureChannel_Protocols(t *testing.T) {
	t.Parallel()

	ch := newConfiguredChannel(t, catalog.ChannelProtocols("kafka", "http"))

	if len(ch.Protocols) != 2 || ch.Protocols[0] != "kafka" {
		t.Errorf("expected [kafka, http], got %v", ch.Protocols)
	}
}

func TestBuilder_ConfigureChannel_Messages(t *testing.T) {
	t.Parallel()

	ch := newConfiguredChannel(t, catalog.ChannelMessages(
		catalog.MessageID("order.created"),
		catalog.MessageID("order.cancelled"),
	))

	if len(ch.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(ch.Messages))
	}
}

func TestBuilder_ConfigureChannel_DeliveryGuarantee(t *testing.T) {
	t.Parallel()

	ch := newConfiguredChannel(t, catalog.ChannelDeliveryGuarantee("at-least-once"))

	if ch.DeliveryGuarantee != "at-least-once" {
		t.Errorf("expected at-least-once, got %s", ch.DeliveryGuarantee)
	}
}

func TestBuilder_ConfigureChannel_Parameters(t *testing.T) {
	t.Parallel()

	ch := newConfiguredChannel(t, catalog.ChannelParameters(map[string]catalog.ChannelParam{
		"orderId": {Description: "The order ID"},
	}))

	if len(ch.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(ch.Parameters))
	}

	if ch.Parameters["orderId"].Description != "The order ID" {
		t.Errorf("expected description, got %v", ch.Parameters["orderId"])
	}
}

func TestBuilder_ConfigureChannel_Routes(t *testing.T) {
	t.Parallel()

	ch := newConfiguredChannel(t, catalog.ChannelRoutes(
		catalog.ChannelRoute{
			ID: catalog.ChannelID("route1"),
			To: []catalog.ChannelID{"orders", "payments"},
		},
	))

	if len(ch.Routes) != 1 || ch.Routes[0].ID != catalog.ChannelID("route1") {
		t.Errorf("expected 1 route with ID route1, got %v", ch.Routes)
	}
}

func TestBuilder_ConfigureChannel_Owners(t *testing.T) {
	t.Parallel()

	ch := newConfiguredChannel(t, catalog.ChannelOwners("team-a", "bob"))

	if len(ch.Owners) != 2 {
		t.Fatalf("expected 2 owners, got %d", len(ch.Owners))
	}
}

func TestBuilder_ConfigureChannel_Badges(t *testing.T) {
	t.Parallel()

	ch := newConfiguredChannel(t, catalog.ChannelBadges(
		catalog.Badge{Content: "Kafka", BackgroundColor: "orange"},
	))

	if len(ch.Badges) != 1 || ch.Badges[0].Content != "Kafka" {
		t.Errorf("expected Kafka badge, got %v", ch.Badges)
	}
}

func TestBuilder_ConfigureChannel_MultipleOptions(t *testing.T) {
	t.Parallel()

	ch := newConfiguredChannel(
		t,
		catalog.ChannelAddress("order.events"),
		catalog.ChannelProtocols("kafka"),
		catalog.ChannelDeliveryGuarantee("exactly-once"),
		catalog.ChannelOwners("team-a"),
		catalog.ChannelBadges(catalog.Badge{Content: "Production"}),
	)

	if ch.Address != "order.events" {
		t.Errorf("expected order.events, got %s", ch.Address)
	}

	if len(ch.Protocols) != 1 {
		t.Errorf("expected 1 protocol, got %d", len(ch.Protocols))
	}

	if ch.DeliveryGuarantee != "exactly-once" {
		t.Errorf("expected exactly-once, got %s", ch.DeliveryGuarantee)
	}

	if len(ch.Owners) != 1 {
		t.Errorf("expected 1 owner, got %d", len(ch.Owners))
	}

	if len(ch.Badges) != 1 {
		t.Errorf("expected 1 badge, got %d", len(ch.Badges))
	}
}

func TestBuilder_ConfigureChannel_NonexistentIgnored(t *testing.T) {
	t.Parallel()

	b := catalog.NewBuilder("Test", "1.0.0")
	b.ConfigureChannel(
		"nonexistent",
		catalog.ChannelBadges(catalog.Badge{Content: "Should not panic"}),
	)

	cat := b.Build()
	if len(cat.Channels) != 0 {
		t.Errorf("expected 0 channels, got %d", len(cat.Channels))
	}
}
