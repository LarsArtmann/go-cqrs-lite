package bench

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	stackmemory "github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

// full_pipeline_test.go — THE #1 gap: measures the complete CQRS pipeline
// end-to-end: create event → save to store → project to read model → query.
// This is the benchmark that answers "how fast is my CQRS system?"

// ─── Backend Factories ───

type pipelineBackend struct {
	name   string
	create func(b testing.TB) (*stack.Bundle, func())
}

func memoryBackend() pipelineBackend {
	return pipelineBackend{
		name: "memory",
		create: func(b testing.TB) (*stack.Bundle, func()) {
			bundle, err := stackmemory.New()
			if err != nil {
				b.Fatal(err)
			}
			return bundle, func() { _ = bundle.Close() }
		},
	}
}

func sqliteBackend() pipelineBackend {
	return pipelineBackend{
		name: "sqlite",
		create: func(b testing.TB) (*stack.Bundle, func()) {
			dir := b.TempDir()
			bundle, err := sqlite.New(
				dir+"/pipeline.db",
				sqlite.WithPragmas(sqlopt.WithOptimizations()),
			)
			if err != nil {
				b.Fatal(err)
			}
			return bundle, func() { _ = bundle.Close() }
		},
	}
}

func pebbleBackend() pipelineBackend {
	return pipelineBackend{
		name: "pebble",
		create: func(b testing.TB) (*stack.Bundle, func()) {
			dir := b.TempDir()
			p, err := pebble.New(dir)
			if err != nil {
				b.Fatal(err)
			}
			return p.Bundle, func() { _ = p.Close() }
		},
	}
}

func allPipelineBackends() []pipelineBackend {
	return []pipelineBackend{memoryBackend(), sqliteBackend(), pebbleBackend()}
}

// ─── Pipeline Setup Helper ───

type pipelineSetup struct {
	bundle      *stack.Bundle
	orderRM     *kv.TypedStore[OrderView, id.StreamID]
	taskRM      *kv.TypedStore[TaskView, id.StreamID]
	userRM      *kv.TypedStore[UserView, id.StreamID]
	project     func(context.Context, event.Event) error
	store       event.Store
	ctx         context.Context
}

func newPipelineSetup(b testing.TB, backend pipelineBackend) *pipelineSetup {
	bundle, cleanup := backend.create(b)
	b.Cleanup(cleanup)

	store, ok := bundle.EventStore()
	if !ok {
		b.Fatal("bundle has no event store")
	}

	var rmBackend kv.Store = kv.NewMemStore()
	if bundle.ReadModels != nil {
		rmBackend = bundle.ReadModels
	}

	orderRM := kv.NewTypedStore[OrderView, id.StreamID](rmBackend)
	taskRM := kv.NewTypedStore[TaskView, id.StreamID](rmBackend)
	userRM := kv.NewTypedStore[UserView, id.StreamID](rmBackend)

	project := newMultiDomainProjection(orderRM, taskRM, userRM)

	return &pipelineSetup{
		bundle:  bundle,
		orderRM: orderRM,
		taskRM:  taskRM,
		userRM:  userRM,
		project: project,
		store:   store,
		ctx:     context.Background(),
	}
}

func (p *pipelineSetup) saveAndProject(evt event.Event, ref id.StreamRef) error {
	if err := p.store.AppendBatch(p.ctx, ref, []event.Event{evt}); err != nil {
		return err
	}
	return p.project(p.ctx, evt)
}

// ─── A3: Full Pipeline (Memory, Single-Goroutine) ───

