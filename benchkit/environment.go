package benchkit

import (
	"fmt"
	"runtime"
)

// defaultLoadWarnRatio is the load-average-per-CPU ratio above which a
// machine counts as oversubscribed when Config.LoadWarnThreshold is unset.
const defaultLoadWarnRatio = 1.0

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

// loadWarnRatio returns the configured oversubscription ratio, defaulting to
// [defaultLoadWarnRatio] for the zero value.
func (r *runner) loadWarnRatio() float64 {
	if r.config.LoadWarnThreshold > 0 {
		return r.config.LoadWarnThreshold
	}

	return defaultLoadWarnRatio
}

// oversubscribed reports whether a load average crosses the configured
// oversubscription line for this machine.
func (r *runner) oversubscribed(load float64) bool {
	cpus := r.result.Environment.NumCPU
	if cpus <= 0 {
		cpus = runtime.NumCPU()
	}

	return load > r.loadWarnRatio()*float64(cpus)
}

// warnIfOversubscribed flags a machine whose 1-minute load average already
// exceeded its CPU count when the run started: every latency in that run then
// carries scheduler wait on top of backend cost. The Environment.LoadAvg1 field
// records the number; this makes it impossible to miss.
func (r *runner) warnIfOversubscribed() {
	env := r.result.Environment
	if r.oversubscribed(env.LoadAvg1) {
		r.warn(fmt.Sprintf(
			"machine oversubscribed at run start: 1-min load average %.1f exceeds %d CPUs, so latencies include scheduler wait",
			env.LoadAvg1,
			env.NumCPU,
		))
	}
}

// recordLoadEnd samples the load average when the run finishes and warns when
// the machine BECAME oversubscribed mid-run. A quiet start with a loud end
// means the second half of the run measured a different machine than the
// first — exactly the pollution a first-sample-only check cannot see.
func (r *runner) recordLoadEnd() {
	r.result.Environment.LoadAvg1End = detectLoadAvg1()

	start := r.result.Environment.LoadAvg1
	end := r.result.Environment.LoadAvg1End

	if end > 0 && start > 0 &&
		r.oversubscribed(end) && !r.oversubscribed(start) {
		r.warn(fmt.Sprintf(
			"machine became oversubscribed during the run: 1-min load average drifted %.1f → %.1f (%d CPUs); later phases are polluted by scheduler wait",
			start,
			end,
			r.result.Environment.NumCPU,
		))
	}
}
