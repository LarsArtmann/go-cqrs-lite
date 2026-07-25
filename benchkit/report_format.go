package benchkit

import (
	"fmt"
	"sort"
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
	switch {
	case f >= 1_000_000:
		return fmt.Sprintf("%.1fM", f/1_000_000)
	case f >= 1_000:
		return fmt.Sprintf("%.1fK", f/1_000)
	default:
		return fmt.Sprintf("%.0f", f)
	}
}

func formatBytes(b uint64) string {
	const (
		kib = 1024
		mib = 1024 * kib
		gib = 1024 * mib
	)

	switch {
	case b >= gib:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gib))
	case b >= mib:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mib))
	case b >= kib:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kib))
	default:
		return fmt.Sprintf("%d B", b)
	}
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
