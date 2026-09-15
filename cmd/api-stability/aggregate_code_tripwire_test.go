package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// renamedAggregateCodes are the error-family codes renamed to the stream
// vocabulary on 2026-09-08 (CHANGELOG "error-family codes renamed to stream
// vocabulary"). The rename must not silently rot back: if any of these code
// strings reappears in Go source (production or test), this tripwire fails
// and points at the file and line.
//
// Deliberately an exact-string table, not a broad `aggregate_` grep:
// legitimate non-error identifiers still contain the word (e.g. the
// "listing.aggregate_projection" projection name), and a substring sweep
// would false-fire on them.
var renamedAggregateCodes = []string{
	"event.nil_aggregate_id",
	"event.empty_aggregate_type",
	"event.aggregate_not_found",
	"command.nil_aggregate_id",
	"command.empty_aggregate_type",
	"memory.aggregate_not_found",
	"storage.parse_aggregate_id",
	"storage.parse_aggregate_type",
	"storage.aggregate_type_mismatch",
	"storage.aggregate_id_mismatch",
	"storage.stream_by_aggregate",
	"storage.delete_by_aggregate",
	"pebble.aggregate_type_mismatch",
	"pebble.aggregate_id_mismatch",
	"watermill.parse_aggregate_id_failed",
	"grpc.command.parse_aggregate_id",
	"grpc.event_client.parse_aggregate_id",
}

// scanGoFileForRenamedCodes reports every renamed-code hit in one .go file as
// "rel:LINE reintroduces CODE" strings. Shared by the repo-wide walk
// (TestNoRenamedAggregateFamilyCodeReappears) and the permanent mutation
// fixture self-assert (TestAggregateTripwireScannerBites), so the two cannot
// drift apart.
func scanGoFileForRenamedCodes(projectRoot, path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var hits []string

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		for _, code := range renamedAggregateCodes {
			if strings.Contains(line, code) {
				rel, _ := filepath.Rel(projectRoot, path)
				hits = append(hits, rel+":"+strconv.Itoa(i+1)+" reintroduces "+code)
			}
		}
	}

	return hits, nil
}

func TestNoRenamedAggregateFamilyCodeReappears(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")

	var hits []string

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := info.Name()
		if info.IsDir() {
			if name == ".git" || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(name, ".go") {
			return nil
		}

		// The tripwire's own table carries every renamed code by design.
		if name == "aggregate_code_tripwire_test.go" {
			return nil
		}

		fileHits, scanErr := scanGoFileForRenamedCodes(projectRoot, path)
		if scanErr != nil {
			return scanErr
		}
		hits = append(hits, fileHits...)

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(hits) > 0 {
		t.Errorf(
			"renamed aggregate_* family codes reappeared (stream vocabulary is canonical):\n%s",
			strings.Join(hits, "\n"),
		)
	}
}

// TestAggregateTripwireScannerBites is the permanent version of the
// 2026-09-11 hand-planted mutation proof (which died with its status
// report): the scanner must fire on a planted reintroduction fixture and
// must NOT fire on the legitimate-identifier negative control, proving the
// exact-string table still bites without a broad-substring false positive.
// The fixture lives under testdata/, which the repo-wide walk skips, so it
// only bites through this self-assert.
func TestAggregateTripwireScannerBites(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")
	fixture := filepath.Join(
		projectRoot,
		"cmd",
		"api-stability",
		"testdata",
		"aggregatetripwire",
		"planted.go",
	)

	hits, err := scanGoFileForRenamedCodes(projectRoot, fixture)
	if err != nil {
		t.Fatalf("scan fixture: %v", err)
	}

	gotCodes := make(map[string]bool, len(hits))
	for _, h := range hits {
		for _, code := range renamedAggregateCodes {
			if strings.Contains(h, " reintroduces "+code) {
				gotCodes[code] = true
			}
		}
	}

	for _, planted := range []string{"event.aggregate_not_found", "storage.parse_aggregate_id"} {
		if !gotCodes[planted] {
			t.Errorf("tripwire scanner did NOT fire on planted %s in the fixture — "+
				"the exact-string table no longer bites (hits: %v)", planted, hits)
		}
	}

	if len(hits) != 2 {
		t.Errorf("scanner produced %d hits on the fixture, want exactly 2 (planted pair only): %v",
			len(hits), hits)
	}

	for _, h := range hits {
		if strings.Contains(h, "listing.aggregate_projection") {
			t.Errorf("scanner false-fired on the legitimate-identifier negative control: %s", h)
		}
	}
}
