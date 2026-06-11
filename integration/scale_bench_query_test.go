package integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// ---------------------------------------------------------------------------
// 5. Thousand Queries — dispatch with many registered handlers
// ---------------------------------------------------------------------------

func BenchmarkScale_QueryDispatch_1000Handlers(b *testing.B) {
	b.ReportAllocs()

	dispatcher := query.NewDispatcher()
	b.Cleanup(func() { _ = dispatcher.Close() })

	handlerCount := 1000

	for i := range handlerCount {
		err := dispatcher.Register(
			query.Type(fmt.Sprintf("query.%d", i)),
			benchNoopQueryHandler,
		)
		if err != nil {
			b.Fatalf("register: %v", err)
		}
	}

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		for i := range handlerCount {
			q := mustNewQuery(query.Type(fmt.Sprintf("query.%d", i)))
			_, err := dispatcher.Dispatch(ctx, q)
			if err != nil {
				b.Fatalf("dispatch: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*handlerCount)/b.Elapsed().Seconds(), "queries/sec")
}

func BenchmarkScale_QueryDispatch_PaginatedResults(b *testing.B) {
	b.ReportAllocs()

	dispatcher := query.NewDispatcher()
	b.Cleanup(func() { _ = dispatcher.Close() })

	pageSizes := []uint{10, 50, 100, 500}
	resultData := make([]string, 1000)
	for i := range resultData {
		resultData[i] = fmt.Sprintf("item-%d", i)
	}

	for _, ps := range pageSizes {
		err := dispatcher.Register(
			query.Type(fmt.Sprintf("list.page%d", ps)),
			func(_ context.Context, _ query.Query) (any, error) {
				p := query.NewPagination(1, ps)
				end := min(int(ps), len(resultData))

				return query.NewPaginatedResult(
					resultData[:end],
					uint(len(resultData)),
					p,
				), nil
			},
		)
		if err != nil {
			b.Fatalf("register: %v", err)
		}
	}

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		for _, ps := range pageSizes {
			q := mustNewQuery(query.Type(fmt.Sprintf("list.page%d", ps)))
			_, err := dispatcher.Dispatch(ctx, q)
			if err != nil {
				b.Fatalf("Dispatch: %v", err)
			}
		}
	}

	b.ReportMetric(float64(b.N*len(pageSizes))/b.Elapsed().Seconds(), "queries/sec")
}
