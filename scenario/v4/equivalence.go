package scenario

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
)

// Interleaved composes projections into one whose Handle runs each in order
// and whose EventTypes is the deduplicated union — the "A + B interleaved"
// composition used by observational-equivalence tests (ADR-0136 context).
// Each embedded projection must share nothing but the event stream with the
// others; the property AssertObservationalEquivalence checks is exactly that
// this sharing is observationally harmless.
//
//nolint:ireturn // the composition IS the projection interface
func Interleaved(projs ...projection.Projection) projection.Projection {
	return &interleavedProjection{projs: projs}
}

type interleavedProjection struct {
	projs []projection.Projection
}

func (p *interleavedProjection) Name() string {
	names := make([]string, len(p.projs))
	for i, sub := range p.projs {
		names[i] = sub.Name()
	}

	return strings.Join(names, "+")
}

func (p *interleavedProjection) EventTypes() []event.Type {
	seen := make(map[event.Type]struct{})

	var types []event.Type

	for _, sub := range p.projs {
		for _, t := range sub.EventTypes() {
			if _, ok := seen[t]; ok {
				continue
			}

			seen[t] = struct{}{}
			types = append(types, t)
		}
	}

	return types
}

func (p *interleavedProjection) Handle(ctx context.Context, evt event.Event) error {
	for _, sub := range p.projs {
		if err := sub.Handle(ctx, evt); err != nil {
			return fmt.Errorf("interleaved %s: %w", sub.Name(), err)
		}
	}

	return nil
}

// EquivalenceProbe pairs the same logical query executed against the
// baseline run and the interleaved run of an observational-equivalence
// assertion.
type EquivalenceProbe struct {
	Baseline    func() (any, error)
	Interleaved func() (any, error)
}

// ScenarioReporter is the minimal testing surface scenario assertions
// require. *testing.T and *testing.B satisfy it; property-based runtimes
// that do not implement the full testing.TB (e.g. pgregory.net/rapid's
// rapid.T on newer Go, which lacks ArtifactDir) satisfy it directly or via
// a one-line adapter.
type ScenarioReporter interface {
	Helper()
	Fatalf(format string, args ...any)
}

// AssertObservationalEquivalence verifies the ADR-0136 observational-
// equivalence property for one projection: running events through the
// projection ALONE (baseline) and through its interleaved composition with
// other projections must yield deeply equal answers to every probe.
// baseline and interleaved must be freshly built over independent stores
// (typically via Interleaved); each probe runs one fixed query against its
// run's store.
//
//	scenario.AssertObservationalEquivalence(t, events,
//	    projectionadapter.New("tasks", storeA, decode),
//	    scenario.Interleaved(taskAdapterAB, auditAdapterAB),
//	    scenario.EquivalenceProbe{
//	        Baseline:    func() (any, error) { return storeA.Execute(findTask{ID: "t1"}) },
//	        Interleaved: func() (any, error) { return storeAB.Execute(findTask{ID: "t1"}) },
//	    },
//	)
//
// Failures name the property and both answers, because a divergence here is
// never a flaky test — it is one projection disturbing another's collection.
func AssertObservationalEquivalence(
	t ScenarioReporter,
	events []event.Event,
	baseline, interleaved projection.Projection,
	probes ...EquivalenceProbe,
) {
	t.Helper()

	feedEquivalenceEvents(t, baseline, events)
	feedEquivalenceEvents(t, interleaved, events)

	for i, probe := range probes {
		want, wantErr := probe.Baseline()
		got, gotErr := probe.Interleaved()

		switch {
		case wantErr != nil:
			t.Fatalf("observational equivalence: probe %d baseline failed: %v", i, wantErr)

		case gotErr != nil:
			t.Fatalf("observational equivalence: probe %d interleaved failed: %v", i, gotErr)

		case !reflect.DeepEqual(want, got):
			t.Fatalf(
				"observational equivalence violated at probe %d: projection answers diverged under interleaving\n"+
					"  baseline:    %#v\n"+
					"  interleaved: %#v",
				i,
				want,
				got,
			)
		}
	}
}

// feedEquivalenceEvents drives a projection through the event sequence,
// failing on the first handler error. Unlike GivenProjection it takes the
// minimal ScenarioReporter, so property-based runtimes can use the
// equivalence assertions without implementing full testing.TB.
func feedEquivalenceEvents(
	t ScenarioReporter,
	proj projection.Projection,
	events []event.Event,
) {
	t.Helper()

	ctx := context.Background()

	for _, evt := range events {
		if err := proj.Handle(ctx, evt); err != nil {
			t.Fatalf("projection %s: handle %s: %v", proj.Name(), evt.Type(), err)
		}
	}
}
