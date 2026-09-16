package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
)

// checkArity parses every fenced Go block and compares the call shape of
// package-qualified exported functions against the real signature. Blocks
// that do not parse (fragments, pseudo-code) are skipped silently — this is
// a spot-check layered on top of doc-check's symbol validation, and a parse
// failure must never fail the gate. Only uniquely-resolvable packages are
// checked, so a wrong-package hit is impossible by construction.
func checkArity(blocks []block, res *resolver) []navIssue {
	res.ensureAliasDirs()

	issues := make([]navIssue, 0, len(blocks))

	for _, b := range blocks {
		issues = append(issues, checkBlockArity(b, res)...)
	}

	return issues
}

func checkBlockArity(b block, res *resolver) []navIssue {
	if strings.Contains(b.src, arityIgnoreDirective) {
		return nil // explicit per-block opt-out for intentional pseudo-code
	}

	src := normalizePlaceholders(b.src)

	fset := token.NewFileSet()

	file, lineOffset := parseDocSnippet(fset, src)
	if file == nil {
		return nil
	}

	srcLines := strings.Split(src, "\n")

	var issues []navIssue

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		alias, symbol, sig := res.qualifiedFuncSig(b, call)
		if sig == nil {
			return true
		}

		parsedLine := fset.Position(call.Pos()).Line

		if aritySkip(srcLines, parsedLine-lineOffset, call, file, fset) {
			return true
		}

		if msg, bad := arityMismatch(alias, symbol, sig, call); bad {
			line := b.line + parsedLine - lineOffset - 1

			issues = append(issues, navIssue{File: b.file, Line: line, Msg: msg})
		}

		return true
	})

	return issues
}

// arityIgnoreDirective opts one fenced block out of the arity spot-check
// (for deliberate pseudo-code). Place it anywhere in the block as a comment.
const arityIgnoreDirective = "doc-check:ignore-arity"

// antiPatternMarker marks deliberately wrong example calls ("// Wrong",
// "// Deprecated: ...") — these are documentation BY design, not lies.
var antiPatternMarker = regexp.MustCompile(
	`(?i)^\s*//\s*(wrong|deprecated|anti-pattern|incorrect|never|don'?t|do not)\b`)

// aritySkip reports whether a call is an intentional doc shape: an
// anti-pattern example (marker comment on the same or previous line), or a
// call whose argument list is comment-only ("f(/* publisher, ... */)") —
// the comment STANDS FOR the arguments, so no arity is checkable.
func aritySkip(
	srcLines []string, srcLine int, call *ast.CallExpr, file *ast.File, fset *token.FileSet,
) bool {
	if len(call.Args) == 0 && commentInside(call, file, fset) {
		return true
	}

	if srcLine-2 >= 0 && srcLine-2 < len(srcLines) &&
		antiPatternMarker.MatchString(srcLines[srcLine-2]) {
		return true
	}

	if srcLine-1 >= 0 && srcLine-1 < len(srcLines) &&
		antiPatternMarker.MatchString(srcLines[srcLine-1]) {
		return true
	}

	return false
}

// commentInside reports whether any comment sits between the call's parens.
func commentInside(call *ast.CallExpr, file *ast.File, fset *token.FileSet) bool {
	for _, group := range file.Comments {
		pos := fset.Position(group.Pos())
		end := fset.Position(group.End())

		if pos.Line >= fset.Position(call.Lparen).Line &&
			end.Line <= fset.Position(call.Rparen).Line {
			return true
		}
	}

	return false
}

