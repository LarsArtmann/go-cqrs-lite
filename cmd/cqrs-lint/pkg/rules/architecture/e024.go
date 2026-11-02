package architecture

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// E024: branded-ID .String() outside display contexts.
//
// cbid.ID-backed IDs whose marker implements BrandNamer render "Brand:value"
// in String() (id.StreamID → "StreamMarker:01J…", scheduling.TimerID →
// "TimerMarker:lock:…"), and id.ActorID's String() drops the actor kind
// entirely. Storing those display forms in keys, columns, or wire payloads
// breaks read-back — the bug class that cost 7+ production fixes (PG snapshot
// binds, typed-decoder lookups, the StreamKey 7th write path; disk-truth plan
// v2 A10). Display positions are whitelisted by design: slog/log arguments,
// log attributes, fmt.Errorf text, and the documented ParseStreamID round-trip
// law. Warn-first per the G4 pattern — hard only at v5. Silent without type
// information (a plain string's .String() is indistinguishable syntactically)
// and silent for cbid instances whose marker lacks Name(), where String()
// already returns the bare value.
//
//nolint:ireturn // factory returns public interface
func NewE024Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E024-branded-id-string-display-form",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest || gf.AST == nil {
					continue
				}

				ranges := e024DisplayRanges(gf)
				e024CollectStringCalls(ctx, gf, ranges, &findings)
			}

			return findings, nil
		},
	)
}

// tokenRange is one whitelisted display-expression span.
type tokenRange struct{ start, end token.Pos }

// contains reports whether pos falls inside the span (inclusive bounds).
func (r tokenRange) contains(pos token.Pos) bool { return pos >= r.start && pos <= r.end }

// e024DisplayRanges records the source spans of arguments passed to
// display-context calls (logs, log attributes, error text, ID re-parsing).
// A .String() call inside any recorded span is display by construction.
func e024DisplayRanges(gf *analyzer.GoFile) []tokenRange {
	var ranges []tokenRange

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if !e024IsDisplayCall(gf, call) {
			return true
		}

		for _, arg := range call.Args {
			ranges = append(ranges, tokenRange{start: arg.Pos(), end: arg.End()})
		}

		return true
	})

	return ranges
}

// e024CollectStringCalls flags branded-receiver .String() calls that sit
// outside every display range.
func e024CollectStringCalls(
	ctx *analyzer.AnalysisContext,
	gf *analyzer.GoFile,
	ranges []tokenRange,
	findings *[]finding.Finding,
) {
	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "String" {
			return true
		}

		kind, name := e024BrandedStringKind(gf, sel)
		if kind == e024BrandedNone {
			return true
		}

		pos := call.Pos()
		for _, r := range ranges {
			if r.contains(pos) {
				return true
			}
		}

		if f, err := e024Finding(ctx, gf, call, kind, name); err == nil {
			*findings = append(*findings, f)
		}

		return true
	})
}

// e024IsDisplayCall reports whether call writes text for humans (or re-parses
// an ID per the documented round-trip law), making .String() the intent.
func e024IsDisplayCall(gf *analyzer.GoFile, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	if path, resolved := analyzer.ResolveQualifierTyped(gf, ident); resolved {
		return e024IsDisplayImportPath(path, sel.Sel.Name)
	}

	return e024DisplayQualifierFallback(ident.Name, sel.Sel.Name)
}

// e024IsDisplayImportPath classifies a resolved package path + callee name.
func e024IsDisplayImportPath(path, name string) bool {
	switch path {
	case "log", "log/slog":
		return true
	}

	last := path
	if _, after, ok := strings.CutLast(path, "/"); ok {
		last = after
	}

	if last == "attribute" { // go.opentelemetry.io/otel/attribute and peers
		return true
	}

	if strings.HasPrefix(name, "Parse") &&
		analyzer.PathContainsFragment(path, "go-cqrs-lite/id") {
		return true // ParseStreamID/ParseActorID strip the display prefix
	}

	return false
}

// e024DisplayQualifierFallback is the syntax-only twin of the path check,
// used when the type checker is unavailable (qualifier names only).
func e024DisplayQualifierFallback(identName, selName string) bool {
	switch identName {
	case "slog", "log", "attribute":
		return true
	case "fmt":
		return selName == "Errorf"
	}

	return false
}

// e024BrandedKind partitions the flagged receiver families.
type e024BrandedKind int

const (
	e024BrandedNone e024BrandedKind = iota
	e024BrandedCBID
	e024BrandedActor
)

