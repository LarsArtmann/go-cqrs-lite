package catalog

import (
	"reflect"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// regressionPayload has no declared parameters, so the cached schema starts
// with an empty Parameters slice and any leakage is observable.
type regressionPayload struct {
	Name string `json:"name"`
}

// TestNewMessageBuilder_ConcurrentOptionsDoNotMutateSharedSchema guards the
// clone-before-configure fix: FromReflect returns the SHARED cached schema
// for a type, and WithParam appends to it. Without the clone in
// newMessageBuilder, concurrent builders race on the cached slice and leak
// parameters into each other (and into every later build of the same type).
// Run with -race.
func TestNewMessageBuilder_ConcurrentOptionsDoNotMutateSharedSchema(t *testing.T) {
	rt := reflect.TypeFor[regressionPayload]()

	base := schema.FromReflect(rt)
	baseParams := len(base.Parameters)

	const workers = 8

	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			cfg := newMessageBuilder[regressionPayload](
				QueryMessage, "test.concurrent.qry", Receives,
				[]MessageOption{WithParam("since", "query", "test param", false)},
			)

			builder, ok := cfg.(*messageBuilder)
			if !ok {
				t.Error("expected *messageBuilder")

				return
			}

			if got := len(builder.schema.Parameters); got != 1 {
				t.Errorf("builder schema params = %d, want exactly 1", got)
			}
		}()
	}

	wg.Wait()

	if got := len(base.Parameters); got != baseParams {
		t.Errorf("cached schema mutated: Parameters = %d, want %d", got, baseParams)
	}
}

// TestSchemaClone_IsolatesMutableContainers verifies Clone copies the
// mutable containers so mutating the clone never touches the original.
func TestSchemaClone_IsolatesMutableContainers(t *testing.T) {
	original := schema.FromReflect(reflect.TypeFor[regressionPayload]())

	wantParams := len(original.Parameters)
	wantRequired := len(original.Required)

	clone := original.Clone()
	if clone == original {
		t.Fatal("Clone must return a new pointer")
	}

	clone.Parameters = append(clone.Parameters, Parameter{Name: "p1"})
	clone.Required = append(clone.Required, "name")

	if got := len(original.Parameters); got != wantParams {
		t.Errorf("original Parameters leaked from clone: %d, want %d", got, wantParams)
	}

	if got := len(original.Required); got != wantRequired {
		t.Errorf("original Required leaked from clone: %d, want %d", got, wantRequired)
	}
}
