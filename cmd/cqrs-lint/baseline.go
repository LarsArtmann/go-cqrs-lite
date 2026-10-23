package main

// Findings baseline: a checked-in record of adjudicated findings, so a lint run
// can answer "which findings are NEW?" instead of re-triaging the whole set
// every time detection width changes.
//
// Precedent: art-dupl's .art-dupl-baseline.json (same workflow: exit 1 on new
// groups only). The baseline is deliberately coarse: entries key by rule,
// project-relative file, line, and column. Line shifts therefore surface as one
// new finding plus one shrunk entry, which is the intended drift signal.

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/larsartmann/go-finding"
)

// baselineVersion is the on-disk schema version. Unknown versions are rejected
// rather than silently misread.
const baselineVersion = 1

// errBaselineInvalid marks malformed or unsupported baseline documents.
var errBaselineInvalid = errors.New("invalid findings baseline")

// baselineDocument is the JSON shape of .cqrs-lint-baseline.json.
type baselineDocument struct {
	Version int             `json:"version"`
	Tool    string          `json:"tool"`
	Entries []baselineEntry `json:"entries"`
}

// baselineEntry records one adjudicated finding. ID is informational (finding
// IDs embed absolute paths and are not machine-portable); matching uses Rule +
// File + Line + Column, with File stored relative to the baseline's directory.
type baselineEntry struct {
	Rule   string `json:"rule"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Reason string `json:"reason,omitempty"`
}

// baselineKey is the portable identity of a finding: the absolute path is
// normalized to a baseline-relative one before it reaches this type.
type baselineKey struct {
	rule   string
	file   string
	line   int
	column int
}

func (e baselineEntry) key() baselineKey {
	return baselineKey{rule: e.Rule, file: e.File, line: e.Line, column: e.Column}
}

// baselineOutcome is the result of splitting a run against a baseline.
type baselineOutcome struct {
	New   []finding.Finding // findings the baseline does not know
	Known []finding.Finding // findings recorded in the baseline (hidden)
	Stale []baselineEntry   // baseline entries no finding matches anymore
}

// baselinePathKey converts a finding's position into the portable key used by
// the baseline file. Paths are made relative to root (the baseline file's
// directory) and slash-normalized so a checked-in baseline works on any
// machine and OS.
func baselinePathKey(f finding.Finding, root string) baselineKey {
	file := string(f.Position.File)

	if file != "" && root != "" {
		if rel, err := filepath.Rel(root, file); err == nil {
			file = rel
		}
	}

	return baselineKey{
		rule:   string(f.Rule),
		file:   filepath.ToSlash(file),
		line:   f.Position.Line,
		column: f.Position.Column,
	}
}

// splitBaseline classifies findings against the document. Findings in the
// document are Known; everything else is New. Entries with no matching finding
// are Stale, so a fixed or renumbered finding is as visible as a new one.
func splitBaseline(doc *baselineDocument, findings []finding.Finding, root string) baselineOutcome {
	recorded := make(map[baselineKey]bool, len(doc.Entries))
	for _, entry := range doc.Entries {
		recorded[entry.key()] = true
	}

	matched := make(map[baselineKey]bool, len(doc.Entries))

	var outcome baselineOutcome

	for _, f := range findings {
		key := baselinePathKey(f, root)

		if recorded[key] {
			outcome.Known = append(outcome.Known, f)
			matched[key] = true

			continue
		}

		outcome.New = append(outcome.New, f)
	}

	for _, entry := range doc.Entries {
		if !matched[entry.key()] {
			outcome.Stale = append(outcome.Stale, entry)
		}
	}

	slices.SortFunc(outcome.Stale, compareBaselineEntries)

	return outcome
}

// entriesFromFindings renders the current findings as baseline entries,
// preserving reasons recorded in the previous document (a regenerated baseline
// must not silently drop the operator's adjudication notes).
func entriesFromFindings(
	findings []finding.Finding,
	root string,
	prior *baselineDocument,
) []baselineEntry {
	reasons := make(map[baselineKey]string)

	if prior != nil {
		for _, entry := range prior.Entries {
			if entry.Reason != "" {
				reasons[entry.key()] = entry.Reason
			}
		}
	}

	seen := make(map[baselineKey]bool, len(findings))
	entries := make([]baselineEntry, 0, len(findings))

	for _, f := range findings {
		key := baselinePathKey(f, root)
		if seen[key] {
			continue
		}

		seen[key] = true

		entries = append(entries, baselineEntry{
			Rule:   key.rule,
			File:   key.file,
			Line:   key.line,
			Column: key.column,
			Reason: reasons[key],
		})
	}

	slices.SortFunc(entries, compareBaselineEntries)

	return entries
}

// compareBaselineEntries orders entries deterministically for a stable file:
// rule, then file, then line, then column.
func compareBaselineEntries(a, b baselineEntry) int {
	if c := strings.Compare(a.Rule, b.Rule); c != 0 {
		return c
	}

	if c := strings.Compare(a.File, b.File); c != 0 {
		return c
	}

	if a.Line != b.Line {
		return a.Line - b.Line
	}

	return a.Column - b.Column
}

// loadBaseline reads and validates a baseline document.
func loadBaseline(path string) (*baselineDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read findings baseline %s: %w", path, err)
	}

	var doc baselineDocument

	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", errBaselineInvalid, path, err)
	}

	if doc.Version != baselineVersion {
		return nil, fmt.Errorf(
			"%w: %s has version %d, want %d (regenerate with --update-baseline)",
			errBaselineInvalid, path, doc.Version, baselineVersion,
		)
	}

	return &doc, nil
}

// loadBaselineIfPresent reads a baseline, returning nil when the file does not
// exist. Used by --update-baseline to preserve reasons on a first write.
func loadBaselineIfPresent(path string) (*baselineDocument, error) {
	doc, err := loadBaseline(path)
	if err == nil {
		return doc, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	return nil, err
}

// writeBaseline writes the document deterministically: entries sorted, two
// space indent, trailing newline. Atomic via a temp file + rename so an
// interrupted run cannot leave a truncated baseline behind.
func writeBaseline(path string, entries []baselineEntry) error {
	doc := baselineDocument{
		Version: baselineVersion,
		Tool:    "cqrs-lint",
		Entries: entries,
	}

	data, err := json.Marshal(doc, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("marshal findings baseline: %w", err)
	}

	data = append(data, '\n')

	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".cqrs-lint-baseline-*.tmp")
	if err != nil {
		return fmt.Errorf("write findings baseline %s: %w", path, err)
	}

	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)

		return fmt.Errorf("write findings baseline %s: %w", path, err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)

		return fmt.Errorf("write findings baseline %s: %w", path, err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)

		return fmt.Errorf("write findings baseline %s: %w", path, err)
	}

	return nil
}

// baselineRun is the effective per-run baseline state, or nil when no baseline
// was requested.
type baselineRun struct {
	Reported []finding.Finding // findings to report and gate the exit code on
	Known    int
	Stale    int
	Updated  bool
}

// applyBaseline loads (or rewrites) the findings baseline and returns the
// findings that should be reported. A nil result means the baseline flag was
// not used.
func applyBaseline(cfg *AppConfig, active []finding.Finding) (*baselineRun, error) {
	if cfg.UpdateBaseline && cfg.Baseline == "" {
		return nil, errors.New("--update-baseline requires --baseline <path>")
	}

	if cfg.Baseline == "" {
		return nil, nil
	}

	absPath, err := filepath.Abs(cfg.Baseline)
	if err != nil {
		return nil, fmt.Errorf("resolve findings baseline path: %w", err)
	}

	root := filepath.Dir(absPath)

	if cfg.UpdateBaseline {
		prior, err := loadBaselineIfPresent(absPath)
		if err != nil {
			return nil, err
		}

		entries := entriesFromFindings(active, root, prior)

		if err := writeBaseline(absPath, entries); err != nil {
			return nil, err
		}

		if !cfg.Quiet {
			fmt.Fprintf(os.Stderr, "Findings baseline written: %s (%d entries)\n", absPath, len(entries))
		}

		return &baselineRun{Reported: nil, Known: len(entries), Updated: true}, nil
	}

	doc, err := loadBaseline(absPath)
	if err != nil {
		return nil, err
	}

	outcome := splitBaseline(doc, active, root)

	if !cfg.Quiet {
		if len(outcome.Known) > 0 {
			fmt.Fprintf(
				os.Stderr,
				"%d finding(s) matched the baseline (not reported; %s)\n",
				len(outcome.Known), absPath,
			)
		}

		if len(outcome.Stale) > 0 {
			fmt.Fprintf(
				os.Stderr,
				"Baseline shrunk: %d recorded finding(s) no longer occur. "+
					"Run with --update-baseline to prune them.\n",
				len(outcome.Stale),
			)

			for _, entry := range outcome.Stale {
				fmt.Fprintf(os.Stderr, "  stale: %s %s:%d:%d\n", entry.Rule, entry.File, entry.Line, entry.Column)
			}
		}
	}

	return &baselineRun{
		Reported: outcome.New,
		Known:    len(outcome.Known),
		Stale:    len(outcome.Stale),
	}, nil
}
