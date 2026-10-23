package main

// Baseline persistence: the on-disk document shape and its atomic read/write
// helpers. Matching and split semantics live in baseline.go.

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// baselineEntry records one adjudicated finding. ID is not stored (finding IDs
// embed absolute paths and are not machine-portable); matching uses Rule +
// File + Line + Column, with File stored relative to the baseline's directory.
type baselineEntry struct {
	Rule   string `json:"rule"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Reason string `json:"reason,omitempty"`
}

func (e baselineEntry) key() baselineKey {
	return baselineKey{rule: e.Rule, file: e.File, line: e.Line, column: e.Column}
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

// writeBaseline writes the document deterministically: entries sorted by the
// caller, two space indent, trailing newline. Atomic via a temp file + rename
// so an interrupted run cannot leave a truncated baseline behind.
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
