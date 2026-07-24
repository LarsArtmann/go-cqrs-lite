// Package benchkit provides a factory-driven benchmarking suite for
// [stack.Bundle] presets — the performance equivalent of
// [stack/contracttest].
//
// A deployer provides a [Factory] (a function that returns a fresh
// *stack.Bundle), and benchkit runs a suite of realistic write, read,
// read-model, and projection workloads while collecting latency percentiles,
// throughput, memory deltas, and storage footprint.
//
// # Quick start
//
//	result, err := benchkit.Run(ctx, benchkit.Config{
//	    Profile:     benchkit.ProfileDev,
//	    PayloadSize: 256,
//	}, func() (*stack.Bundle, error) {
//	    return sqlite.New(filepath.Join(dir, "bench.db"))
//	})
//	if err != nil { ... }
//	benchkit.PrintReport(os.Stdout, result)
//
// # Cross-backend comparison
//
//	results, err := benchkit.Compare(ctx, config, map[string]benchkit.Factory{
//	    "memory": func() (*stack.Bundle, error) { return memory.New() },
//	    "sqlite": func() (*stack.Bundle, error) { return sqlite.New(":memory:") },
//	})
//	benchkit.PrintComparison(os.Stdout, results)
//
// The tool never imports a backend driver. Switching backends is a one-line
// factory change, mirroring the library's deployer-first philosophy.
//
// # Build tag requirement
//
// This package imports encoding/json/v2 and requires the build tag
// "goexperiment.jsonv2". Use:
//
//	go build -tags "goexperiment.jsonv2" ./benchkit/...
//	go test -tags "goexperiment.jsonv2" ./benchkit/...
//
// or set GOEXPERIMENT=jsonv2 in the environment.
package benchkit