// e024BrandedStringKind resolves the .String() receiver's type: a cbid.ID
// instance with a BrandNamer marker (display form is "Marker:value") or an
// id.ActorID (String() drops the kind). Everything else — including cbid
// instances whose marker has no Name(), where String() already returns the
// bare value — is not flagged.
func e024BrandedStringKind(gf *analyzer.GoFile, sel *ast.SelectorExpr) (e024BrandedKind, string) {
	if gf == nil || gf.Pkg == nil || gf.Pkg.TypesInfo == nil {
		return e024BrandedNone, ""
	}

	tv, ok := gf.Pkg.TypesInfo.Types[sel.X]
	if !ok || tv.Type == nil {
		return e024BrandedNone, ""
	}

	t := tv.Type
	if ptr, isPtr := t.(*types.Pointer); isPtr {
		t = ptr.Elem()
	}

	named, isNamed := t.(*types.Named)
	if !isNamed || named.Obj() == nil || named.Obj().Pkg() == nil {
		return e024BrandedNone, ""
	}

	obj := named.Obj()
	pkgPath := obj.Pkg().Path()

	if obj.Name() == "ID" && analyzer.PathContainsFragment(pkgPath, "go-branded-id") {
		return e024CBIDKind(named)
	}

	if obj.Name() == "ActorID" && analyzer.PathContainsFragment(pkgPath, "go-cqrs-lite/id") {
		return e024BrandedActor, "ActorID"
	}

	return e024BrandedNone, ""
}

// e024CBIDKind classifies one cbid.ID instance by its marker type argument.
func e024CBIDKind(named *types.Named) (e024BrandedKind, string) {
	if named.TypeArgs().Len() == 0 {
		return e024BrandedNone, ""
	}

	marker := named.TypeArgs().At(0)
	if !e024MarkerImplementsBrandNamer(marker) {
		return e024BrandedNone, ""
	}

	return e024BrandedCBID, e024TypeName(marker)
}

// brandNamerIface is the { Name() string } marker contract from go-branded-id;
// only markers implementing it make String() a prefixed display form.
//
//nolint:gochecknoglobals // read-only type-level constant
var brandNamerIface = types.NewInterfaceType([]*types.Func{
	types.NewFunc(
		token.NoPos,
		nil,
		"Name",
		types.NewSignatureType(
			nil, nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false,
		),
	),
}, nil).Complete()

// e024MarkerImplementsBrandNamer accepts value- and pointer-receiver markers.
func e024MarkerImplementsBrandNamer(t types.Type) bool {
	if t == nil {
		return false
	}

	return types.Implements(t, brandNamerIface) ||
		types.Implements(types.NewPointer(t), brandNamerIface)
}

// e024TypeName renders a short marker name for the finding message.
func e024TypeName(t types.Type) string {
	if named, ok := t.(*types.Named); ok && named.Obj() != nil {
		return named.Obj().Name()
	}

	return "?"
}

// e024Finding builds the finding for one flagged .String() call, anchored at
// the call site where the fix lands.
func e024Finding(
	ctx *analyzer.AnalysisContext,
	gf *analyzer.GoFile,
	call *ast.CallExpr,
	kind e024BrandedKind,
	name string,
) (finding.Finding, error) {
	pos := ctx.Fset.Position(call.Pos())

	msg, suggestion := e024CBIDMessage(name)
	if kind == e024BrandedActor {
		msg, suggestion = e024ActorMessage()
	}

	return findingTemplate.Builder(
		"E024",
		msg,
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(gf.Path), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryStructure).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion(suggestion).
		WithSnippet(ctx.SourceLine(gf.Path, pos.Line)).
		Build()
}

// e024CBIDMessage words the cbid.ID finding.
func e024CBIDMessage(marker string) (msg, suggestion string) {
	return fmt.Sprintf(
			"%s-branded ID: .String() renders the display form (%s:<value>) — bare value required here",
			marker, marker,
		),
		"Use .Get() for the storage/key/wire value (or MarshalText); keep .String() for " +
			"logs and error text (whitelisted there by design); dismiss genuine display uses " +
			"with //cqrs-lint:ignore(E024) <reason> or a .cqrs-lint.json scorecard waiver"
}

// e024ActorMessage words the id.ActorID finding.
func e024ActorMessage() (msg, suggestion string) {
	return "id.ActorID: .String() drops the actor kind — lossy outside display contexts",
		"Use PrefixedString() at SQL/boundary serialization (read back with ParseActorID) " +
			"or carry the typed id.ActorID; logs and error text are whitelisted; dismiss " +
			"genuine display uses with //cqrs-lint:ignore(E024) <reason>"
}
