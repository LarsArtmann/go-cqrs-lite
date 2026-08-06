package benchkit

import (
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// FormatDuration rounds a duration to a readable precision bucket and returns
// its string form. It eliminates sub-microsecond noise so that a p99 of
// 1.234ms is shown as "1.234ms" but 1.00004s becomes "1s".
func FormatDuration(d time.Duration) string {
	return roundDuration(d).String()
}

// FormatBytes formats a byte count as a human-readable IEC string (e.g. "1.2 MiB").
func FormatBytes(b uint64) string {
	return formatBytes(b)
}

// FormatFloat formats a float using SI prefixes with one decimal digit
// (e.g. 1500 → "1.5 k").
func FormatFloat(f float64) string {
	return formatFloat(f)
}

// FormatInt formats an integer with comma thousands separators (e.g. 10000 → "10,000").
func FormatInt(n int) string {
	return formatInt(n)
}

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
