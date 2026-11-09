package metaengine

import (
	"sync/atomic"
	"testing"
)

// BenchmarkWorkloadMeterContention measures the cost of concurrent read/write
// counter traffic on workloadMeter — the pattern produced by Store.Execute
// (IncRead) racing Store.Append/Save (IncWrite) across goroutines.
// Compare across code changes: cache-line separation between writeCount and
// readCount should show up as a per-op delta here.
func BenchmarkWorkloadMeterContention(b *testing.B) {
	meter := newWorkloadMeter()

	var role atomic.Int64

	b.RunParallel(func(pb *testing.PB) {
		if role.Add(1)%2 == 1 {
			for pb.Next() {
				meter.IncWrite()
			}
			return
		}

		for pb.Next() {
			meter.IncRead()
		}
	})

	if got := meter.readCount.Load() + meter.writeCount.Load(); got != int64(b.N) {
		b.Fatalf("lost updates: counted %d of %d", got, b.N)
	}
}
