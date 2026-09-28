package system

import (
	"context"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type explainItemCreated struct{ Name string }

type explainInput struct{ Key string }

type explainRow struct{ Name string }

// TestExplainRendersQueryPlacements pins the topology view's per-query
// placement section: Explain must surface the planner's engine assignment
// and the declared Volume hint for every projection — not just the
// collections count (the gap recorded in TODO_LIST:1099).
func TestExplainRendersQueryPlacements(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		metaengine.Query[explainInput, explainRow](
			"items_by_name",
			metaengine.OnRecord(
				explainItemCreated{},
				func(_ record.Record, e explainItemCreated) (string, explainRow) {
					return e.Name, explainRow(e)
				},
			),
			metaengine.Volume(2_500),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	sys := &System{projStore: store}

	out := sys.Explain(context.Background())

	for _, want := range []string{
		"ProjectionStore:",
		"~ items_by_name: memory (map",
		"volume=2500/s",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Explain output missing %q:\n%s", want, out)
		}
	}
}

// TestExplainWithoutProjectionStore pins the no-projection path: no
// placement section renders, and Explain does not panic on an empty System.
func TestExplainWithoutProjectionStore(t *testing.T) {
	t.Parallel()

	out := (&System{}).Explain(context.Background())

	if strings.Contains(out, "~") {
		t.Errorf("Explain without a projection store must not render placements:\n%s", out)
	}

	if !strings.Contains(out, "System Topology:") {
		t.Errorf("Explain lost its topology header:\n%s", out)
	}
}
