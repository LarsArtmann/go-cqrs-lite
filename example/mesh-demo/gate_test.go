package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestGate_TypoCaught pins the hard side of the runtime coeffect gate: the
// typo'd import fails system.New and the error names the dangling
// subscription — the silently-dead-projection class, caught at composition.
func TestGate_TypoCaught(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx, ordersGateDomain(typoUniverse()), gateDeployment())
	if !errors.Is(err, system.ErrDanglingEventSubscription) {
		t.Fatalf("want ErrDanglingEventSubscription, got %v", err)
	}

	if sys != nil {
		t.Fatal("system should be nil on gate failure")
	}

	if want := string(evtInvoiceIssued); !strings.Contains(err.Error(), want) {
		t.Fatalf("error should name the dangling type %q: %v", want, err)
	}
}

// TestGate_ContractUniverseComposes pins the clean side: the universe the
// bilateral contracts describe composes without a peep.
func TestGate_ContractUniverseComposes(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx, ordersGateDomain(contractUniverse()), gateDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
