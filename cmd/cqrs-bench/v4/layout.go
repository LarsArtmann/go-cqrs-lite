package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/larsartmann/go-output"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// layoutRuleWidth is the width of the text layout's heading rule.
const layoutRuleWidth = 72

// layoutEntry holds the scoring result for one storage-layout and priority
// pair.
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

// layoutMetaDescriptor describes one storage layout for the exploration tool.
type layoutMetaDescriptor struct {
	layout  metaengine.StorageLayout
	name    string
	engines string
}

// allLayoutMeta returns the storage layouts the cost model understands.
func allLayoutMeta() []layoutMetaDescriptor {
	return []layoutMetaDescriptor{
		{metaengine.LayoutKV, "KV", "Memory (hash map)"},
		{metaengine.LayoutLSM, "LSM", "Pebble, bbolt"},
		{metaengine.LayoutRow, "Row", "SQLite, PostgreSQL, MySQL"},
		{metaengine.LayoutColumnar, "Columnar", "DuckDB"},
	}
}

// allPriorities returns the four operator priorities in display order.
func allPriorities() []metaengine.Priority {
	return []metaengine.Priority{
		metaengine.PriorityBalanced,
		metaengine.PriorityReadSpeed,
		metaengine.PriorityWriteSpeed,
		metaengine.PriorityStorageSpace,
	}
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

// parsePriority resolves --priority; empty selects the balanced priority.
func parsePriority(s string) (metaengine.Priority, error) {
	priority, ok := priorityByName(s)
	if !ok {
		return "", fmt.Errorf("%w: unknown --priority %q (use: balanced, read-speed, write-speed, storage-space)",
			errFlagValue, s)
	}

	return priority, nil
}

func priorityByName(s string) (metaengine.Priority, bool) {
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

// parseLayoutFilter resolves --layout into the canonical layout name; empty
// keeps every layout.
func parseLayoutFilter(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", nil
	}

	for _, meta := range allLayoutMeta() {
		if strings.EqualFold(meta.name, s) || strings.EqualFold(string(meta.layout), s) {
			return meta.name, nil
		}
	}

	return "", fmt.Errorf("%w: unknown --layout %q (use: kv, lsm, row, columnar)", errFlagValue, s)
}

// layoutHandler is the layout command: a pre-deployment "what if" look at
// the metaengine layout cost model.
func layoutHandler(_ context.Context, _ *appConfig, flags layoutFlags) error {
	priorities, priorityErr := flags.priorities()

	layoutFilter, layoutErr := parseLayoutFilter(flags.Layout)

	err := errors.Join(validateFormat("layout", flags.Format, layoutFormats()), priorityErr, layoutErr)
	if err != nil {
		return err
	}

	groups, err := scoreLayouts(priorities, layoutFilter, flags.Verbose)
	if err != nil {
		return err
	}

	target := fileTarget(flags.Output)
	format := target.resolveFormat(flags.Format)

	return target.render(func(out reportWriter) error {
		return renderLayout(out, format, groups, flags.Verbose)
	})
}

// priorities resolves --priority into the priorities to score.
func (f layoutFlags) priorities() ([]metaengine.Priority, error) {
	if f.Priority == "" {
		return allPriorities(), nil
	}

	priority, err := parsePriority(f.Priority)
	if err != nil {
		return nil, err
	}

	return []metaengine.Priority{priority}, nil
}

// layoutCosts are the embed and normalize cost of one engine profile.
type layoutCosts struct {
	embed metaengine.LayoutCost
	norm  metaengine.LayoutCost
}

// scoreLayouts scores every selected layout for every priority, asking the
// planner's own SelectLayout which option it would pick.
func scoreLayouts(
	priorities []metaengine.Priority, layoutFilter string, verbose bool,
) ([]layoutGroup, error) {
	var groups []layoutGroup

	for _, meta := range allLayoutMeta() {
		if layoutFilter != "" && meta.name != layoutFilter {
			continue
		}

		profile := layoutProfile(meta.layout)

		costs, err := scoreCosts(profile)
		if err != nil {
			return nil, err
		}

		entries := make([]layoutEntry, 0, len(priorities))

		for _, priority := range priorities {
			entries = append(entries, scorePriority(profile, priority, costs, verbose))
		}

		groups = append(groups, layoutGroup{Layout: meta.name, Engines: meta.engines, Entries: entries})
	}

	return groups, nil
}

// scoreCosts reads the embed and normalize cost out of the cost model.
func scoreCosts(profile metaengine.EngineProfile) (layoutCosts, error) {
	var costs layoutCosts

	for _, cost := range metaengine.ScoreLayouts(profile) {
		switch cost.Option {
		case metaengine.LayoutEmbed:
			costs.embed = cost
		case metaengine.LayoutNormalize:
			costs.norm = cost
		}
	}

	if costs.embed.Option == "" || costs.norm.Option == "" {
		return costs, fmt.Errorf("cost model returned no embed or normalize option for %s", profile.Name)
	}

	return costs, nil
}

// scorePriority scores one priority. The selected option comes from
// metaengine.SelectLayout, so the tool shows what the planner would do
// rather than a second copy of its decision rule.
func scorePriority(
	profile metaengine.EngineProfile,
	priority metaengine.Priority,
	costs layoutCosts,
	verbose bool,
) layoutEntry {
	weights := priority.Weights()

	embedScore := costs.embed.ScoreWeighted(weights)
	normScore := costs.norm.ScoreWeighted(weights)

	option, _ := metaengine.SelectLayout(profile, priority)

	entry := layoutEntry{
		Priority:   string(priority),
		Selected:   string(option),
		EmbedScore: embedScore,
		NormScore:  normScore,
		MarginPct:  math.Abs(embedScore-normScore) / math.Max(embedScore, normScore) * 100,
	}

	if verbose {
		entry.EmbedRead = costs.embed.ReadCost
		entry.EmbedWrite = costs.embed.WriteCost
		entry.EmbedStorage = costs.embed.StorageCost
		entry.NormRead = costs.norm.ReadCost
		entry.NormWrite = costs.norm.WriteCost
		entry.NormStorage = costs.norm.StorageCost
	}

	return entry
}

// renderLayout writes the scored layout groups in format.
func renderLayout(out reportWriter, format string, groups []layoutGroup, verbose bool) error {
	switch format {
	case formatJSON:
		encoder := jsontext.NewEncoder(out.w, jsontext.WithIndent("  "))
		if err := json.MarshalEncode(encoder, groups); err != nil {
			return writeError(err)
		}

		return nil
	case formatTable:
		return out.table(buildLayoutTable(groups))
	default:
		renderLayoutText(out.w, groups, verbose)

		return nil
	}
}

// buildLayoutTable renders all layout groups as one bordered table, the
// tabular twin of renderLayoutText (same columns).
func buildLayoutTable(groups []layoutGroup) *output.Table {
	table := output.NewTable([]string{"Layout", "Priority", "Selected", "Embed", "Normalize", "Margin"})

	for _, group := range groups {
		for _, entry := range group.Entries {
			table.AddRow([]string{
				group.Layout,
				entry.Priority,
				entry.Selected,
				fmt.Sprintf("%.2f", entry.EmbedScore),
				fmt.Sprintf("%.2f", entry.NormScore),
				fmt.Sprintf("%.1f%%", entry.MarginPct),
			})
		}
	}

	return table
}

// renderLayoutText writes the scored layout groups as aligned text.
func renderLayoutText(w io.Writer, groups []layoutGroup, verbose bool) {
	for _, group := range groups {
		fmt.Fprintf(w, "\n%s (%s)\n", group.Layout, group.Engines)
		fmt.Fprintln(w, strings.Repeat("-", layoutRuleWidth))

		if verbose {
			first := group.Entries[0]
			fmt.Fprintf(w, "  Embed:      Read=%.2f  Write=%.2f  Storage=%.2f\n",
				first.EmbedRead, first.EmbedWrite, first.EmbedStorage)
			fmt.Fprintf(w, "  Normalize:  Read=%.2f  Write=%.2f  Storage=%.2f\n",
				first.NormRead, first.NormWrite, first.NormStorage)
			fmt.Fprintln(w)
		}

		fmt.Fprintf(w, "  %-16s %-12s %12s %12s %10s\n",
			"Priority", "Selected", "Embed", "Normalize", "Margin")

		for _, entry := range group.Entries {
			fmt.Fprintf(w, "  %-16s %-12s %12.2f %12.2f %9.1f%%\n",
				entry.Priority, entry.Selected, entry.EmbedScore, entry.NormScore, entry.MarginPct)
		}
	}

	fmt.Fprintln(w)
}
