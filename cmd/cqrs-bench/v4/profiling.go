package main

import (
	"os"
	"runtime/pprof"
	"slices"
)

// startProfiling wires the --cpuprofile/--memprofile flags onto a handler:
// CPU profiling starts NOW and the returned stop func (intended for
// `defer startProfiling(...)()`) stops it and writes the heap profile when
// the handler returns — matching the defers this extracted.
func startProfiling(cpuProfile, memProfile string) func() {
	var teardown []func()

	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			fatalf("create cpu profile: %v", err)
		}

		if err := pprof.StartCPUProfile(f); err != nil {
			fatalf("start cpu profile: %v", err)
		}

		teardown = append(teardown,
			func() { _ = f.Close() },
			pprof.StopCPUProfile,
		)
	}

	if memProfile != "" {
		teardown = append(teardown, func() {
			f, err := os.Create(memProfile)
			if err != nil {
				fatalf("create mem profile: %v", err)
			}

			defer f.Close()

			_ = pprof.WriteHeapProfile(f)
		})
	}

	return func() {
		// Reverse order — the same LIFO a defer stack would give.
		for _, fn := range slices.Backward(teardown) {
			fn()
		}
	}
}
