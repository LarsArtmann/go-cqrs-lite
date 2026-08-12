package pebbleengine_test

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// adt_matrix_test.go runs the full 7-ADT test matrix across the Pebble
// engine and the memory engine, asserting cross-engine parity. The
// metaengine module's own adt_matrix_test.go covers memory↔sqlite parity;
// this test covers memory↔pebble parity. By transitivity, all three
// engines produce identical results.

func TestPebbleADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "pebble",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				eng, err := pebbleengine.NewPebbleEngine("")
				gomega.NewWithT(t).Expect(err).NotTo(gomega.HaveOccurred())

				return eng
			},
		},
	})
}
