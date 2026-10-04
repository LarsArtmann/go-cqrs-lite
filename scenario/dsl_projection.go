package scenario

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// --- Projection DSL ---

// ProjectionScenario tests that a projection handles events without error.
type ProjectionScenario struct {
	proj     projection.Projection
	t        testing.TB
	errs     []error
	asserted bool
}

// GivenProjection creates a projection scenario and feeds it the given events.
// The testing.TB parameter accepts *testing.T and property-based testing
// runtimes (e.g. rapid.T) alike.
func GivenProjection(
	tb testing.TB,
	proj projection.Projection,
	events ...event.Event,
) *ProjectionScenario {
	tb.Helper()
	scenario := &ProjectionScenario{ //nolint:exhaustruct_v5 // errs populated below; asserted flips in Then*
		proj: proj,
		t:    tb,
	}
	tb.Cleanup(func() {
		if !scenario.asserted {
			tb.Errorf(
				"scenario: no Then* assertion ran — this test passes vacuously and " +
					"swallows every handler error; end the chain with " +
					"ThenNoError, ThenError, or ThenQueryResult",
			)
		}
	})

	ctx := context.Background()
	for _, evt := range events {
		if err := proj.Handle(ctx, evt); err != nil {
			scenario.errs = append(scenario.errs, errorfamily.Wrap(err, errorfamily.Classify(err),
				"scenario.projection.handle",
				fmt.Sprintf("event %s", evt.Type())))
		}
	}

	return scenario
}

// ThenNoError asserts the projection handled all events without error.
func (s *ProjectionScenario) ThenNoError() {
	s.t.Helper()
	s.asserted = true

	if len(s.errs) > 0 {
		for _, err := range s.errs {
			s.t.Errorf("projection %q: %v", s.proj.Name(), err)
		}

		s.t.FailNow()
	}
}

// ThenError asserts the projection returned at least one error.
func (s *ProjectionScenario) ThenError() {
	s.t.Helper()
	s.asserted = true

	if len(s.errs) == 0 {
		s.t.Fatalf("projection %q: expected at least one error, got none", s.proj.Name())
	}
}

// ThenQueryResult calls queryFn and asserts the result matches expected via
// reflect.DeepEqual. The function signature is generic (func() (any, error))
// so scenario does not depend on any specific query engine. Wrap metaengine's
// ExecuteTyped or any other query in a closure:
//
//	scenario.GivenProjection(t, adapter, evt1).
//	    ThenNoError().
//	    ThenQueryResult(
//	        func() (any, error) {
//	            return metaengine.ExecuteTyped[Input, Result](ctx, store, Input{})
//	        },
//	        Result{"count": 5},
//	    )
func (s *ProjectionScenario) ThenQueryResult(
	queryFn func() (any, error),
	expected any,
) *ProjectionScenario {
	s.t.Helper()
	s.asserted = true

	if len(s.errs) > 0 {
		s.t.Fatalf(
			"ThenQueryResult: cannot assert query result — projection had %d errors",
			len(s.errs),
		)
	}

	result, err := queryFn()
	if err != nil {
		s.t.Fatalf("ThenQueryResult: query returned error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		s.t.Fatalf("ThenQueryResult: expected %v, got %v", expected, result)
	}

	return s
}
