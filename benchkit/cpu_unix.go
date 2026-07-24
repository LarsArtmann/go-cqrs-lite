//go:build unix

package benchkit

import "syscall"

// cpuTimeProc reads process CPU time (user + sys) via getrusage, which has
// microsecond resolution on most Unix systems. This fixes the "n/a" CPU
// issue for fast benchmarks that completed within a single 10ms clock tick
// when the old /proc/self/stat approach was used.
func cpuTimeProc() uint64 {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return 0
	}

	userNs := uint64(usage.Utime.Sec)*1e9 + uint64(usage.Utime.Usec)*1e3
	sysNs := uint64(usage.Stime.Sec)*1e9 + uint64(usage.Stime.Usec)*1e3

	return userNs + sysNs
}
