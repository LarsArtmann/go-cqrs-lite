package benchkit

import (
	"fmt"
	"runtime"
)

// captureEnvironment records the machine and runtime context a result was
// produced in, so comparisons across machines, Go versions, or CPU limits can
// be judged honestly rather than by assuming the numbers are comparable.
func captureEnvironment() Environment {
	return Environment{
		GoVersion:     runtime.Version(),
		NumCPU:        runtime.NumCPU(),
		GOMAXPROCS:    runtime.GOMAXPROCS(0),
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		CPUModel:      detectCPUModel(),
		TotalRAMBytes: detectTotalRAM(),
		LoadAvg1:      detectLoadAvg1(),
	}
}

// warnIfOversubscribed flags a machine whose 1-minute load average already
// exceeded its CPU count when the run started: every latency in that run then
// carries scheduler wait on top of backend cost. The Environment.LoadAvg1 field
// records the number; this makes it impossible to miss.
func (r *runner) warnIfOversubscribed() {
	env := r.result.Environment
	if env.LoadAvg1 > float64(env.NumCPU) {
		r.warn(fmt.Sprintf(
			"machine oversubscribed at run start: 1-min load average %.1f exceeds %d CPUs, so latencies include scheduler wait",
			env.LoadAvg1, env.NumCPU,
		))
	}
}
