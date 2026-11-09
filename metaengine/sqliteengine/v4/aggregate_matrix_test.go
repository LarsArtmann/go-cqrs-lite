package sqliteengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

func TestSQLiteAggregateMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunAggregateMatrix(t, []adttest.Factory{
		{
			Name: "sqlite",
			Create: func(t *testing.T) metaengine.Engine {
				eng, cleanup := newAggSQLiteEngine(t)
				t.Cleanup(cleanup)

				return eng
			},
		},
	})
}
