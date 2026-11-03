package correctness

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// C044: branded-ID .String() outside display contexts. `cbid.ID.String()`
// (and the id-package display overrides) render the PREFIXED display form
// ("StreamMarker:<raw>"); the wire/storage form is the bare value (.Get(),
// MarshalText, driver.Valuer). Every display-form leak shipped as a bug was
// a `.String()` call on a branded receiver in a storage, key, cursor, or
// SQL-argument position (scheduling/sqlstore queries, system typed-decoder
// lookups, the getting-started example, the meta_stream_log stream-key path,
// PG snapshot binds — 2026-08..10). This rule flags those calls WARN-first:
// the display intent is legitimate in logging and message-formatting
// positions, so those are whitelisted (arguments of display calls are
// whitelisted at ANY nesting depth); hard severity waits for v5.
//
// Precision carve-outs (2026-11-03 strengthening, FP control):
//   - cbid.ID instances whose marker does NOT implement BrandNamer render the
//     BARE value in String() — not flagged (id.EventID and the ULID family).
//   - id/v4 benign passthroughs (StreamType.String, ActorKind.String) — not
//     flagged; the lossy id overrides (ActorID drops the kind, StreamRef
//     composes the display form) ARE flagged with tailored guidance.
//   - id Parse*ID(s.String()) re-parses per the documented round-trip law —
//     whitelisted.
//   - log-attribute packages (log/slog, otel attribute) — whitelisted.
//
//nolint:ireturn // factory returns public interface
func NewC044Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C044-branded-id-display-string",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if gf.Pkg == nil || gf.Pkg.TypesInfo == nil || gf.AST == nil {
					continue
				}

				findings = append(findings, detectBrandedStringCalls(ctx, gf)...)
			}

			return findings, nil
		},
	)
}

// displayContextCalleeNames are the enclosing-call method names under which
// a branded .String() is display intent (logging, structured attributes,
// message formatting) rather than a wire/storage leak.
var displayContextCalleeNames = map[string]bool{ //nolint:gochecknoglobals // static lookup table
	// log/slog family
	"Info": true, "Infof": true, "Warn": true, "Warnf": true, "Warning": true,
	"Error": true, "Errorf": true, "Debug": true, "Debugf": true,
	"Log": true, "Logf": true, "LogAttrs": true,
	// stdlib log family
	"Printf": true, "Print": true, "Println": true, "Fatalf": true, "Fatal": true,
	"Panicf": true, "Panic": true,
	// fmt display formatting (Sprintf/Errorf render for humans)
	"Sprintf": true, "Fprintf": true,
	// slog attribute construction
	"Attr": true, "String": true, "Any": true,
}

// brandedStringClass partitions flagged receivers by what makes them lossy
// outside display contexts — the message and the fix differ per class.
type brandedStringClass int

const (
	brandedStringNone brandedStringClass = iota
	brandedStringDisplay
	brandedStringActor
	brandedStringRef
)

func detectBrandedStringCalls(
	ctx *analyzer.AnalysisContext,
	gf *analyzer.GoFile,
) []finding.Finding {
	var findings []finding.Finding

	ranges := displayArgRanges(gf)

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 0 {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "String" {
			return true
		}

		class := brandedStringClassOf(gf.Pkg.TypesInfo, sel)
		if class == brandedStringNone {
			return true
		}

		for _, r := range ranges {
			if call.Pos() >= r.start && call.Pos() <= r.end {
				return true
			}
		}

		if f, err := c044Finding(ctx, call, class); err == nil {
			findings = append(findings, f)
		}

		return true
	})

	return findings
}

// brandedStringClassOf resolves the .String() method's DECLARING receiver
// type: the generic cbid.ID (display form only when its marker is a
// BrandNamer), or the lossy id-package overrides. Benign passthroughs
// (StreamType, ActorKind, marker-less cbid instances) classify as None.
func brandedStringClassOf(info *types.Info, sel *ast.SelectorExpr) brandedStringClass {
	obj, ok := info.ObjectOf(sel.Sel).(*types.Func)
	if !ok {
		return brandedStringNone
	}

	sig, ok := obj.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return brandedStringNone
	}

	named, ok := resolveNamed(sig.Recv().Type())
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return brandedStringNone
	}

	switch named.Obj().Pkg().Path() {
	case "github.com/larsartmann/go-branded-id":
		if !receiverMarkerIsBrandNamer(info, sel) {
			return brandedStringNone
		}

		return brandedStringDisplay
	case "github.com/larsartmann/go-cqrs-lite/id/v4",
		"github.com/larsartmann/go-cqrs-lite/core/v5/id":
		switch named.Obj().Name() {
		case "ActorID":
			return brandedStringActor
		case "StreamRef":
			return brandedStringRef
		}

		return brandedStringNone
	default:
		return brandedStringNone
	}
}

// brandNamerIface is the { Name() string } marker contract from
// go-branded-id; only markers implementing it make String() prefixed.
//
//nolint:gochecknoglobals // read-only type-level constant
var brandNamerIface = types.NewInterfaceType([]*types.Func{
	types.NewFunc(
		token.NoPos, nil, "Name",
		types.NewSignatureType(
			nil, nil, nil,
			nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
			false,
		),
	),
}, nil).Complete()

// receiverMarkerIsBrandNamer checks the CALL-SITE receiver instance: method
// declarations carry uninstantiated receivers, so the marker type argument
// is read from the receiver expression's type (id.StreamID et al. arrive as
// instantiated aliases).
func receiverMarkerIsBrandNamer(info *types.Info, sel *ast.SelectorExpr) bool {
	tv, ok := info.Types[sel.X]
	if !ok || tv.Type == nil {
		return false
	}

	t := types.Unalias(tv.Type)
	if ptr, isPtr := t.(*types.Pointer); isPtr {
		t = types.Unalias(ptr.Elem())
	}

	named, ok := t.(*types.Named)
	if !ok || named.TypeArgs().Len() == 0 {
		return false
	}

	marker := named.TypeArgs().At(0)

	return types.Implements(marker, brandNamerIface) ||
		types.Implements(types.NewPointer(marker), brandNamerIface)
}

// displaySpan is one whitelisted display-expression span.

func c044Finding(
	ctx *analyzer.AnalysisContext,
	call *ast.CallExpr,
	class brandedStringClass,
) (finding.Finding, error) {
	pos := ctx.Fset.Position(call.Pos())

	msg := "branded-ID .String() renders the PREFIXED display form — " +
		"use .Get()/the typed value for keys, SQL args, cursors, and wire payloads"
	if class == brandedStringActor {
		msg = "id.ActorID.String() drops the actor kind — use PrefixedString() at " +
			"serialization boundaries (ParseActorID reads it back) or carry the typed value"
	}

	if class == brandedStringRef {
		msg = "StreamRef.String() is the display composite — storage/stream keys use " +
			"StreamKey() (the bare form); display positions are whitelisted"
	}

	return findingTemplate.Builder(
		"C044",
		msg,
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithConfidence(finding.ConfidenceHigh).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
}
