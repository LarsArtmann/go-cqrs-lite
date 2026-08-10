package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// sortStats orders stats by descending hotspot, then commits, then path.
func sortStats(in []*FileStat) {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Hotspot() != in[j].Hotspot() {
			return in[i].Hotspot() > in[j].Hotspot()
		}
		if in[i].Commits != in[j].Commits {
			return in[i].Commits > in[j].Commits
		}
		return in[i].Path < in[j].Path
	})
}

// writeReport renders stats in the requested format, limited to top rows.
func writeReport(w io.Writer, rows []*FileStat, format string, top int) {
	sortStats(rows)
	if top > 0 && top < len(rows) {
		rows = rows[:top]
	}
	switch format {
	case "markdown", "md":
		writeMarkdown(w, rows)
	case "csv":
		writeCSV(w, rows)
	case "json":
		writeJSON(w, rows)
	default:
		writeTable(w, rows)
	}
}

func writeTable(w io.Writer, rows []*FileStat) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "RANK\tPATH\tCOMMITS\tCHURN\tAUTHORS\tSLOC\tHOTSPOT")
	for i, r := range rows {
		fmt.Fprintf(tw, "%d\t%s\t%d\t%d\t%d\t%d\t%s\n",
			i+1, truncatePath(r.Path, 52), r.Commits, r.Churn(), r.AuthorCount(),
			r.Complexity, formatHotspot(r.Hotspot()))
	}
	tw.Flush()
}

func writeMarkdown(w io.Writer, rows []*FileStat) {
	fmt.Fprintln(w, "| # | Path | Commits | Churn | Authors | SLOC | Hotspot |")
	fmt.Fprintln(w, "|--:|:--|--:|--:|--:|--:|--:|")
	for i, r := range rows {
		fmt.Fprintf(w, "| %d | `%s` | %d | %d | %d | %d | %s |\n",
			i+1, r.Path, r.Commits, r.Churn(), r.AuthorCount(), r.Complexity,
			formatHotspot(r.Hotspot()))
	}
}

func writeCSV(w io.Writer, rows []*FileStat) {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"path", "commits", "added", "deleted", "churn", "authors", "sloc", "hotspot"})
	for _, r := range rows {
		_ = cw.Write([]string{
			r.Path,
			strconv.Itoa(r.Commits),
			strconv.Itoa(r.Added),
			strconv.Itoa(r.Deleted),
			strconv.Itoa(r.Churn()),
			strconv.Itoa(r.AuthorCount()),
			strconv.Itoa(r.Complexity),
			strconv.FormatFloat(r.Hotspot(), 'f', 0, 64),
		})
	}
	cw.Flush()
}

// jsonRow is the JSON representation of a FileStat.
type jsonRow struct {
	Path       string  `json:"path"`
	Commits    int     `json:"commits"`
	Added      int     `json:"added"`
	Deleted    int     `json:"deleted"`
	Churn      int     `json:"churn"`
	Authors    int     `json:"authors"`
	Complexity int     `json:"complexity"`
	Hotspot    float64 `json:"hotspot"`
}

func writeJSON(w io.Writer, rows []*FileStat) {
	out := make([]jsonRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, jsonRow{
			Path: r.Path, Commits: r.Commits, Added: r.Added, Deleted: r.Deleted,
			Churn: r.Churn(), Authors: r.AuthorCount(), Complexity: r.Complexity,
			Hotspot: r.Hotspot(),
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

// formatHotspot renders large hotspot scores with k/M suffixes for readability.
func formatHotspot(h float64) string {
	switch {
	case h >= 1_000_000:
		return strconv.FormatFloat(h/1_000_000, 'f', 1, 64) + "M"
	case h >= 1_000:
		return strconv.FormatFloat(h/1_000, 'f', 1, 64) + "k"
	default:
		return strconv.FormatFloat(h, 'f', 0, 64)
	}
}

// truncatePath keeps a path within width, preserving the final segments.
func truncatePath(p string, width int) string {
	if len(p) <= width {
		return p
	}
	segs := strings.Split(p, "/")
	// keep the last two segments, abbreviate the prefix with …
	if len(segs) > 2 {
		short := strings.Join(segs[len(segs)-2:], "/")
		return "…" + short
	}
	return p
}
