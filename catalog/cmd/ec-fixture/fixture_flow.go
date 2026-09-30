package main

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// fixtureDataContract is the data product output contract the fixture
// places at contracts/orders.yaml.
const fixtureDataContract = `# Data contract: orders (fixture)
id: orders
owner: order-team
type: table
description: One row per order event aggregate
fields:
  - name: order_id
    type: string
    required: true
`

func checkoutSteps(withAgent bool) []catalog.FlowStep {
	steps := []catalog.FlowStep{
		{
			ID:    "s1",
			Title: "Customer",
			Actor: &catalog.FlowActor{Name: "Customer", Summary: "Places orders"},
		},
		{
			ID:       "s2",
			Title:    "Submit",
			Service:  &catalog.FlowStepRef{ID: fixtureServiceID},
			NextStep: &catalog.FlowEdge{ID: "s3"},
		},
		{
			ID:        "s3",
			Title:     "Create",
			Message:   &catalog.FlowStepRef{ID: "CreateOrder"},
			NextSteps: []catalog.FlowEdge{{ID: "s4"}},
		},
		{ID: "s4", Title: "Persist", DataStore: &catalog.FlowStepRef{ID: "orders-db"}},
	}
	if withAgent {
		steps = append(
			steps,
			catalog.FlowStep{
				ID:    "s5",
				Title: "AI summary",
				Agent: &catalog.FlowStepRef{ID: "order-bot"},
			},
		)
	}

	return append(
		steps,
		catalog.FlowStep{
			ID:          "s6",
			Title:       "Analytics",
			DataProduct: &catalog.FlowStepRef{ID: "order-analytics"},
		},
		catalog.FlowStep{
			ID:       "s7",
			Title:    "Payment provider",
			External: &catalog.FlowActor{Name: "Stripe", URL: "https://stripe.com"},
		},
		catalog.FlowStep{
			ID:     "s8",
			Title:  "Audit",
			Custom: &catalog.FlowCustomNode{Title: "Audit log", Icon: "clipboard"},
		},
	)
}
