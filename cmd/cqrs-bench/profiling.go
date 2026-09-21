package main

import (
	"os"
	"runtime/pprof"
)

// startProfiling wires the --cpuprofile/--memprofile flags onto a handler:
// CPU profiling starts immediately and stops via defer; the heap profile is
// written in a defer so it captures the heap after the benchmark ran.
func startProfiling(cpuProfile, memProfile string) {
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			fatalf("create cpu profile: %v", err)
		}

		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			fatalf("start cpu profile: %v", err)
		}

		defer pprof.StopCPUProfile()
	}

	if memProfile != "" {
		defer func() {
			f, err := os.Create(memProfile)
			if err != nil {
				fatalf("create mem profile: %v", err)
			}

			defer f.Close()

			_ = pprof.WriteHeapProfile(f)
		}()
	}
}
