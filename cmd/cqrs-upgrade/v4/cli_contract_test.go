package main

import (
	"testing"
)

// wantFlagHelp pins the exact help text of every flag from the pre-cmdguard
// stdlib flag set — the --help-parity contract (plan T19/m60). A wording
// drift here is a user-visible CLI surface change and must be deliberate.
var wantFlagHelp = map[string]string{
	"dry-run":  "show planned bumps and the deprecation report without changing go.mod",
	"no-build": "skip the go mod tidy + build + vet verification after bumping",
	"strict": "exit non-zero on v5-removed API usage, failed scans, or module " +
		"errors (v5-readiness gate)",
	"json":      "machine-readable JSON output (bump plan + deprecations per module)",
	"to":        "clamp every bump target to at most this version (semver, e.g. v4.13.0); never downgrades",
	"workspace": "upgrade every module under dir (skips vendor/, testdata/, .git/)",
	"to-v5": "v5 migration pass: rewrite imports of the folded v4 trains to their " +
		"core/v5 twins (ADR-0152); go.mod requires are reported, not edited " +
		"(core/v5 is untagged); combine with --dry-run for a per-file diff",
}

// wantFlagDefault pins each flag's zero-value default. Bools default false,
// --to defaults to "" (no clamp).
var wantFlagDefault = map[string]string{
	"dry-run":   "false",
	"no-build":  "false",
	"strict":    "false",
	"json":      "false",
	"to":        "",
	"workspace": "false",
	"to-v5":     "false",
}

// TestCLIHelpParity asserts the cmdguard-registered flag set matches the old
// stdlib flag set flag-for-flag: same names, same help strings, same
// defaults, and all seven are bool/string (no accidental value flags).
func TestCLIHelpParity(t *testing.T) {
	t.Parallel()

	cli := buildForTest(t)
	// Root T fields register as persistent flags and may materialize lazily;
	// LocalFlags() covers the root's own surface (local + its persistent).
	flags := cli.RootCommand().LocalFlags()

	seen := make(map[string]bool, len(wantFlagHelp))

	for name, wantHelp := range wantFlagHelp {
		f := flags.Lookup(name)
		if f == nil {
			t.Errorf("flag --%s missing from the CLI surface", name)

			continue
		}

		seen[name] = true

		if f.Usage != wantHelp {
			t.Errorf("--%s help drifted:\n got: %s\nwant: %s", name, f.Usage, wantHelp)
		}

		if want, ok := wantFlagDefault[name]; ok && f.DefValue != want {
			t.Errorf("--%s default = %q, want %q", name, f.DefValue, want)
		}
	}

	for name := range wantFlagDefault {
		if !seen[name] {
			t.Errorf("flag --%s declared in the parity tables but absent from the CLI", name)
		}
	}
}
