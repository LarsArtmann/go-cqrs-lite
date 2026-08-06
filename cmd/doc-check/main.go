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
	"regexp"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/spf13/cobra"
)

const repoImportPrefix = "github.com/larsartmann/go-cqrs-lite/"

var errBrokenReferences = errors.New("broken documentation reference(s)")

type ref struct {
	pkg    string
	symbol string
	file   string
	line   int
}

type AppConfig struct {
	cmdguard.Config
}

func main() {
	cli, err := cmdguard.NewCLI(
		"doc-check",
		"Verify Go import paths and qualified symbols in documentation files",
		AppConfig{Config: cmdguard.Config{}}, //nolint:exhaustruct // defaults are fine for this one-shot CLI
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
	rootCmd.RunE = func(_ *cobra.Command, args []string) error {
		return run(args)
	}

	cli.ExecuteAndExit(context.Background())
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

	var allRefs []ref

	var allImports []string

	for _, file := range files {
		refs, imports, err := scanMarkdown(file)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", file, err)
		}

		allRefs = append(allRefs, refs...)
		allImports = append(allImports, imports...)
	}

	// Build package export index from cqrs-lite imports.
	exportIndex := buildExportIndex(allImports, repoRoot)

	// Verify references.
	broken := 0

	for _, r := range allRefs {
		if _, ok := exportIndex[r.pkg]; !ok {
			continue // external package, skip
		}

		if !exportIndex[r.pkg][r.symbol] {
			log.Printf("  ✗ %s:%d: %s.%s not found", r.file, r.line, r.pkg, r.symbol)

			broken++
		}
	}

	if broken > 0 {
		return fmt.Errorf("%w: %d broken reference(s) found", errBrokenReferences, broken)
	}

	if len(allRefs) == 0 {
		log.Printf( //nolint:lll // CLI tool, no untrusted input
			"⚠  WARNING: 0 Go references found — no fenced ```go code blocks detected.\n" +
				"Documents were NOT verified. Add a verification code block or pass files with Go samples.",
		)

		return nil
	}

	log.Printf( //nolint:lll
		"✓ All %d references valid across %d package(s).",
		len(allRefs), len(exportIndex),
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

var (
	goBlockRe = regexp.MustCompile("(?s)```go\n(.*?)```")
	importRe  = regexp.MustCompile(`"(` + regexp.QuoteMeta(repoImportPrefix) + `[^"]+)"`)
	refRe     = regexp.MustCompile(`\b([a-z][a-z0-9]*)\.([A-Z][A-Za-z0-9]*)\b`)
)

func scanMarkdown(path string) ([]ref, []string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // tool exit
	}

	content := string(data)

	var refs []ref

	var imports []string

	lineNum := 0

	for _, match := range goBlockRe.FindAllStringSubmatchIndex(content, -1) {
		blockStart := match[2]
		blockEnd := match[3]
		block := content[blockStart:blockEnd]

		// Approximate line number of block start.
		lineNum += strings.Count(content[:blockStart], "\n") + 1

		// Extract imports.
		for _, imp := range importRe.FindAllStringSubmatch(block, -1) {
			imports = append(imports, imp[1])
		}

		// Extract qualified references.
		for _, refMatch := range refRe.FindAllStringSubmatch(block, -1) {
			pkgAlias := refMatch[1]
			symbol := refMatch[2]

			// Skip common non-package prefixes.
			if isStdlibOrBuiltin(pkgAlias) {
				continue
			}

			refs = append(refs, ref{
				pkg:    pkgAlias,
				symbol: symbol,
				file:   path,
				line:   lineNum,
			})
		}
	}

	return refs, imports, nil
}

func isStdlibOrBuiltin(alias string) bool {
	skip := map[string]bool{
		"fmt": true, "os": true, "time": true, "sync": true,
		"context": true, "errors": true, "strings": true, "strconv": true,
		"log": true, "testing": true, "bytes": true, "io": true,
		"json": true, "database": true, "sql": true, "net": true,
		"http": true, "reflect": true, "sort": true, "math": true,
		"filepath": true, "regexp": true, "slog": true, "rand": true,
		"otel":         true,
		"grpc":         true,
		"pebble":       true,
		"projection":   true,
		"turso":        true,
		"asyncapi":     true,
		"openapi":      true,
		"eventcatalog": true,
		"d2":           true,
	}

	return skip[alias]
}
