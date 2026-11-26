package main

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/markdown"
	gotable "github.com/larsartmann/go-output/table"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// Report formats, as chosen by --format.
const (
	formatAuto      = "auto"
	formatTable     = "table"
	formatText      = "text"
	formatJSON      = "json"
	formatCSV       = "csv"
	formatTSV       = "tsv"
	formatMarkdown  = "markdown"
	formatBenchstat = "benchstat"
	formatManifest  = "manifest"
)

// reportWriter renders into a report target with the target's color mode: a
// terminal may use color, a file never does.
type reportWriter struct {
	w     io.Writer
	color output.ColorMode
}

// render writes a report into the target.
func (t reportTarget) render(report func(out reportWriter) error) error {
	return t.write(func(w io.Writer) error {
		return report(reportWriter{w: w, color: t.color()})
	})
}

// table writes a styled table.
func (r reportWriter) table(data *output.Table) error {
	if err := gotable.Write(r.w, data, gotable.WithColorMode(r.color)); err != nil {
		return fmt.Errorf("%w: %w", errRender, err)
	}

	return nil
}

// markdownTable writes a go-output table as a markdown pipe table (the
// leading border pipe go-output emits is dropped).
func (r reportWriter) markdownTable(data *output.Table) error {
	rendered, err := markdown.Render(data, markdown.WithColorMode(output.ColorModeNever))
	if err != nil {
		return fmt.Errorf("%w: %w", errRender, err)
	}

	fmt.Fprint(r.w, strings.TrimPrefix(rendered, "|"))
	fmt.Fprintln(r.w)

	return nil
}

// writeError wraps a failed report write.
func writeError(err error) error {
	return fmt.Errorf("%w: %w", errRender, err)
}

// ── formatting helpers ─

func sortedResultKeys(results map[string]*benchkit.Result) []string {
	return slices.Sorted(maps.Keys(results))
}

// fmtDur wraps benchkit.FormatDuration so render.go callers stay short.
func fmtDur(d time.Duration) string { return benchkit.FormatDuration(d) }

// fmtBytes wraps benchkit.FormatBytes.
func fmtBytes(bytes uint64) string { return benchkit.FormatBytes(bytes) }

// fmtFloat wraps benchkit.FormatFloat.
func fmtFloat(f float64) string { return benchkit.FormatFloat(f) }

// fmtInt wraps benchkit.FormatInt.
func fmtInt(n int) string { return benchkit.FormatInt(n) }

// fmtGCDash renders a GC pause, "-" when none was recorded.
func fmtGCDash(d time.Duration) string {
	if d == 0 {
		return "-"
	}

	return fmtDur(d)
}

// fmtRatioDash renders a ratio such as write amplification, "-" when it was
// not measured.
func fmtRatioDash(ratio float64) string {
	if ratio <= 0 {
		return "-"
	}

	return fmt.Sprintf("%.1fx", ratio)
}

// fmtAllocDash renders allocations per operation, "-" when not measured.
func fmtAllocDash(allocs float64) string {
	if allocs <= 0 {
		return "-"
	}

	return fmt.Sprintf("%.0f", allocs)
}

// fmtCoVDash renders a coefficient of variation in percent, "-" when there
// is no dispersion to show.
func fmtCoVDash(cov float64) string {
	if cov <= 0 {
		return "-"
	}

	return fmt.Sprintf("%.1f%%", cov*100)
}

// dashRow builds a FAILED table row: the leading cells, then "-" for every
// remaining column.
func dashRow(headers []string, leading ...string) []string {
	row := append(slices.Clone(leading), make([]string, len(headers)-len(leading))...)
	for i := len(leading); i < len(row); i++ {
		row[i] = "-"
	}

	return row
}

// withCoV says whether any sweep point carries repeat dispersion, which is
// exactly when the sweep table grows its CoV column.
func withCoV(results []benchkit.SweepResult) bool {
	return slices.ContainsFunc(results, func(sr benchkit.SweepResult) bool {
		return sr.Result != nil && sr.Result.RepeatCount > 1
	})
}
