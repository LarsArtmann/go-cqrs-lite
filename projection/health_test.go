package projection_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/event/eventtest"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

func TestRunner_HealthCheck_Healthy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bus, checkpoint := newTestBusAndCheckpoint(t)

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	g.Expect(err).ToNot(HaveOccurred())

	err = runner.HealthCheck(context.Background())
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunner_HealthCheck_DetailedHealth(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bus, checkpoint := newTestBusAndCheckpoint(t)

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	g.Expect(err).ToNot(HaveOccurred())

	err = runner.Register(event.NewProjection("test-proj", eventtest.NoopEventHandler(), nil))
	g.Expect(err).ToNot(HaveOccurred())

	status := runner.DetailedHealthCheck(context.Background())
	g.Expect(status.Healthy).To(BeTrue())
	g.Expect(status.Projections).To(HaveLen(1))
	g.Expect(status.Projections[0].Name).To(Equal("test-proj"))
	g.Expect(status.Projections[0].Healthy).To(BeTrue())
}

func TestRunner_RegisteredProjections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bus, checkpoint := newTestBusAndCheckpoint(t)

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	g.Expect(err).ToNot(HaveOccurred())

	err = runner.Register(event.NewProjection("proj-a", eventtest.NoopEventHandler(), nil))
	g.Expect(err).ToNot(HaveOccurred())

	err = runner.Register(event.NewProjection("proj-b", eventtest.NoopEventHandler(), nil))
	g.Expect(err).ToNot(HaveOccurred())

	names := runner.RegisteredProjections()
	g.Expect(names).To(HaveLen(2))
	g.Expect(names).To(ContainElement("proj-a"))
	g.Expect(names).To(ContainElement("proj-b"))
}

func TestHealthCheckAll_AllHealthy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bus, checkpoint := newTestBusAndCheckpoint(t)

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	g.Expect(err).ToNot(HaveOccurred())

	err = projection.HealthCheckAll(context.Background(), runner)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestHealthCheckAll_NilChecker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := projection.HealthCheckAll(context.Background())
	g.Expect(err).ToNot(HaveOccurred())
}
