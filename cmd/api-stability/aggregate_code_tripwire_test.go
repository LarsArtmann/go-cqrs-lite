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

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			for _, code := range renamedAggregateCodes {
				if strings.Contains(line, code) {
					rel, _ := filepath.Rel(projectRoot, path)
					hits = append(hits, rel+":"+strconv.Itoa(i+1)+" reintroduces "+code)
				}
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(hits) > 0 {
		t.Errorf("renamed aggregate_* family codes reappeared (stream vocabulary is canonical):\n%s",
			strings.Join(hits, "\n"))
	}
}
