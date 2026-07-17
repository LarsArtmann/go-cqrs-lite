package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v3/pkg/cmdguard/v3"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func setupDoctorCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand[AppConfig, cmdguard.NoFlags](
		"doctor",
		cmdguard.NoFlags{},
		func(ctx context.Context, cfg *AppConfig, _ cmdguard.NoFlags) error {
			actx, err := analyzer.BuildContext(cfg.Path)
			if err != nil {
				return fmt.Errorf("load packages: %w", err)
			}

			profile := actx.FeatureProfile

			fmt.Println("Detected go-cqrs-lite feature profile:")
			fmt.Println()
			fmt.Print(profile)
			fmt.Println()

			features := profile.ToConfigFeatures()
			raw, err := json.MarshalIndent(
				map[string]analyzer.ConfigFeatures{"features": features},
				"",
				"  ",
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
				rulesRaw, err := json.MarshalIndent(
					map[string]analyzer.RulesConfig{"rules": cfg.Rules},
					"",
					"  ",
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
	if err != nil {
		return fmt.Errorf("create doctor command: %w", err)
	}

	if err := cmdguard.AddCommand(cli, cmd); err != nil {
		return fmt.Errorf("add doctor command: %w", err)
	}

	return nil
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
