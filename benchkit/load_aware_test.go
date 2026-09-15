package benchkit

import (
	"runtime"
	"time"
)

// art-dupl:accept local copy — benchkit is dep-budget-lean, testutil would exceed it
// loadScaledCeiling widens a wall-clock assertion ceiling by the ambient load
// factor. The benchkit timing tests assert "a Duration-bounded run returns
// promptly"; under a loaded host (shared box, parallel sessions) even trivial
// runs inflate 10x+, turning a fixed 30s ceiling into a coin flip.
//
// The factor is 1-minute load average divided by GOMAXPROCS (1.0 = idle,
// 4.0 = four runnable threads per core). The scaled ceiling keeps real hang
// detection meaningful while surviving load spikes; a true hang still dies
// at the go test per-package timeout, which is the structural backstop.
//
// Local copy (not testutil): benchkit is a lean-budget module — see
// AGENTS.md "Race-aware test thresholds".
func loadScaledCeiling(base time.Duration) time.Duration {
	return base * time.Duration(ambientLoadFactor())
}

// ambientLoadFactor returns max(1, load1/GOMAXPROCS), capped at 8, using the
// same /proc/loadavg reader as Environment.LoadAvg1 (detectLoadAvg1).
// Non-Linux hosts and unreadable files yield 0 → factor 1 (fixed ceilings,
// prior behavior).
func ambientLoadFactor() float64 {
	load1 := detectLoadAvg1()

	cores := float64(runtime.GOMAXPROCS(0))
	if cores < 1 {
		cores = 1
	}

	factor := load1 / cores
	if factor < 1 {
		return 1
	}
	if factor > 8 {
		return 8
	}

	return factor
}
