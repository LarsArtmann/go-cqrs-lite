package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
)

// funcSig is the call-shape of one package-level exported function.
type funcSig struct {
	slots    int  // argument positions (a, b int counts as 2)
	variadic bool // last position is ...
}

// parseFuncSignatures records package-level exported function signatures for
// one package directory. Methods are excluded: resolving their receiver
// types from doc snippets is out of scope for a spot-check (zero false
// positives beats recall).
func parseFuncSignatures(dir string) map[string]funcSig {
	sigs := make(map[string]funcSig)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return sigs
	}

	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() || !shouldParseFile(entry.Name()) {
			continue
		}

		file, err := parser.ParseFile(
			fset, filepath.Join(dir, entry.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() || fn.Type.Params == nil {
				continue
			}

			var sig funcSig

			for _, field := range fn.Type.Params.List {
				if len(field.Names) == 0 {
					sig.slots++
				} else {
					sig.slots += len(field.Names)
				}

				if _, isVariadic := field.Type.(*ast.Ellipsis); isVariadic {
					sig.variadic = true
				}
			}

			sigs[fn.Name.Name] = sig
		}
	}

	return sigs
}

// signatures loads (and memoizes) the function signatures of one directory.
func (r *resolver) signatures(dir string) map[string]funcSig {
	if sigs, ok := r.sigs[dir]; ok {
		return sigs
	}

	sigs := parseFuncSignatures(dir)

	r.sigs[dir] = sigs

	return sigs
}

// ensureAliasDirs loads the repo-wide package-name index once.
func (r *resolver) ensureAliasDirs() {
	if !r.aliasLoaded {
		r.loadAliasDirs()

		r.aliasLoaded = true
	}
}

// docPlaceholder stands in for doc ellipsis arguments ("...", "…") so
// abbreviated snippets still parse; the token is never valid Go itself.
const docPlaceholder = "_docPlaceholder"

var (
	placeholderArg  = regexp.MustCompile(`([(,{]\s*)(?:\.\.\.|…)(\s*[,)}])`)
	unicodeEllipsis = regexp.MustCompile(`…`)
)

// normalizePlaceholders rewrites doc-style ellipsis abbreviations into
// parseable placeholders. Standalone "..." in argument/composite-literal
// positions becomes one placeholder argument (it counts as one value);
// pass-through variadic calls ("args...") keep their ellipsis. Unicode "…"
// is never valid Go and is replaced everywhere.
func normalizePlaceholders(src string) string {
	src = unicodeEllipsis.ReplaceAllString(src, docPlaceholder)

	for prev := ""; prev != src; {
		prev = src
		src = placeholderArg.ReplaceAllString(src, "${1}"+docPlaceholder+"${2}")
	}

	return src
}
