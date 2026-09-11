package correctness

// F091 Tier 3 — payload-shape confirmation for the C013/C035 name heuristics.
// Names select candidates; under the typed-confirmation tier (--typed-info
// on|auto with type info available) a candidate picked ONLY by file-location
// vibes additionally needs structural evidence that it really is the kind of
// thing the rule assumes. Same tradeoff as the C008 Tier-2 gate: an
// evidence-free weak candidate stays silent rather than guess, strong
// name-suffix candidates fire unchanged, and syntax-only loads /
// --typed-info=off keep the historical heuristic.

import (
	"go/ast"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// typedEvidence carries package-wide structural signals consulted to confirm
// or reject file-name-only candidates.
type typedEvidence struct {
	payloadTypes map[string]bool // struct names seen as event.New payload args
	typeMethods  map[string]bool // struct names declaring a Type() string method
	fieldUses    map[string]bool // field names referenced via selector expressions
}

// buildTypedEvidence walks the analyzed (non-test) files once, collecting the
// confirmation signals. Only built when the typed tier is active.
func buildTypedEvidence(ctx *analyzer.AnalysisContext) *typedEvidence {
	ev := &typedEvidence{
		payloadTypes: make(map[string]bool),
		typeMethods:  make(map[string]bool),
		fieldUses:    make(map[string]bool),
	}
	if ctx.Registry != nil {
		for name := range ctx.Registry.EventPayloadTypes {
			ev.payloadTypes[name] = true
		}
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}
		collectTypeMethods(gf.AST, ev.typeMethods)
		collectFieldUses(gf.AST, ev.fieldUses)
	}

	return ev
}

// payloadConfirmed reports structural evidence that structName really is an
// event payload: it flowed into an event.New/NewEvent payload position
// (scanner registry) or declares the payload-conventional Type() string
// method.
func (e *typedEvidence) payloadConfirmed(structName string) bool {
	return e.payloadTypes[structName] || e.typeMethods[structName]
}

// mapFieldUsed reports whether any of the named fields is referenced via a
// selector expression in the analyzed files — evidence that a C035 map field
// is live shared state rather than an inert DTO field.
func (e *typedEvidence) mapFieldUsed(fieldNames []string) bool {
	for _, name := range fieldNames {
		if e.fieldUses[name] {
			return true
		}
	}

	return false
}

// collectTypeMethods records struct names whose method set carries the
// payload-conventional Type() string method (receiver T or *T).
func collectTypeMethods(file *ast.File, into map[string]bool) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != "Type" || !returnsString(fn) {
			continue
		}

		if name := receiverTypeName(fn); name != "" {
			into[name] = true
		}
	}
}

// receiverTypeName extracts the named type from a method receiver (T or *T).
func receiverTypeName(fn *ast.FuncDecl) string {
	if len(fn.Recv.List) == 0 {
		return ""
	}

	switch t := fn.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.Ident:
		return t.Name
	}

	return ""
}

// returnsString reports whether the function declares a single string result.
func returnsString(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}

	id, ok := fn.Type.Results.List[0].Type.(*ast.Ident)

	return ok && id.Name == "string"
}

// collectFieldUses records every selector field name referenced in the file
// (x.Field). Package qualifiers land in the set too — the evidence check is
// deliberately lenient: it exists to kill inert-DTO candidates, not to prove
// liveness.
func collectFieldUses(file *ast.File, into map[string]bool) {
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			into[sel.Sel.Name] = true
		}

		return true
	})
}

// viewSerialized reports whether the struct carries any json tag — the
// serialization reflection point distinguishing a served view from an
// unrelated struct that merely lives in views.go.
func viewSerialized(st *ast.StructType) bool {
	if st.Fields == nil {
		return false
	}

	for _, field := range st.Fields.List {
		if field.Tag != nil && strings.Contains(field.Tag.Value, "json:") {
			return true
		}
	}

	return false
}

// c013Candidate classifies a struct for C013 by the strength of its
// payload/view signal. The ordering mirrors the historical branch precedence:
// payload signals beat view signals, name suffixes beat file location.
type c013Candidate int

const (
	c013NotCandidate c013Candidate = iota
	c013PayloadByName
	c013PayloadByFile
	c013ViewByName
	c013ViewByFile
)

// classifyC013Candidate splits the payload heuristic (lintutil.
// LooksLikeEventPayload) and the view heuristic into strong (name-suffix) and
// weak (file-only) halves so the typed tier can gate the weak half.
func classifyC013Candidate(structName, filePath string) c013Candidate {
	switch {
	case lintutil.HasEventPayloadNameSuffix(structName):
		return c013PayloadByName
	case lintutil.IsPayloadFileName(filePath):
		return c013PayloadByFile
	case lintutil.HasReadModelNameSuffix(structName):
		return c013ViewByName
	case lintutil.IsReadModelFileName(filePath):
		return c013ViewByFile
	default:
		return c013NotCandidate
	}
}

// c035CandidateStrength classifies a C035 read-model candidate by signal
// strength: explicit read-model names are strong; generic handler/store/cache
// suffixes and file-location matches are ambient.
type c035CandidateStrength int

const (
	c035NotCandidate c035CandidateStrength = iota
	c035StrongCandidate
	c035WeakCandidate
)

// classifyC035Candidate splits the C035 read-model heuristic: name suffixes
// VIEW/READMODEL/READMODELSTATE/PROJECTION are strong; HANDLER/PROJECTOR/
// STORE/CACHE and any file-location match are weak (ambient).
func classifyC035Candidate(structName, filePath string) c035CandidateStrength {
	switch {
	case lintutil.HasReadModelNameSuffix(structName):
		return c035StrongCandidate
	case hasWeakReadModelSuffix(structName), isReadModelishFileName(filePath):
		return c035WeakCandidate
	default:
		return c035NotCandidate
	}
}

// hasWeakReadModelSuffix reports generic suffixes (HANDLER, PROJECTOR, STORE,
// CACHE) that appear far outside read models — they need typed evidence.
func hasWeakReadModelSuffix(structName string) bool {
	upper := strings.ToUpper(structName)

	for _, suffix := range []string{"HANDLER", "PROJECTOR", "STORE", "CACHE"} {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}

	return false
}

// isReadModelishFileName extends the shared read-model file vocabulary with
// handler.go/handlers.go, matching the original looksLikeReadModel heuristic.
func isReadModelishFileName(filePath string) bool {
	base := lintutil.BaseFileName(filePath)

	return base == "views" || base == "projection" || base == "readmodel" ||
		base == "handler" || base == "handlers"
}
