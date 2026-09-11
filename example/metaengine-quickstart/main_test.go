package main

import (
	"context"
	"testing"
)

// TestQuickstart_AllDemoSectionsGreen is the smoke test for the four demo
// sections main() runs. Until now "all 4 sections green" rested on manual
// runs; this pins it in CI. Each run*Demo already fails loudly (returns an
// error) on wrong query results, so err == nil is the "green" contract.
func TestQuickstart_AllDemoSectionsGreen(t *testing.T) {
	t.Parallel()

	sections := []struct {
		title string
		run   func(context.Context) error
	}{
		{"1/4 Map ADT: CRUD task view", runTaskDemo},
		{"2/4 Graph ADT: follow network traversal", runGraphDemo},
		{"3/4 Vector ADT: k-NN semantic search", runVectorDemo},
		{"4/4 Operator config: boot from cqrs.yaml", runConfigFileDemo},
	}

	ctx := context.Background()

	for _, section := range sections {
		if err := section.run(ctx); err != nil {
			t.Errorf("%s: %v", section.title, err)
		}
	}
}
