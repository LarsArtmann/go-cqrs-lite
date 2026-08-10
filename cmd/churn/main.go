package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// filterOpts controls which files survive into the final report.
type filterOpts struct {
	exts             []string
	includeTests     bool
	includeGenerated bool
	prefixes         []string
}

func main() {
	if err := run(os.Args[1:], os.Stdout, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "churn:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer, now time.Time) error {
	fs := flag.NewFlagSet("churn", flag.ContinueOnError)
	since := fs.String("since", "1 year ago", "analyze commits since this date (git date spec)")
	until := fs.String("until", "", "analyze commits until this date")
	branch := fs.String("branch", "", "git revision to analyze (default: HEAD)")
	format := fs.String("format", "table", "output format: table|markdown|csv|json")
	top := fs.Int("top", 25, "rows to show (0 = all)")
	mode := fs.String("aggregate", "file", "group by: file|dir|module")
	recency := fs.Float64("recency", 0, "recency half-life in days; 0 weights all change equally")
	ext := fs.String("ext", ".go", "comma-separated extensions to include (e.g. .go,.ts)")
	includeTests := fs.Bool("include-tests", true, "include _test.go files")
	includeGen := fs.Bool("include-generated", false, "include generated files (*.gen.go, *.pb.go)")
	paths := fs.String("paths", "", "comma-separated path prefixes to include (default: all)")
	quiet := fs.Bool("quiet", false, "suppress the summary header")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fo := filterOpts{
		exts:             splitCSV(*ext),
		includeTests:     *includeTests,
		includeGenerated: *includeGen,
		prefixes:         splitCSV(*paths),
	}

	h, err := collectStats(analyzeOptions{
		since: *since, until: *until, branch: *branch, halfLifeDay: *recency,
	}, now)
	if err != nil {
		return err
	}

	h.Files = filterFiles(h.Files, fo)
	annotateComplexity(h.Files)

	rows := aggregate(h.Files, parseAggregateMode(*mode))

	if !*quiet {
		writeSummary(out, h, *mode, *recency)
	}
	writeReport(out, rows, *format, *top)
	return nil
}

// filterFiles removes files that do not satisfy the filter options.
func filterFiles(files map[string]*FileStat, fo filterOpts) map[string]*FileStat {
	out := make(map[string]*FileStat, len(files))
	for path, s := range files {
		if !keep(path, fo) {
			continue
		}
		out[path] = s
	}
	return out
}

// keep reports whether a path passes the filter.
func keep(path string, fo filterOpts) bool {
	if strings.Contains("/"+path+"/", "/vendor/") {
		return false
	}
	if !fo.includeGenerated && isGenerated(path) {
		return false
	}
	if !fo.includeTests && strings.HasSuffix(path, "_test.go") {
		return false
	}
	if len(fo.prefixes) > 0 && !hasAnyPrefix(path, fo.prefixes) {
		return false
	}
	return hasAnySuffix(path, fo.exts)
}

func hasAnyPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func hasAnySuffix(path string, suffixes []string) bool {
	if len(suffixes) == 0 {
		return true
	}
	for _, s := range suffixes {
		if s != "" && strings.HasSuffix(path, s) {
			return true
		}
	}
	return false
}

// annotateComplexity fills in the SLOC-based complexity for each existing file.
func annotateComplexity(files map[string]*FileStat) {
	for path, s := range files {
		s.Complexity = countSLOC(path)
	}
}

func writeSummary(w io.Writer, h *history, mode string, recency float64) {
	fmt.Fprintln(w, "─ churn analysis ─")
	fmt.Fprintf(w, "window:     %s", h.FirstCommit.Format("2006-01-02"))
	fmt.Fprintf(w, " → %s\n", h.LastCommit.Format("2006-01-02"))
	fmt.Fprintf(w, "commits:    %d\n", h.TotalCommits)
	fmt.Fprintf(w, "files:      %d\n", len(h.Files))
	if recency > 0 {
		fmt.Fprintf(w, "recency:    %.0f-day half-life\n", recency)
	}
	fmt.Fprintf(w, "group by:   %s\n\n", mode)
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			res = append(res, p)
		}
	}
	return res
}
