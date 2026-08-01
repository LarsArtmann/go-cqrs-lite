package benchkit

import (
	"context"
	"runtime"
	"sync"
)

func (r *runner) finalizeResult(peakMem uint64, baseline memSnapshot) {
	r.result.Memory = ResourceStats{
		Before: baseline.heapAlloc,
		After:  peakMem,
	}

	if peakMem > baseline.heapAlloc {
		r.result.Memory.Delta = peakMem - baseline.heapAlloc
	}

	endCPU := cpuTime()

	r.result.CPU = ResourceStats{
		Before: r.startCPU,
		After:  endCPU,
	}

	if endCPU > r.startCPU {
		r.result.CPU.Delta = endCPU - r.startCPU
	}

	r.result.Disk.EventBytes = int64(r.result.TotalEvents) * int64(r.gen.MeanSize())

	if r.result.Disk.DatabaseBytes > 0 {
		r.result.Disk.OverheadBytes = r.result.Disk.DatabaseBytes - r.result.Disk.EventBytes
		r.result.Disk.OverheadPct = float64(r.result.Disk.OverheadBytes) /
			float64(r.result.Disk.DatabaseBytes) * 100

		if r.result.Disk.EventBytes > 0 {
			r.result.Disk.WriteAmplification =
				float64(r.result.Disk.DatabaseBytes) / float64(r.result.Disk.EventBytes)
		}
	}

	// GC pause metrics — capture final MemStats and compute deltas from baseline.
	var finalStats runtime.MemStats
	runtime.ReadMemStats(&finalStats)

	gc := computeGCMetrics(r.baselineMemStats, finalStats)
	r.result.GCCount = gc.Count
	r.result.GCTotalPause = gc.TotalPause
	r.result.GCMaxPause = gc.MaxPause
	r.result.GCMeanPause = gc.MeanPause

	// Allocation deltas — total heap allocations during the benchmark.
	if finalStats.Mallocs >= r.baselineMemStats.Mallocs {
		r.result.AllocCount = finalStats.Mallocs - r.baselineMemStats.Mallocs
	}

	if finalStats.TotalAlloc >= r.baselineMemStats.TotalAlloc {
		r.result.AllocBytes = finalStats.TotalAlloc - r.baselineMemStats.TotalAlloc
	}
}

// runConcurrent runs op for each index in [0, total) using at most
// concurrency goroutines. Returns the first error encountered.
func runConcurrent(
	ctx context.Context,
	total, concurrency int,
	op func(ctx context.Context, idx int) error,
) error {
	if concurrency <= 0 {
		concurrency = 1
	}

	if concurrency > total {
		concurrency = total
	}

	work := make(chan int)
	errCh := make(chan error, 1)

	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
	)

	for range concurrency {
		wg.Go(func() {
			for idx := range work {
				if err := op(cancelCtx, idx); err != nil {
					errOnce.Do(func() { errCh <- err })
					cancel()

					return
				}
			}
		})
	}

	go func() {
		defer close(work)

		for i := range total {
			select {
			case work <- i:
			case <-cancelCtx.Done():
				return
			}
		}
	}()

	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}
