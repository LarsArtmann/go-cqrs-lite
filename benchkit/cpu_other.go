//go:build !unix

package benchkit

// cpuTimeProc returns 0 on non-Unix platforms (Windows, js/wasm) where
// syscall.Getrusage is unavailable. CPU metrics will display as "n/a".
func cpuTimeProc() uint64 { return 0 }
