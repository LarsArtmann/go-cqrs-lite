package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v3/pkg/cmdguard/v3"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func setupDoctorCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand[AppConfig, cmdguard.NoFlags](
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

			fmt.Println("Detected go-cqrs-lite feature profile:")
			fmt.Println()
			fmt.Print(profile)
			fmt.Println()

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
