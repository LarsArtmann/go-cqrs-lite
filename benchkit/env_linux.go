//go:build linux

package benchkit

import (
	"os"
	"strconv"
	"strings"
)

// detectCPUModel reads the CPU model name from /proc/cpuinfo on Linux.
// Returns empty string when unavailable.
func detectCPUModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

// detectTotalRAM reads total system RAM from /proc/meminfo on Linux.
// Returns 0 when unavailable.
func detectTotalRAM() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)

			if len(parts) >= 2 {
				var kilobytes uint64

				for _, c := range parts[1] {
					if c < '0' || c > '9' {
						return 0
					}

					kilobytes = kilobytes*10 + uint64(c-'0')
				}

				return kilobytes * 1024
			}
		}
	}

	return 0
}

// detectLoadAvg1 reads the 1-minute system load average from /proc/loadavg.
// Returns 0 when unavailable. A load average above the CPU count means more
// runnable work existed than cores, so measured latencies include scheduler
// wait rather than pure backend cost.
func detectLoadAvg1() float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}

	load, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}

	return load
}
