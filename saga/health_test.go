package saga_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/saga"
	"github.com/stretchr/testify/require"
)

func TestRunner_HealthCheck_WithMemoryStore(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, nopDispatcher{})

	err := runner.HealthCheck(context.Background())
	require.NoError(t, err)
}

func TestRunner_RegisteredSagas(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, nopDispatcher{})

	err := runner.Register(testDefinition{sagaType: "order-saga"})
	require.NoError(t, err)

	err = runner.Register(testDefinition{sagaType: "payment-saga"})
	require.NoError(t, err)

	types := runner.RegisteredSagas()
	require.Len(t, types, 2)
	require.Contains(t, types, "order-saga")
	require.Contains(t, types, "payment-saga")
}

func TestRunner_HealthCheck_NilStore(t *testing.T) {
	t.Parallel()

	runner := saga.NewRunner(nil, nopDispatcher{})

	err := runner.HealthCheck(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "store is nil")
}
