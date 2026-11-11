package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	errExportMismatch = errors.New("API export mismatch")
	errGoldenMissing  = errors.New("API golden file missing")
)

const (
	goldenDirPerms  = 0o750
	goldenFilePerms = 0o600
)

func writeGoldenFile(goldenPath string, exports []string) error {
	cleanPath := filepath.Clean(goldenPath)

	err := os.MkdirAll(filepath.Dir(cleanPath), goldenDirPerms)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	err = os.WriteFile(cleanPath, []byte(strings.Join(exports, "\n")+"\n"), goldenFilePerms)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if _, err := fmt.Fprintf(
		os.Stdout,
		"Updated %s (%d exports)\n",
		cleanPath,
		len(exports),
	); err != nil {
		return fmt.Errorf("write status: %w", err)
	}

	return nil
}

func verifyGoldenFile(goldenPath string, exports []string) error {
	cleanPath := filepath.Clean(goldenPath)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return handleReadError(cleanPath, err)
	}

	expected := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(exports) != len(expected) {
		return reportMismatch(expected, exports)
	}

	for i, exp := range expected {
		if exports[i] != exp {
			return fmt.Errorf(
				"%w: export %d: expected %q, got %q",
				errExportMismatch,
				i,
				exp,
				exports[i],
			)
		}
	}

	if _, err := fmt.Fprintf(
		os.Stdout,
		"API surface OK: %d exports verified\n",
		len(exports),
	); err != nil {
		return fmt.Errorf("write status: %w", err)
	}

	return nil
}

func handleReadError(goldenPath string, err error) error {
	if os.IsNotExist(err) {
		return fmt.Errorf("%w: %s; run with --update to create", errGoldenMissing, goldenPath)
	}

	return fmt.Errorf("read: %w", err)
}

func reportMismatch(expected, exports []string) error {
	missing, added := diff(expected, exports)
	fmt.Fprintf(
		os.Stderr,
		"API surface mismatch: %d expected, %d actual\n",
		len(expected),
		len(exports),
	)

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "REMOVED exports:\n  %s\n", strings.Join(missing, "\n  "))
	}

	if len(added) > 0 {
		fmt.Fprintf(os.Stderr, "NEW exports:\n  %s\n", strings.Join(added, "\n  "))
	}

	return fmt.Errorf("%w: %d expected, %d actual", errExportMismatch, len(expected), len(exports))
}

func diff(expected, actual []string) ([]string, []string) {
	expSet := make(map[string]struct{}, len(expected))
	for _, e := range expected {
		expSet[e] = struct{}{}
	}

	actSet := make(map[string]struct{}, len(actual))
	for _, a := range actual {
		actSet[a] = struct{}{}
	}

	var missing []string

	for _, e := range expected {
		if _, ok := actSet[e]; !ok {
			missing = append(missing, e)
		}
	}

	var added []string

	for _, a := range actual {
		if _, ok := expSet[a]; !ok {
			added = append(added, a)
		}
	}

	return missing, added
}
