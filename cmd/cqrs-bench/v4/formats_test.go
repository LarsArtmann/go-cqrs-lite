package main

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// TestValidateFormatAcceptsEveryFormat walks each command's vocabulary
// through validateFormat and pins that no supported value is rejected.
func TestValidateFormatAcceptsEveryFormat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		command string
		allowed []string
	}{
		{"run", runFormats()},
		{"compare", reportFormats()},
		{"sweep", reportFormats()},
		{"run --soak", reportFormats()},
		{"layout", layoutFormats()},
	}

	for _, tc := range cases {
		for _, format := range tc.allowed {
			if err := validateFormat(tc.command, format, tc.allowed); err != nil {
				t.Errorf("validateFormat(%q, %q) = %v, want nil", tc.command, format, err)
			}
		}
	}
}

// TestValidateFormatRejectsAndTeaches: a format the command cannot render
// is rejected before any work, naming the supported ones.
func TestValidateFormatRejectsAndTeaches(t *testing.T) {
	t.Parallel()

	err := validateFormat("compare", formatManifest, reportFormats())
	if !errors.Is(err, errFormat) {
		t.Fatalf("validateFormat(compare, manifest) = %v, want errFormat", err)
	}

	for _, teaches := range []string{`"manifest"`, "compare", "supported:"} {
		if !strings.Contains(err.Error(), teaches) {
			t.Errorf("error %q should mention %s", err, teaches)
		}
	}
}

// TestFormatVocabularies: the report formats are the run formats minus the
// two that describe a single run, so a format can never be missing from one
// list by mistake.
func TestFormatVocabularies(t *testing.T) {
	t.Parallel()

	runOnly := slices.DeleteFunc(runFormats(), func(format string) bool {
		return slices.Contains(reportFormats(), format)
	})

	if !slices.Equal(runOnly, []string{formatBenchstat, formatManifest}) {
		t.Errorf("run-only formats = %q, want [benchstat manifest]", runOnly)
	}

	if !slices.Equal(layoutFormats(), []string{formatAuto, formatTable, formatText, formatJSON}) {
		t.Errorf("layout formats = %q", layoutFormats())
	}
}
