package benchkit

import (
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// roundDuration snaps a latency to a readable precision bucket so that a
// reported p99 of, say, 1.234ms is shown as "1.234ms" but 1.00004s becomes
// "1s" instead of carrying meaningless sub-microsecond noise.
func roundDuration(d time.Duration) time.Duration {
	switch {
	case d < time.Microsecond:
		return d.Round(time.Nanosecond)
	case d < time.Millisecond:
		return d.Round(100 * time.Nanosecond)
	case d < time.Second:
		return d.Round(time.Microsecond)
	default:
		return d.Round(time.Millisecond)
	}
}

func formatInt(n int) string {
	return humanize.Comma(int64(n))
}

func formatFloat(f float64) string {
	return strings.TrimSpace(humanize.SIWithDigits(f, 1, ""))
}

func formatBytes(b uint64) string {
	return humanize.IBytes(b)
}

func formatCPUDuration(ns uint64) string {
	if ns == 0 {
		return "n/a"
	}

	return roundDuration(time.Duration(ns)).String()
}

func sortedKeys(m map[string]*Result) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
