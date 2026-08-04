package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"sort"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func setupDoctorCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"doctor",
		cmdguard.NoFlags{},
		func(_ context.Context, cfg *AppConfig, _ cmdguard.NoFlags) error {
			actx, err := analyzer.BuildContext(cfg.Path)
			if err != nil {
				return fmt.Errorf("load packages: %w", err)
			}

			if len(actx.LoadErrors) > 0 {
				fmt.Fprintln(
					os.Stderr,
					"WARNING: package loading was partial; the profile below may be incomplete or misleading.",
				)
				for _, le := range actx.LoadErrors {
					if le.PkgPath != "" {
						fmt.Fprintf(os.Stderr, "  %s (%s):\n", le.Module, le.PkgPath)
					} else {
						fmt.Fprintf(os.Stderr, "  %s:\n", le.Module)
					}
					for _, msg := range le.Errors {
						fmt.Fprintf(os.Stderr, "    %s\n", msg)
					}
				}
				fmt.Fprintln(os.Stderr)
			}

			profile := actx.FeatureProfile

			// Show active preset and what it resolved to, so users have full
			// visibility into how their config interacts with preset defaults.
			if cfg.Preset != "" {
				presetDef := analyzer.ResolvePresetDefinition(cfg.Preset)
				fmt.Printf("Active preset: %s\n", cfg.Preset)
				if len(presetDef.Rules.Disable) > 0 {
					fmt.Printf("  Preset-disabled rules: %s\n",
						strings.Join(presetDef.Rules.Disable, ", "))
				}
				if presetDef.MinSeverity != "" {
					fmt.Printf("  Preset severity floor: %s\n", presetDef.MinSeverity)
				}
				fmt.Println()
			} else {
				fmt.Println("Active preset: (none)")
				fmt.Println()
			}

			fmt.Println("Detected go-cqrs-lite feature profile:")
			fmt.Println()
			fmt.Print(profile)
			fmt.Println()

			// In a multi-module workspace, show each module's own profile. This is
			// the key diagnostic for the per-module feature detection: a library
			// module and an examples/ module can (and should) differ. Without this
			// view a consumer cannot tell why a rule fired or didn't.
			if len(actx.FeatureProfiles) > 1 {
				fmt.Println()
				fmt.Printf("Per-module feature profiles (%d modules):\n", len(actx.FeatureProfiles))
				fmt.Println()

				type modProfile struct {
					dir     string
					profile analyzer.FeatureProfile
				}

				mods := make([]modProfile, 0, len(actx.FeatureProfiles))
				for dir, p := range actx.FeatureProfiles {
					mods = append(mods, modProfile{dir, p})
				}

				sort.Slice(mods, func(i, j int) bool {
					return len(mods[i].dir) < len(mods[j].dir)
				})

				for _, m := range mods {
					rel := m.dir
					if rel == "" {
						rel = "(root)"
					}

					fmt.Printf("=== %s ===\n", rel)
					fmt.Print(m.profile)
					fmt.Println()
				}
			}

			features := profile.ToConfigFeatures()
			raw, err := json.Marshal(
				map[string]analyzer.ConfigFeatures{"features": features},
				jsontext.WithIndentPrefix(
					"",
				), jsontext.WithIndent(
					"  ",
				),
			)
			if err != nil {
				return fmt.Errorf("marshal suggested features: %w", err)
			}

			fmt.Println("Suggested .cqrs-lint.json features section:")
			fmt.Println()
			fmt.Println(string(raw))

			// Surface any loaded rule overrides so consumers can verify their
			// config (e.g. that D002 external-api-struct-prefixes were picked
			// up) without having to re-read the JSON file.
			if len(cfg.Rules.ExternalAPIStructPrefixes) > 0 {
				rulesRaw, err := json.Marshal(
					map[string]analyzer.RulesConfig{"rules": cfg.Rules},
					jsontext.WithIndentPrefix(
						"",
					), jsontext.WithIndent(
						"  ",
					),
				)
				if err != nil {
					return fmt.Errorf("marshal rules config: %w", err)
				}

				fmt.Println()
				fmt.Println("Loaded rules overrides from .cqrs-lint.json:")
				fmt.Println()
				fmt.Println(string(rulesRaw))
			}

			// Count inline suppressions per rule so consumers see which rules
			// they are ignoring - a high suppression rate signals the heuristic
			// may need tuning.
			suppressionCounts := countSuppressions(actx)
			if len(suppressionCounts) > 0 {
				fmt.Println()
				fmt.Println("Inline suppression counts (//cqrs-lint:ignore(RULE)):")
				fmt.Println()

				for rule, count := range suppressionCounts {
					fmt.Printf("  %s: %d suppressed\n", rule, count)
				}
			}

			return nil
		},
		cmdguard.WithShort("Detect and display the project's go-cqrs-lite feature profile"),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "doctor", cmd, err)
}

// countSuppressions scans all Go files for //cqrs-lint:ignore(RULE) comments
// and returns a map of rule ID to suppression count. This helps consumers spot
// rules with high suppression rates, which signals the heuristic may need tuning.
func countSuppressions(actx *analyzer.AnalysisContext) map[string]int {
	counts := make(map[string]int)

	for _, gf := range actx.GoFiles {
		if gf.AST == nil {
			continue
		}

		for _, group := range gf.AST.Comments {
			for _, c := range group.List {
				text := c.Text
				idx := strings.Index(text, "cqrs-lint:ignore(")
				if idx < 0 {
					continue
				}

				start := idx + len("cqrs-lint:ignore(")
				end := strings.Index(text[start:], ")")
				if end < 0 {
					continue
				}

				rule := strings.TrimSpace(text[start : start+end])
				if rule != "" {
					counts[rule]++
				}
			}
		}
	}

	return counts
}
