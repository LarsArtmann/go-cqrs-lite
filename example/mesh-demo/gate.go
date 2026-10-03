package main

// The runtime twin of the catalog-side coeffect story (core.md §3.9):
// deploying ONE bounded context (orders) as a system.New composition
// declares the same event universe the bilateral contracts describe — own
// emissions plus the imported invoice.issued — and the coeffect gate
// catches a contract typo at COMPOSITION time instead of as a silently
// dead projection at runtime.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// OrderStatusView is the orders read model the gate demo folds: placed →
// invoiced → completed. Keyed by OrderID — the bilateral contract payloads
// key on OrderID, so the evolution declares EvolveKey("OrderID").
type OrderStatusView struct {
	OrderID    string
	Placed     bool
	TotalCents int64
	InvoiceRef string
	Completed  bool
}

// ordersEvolution folds the context's own emissions plus the consumed
// foreign event — cross-context integration is a fold, not a call
// (ADR-0146). The payload types ARE the bilateral contracts from
// orders.go/billing.go: the same shapes the catalogs declare.
func ordersEvolution() system.EvolutionSpec {
	return system.OnEvolution(
		system.OnEvolution(
			system.OnEvolution(
				system.Evolve[OrderStatusView]("orders_status", system.EvolveKey("OrderID")),
				string(evtOrderPlaced), OrderPlacedPayload{},
				func(e OrderPlacedPayload, v *OrderStatusView) {
					v.Placed, v.TotalCents = true, e.TotalCents
				},
			),
			string(evtInvoiceIssued), InvoiceReceivedPayload{},
			func(e InvoiceReceivedPayload, v *OrderStatusView) { v.InvoiceRef = e.InvoiceRef },
		),
		string(evtOrderCompleted), OrderCompletedPayload{},
		func(_ OrderCompletedPayload, v *OrderStatusView) { v.Completed = true },
	).Done()
}

// ordersGateDomain parameterizes the declared universe: the typo case and
// the contract case differ ONLY in the Events declaration.
func ordersGateDomain(events []event.Type) system.DomainConfig {
	return system.DomainConfig{
		Evolutions: []system.EvolutionSpec{ordersEvolution()},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[OrderStatusView]("orders_lookup").Done(),
		},
		Events: events,
	}
}

// gateDeployment is the operator side: engines are a deployment decision,
// not a domain one — the demo runs the in-core memory engine.
func gateDeployment() system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}
}

// contractUniverse declares what the bilateral contracts describe: the
// orders context's own emissions plus the imported invoice.issued.
func contractUniverse() []event.Type {
	return []event.Type{evtOrderPlaced, evtOrderCompleted, evtInvoiceIssued}
}

// typoUniverse carries the class the gate exists to catch: a typo'd import
// that no producer will ever emit.
func typoUniverse() []event.Type {
	return []event.Type{evtOrderPlaced, evtOrderCompleted, event.Type("invoice.issud")}
}

// runGate composes the orders context twice: with the typo'd import (the
// gate must reject it, naming the dangling subscription) and with the
// contract universe (must compose cleanly).
func runGate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Println("==> composing orders with a typo'd import (invoice.issud)...")

	_, err := system.New(ctx, ordersGateDomain(typoUniverse()), gateDeployment())
	if !errors.Is(err, system.ErrDanglingEventSubscription) {
		return fmt.Errorf(
			"gate demo: typo composition: want ErrDanglingEventSubscription, got %w",
			err,
		)
	}

	fmt.Printf("    caught at composition: %v\n", err)

	fmt.Println("==> composing orders with the contract universe...")

	sys, err := system.New(ctx, ordersGateDomain(contractUniverse()), gateDeployment())
	if err != nil {
		return fmt.Errorf("gate demo: contract composition: %w", err)
	}

	fmt.Println("    composed cleanly — folds wired, lookup routed")

	return sys.Close()
}