// BenchmarkFullPipeline_Memory measures the complete pipeline:
// create event → save → project to read model → query result.
// Single-goroutine, memory backend, round-robin across 3 domains.
func BenchmarkFullPipeline_Memory(b *testing.B) {
	ps := newPipelineSetup(b, memoryBackend())
	ctx := context.Background()

	domains := []string{"order", "task", "user"}

	b.ResetTimer()

	for b.Loop() {
		i := int(b.N)
		domain := domains[i%3]
		streamID := id.NewStreamID()
		ref := id.NewStreamRef(id.StreamType(domain), streamID)

		switch domain {
		case "order":
			evt, err := event.New(
				"order.created", streamID, "order", event.Version(1),
				OrderCreatedEvt{
					ID: streamID.String(), Customer: "cus-001",
					Amount: 5000, Product: "prod-001", At: time.Now(),
				},
			)
			if err != nil {
				b.Fatal(err)
			}
			if err := ps.saveAndProject(evt, ref); err != nil {
				b.Fatal(err)
			}
			view, err := ps.orderRM.Get(ctx, streamID)
			if err != nil || view == nil {
				b.Fatal("order read model not found")
			}

		case "task":
			evt, err := event.New(
				"task.created", streamID, "task", event.Version(1),
				TaskCreatedEvt{
					ID: streamID.String(), Title: "Write tests",
					Owner: "alice", Status: "open",
				},
			)
			if err != nil {
				b.Fatal(err)
			}
			if err := ps.saveAndProject(evt, ref); err != nil {
				b.Fatal(err)
			}
			view, err := ps.taskRM.Get(ctx, streamID)
			if err != nil || view == nil {
				b.Fatal("task read model not found")
			}

		case "user":
			evt, err := event.New(
				"user.registered", streamID, "user", event.Version(1),
				UserRegisteredEvt{
					ID: streamID.String(), Email: "user@test.com", Name: "Alice",
				},
			)
			if err != nil {
				b.Fatal(err)
			}
			if err := ps.saveAndProject(evt, ref); err != nil {
				b.Fatal(err)
			}
			view, err := ps.userRM.Get(ctx, streamID)
			if err != nil || view == nil {
				b.Fatal("user read model not found")
			}
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pipelines/sec")
}

// ─── A4: Concurrent Full Pipeline ───

// BenchmarkFullPipeline_Concurrent measures pipeline throughput under
// concurrent load. 8 goroutines each run the full pipeline.
func BenchmarkFullPipeline_Concurrent(b *testing.B) {
	ps := newPipelineSetup(b, memoryBackend())
	ctx := context.Background()
	concurrency := 8

	b.ResetTimer()

	for b.Loop() {
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for w := range concurrency {
			go func(worker int) {
				defer wg.Done()
				for range b.N / concurrency {
					streamID := id.NewStreamID()
					ref := id.NewStreamRef("order", streamID)

					evt, err := event.New(
						"order.created", streamID, "order", event.Version(1),
						OrderCreatedEvt{
							ID: streamID.String(), Customer: "cus-001",
							Amount: 5000, Product: "prod-001", At: time.Now(),
						},
					)
					if err != nil {
						b.Error(err)
						return
					}
					if err := ps.saveAndProject(evt, ref); err != nil {
						b.Error(err)
						return
					}
					if _, err := ps.orderRM.Get(ctx, streamID); err != nil {
						b.Error(err)
						return
					}
				}
			}(w)
		}
		wg.Wait()
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pipelines/sec")
}

// ─── A5: Parameterize by Backend ───

// BenchmarkFullPipeline_Backends runs the pipeline against memory, SQLite, and
// Pebble backends. Compare to see the storage overhead in the full pipeline.
func BenchmarkFullPipeline_Backends(b *testing.B) {
	for _, backend := range allPipelineBackends() {
		b.Run(backend.name, func(b *testing.B) {
			ps := newPipelineSetup(b, backend)
			ctx := context.Background()

			b.ResetTimer()

			for b.Loop() {
				streamID := id.NewStreamID()
				ref := id.NewStreamRef("order", streamID)

				evt, err := event.New(
					"order.created", streamID, "order", event.Version(1),
					OrderCreatedEvt{
						ID: streamID.String(), Customer: "cus-001",
						Amount: 5000, Product: "prod-001", At: time.Now(),
					},
				)
				if err != nil {
					b.Fatal(err)
				}
				if err := ps.saveAndProject(evt, ref); err != nil {
					b.Fatal(err)
				}
				if _, err := ps.orderRM.Get(ctx, streamID); err != nil {
					b.Fatal(err)
				}
			}

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pipelines/sec")
		})
	}
}

// ─── A7: Middleware Stack Variants ───

// BenchmarkFullPipeline_MiddlewareStacks measures the overhead of middleware
// chains in the pipeline. Production systems run logging + recovery + retry +
// OTel simultaneously.
func BenchmarkFullPipeline_MiddlewareStacks(b *testing.B) {
	// Baseline: no middleware.
	b.Run("no-middleware", func(b *testing.B) {
		runPipelineWithOverhead(b, func(evt event.Event, ref id.StreamRef, ps *pipelineSetup) error {
			return ps.saveAndProject(evt, ref)
		})
	})

	// Simulated middleware overhead: measure the cost of wrapping each event
	// through a chain of functions (representing logging, recovery, etc.).
	b.Run("5-chain-simulation", func(b *testing.B) {
		runPipelineWithOverhead(b, func(evt event.Event, ref id.StreamRef, ps *pipelineSetup) error {
			// Simulate 5 middleware layers wrapping the save+project call.
			chain := func(e event.Event) error { return ps.saveAndProject(e, ref) }
			for range 5 {
				inner := chain
				chain = func(e event.Event) error { return inner(e) }
			}
			return chain(evt)
		})
	})
}

func runPipelineWithOverhead(b *testing.B, run func(event.Event, id.StreamRef, *pipelineSetup) error) {
	b.Helper()
	ps := newPipelineSetup(b, memoryBackend())

	b.ResetTimer()

	for b.Loop() {
		streamID := id.NewStreamID()
		ref := id.NewStreamRef("order", streamID)

		evt, err := event.New(
			"order.created", streamID, "order", event.Version(1),
			OrderCreatedEvt{
				ID: streamID.String(), Customer: "cus-001",
				Amount: 5000, Product: "prod-001", At: time.Now(),
			},
		)
		if err != nil {
			b.Fatal(err)
		}
		if err := run(evt, ref, ps); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pipelines/sec")
}

// ─── A8: Payload Size Variants ───

// BenchmarkFullPipeline_PayloadSizes measures how payload size affects the
// full pipeline. Real events range from 64B status changes to 10KB embedded
// collections.
func BenchmarkFullPipeline_PayloadSizes(b *testing.B) {
	for _, size := range []int{64, 256, 1024, 10240} {
		b.Run(fmt.Sprintf("size=%dB", size), func(b *testing.B) {
			ps := newPipelineSetup(b, memoryBackend())
			ctx := context.Background()

			// Create a payload of the target size.
			payload := make([]byte, size)
			for i := range payload {
				payload[i] = 'x'
			}

			b.ResetTimer()

			for b.Loop() {
				streamID := id.NewStreamID()
				ref := id.NewStreamRef("order", streamID)

				evt, err := event.NewEvent(
					"order.created", streamID, "order", event.Version(1),
					payload,
				)
				if err != nil {
					b.Fatal(err)
				}
				if err := ps.store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
					b.Fatal(err)
				}
			}

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
			b.ReportMetric(float64(size), "payload_bytes")
		})
	}
}

// ─── A9: Concurrency Variants ───

// BenchmarkFullPipeline_Concurrency measures how concurrency affects pipeline
// throughput. 1, 4, 8, 16 workers each running the full pipeline.
func BenchmarkFullPipeline_Concurrency(b *testing.B) {
	for _, workers := range []int{1, 4, 8, 16} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			ps := newPipelineSetup(b, memoryBackend())

			b.ResetTimer()

			for b.Loop() {
				var wg sync.WaitGroup
				wg.Add(workers)

				for range workers {
					go func() {
						defer wg.Done()
						for range b.N / workers {
							streamID := id.NewStreamID()
							ref := id.NewStreamRef("order", streamID)

							evt, err := event.New(
								"order.created", streamID, "order", event.Version(1),
								OrderCreatedEvt{
									ID: streamID.String(), Customer: "cus-001",
									Amount: 5000, Product: "prod-001", At: time.Now(),
								},
							)
							if err != nil {
								b.Error(err)
								return
							}
							if err := ps.saveAndProject(evt, ref); err != nil {
								b.Error(err)
								return
							}
						}
					}()
				}
				wg.Wait()
			}

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pipelines/sec")
		})
	}
}

// suppress unused import warning
var _ = stackmemory.New
