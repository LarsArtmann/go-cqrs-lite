package suppression

import (
	"os"
	"sort"
	"strings"
)

// FixResult reports the outcome of a --doctor --fix run.
type FixResult struct {
	// Removed holds one entry per deleted stale directive LINE, deduplicated
	// by line (a combined ignore(A,B) directive that went fully stale emits
	// one audit entry per rule but is a single line — it appears once here).
	Removed []SuppressionAuditEntry
	// Skipped holds stale entries left untouched: the directive trails on a
	// line of code, shares its line with other content, or the file could
	// not be read or rewritten. These need manual removal.
	Skipped []SuppressionAuditEntry
	// Files lists the rewritten files (unique, sorted).
	Files []string
}

// RemoveStaleInlineSuppressions deletes stale inline suppression directives
// from disk — but ONLY lines whose sole non-whitespace content is the
// //cqrs-lint:ignore(...) comment. Trailing-on-code directives are never
// auto-removed (deleting the line would delete the code too); block markers
// (ignore-start/ignore-end) do not parse as inline directives and are immune
// by construction. Both classes surface in Skipped for manual cleanup.
// Unknown-rule entries are not touched: they may be typos the user wants to
// correct rather than delete. Non-stale entries are ignored.
func RemoveStaleInlineSuppressions(entries []SuppressionAuditEntry) FixResult {
	var res FixResult

	byFile := make(map[string][]SuppressionAuditEntry)
	for _, e := range entries {
		if e.Status == AuditStale {
			byFile[e.File] = append(byFile[e.File], e)
		}
	}

	for file, fileEntries := range byFile {
		removeStaleLinesInFile(file, fileEntries, &res)
	}

	sortAuditEntries(res.Removed)
	dedupeByLine(&res.Removed)
	sortAuditEntries(res.Skipped)
	sort.Strings(res.Files)

	return res
}

// PlanStaleInlineSuppressions classifies stale directives exactly like
// [RemoveStaleInlineSuppressions] would, WITHOUT touching any file:
// Removed holds the lines that WOULD be deleted, Skipped those that would
// be left for manual removal, and Files the files that would be rewritten.
// It backs `doctor --fix --dry-run`.
func PlanStaleInlineSuppressions(entries []SuppressionAuditEntry) FixResult {
	var res FixResult

	byFile := make(map[string][]SuppressionAuditEntry)
	for _, e := range entries {
		if e.Status == AuditStale {
			byFile[e.File] = append(byFile[e.File], e)
		}
	}

	for file, fileEntries := range byFile {
		planStaleLinesInFile(file, fileEntries, &res)
	}

	sortAuditEntries(res.Removed)
	dedupeByLine(&res.Removed)
	sortAuditEntries(res.Skipped)
	sort.Strings(res.Files)

	return res
}

// planStaleLinesInFile classifies each stale entry of one file without
// rewriting anything, recording the outcome in res.
func planStaleLinesInFile(file string, entries []SuppressionAuditEntry, res *FixResult) {
	data, err := os.ReadFile(file)
	if err != nil {
		res.Skipped = append(res.Skipped, entries...)
		return
	}

	lines := strings.Split(string(data), "\n")

	removable := false
	for _, e := range entries {
		if e.Line >= 1 && e.Line <= len(lines) && isWholeLineDirective(lines[e.Line-1]) {
			res.Removed = append(res.Removed, e)
			removable = true
		} else {
			res.Skipped = append(res.Skipped, e)
		}
	}

	if removable {
		res.Files = append(res.Files, file)
	}
}

// removeStaleLinesInFile classifies each stale entry of one file, deletes the
// whole-line directives, and records the outcome in res.
func removeStaleLinesInFile(file string, entries []SuppressionAuditEntry, res *FixResult) {
	data, err := os.ReadFile(file)
	if err != nil {
		res.Skipped = append(res.Skipped, entries...)
		return
	}

	lines := strings.Split(string(data), "\n")

	remove := make(map[int]bool)
	for _, e := range entries {
		if e.Line >= 1 && e.Line <= len(lines) && isWholeLineDirective(lines[e.Line-1]) {
			remove[e.Line] = true
		}
	}

	for _, e := range entries {
		if !remove[e.Line] {
			res.Skipped = append(res.Skipped, e)
		}
	}

	if len(remove) == 0 {
		return
	}

	kept := make([]string, 0, len(lines)-len(remove))
	for i, line := range lines {
		if !remove[i+1] {
			kept = append(kept, line)
		}
	}

	if err := writeFilePreservingPerm(file, strings.Join(kept, "\n")); err != nil {
		for _, e := range entries {
			if remove[e.Line] {
				res.Skipped = append(res.Skipped, e)
			}
		}
		return
	}

	res.Files = append(res.Files, file)
	for _, e := range entries {
		if remove[e.Line] {
			res.Removed = append(res.Removed, e)
		}
	}
}

// isWholeLineDirective reports whether the line contains nothing but a
// suppression directive comment (plus surrounding whitespace). Only such
// lines can be deleted safely.
func isWholeLineDirective(line string) bool {
	cs := commentTextStart(line)
	if cs < 0 {
		return false
	}

	if strings.TrimSpace(line[:cs]) != "" {
		return false // code or another comment precedes the directive
	}

	// A non-empty parse result excludes block markers: ignore-start/ignore-end
	// comment text does not match the ignore(RULES) shape, so ParseSuppressions
	// returns nothing for them.
	return len(ParseSuppressions(line)) > 0
}

// writeFilePreservingPerm writes content to an existing path, keeping the
// file's current permissions (0o644 fallback when stat fails).
func writeFilePreservingPerm(file, content string) error {
	perm := os.FileMode(0o644)
	if info, err := os.Stat(file); err == nil {
		perm = info.Mode().Perm()
	}

	return os.WriteFile(file, []byte(content), perm)
}

// sortAuditEntries sorts by file, then line, then rule.
func sortAuditEntries(entries []SuppressionAuditEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].File != entries[j].File {
			return entries[i].File < entries[j].File
		}
		if entries[i].Line != entries[j].Line {
			return entries[i].Line < entries[j].Line
		}
		return entries[i].Rule < entries[j].Rule
	})
}

// dedupeByLine collapses sorted entries that share (file, line), keeping the
// first — combined directives are represented once in FixResult.Removed.
func dedupeByLine(entries *[]SuppressionAuditEntry) {
	if len(*entries) < 2 {
		return
	}

	deduped := (*entries)[:1]
	for _, e := range (*entries)[1:] {
		last := deduped[len(deduped)-1]
		if e.File != last.File || e.Line != last.Line {
			deduped = append(deduped, e)
		}
	}

	*entries = deduped
}
