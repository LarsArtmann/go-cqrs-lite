package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ref is one qualified pkg.Symbol reference extracted from a doc code block.
type ref struct {
	pkg    string
	symbol string
	file   string
	line   int
}

// block is one fenced ```go code block with its own imports and references.
// References are resolved against the block's OWN imports first, so repo
// packages sharing a package name (e.g. scheduling/sqlstore and
// idempotency/sqlstore) cannot contaminate each other's verification.
type block struct {
	file    string
	line    int
	imports []string
	refs    []ref
}

var (
	goBlockRe = regexp.MustCompile("(?s)```go\n(.*?)```")
	importRe  = regexp.MustCompile(`"(` + regexp.QuoteMeta(repoImportPrefix) + `[^"]+)"`)
	refRe     = regexp.MustCompile(`\b([a-z][a-z0-9]*)\.([A-Z][A-Za-z0-9]*)\b`)
)

// scanMarkdownBlocks parses one markdown file into per-block import/ref sets.
func scanMarkdownBlocks(path string) ([]block, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err //nolint:wrapcheck // tool exit
	}

	content := string(data)

	var blocks []block

	for _, match := range goBlockRe.FindAllStringSubmatchIndex(content, -1) {
		blockStart, blockEnd := match[2], match[3]
		raw := content[blockStart:blockEnd]

		blocks = append(blocks, parseBlock(raw, path, blockLine(content, blockStart)))
	}

	return blocks, nil
}

// blockLine returns the 1-based source line of a byte offset.
func blockLine(content string, offset int) int {
	return strings.Count(content[:offset], "\n") + 1
}

// parseBlock extracts imports and qualified references from one code block.
func parseBlock(raw, file string, line int) block {
	b := block{file: file, line: line} //nolint:exhaustruct_v5 // imports/refs are appended below

	for _, imp := range importRe.FindAllStringSubmatch(raw, -1) {
		b.imports = append(b.imports, imp[1])
	}

	for _, refMatch := range refRe.FindAllStringSubmatch(raw, -1) {
		pkgAlias, symbol := refMatch[1], refMatch[2]
		if isStdlibOrBuiltin(pkgAlias) {
			continue
		}

		b.refs = append(b.refs, ref{pkg: pkgAlias, symbol: symbol, file: file, line: line})
	}

	return b
}

// isStdlibOrBuiltin reports whether an alias should never be resolved against
// repo packages (stdlib names, well-known external aliases, local variables).
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
