// Package main implements doc-check: a tool that verifies Go import paths and
// qualified symbol references in documentation files actually exist in the
// codebase.
//
// It scans markdown files for Go code blocks, extracts import paths and
// qualified references (e.g. storage.NewSQLiteViewStore, kv.ViewStore), and
// verifies:
//
//  1. Every cqrs-lite import path maps to a real directory with a go.mod.
//  2. Every qualified symbol reference (pkg.Symbol) exists as an exported
//     declaration in that package.
//
// Usage:
//
//	go run ./cmd/doc-check/ [files...]
//
// Defaults to SKILL.md, AGENTS.md, and any .agents/skills/*/references/*.md
// if no files are given.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/spf13/cobra"
)

const repoImportPrefix = "github.com/larsartmann/go-cqrs-lite/"

var (
	errBrokenReferences = errors.New("broken documentation reference(s)")
	errWarningsFound    = errors.New("doc-check warning(s) found")
	errNoReferences     = errors.New("no Go references found")
)

type AppConfig struct {
	cmdguard.Config
}

func main() {
	cli, err := cmdguard.NewCLI(
		"doc-check",
		"Verify Go import paths and qualified symbols in documentation files",
		//nolint:exhaustruct_v5 // defaults are fine for this one-shot CLI
		AppConfig{
			Config: cmdguard.Config{},
		},
		cmdguard.WithCLILong(
			"doc-check scans markdown files for Go code blocks, extracts import paths and "+
				"qualified references (e.g. storage.NewSQLiteViewStore), and verifies they exist in the codebase.\n\n"+
				"Defaults to SKILL.md, AGENTS.md, and .agents/skills/*/references/*.md if no files are given.",
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CLI: %v\n", err)
		os.Exit(1)
	}

	rootCmd := cli.RootCommand()
	rootCmd.Use = "doc-check [files...]"
	rootCmd.Args = fileArgs
	rootCmd.RunE = func(_ *cobra.Command, args []string) error {
		return run(args)
	}

	cli.ExecuteAndExit(context.Background())
}

// fileArgs is a cobra.PositionalArgs validator that accepts zero or more
// markdown file paths. Zero args triggers auto-discovery. Each provided arg
// must be an existing file with a .md extension.
func fileArgs(_ *cobra.Command, args []string) error {
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return fmt.Errorf("argument %q: %w", arg, err)
		}

		if info.IsDir() {
			return fmt.Errorf("argument %q is a directory, not a file", arg)
		}

		if !strings.HasSuffix(arg, ".md") {
			return fmt.Errorf("argument %q is not a markdown file (.md)", arg)
		}
	}

	return nil
}

func run(files []string) error {
	if len(files) == 0 {
		// Auto-discover from the repo root so the tool works regardless of CWD
		// (cmd/doc-check is its own module, so it's often run from inside cmd/doc-check/).
		root := findRepoRoot()
		files = []string{
			filepath.Join(root, "SKILL.md"),
			filepath.Join(root, "AGENTS.md"),
			filepath.Join(root, "docs/DOMAIN_LANGUAGE.md"),
			filepath.Join(root, "docs/METAENGINE_DOMAIN_LANGUAGE.md"),
		}
		// Auto-discover skill reference files so split SKILL.md content stays checked.
		if refFiles, err := filepath.Glob(
			filepath.Join(root, ".agents/skills/*/references/*.md"),
		); err == nil {
			files = append(files, refFiles...)
		}
	}

	// Resolve repo root by walking up from the first file's directory.
	repoRoot := findRepoRootFromPath(filepath.Dir(files[0]))

	var allBlocks []block

	var allImports []string

	for _, file := range files {
		blocks, err := scanMarkdownBlocks(file)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", file, err)
		}

		for _, b := range blocks {
			allImports = append(allImports, b.imports...)
		}

		allBlocks = append(allBlocks, blocks...)
	}

	// References resolve against each block's own imports first, then the
	// repo-wide package-name index; same-named packages cannot cross-resolve.
	res := newResolver(repoRoot)

	broken, totalRefs, warnings := verifyBlocks(allBlocks, allImports, res)

	if broken > 0 {
		return fmt.Errorf("%w: %d broken reference(s) found", errBrokenReferences, broken)
	}

	// 0-warning tripwire: doc-check warnings (unreadable dirs, empty package
	// exports, unparseable files) have been at zero since 2026-08-15. Gate the
	// count at zero so warning spam cannot silently creep back — a warning
	// means a document verified against a MISSING package index, i.e. it was
	// not really verified.
	if len(warnings) > 0 {
		for _, w := range warnings {
			log.Printf("  ⚠ %s", w)
		}

		return fmt.Errorf(
			"%w: %d warning(s) — fix them or extend the skip-list",
			errWarningsFound,
			len(warnings),
		)
	}

	if totalRefs == 0 {
		return fmt.Errorf( //nolint:lll // CLI tool, no untrusted input
			"%w: no fenced ```go code blocks detected — documents were NOT verified. "+
				"Add a verification code block or pass files with Go samples",
			errNoReferences,
		)
	}

	log.Printf( //nolint:lll
		"✓ All %d references valid across %d package(s).",
		totalRefs, len(res.clauses),
	)

	return nil
}

// findRepoRoot walks up from the working directory to the nearest directory
// containing a .git marker. Falls back to "." if none is found.
func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	return findRepoRootFromPath(dir)
}

// findRepoRootFromPath walks up from the given directory to the nearest .git marker.
func findRepoRootFromPath(start string) string {
	dir := start

	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return dir
		}
		// .git can also be a file (worktrees); accept either.
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return "."
}
