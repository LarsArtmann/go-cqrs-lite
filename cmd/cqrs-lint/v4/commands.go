package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	output "github.com/larsartmann/go-output"
)

// registerCommand wraps the create-and-add pattern shared by every subcommand:
// it surfaces NewCommand failures as "create <name> command" and AddCommand
// failures as "add <name> command".
func registerCommand[F any](
	cli *cmdguard.CLI[AppConfig],
	name string,
	cmd cmdguard.Command[AppConfig, F],
	err error,
) error {
	if err != nil {
		return fmt.Errorf("create %s command: %w", name, err)
	}
	if err := cmdguard.AddCommand(cli, cmd); err != nil {
		return fmt.Errorf("add %s command: %w", name, err)
	}
	return nil
}

func setupRulesCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"rules",
		rulesFlags{},
		func(_ context.Context, cfg *AppConfig, flags rulesFlags) error {
			format, err := rulesFormat(flags, cfg)
			if err != nil {
				return err
			}

			out, err := renderRules(format, parseColorMode(cfg.Color))
			if err != nil {
				return err
			}

			fmt.Print(out)

			return nil
		},
		cmdguard.WithShort(withFormatsSuffix("List all available rules", formatsRules)),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "rules", cmd, err)
}

// rulesFormat resolves the rules command's output format. The legacy
// --json/--markdown booleans win when set; otherwise the shared --format
// flag (or config file) decides — the same vocabulary as every other
// command, restricted to the formats the rules catalog supports.
func rulesFormat(flags rulesFlags, cfg *AppConfig) (string, error) {
	switch {
	case flags.Markdown:
		return "markdown", nil
	case flags.JSON:
		return "json", nil
	}

	switch f := strings.ToLower(strings.TrimSpace(cfg.Format)); f {
	case "", "text":
		return "text", nil
	case "json", "markdown":
		return f, nil
	default:
		return "", validateFormatFlag(cfg.Format, formatsRules...)
	}
}

// renderRules renders the rule catalog in the given format. Each branch
// reproduces the exact byte layout the previous per-format print calls
// produced (Println for json/table, raw Print for markdown) so redirected
// output such as `rules --markdown > RULES.md` stays identical.
func renderRules(format string, colorMode output.ColorMode) (string, error) {
	switch format {
	case "json":
		out, err := renderRulesJSON()
		if err != nil {
			return "", fmt.Errorf("render rules json: %w", err)
		}

		return out + "\n", nil
	case "markdown":
		return renderRulesMarkdown(), nil
	default:
		out, err := renderRulesTable(colorMode)
		if err != nil {
			return "", fmt.Errorf("render rules: %w", err)
		}

		return out + "\n", nil
	}
}

// rulesFlags carries the rules subcommand's output-format flag.
type rulesFlags struct {
	JSON     bool `default:"false" flag:"json"     help:"Emit the catalog as JSON for tooling consumers"`
	Markdown bool `default:"false" flag:"markdown" help:"Emit the anchored RULES.md doc page"`
}

func setupVersionCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"version",
		versionFlags{},
		func(_ context.Context, _ *AppConfig, flags versionFlags) error {
			if flags.Verbose {
				fmt.Println(versionVerbose())
			} else {
				fmt.Println(versionString())
			}
			return nil
		},
		cmdguard.WithShort("Print version"),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "version", cmd, err)
}

// versionFlags adds --verbose to the version subcommand.
type versionFlags struct {
	Verbose bool `default:"false" flag:"verbose" help:"Show Go version, OS/arch, and module path"`
}

// versionVerbose returns the full version string with build environment details.
func versionVerbose() string {
	var b strings.Builder
	b.WriteString(versionString())
	b.WriteString("\n  go:      ")
	b.WriteString(runtime.Version())
	b.WriteString("\n  arch:    ")
	b.WriteString(runtime.GOOS + "/" + runtime.GOARCH)
	b.WriteString("\n  module:  github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4")
	return b.String()
}

func setupChangelogCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"changelog",
		cmdguard.NoFlags{},
		func(ctx context.Context, _ *AppConfig, _ cmdguard.NoFlags) error {
			result, err := computeChangelog(ctx, resolvedVersion())
			if err != nil {
				return err
			}

			if result.fallback {
				fmt.Fprintf(os.Stderr,
					"no release tag cmd/cqrs-lint/v%s yet — showing the last 20 commits instead\n",
					resolvedVersion())
			}

			fmt.Print(result.commits)
			return nil
		},
		cmdguard.WithShort("Print changelog (commits since last release tag)"),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "changelog", cmd, err)
}

// changelogResult carries the git log output plus whether the release-tag
// range was unusable (tag missing or git refused) and the last-20 fallback
// ran — the caller turns that into an honest stderr notice instead of
// silently truncating.
type changelogResult struct {
	commits  string
	fallback bool
}

// computeChangelog resolves the changelog for the module's release tag:
// commits since cmd/cqrs-lint/v<version>..HEAD. When that range fails
// (typically: the tag does not exist yet for a dev build), it falls back to
// the last 20 commits and reports fallback=true.
func computeChangelog(ctx context.Context, version string) (changelogResult, error) {
	tag := "cmd/cqrs-lint/v" + version

	out, err := exec.CommandContext(ctx, "git", "log", "--oneline", tag+"..HEAD").Output()
	if err == nil {
		return changelogResult{commits: string(out)}, nil
	}

	tagPresent := exec.CommandContext(
		ctx, "git", "rev-parse", "--verify", "--quiet", "refs/tags/"+tag,
	).Run() == nil

	out, err = exec.CommandContext(ctx, "git", "log", "--oneline", "-20").Output()
	if err != nil {
		return changelogResult{}, fmt.Errorf("git log: %w", err)
	}

	return changelogResult{commits: string(out), fallback: !tagPresent}, nil
}

// versionString returns the full version string, including commit hash and
// build date when they were injected via ldflags. Local `go build` runs
// produce a bare "cqrs-lint X.Y.Z"; Nix builds include provenance.
func versionString() string {
	var parts []string

	if commitHash != "" {
		parts = append(parts, "commit: "+commitHash)
	}

	if buildDate != "" {
		parts = append(parts, "built: "+buildDate)
	}

	if dep := moduleVersion("github.com/larsartmann/go-finding"); dep != "" {
		parts = append(parts, "go-finding: "+dep)
	}

	if len(parts) == 0 {
		return "cqrs-lint " + resolvedVersion()
	}

	return fmt.Sprintf("cqrs-lint %s (%s)", resolvedVersion(), strings.Join(parts, ", "))
}