// parseDocSnippet parses a doc fence as whole program, top-level unit, or
// wrapped function body — whichever shape the snippet is. Import statements
// mixed with statements are hoisted to a synthetic file head. The second
// result is the synthetic-line offset for doc-line mapping (already net of
// hoisted lines).
func parseDocSnippet(fset *token.FileSet, src string) (*ast.File, int) {
	wrapped := "package _doc\n\nfunc _docWrap() {\n" + src + "\n}"

	shapes := []struct {
		src string
		off int
	}{
		{src, 0},
		{"package _doc\n\n" + src, 2},
		{wrapped, 3},
	}

	if imports, rest, hoisted := hoistImports(src); hoisted > 0 {
		// package, blank, imports..., blank, func = hoisted+4 lines before
		// rest; the hoisted lines fold back into the offset (4).
		shapes = append(shapes, struct {
			src string
			off int
		}{
			"package _doc\n\n" + imports + "\n\nfunc _docWrap() {\n" + rest + "\n}", 4,
		})
	}

	for _, s := range shapes {
		file, err := parser.ParseFile(
			fset,
			"doc.go",
			s.src,
			parser.SkipObjectResolution|parser.ParseComments,
		)
		if err == nil {
			return file, s.off
		}
	}

	return nil, 0
}

// hoistImports splits leading import declarations from a snippet so
// "import + statements" fences can parse as one synthetic file.
func hoistImports(src string) (string, string, int) {
	var (
		imp   []string
		body  []string
		inImp bool
	)

	for line := range strings.SplitSeq(src, "\n") {
		t := strings.TrimSpace(line)

		switch {
		case inImp:
			imp = append(imp, line)
			if strings.Contains(t, ")") {
				inImp = false
			}
		case strings.HasPrefix(t, "import"):
			imp = append(imp, line)
			inImp = strings.Contains(t, "(")
		default:
			body = append(body, line)
		}
	}

	if len(imp) == 0 {
		return "", src, 0
	}

	return strings.Join(imp, "\n"), strings.Join(body, "\n"), len(imp)
}

// qualifiedFuncSig resolves a call's package-qualified exported function to
// its real signature: the alias must map to exactly one package (block
// import or unique repo package name), and the symbol must be a
// package-level function. Anything else returns nil (skipped).
func (r *resolver) qualifiedFuncSig(b block, call *ast.CallExpr) (string, string, *funcSig) {
	fun := call.Fun

	for unwrap := true; unwrap; {
		switch v := fun.(type) {
		case *ast.ParenExpr:
			fun = v.X
		case *ast.IndexExpr: // generic instantiation pkg.Foo[T](...)
			fun = v.X
		case *ast.IndexListExpr:
			fun = v.X
		default:
			unwrap = false
		}
	}

	sel, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return "", "", nil
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || !sel.Sel.IsExported() || isStdlibOrBuiltin(ident.Name) {
		return "", "", nil
	}

	alias := ident.Name

	var dir string

	switch paths := r.blockPaths(b, alias); {
	case len(paths) == 1:
		dir = r.fullDir(paths[0])
	case len(paths) == 0 && len(r.aliasDirs[alias]) == 1:
		dir = filepath.Join(r.repoRoot, r.aliasDirs[alias][0])
	default:
		return "", "", nil // ambiguous or external: stay silent
	}

	if sig, ok := r.signatures(dir)[sel.Sel.Name]; ok {
		return alias, sel.Sel.Name, &sig
	}

	return "", "", nil // type, var, or method: not an arity target
}

// arityMismatch validates one call against its signature.
func arityMismatch(alias, symbol string, sig *funcSig, call *ast.CallExpr) (string, bool) {
	count := len(call.Args)

	required := sig.slots

	if sig.variadic {
		required--
	}

	name := alias + "." + symbol

	switch {
	case call.Ellipsis.IsValid() && !sig.variadic:
		return fmt.Sprintf("arity: %s(...): spread call but signature takes exactly %d arg(s)",
			name, sig.slots), true
	case count < required:
		return fmt.Sprintf("arity: %s called with %d arg(s), signature wants %s",
			name, count, sigShape(sig)), true
	case !sig.variadic && count > sig.slots:
		return fmt.Sprintf("arity: %s called with %d arg(s), signature wants %s",
			name, count, sigShape(sig)), true
	}

	return "", false
}

func sigShape(sig *funcSig) string {
	if sig.variadic {
		return fmt.Sprintf("at least %d", sig.slots-1)
	}

	return fmt.Sprintf("exactly %d", sig.slots)
}
