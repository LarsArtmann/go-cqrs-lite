package main

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/larsartmann/go-finding/pipeline"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func printLoadErrors(w io.Writer, errors []analyzer.PackageLoadError) {
	for _, le := range errors {
		if le.PkgPath != "" {
			_, _ = fmt.Fprintf(w, "  %s (%s):\n", le.Module, le.PkgPath)
		} else {
			_, _ = fmt.Fprintf(w, "  %s:\n", le.Module)
		}
		for _, msg := range le.Errors {
			_, _ = fmt.Fprintf(w, "    %s\n", msg)
		}
	}
}

func countModules(files []*analyzer.GoFile) int {
	seen := make(map[string]bool)
	for _, f := range files {
		seen[filepath.Dir(f.Path)] = true
	}
	return len(seen)
}

// loadRawRulesJSON reads .cqrs-lint.json from the current directory and returns
// the raw JSON of the "rules" key (for unknown-key validation). Returns nil if
// the file is absent, unreadable, or has no "rules" key — validation is
// best-effort and must never block a lint run.
func loadRawRulesJSON() []byte {
	data, err := os.ReadFile(".cqrs-lint.json")
	if err != nil {
		return nil
	}

	var top map[string]any
	if err := json.Unmarshal(data, &top); err != nil {
		return nil
	}

	if rules, ok := top["rules"]; ok {
		if reencoded, err := json.Marshal(rules); err == nil {
			return reencoded
		}
	}

	return nil
}

func printDetectorTimings(w io.Writer, snap pipeline.MetricsSnapshot) {
	if len(snap.DetectorTimes) == 0 {
		return
	}

	type detStat struct {
		name     string
		duration time.Duration
		findings int
	}

	stats := make([]detStat, 0, len(snap.DetectorTimes))
	for name, d := range snap.DetectorTimes {
		stats = append(stats, detStat{name: name, duration: d, findings: snap.FindingsFound[name]})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].duration > stats[j].duration
	})

	_, _ = fmt.Fprintln(w, "Detector timings (slowest first):")
	for _, s := range stats {
		if s.duration < time.Millisecond {
			continue
		}
		_, _ = fmt.Fprintf(
			w,
			"  %-40s %8s  %d findings\n",
			s.name,
			s.duration.Round(time.Millisecond),
			s.findings,
		)
	}
	_, _ = fmt.Fprintln(w)
}
