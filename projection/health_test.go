package projection_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/projection"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestRunner_HealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	bus, checkpoint := newTestBusAndCheckpoint(t)

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	require.NoError(t, err)

	err = runner.HealthCheck(context.Background())
	require.NoError(t, err)
}

func TestRunner_HealthCheck_DetailedHealth(t *testing.T) {
	t.Parallel()

	bus, checkpoint := newTestBusAndCheckpoint(t)

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	require.NoError(t, err)

	err = runner.Register(event.NewProjection("test-proj", testhelpers.NoopEventHandler(), nil))
	require.NoError(t, err)

	status := runner.DetailedHealthCheck(context.Background())
	require.True(t, status.Healthy)
	require.Len(t, status.Projections, 1)
	require.Equal(t, "test-proj", status.Projections[0].Name)
	require.True(t, status.Projections[0].Healthy)
}

func TestRunner_RegisteredProjections(t *testing.T) {
	t.Parallel()

	bus, checkpoint := newTestBusAndCheckpoint(t)

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	require.NoError(t, err)

	err = runner.Register(event.NewProjection("proj-a", testhelpers.NoopEventHandler(), nil))
	require.NoError(t, err)

	err = runner.Register(event.NewProjection("proj-b", testhelpers.NoopEventHandler(), nil))
	require.NoError(t, err)

	names := runner.RegisteredProjections()
	require.Len(t, names, 2)
	require.Contains(t, names, "proj-a")
	require.Contains(t, names, "proj-b")
}

func TestHealthCheckAll_AllHealthy(t *testing.T) {
	t.Parallel()

	bus, checkpoint := newTestBusAndCheckpoint(t)

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	require.NoError(t, err)

	err = projection.HealthCheckAll(context.Background(), runner)
	require.NoError(t, err)
}

func TestHealthCheckAll_NilChecker(t *testing.T) {
	t.Parallel()

	err := projection.HealthCheckAll(context.Background())
	require.NoError(t, err)
}
