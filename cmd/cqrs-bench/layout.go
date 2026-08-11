package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func registerLayoutCommand(cli *cmdguard.CLI[AppConfig]) {
	layoutCmd, err := cmdguard.NewCommand[AppConfig, *LayoutFlags]("layout", &LayoutFlags{},
		layoutHandler,
		cmdguard.WithShort("Explore layout cost model — pre-deployment \"what if\" analysis"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating layout command: %v\n", err)
		os.Exit(1)
	}

	if err := cmdguard.AddCommand(cli, layoutCmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding layout command: %v\n", err)
		os.Exit(1)
	}
}

// layoutEntry holds the scoring result for one storage-layout × priority pair.
type layoutEntry struct {
	Priority     string  `json:"priority"`
	Selected     string  `json:"selected"`
	EmbedScore   float64 `json:"embedScore"`
	NormScore    float64 `json:"normalizeScore"`
	MarginPct    float64 `json:"marginPct"`
	EmbedRead    float64 `json:"embedRead,omitempty"`
	EmbedWrite   float64 `json:"embedWrite,omitempty"`
	EmbedStorage float64 `json:"embedStorage,omitempty"`
	NormRead     float64 `json:"normRead,omitempty"`
	NormWrite    float64 `json:"normWrite,omitempty"`
	NormStorage  float64 `json:"normStorage,omitempty"`
}

// layoutGroup holds all priority results for one storage layout.
type layoutGroup struct {
	Layout  string        `json:"layout"`
	Engines string        `json:"engines"`
	Entries []layoutEntry `json:"entries"`
}

var layoutMeta = []struct {
	layout  metaengine.StorageLayout
	name    string
	engines string
}{
	{metaengine.LayoutKV, "KV", "Memory (hash map)"},
	{metaengine.LayoutLSM, "LSM", "Pebble, bbolt"},
	{metaengine.LayoutRow, "Row", "SQLite, PostgreSQL, MySQL"},
	{metaengine.LayoutColumnar, "Columnar", "DuckDB"},
}

var allPriorities = []metaengine.Priority{
	metaengine.PriorityBalanced,
	metaengine.PriorityReadSpeed,
	metaengine.PriorityWriteSpeed,
	metaengine.PriorityStorageSpace,
}

func layoutProfile(layout metaengine.StorageLayout) metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name: string(layout),
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap: layout,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}
}

func parsePriority(s string) (metaengine.Priority, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "balanced":
		return metaengine.PriorityBalanced, true
	case "read-speed", "read", "readspeed":
		return metaengine.PriorityReadSpeed, true
	case "write-speed", "write", "writespeed":
		return metaengine.PriorityWriteSpeed, true
	case "storage-space", "storage", "storagespace":
		return metaengine.PriorityStorageSpace, true
	default:
		return "", false
	}
}

func parseLayoutFilter(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", true
	}
	for _, m := range layoutMeta {
		if strings.EqualFold(m.name, s) || strings.EqualFold(string(m.layout), s) {
			return m.name, true
		}
	}
	return "", false
}

func layoutHandler(_ context.Context, _ *AppConfig, flags *LayoutFlags) error {
	priorities := allPriorities
	if flags.Priority != "" {
		p, ok := parsePriority(flags.Priority)
		if !ok {
			fatalf("unknown priority %q (use: balanced, read-speed, write-speed, storage-space)", flags.Priority)
		}
		priorities = []metaengine.Priority{p}
	}

	layoutFilter, ok := parseLayoutFilter(flags.Layout)
	if !ok {
		fatalf("unknown layout %q (use: kv, lsm, row, columnar)", flags.Layout)
	}

	var groups []layoutGroup
	for _, m := range layoutMeta {
		if layoutFilter != "" && m.name != layoutFilter {
			continue
		}
		profile := layoutProfile(m.layout)
		costs := metaengine.ScoreLayouts(profile)

		var embedCost, normCost metaengine.LayoutCost
		for _, c := range costs {
			if c.Option == metaengine.LayoutEmbed {
				embedCost = c
			}
			if c.Option == metaengine.LayoutNormalize {
				normCost = c
			}
		}

		grp := layoutGroup{Layout: m.name, Engines: m.engines}
		for _, p := range priorities {
			w := p.Weights()
			embedScore := embedCost.ScoreWeighted(w)
			normScore := normCost.ScoreWeighted(w)
			selected := "Embed"
			if normScore < embedScore {
				selected = "Normalize"
			}
			margin := math.Abs(embedScore-normScore) / math.Max(embedScore, normScore) * 100

			entry := layoutEntry{
				Priority:   string(p),
				Selected:   selected,
				EmbedScore: embedScore,
				NormScore:  normScore,
				MarginPct:  margin,
			}
			if flags.Verbose {
				entry.EmbedRead = embedCost.ReadCost
				entry.EmbedWrite = embedCost.WriteCost
				entry.EmbedStorage = embedCost.StorageCost
				entry.NormRead = normCost.ReadCost
				entry.NormWrite = normCost.WriteCost
				entry.NormStorage = normCost.StorageCost
			}
			grp.Entries = append(grp.Entries, entry)
		}
		groups = append(groups, grp)
	}

	withOutput(flags.Output, func(w *os.File) {
		switch resolveFormat(flags.Format) {
		case formatJSON:
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			_ = enc.Encode(groups)
		default:
			renderLayoutText(w, groups, flags.Verbose)
		}
	})

	return nil
}

func renderLayoutText(w *os.File, groups []layoutGroup, verbose bool) {
	for _, grp := range groups {
		fmt.Fprintf(w, "\n%s (%s)\n", grp.Layout, grp.Engines)
		fmt.Fprintln(w, strings.Repeat("-", 72))

		if verbose {
			first := grp.Entries[0]
			fmt.Fprintf(w, "  Embed:      Read=%.2f  Write=%.2f  Storage=%.2f\n",
				first.EmbedRead, first.EmbedWrite, first.EmbedStorage)
			fmt.Fprintf(w, "  Normalize:  Read=%.2f  Write=%.2f  Storage=%.2f\n",
				first.NormRead, first.NormWrite, first.NormStorage)
			fmt.Fprintln(w)
		}

		fmt.Fprintf(w, "  %-16s %-12s %12s %12s %10s\n",
			"Priority", "Selected", "Embed", "Normalize", "Margin")
		for _, e := range grp.Entries {
			fmt.Fprintf(w, "  %-16s %-12s %12.2f %12.2f %9.1f%%\n",
				e.Priority, e.Selected, e.EmbedScore, e.NormScore, e.MarginPct)
		}
	}
	fmt.Fprintln(w)
}
