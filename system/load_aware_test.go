package system_test

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// loadScaledDeadline widens a catch-up-poll deadline by the ambient load
// factor. Projection catch-up is eventual by construction: the poll loop is
// timing-proof, but a fixed wall-clock deadline expires under a loaded host
// (shared box, parallel sessions) even though the projection is healthy.
// Scaling the deadline by 1-minute load average / GOMAXPROCS keeps genuine
// regressions failing (a broken projection never arrives) while surviving
// load spikes. Capped at 8x so a true hang still dies inside the go test
// per-package timeout.
//
// Local copy (not testutil): mirrors benchkit's loadScaledCeiling.
func loadScaledDeadline(base time.Duration) time.Time {
	factor := 1.0

	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		if fields := strings.Fields(string(data)); len(fields) > 0 {
			if load1, err := strconv.ParseFloat(fields[0], 64); err == nil {
				cores := float64(runtime.GOMAXPROCS(0))
				if cores < 1 {
					cores = 1
				}

				factor = load1 / cores
				if factor < 1 {
					factor = 1
				}
				if factor > 8 {
					factor = 8
				}
			}
		}
	}

	return time.Now().Add(time.Duration(factor * float64(base)))
}
