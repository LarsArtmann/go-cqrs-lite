package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// rebindForeignCwdConfig undoes the one cwd-sensitivity in cqrs-lint's config
// binding: cmdguard loads .cqrs-lint.json from the PROCESS CWD (single
// relative path, no ancestor walk), so analyzing a project from an unrelated,
// config-carrying directory silently applies the FOREIGN project's preset,
// features, rules, health, and scorecard keys to the target. The concrete
// trap (2026-11-20): this library's own root config sets `preset: library`,
// which disables F030/V007 — every consumer linted from ~/projects/go-cqrs-lite
// looked clean while carrying deleted-module imports, and the tool was blamed
// for ~90 minutes before the invocation was.
//
// When the lint target lies OUTSIDE the cwd (a sibling checkout, not a
// monorepo subdirectory — ancestor configs legitimately govern descendants),
// the five config-file-only project keys are re-bound to what running from
// the target directory would produce: the target's own .cqrs-lint.json when
// it has one, tag defaults when it does not. Flag-carrying keys (format,
// color, min-severity, typed-info, ...) are untouched — no flag exists for
// the re-bound keys, so an explicit CLI value can never be clobbered.
func rebindForeignCwdConfig(w io.Writer, cfg *AppConfig) {
	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	if !pathOutsideDir(absPath, cwd) {
		return
	}

	if loadConfigFileBytes(analyzer.ConfigFileName) == nil {
		return
	}

	foreignPreset := string(cfg.Preset)

	cfg.Preset = ""
	cfg.Features = analyzer.ConfigFeatures{}
	cfg.Rules = analyzer.RulesConfig{}
	cfg.Health = HealthConfig{}
	cfg.ScorecardSettings = analyzer.ScorecardSettings{}

	target := AppConfig{}
	source := "tag defaults"
	if data := loadConfigFileBytes(filepath.Join(absPath, analyzer.ConfigFileName)); data != nil {
		if _, err := (JSONCLoader{}).Load(data, &target); err != nil {
			fmt.Fprintf(w,
				"cqrs-lint: warning: config was bound from the process cwd (%s), not from --path (%s),\n"+
					"and the target's own %s could not be parsed: %v\n"+
					"The foreign project keys were dropped (running on tag defaults). Fix the target\n"+
					"config, or re-run from %s.\n",
				cwd, absPath, analyzer.ConfigFileName, err, absPath,
			)
			return
		}

		cfg.Preset = target.Preset
		cfg.Features = target.Features
		cfg.Rules = target.Rules
		cfg.Health = target.Health
		cfg.ScorecardSettings = target.ScorecardSettings
		source = "the target's own " + analyzer.ConfigFileName
	}

	fmt.Fprintf(w,
		"cqrs-lint: warning: config was bound from the process cwd (%s), not from --path (%s).\n"+
			"The cwd's project keys (preset %q, features, rules, health, scorecard) would have silently\n"+
			"applied to a directory outside it. Re-bound to %s instead. Re-run from %s\n"+
			"(or pass --config %s) for fully native behavior.\n",
		cwd, absPath, foreignPreset, source, absPath,
		filepath.Join(absPath, analyzer.ConfigFileName),
	)
}

// pathOutsideDir reports whether path is neither equal to dir nor inside it.
func pathOutsideDir(path, dir string) bool {
	if path == dir {
		return false
	}

	return !strings.HasPrefix(path, dir+string(filepath.Separator))
}
