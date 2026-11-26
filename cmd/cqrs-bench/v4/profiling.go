package main

import (
	"fmt"
	"os"
	"slices"
	"runtime/pprof"
)

// startProfiling wires the --cpuprofile/--memprofile flags onto a command:
// CPU profiling starts now and the returned stop func (meant for
// `defer stop()`) stops it and writes the heap profile when the command
// returns. An error means profiling could not start and nothing is running.
func startProfiling(cpuProfile, memProfile string) (func(), error) {
	var (
		teardown []func()
		err      error
	)

	if cpuProfile != "" {
		teardown, err = startCPUProfile(cpuProfile, teardown)
		if err != nil {
			return nil, err
		}
	}

	if memProfile != "" {
		teardown = append(teardown, func() { writeHeapProfile(memProfile) })
	}

	return func() {
		// Reverse order: the LIFO a defer stack would give. The heap profile
		// is written first, then the CPU profile is stopped, then its file
		// is closed (a close before the stop leaves an EMPTY CPU profile).
		for _, fn := range slices.Backward(teardown) {
			fn()
		}
	}, nil
}

// startCPUProfile starts CPU profiling into cpuProfile and registers the
// matching teardown on teardown.
func startCPUProfile(cpuProfile string, teardown []func()) ([]func(), error) {
	file, err := os.Create(cpuProfile)
	if err != nil {
		return nil, fmt.Errorf("%w: --cpuprofile %s: %w", errFlagValue, cpuProfile, err)
	}

	if err := pprof.StartCPUProfile(file); err != nil {
		_ = file.Close()

		return nil, fmt.Errorf("start CPU profile: %w", err)
	}

	return append(teardown,
		func() { _ = file.Close() },
		pprof.StopCPUProfile,
	), nil
}

// writeHeapProfile writes the heap profile to path. It runs during teardown,
// when nothing can act on the failure, so it is reported as a warning.
func writeHeapProfile(path string) {
	file, err := os.Create(path)
	if err != nil {
		warnf("--memprofile %s: %v", path, err)

		return
	}

	defer func() { _ = file.Close() }()

	if err := pprof.WriteHeapProfile(file); err != nil {
		warnf("--memprofile %s: %v", path, err)
	}
}
