package bbolt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v4/querytest"
)

func TestQueryStoreSuite(t *testing.T) {
	t.Parallel()

	querytest.RunStoreSuite(t, func(t *testing.T) querytest.StoreSuite {
		return newTestBackend(t).QueryStore()
	})
}
