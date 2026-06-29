// Package main demonstrates the deriver module (event→command derivation).
//
// It shows the stateless-saga pattern: when a "user.signed_up" event occurs, a
// composed deriver produces two derived commands (send welcome email + sync to
// CRM). Derivers compose via Then (fan-out — both see the same event) and
// Filter (event-type matching). Idempotent re-stamps each command with a
// deterministic ID derived from the source event, so re-processing the same
// event yields the same command IDs — the command-side idempotency layer
// (idempotency.CommandIdempotency) deduplicates at-least-once redeliveries.
//
// In a real app the composed deriver's AsHandler would be wired into the event
// bus via bus.SubscribeAll. This demo calls the deriver directly to keep the
// output inspectable.
//
// Run with: go run ./example/deriver
package main

import (
	"context"
	"encoding/json"
	"fmt"

	cqrscommand "github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/deriver/v3"
	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// userSignedUp is the payload of the source event.
type userSignedUp struct {
	Email string `json:"email"`
}

func main() {
	ctx := context.Background()

	// sendWelcomeEmail: derive an email.send_welcome command from the signup.
	welcomeEmail := deriver.Deriver(
		func(_ context.Context, evt cqrsevent.Event) ([]cqrscommand.Command, error) {
			var payload userSignedUp
			if err := json.Unmarshal(evt.Payload(), &payload); err != nil {
				return nil, fmt.Errorf("welcome email: unmarshal: %w", err)
			}

			cmd, err := cqrscommand.New("email.send_welcome", evt.AggregateID())
			if err != nil {
				return nil, err
			}

			return []cqrscommand.Command{cmd}, nil
		},
	)

	// syncToCRM: derive a crm.upsert_user command from the same signup.
	syncToCRM := deriver.Deriver(
		func(_ context.Context, evt cqrsevent.Event) ([]cqrscommand.Command, error) {
			var payload userSignedUp
			if err := json.Unmarshal(evt.Payload(), &payload); err != nil {
				return nil, fmt.Errorf("crm sync: unmarshal: %w", err)
			}

			cmd, err := cqrscommand.New("crm.upsert_user", evt.AggregateID())
			if err != nil {
				return nil, err
			}

			return []cqrscommand.Command{cmd}, nil
		},
	)

	// Compose: fan-out (both derivers see the same event), scoped to one type,
	// with deterministic command IDs for at-least-once delivery safety.
	// Then runs both; Filter ignores non-matching events; Idempotent re-stamps
	// each command's ID from the source event so re-processing is dedup-able.
	composed := welcomeEmail.Then(syncToCRM).Filter("user.signed_up").Idempotent()

	// Simulate the source event.
	payloadBytes, _ := json.Marshal(userSignedUp{Email: "alice@example.com"})
	aggID := id.NewAggregateID()
	evt, _ := cqrsevent.NewEvent(
		"user.signed_up",
		aggID,
		"User",
		cqrsevent.Version(1),
		payloadBytes,
	)

	fmt.Printf("Source event: user.signed_up (id %s, aggregate %s)\n", evt.ID(), aggID)
	fmt.Println("Derived commands (first call):")

	cmds, err := composed(ctx, evt)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)

		return
	}

	for _, c := range cmds {
		fmt.Printf("  → %s (id %s, aggregate %s)\n", c.Type(), c.ID(), c.AggregateID())
	}

	// Demonstrate Idempotent: re-processing the same event yields the SAME
	// command IDs, so an idempotency store keyed on command ID deduplicates.
	fmt.Println("\nDerived commands (second call — same event, same IDs):")

	cmds2, _ := composed(ctx, evt)
	for i, c := range cmds2 {
		match := "MISMATCH"
		if c.ID() == cmds[i].ID() {
			match = "match"
		}

		fmt.Printf("  → %s (id %s) — %s\n", c.Type(), c.ID(), match)
	}

	// Demonstrate Filter: an unrelated event type produces no commands.
	otherEvt, _ := cqrsevent.NewEvent(
		"order.placed",
		aggID,
		"Order",
		cqrsevent.Version(1),
		payloadBytes,
	)
	filtered, _ := composed(ctx, otherEvt)

	fmt.Printf("\nFilter check: order.placed produced %d commands (expected 0)\n", len(filtered))
}
