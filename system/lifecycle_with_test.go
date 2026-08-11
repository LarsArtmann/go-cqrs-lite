package system_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystore "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestWithCommandLifecycle_ReturnsAllComponents(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := memorystore.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	cl := system.WithCommandLifecycle(store)

	g.Expect(cl.Recorder).NotTo(BeNil())
	g.Expect(cl.OuterMiddleware).NotTo(BeNil())
	g.Expect(cl.AttemptMiddleware).NotTo(BeNil())
	g.Expect(cl.Projections).To(HaveLen(4))
}

func TestWithCommandLifecycle_MiddlewareEmitsEvents(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	store := memorystore.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	cl := system.WithCommandLifecycle(store)

	cmd, err := command.New("create_user", id.NewStreamID())
	g.Expect(err).NotTo(HaveOccurred())

	handler := cl.OuterMiddleware(cl.AttemptMiddleware(
		func(_ context.Context, _ command.Command) error { return nil },
	))

	g.Expect(handler(context.Background(), cmd)).To(Succeed())

	ref := commandlifecycle.LifecycleStreamRef(cmd)
	events, loadErr := store.Load(context.Background(), ref)
	g.Expect(loadErr).NotTo(HaveOccurred())
	g.Expect(events).To(HaveLen(2))
	g.Expect(string(events[0].Type())).To(Equal("command.received"))
	g.Expect(string(events[1].Type())).To(Equal("command.completed"))
}
