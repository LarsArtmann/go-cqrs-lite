//go:build unix

package benchkit

import "syscall"

// cpuTimeProc reads process CPU time (user + sys) via getrusage, which has
// microsecond resolution on most Unix systems. This fixes the "n/a" CPU
// issue for fast benchmarks that completed within a single 10ms clock tick
// when the old /proc/self/stat approach was used.
func cpuTimeProc() uint64 {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}

	userNs := uint64(ru.Utime.Sec)*1e9 + uint64(ru.Utime.Usec)*1e3
	sysNs := uint64(ru.Stime.Sec)*1e9 + uint64(ru.Stime.Usec)*1e3

	return userNs + sysNs
}
