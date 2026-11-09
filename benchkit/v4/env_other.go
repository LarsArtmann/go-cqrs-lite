//go:build !linux

package benchkit

// detectCPUModel returns empty on non-Linux platforms.
func detectCPUModel() string { return "" }

// detectTotalRAM returns 0 on non-Linux platforms.
func detectTotalRAM() uint64 { return 0 }

// detectLoadAvg1 returns 0 on non-Linux platforms.
func detectLoadAvg1() float64 { return 0 }
