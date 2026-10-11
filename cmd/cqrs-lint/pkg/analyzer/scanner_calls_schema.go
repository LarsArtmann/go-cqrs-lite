package analyzer

import (
	"go/ast"
	"go/token"
)

// recordSchemaDeclaredEvent records one schema-declaration event type
// (schema.Event/EventOf or the Schemas-builder .Event method): the first
// argument is the event type as a string literal, or a constant reference
// that resolves in the post-pass (ResolveEmittedEventTypeConsts).
func recordSchemaDeclaredEvent(
	ctx *AnalysisContext,
	gf *GoFile,
	call *ast.CallExpr,
	pos token.Position,
) {
	if len(call.Args) == 0 {
		return
	}

	if eventTypeStr := StringLit(call.Args[0]); eventTypeStr != "" {
		ctx.Registry.EventTypesInSchemaDecl[eventTypeStr] = EventEmission{
			File: gf.Path,
			Line: pos.Line,
		}

		// Ladder tracking (E022): the current version is the second argument
		// when present (schema.Event/EventOf and the builder's .Event both
		// take it positionally). Literal only; unknown stays -1 (silent).
		if len(call.Args) > 1 {
			if ladder := ctx.Registry.schemaLadder(eventTypeStr); ladder != nil {
				if version := IntLit(call.Args[1]); version > 0 {
					ladder.Current = version
					ladder.Decl = EventEmission{File: gf.Path, Line: pos.Line}
				}
			}
		}
	} else if name := ExprIdentName(call.Args[0]); name != "" {
		ctx.Registry.pendingSchemaEventTypeRefs = append(
			ctx.Registry.pendingSchemaEventTypeRefs,
			pendingEventTypeRef{constName: name, file: gf.Path, line: pos.Line},
		)
	}
}

// recordSchemaOp records one migration op call (schema.RenameField, AddField,
// RemoveField, Transform, Split, RenameType): first argument the event type
// string literal, second the source version literal. RenameType's second
// argument is the TARGET type, not a version — it does not record a rung.
func recordSchemaOp(
	ctx *AnalysisContext,
	gf *GoFile,
	call *ast.CallExpr,
	pos token.Position,
) {
	if len(call.Args) < 2 {
		return
	}

	eventTypeStr := StringLit(call.Args[0])
	if eventTypeStr == "" {
		return
	}

	version := IntLit(call.Args[1])
	if version <= 0 {
		return
	}

	ladder := ctx.Registry.schemaLadder(eventTypeStr)
	if ladder == nil {
		return
	}

	if ladder.OpVersions == nil {
		ladder.OpVersions = make(map[int]EventEmission)
	}

	ladder.OpVersions[version] = EventEmission{File: gf.Path, Line: pos.Line}
}

// isSchemaBuilderEventCall reports whether an .Event method call targets the
// system.Schemas() fluent builder: the receiver is a system.Schemas() call
// (directly or through a chain of builder .Event calls) or a local variable
// initialized from one (tracked per file by trackSchemaBuilderAssignments).
// A false positive only over-declares — suppressing an E021 finding — which
// is the safe direction; a false negative keeps the finding alive.
func isSchemaBuilderEventCall(ctx *AnalysisContext, gf *GoFile, sel *ast.SelectorExpr) bool {
	switch recv := sel.X.(type) {
	case *ast.Ident:
		return ctx.Registry.schemaBuilderIdents[gf.Path][recv.Name]
	case *ast.CallExpr:
		return isSchemaBuilderChainCall(ctx, gf, recv)
	}

	return false
}

// isSchemaBuilderChainCall walks a receiver call chain that must end in a
// system-qualified Schemas() call or a tracked builder variable:
// system.Schemas().Event[a](...).Event[b](...).
func isSchemaBuilderChainCall(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) bool {
	chainSel, ok := SelectorFromExpr(call.Fun)
	if !ok {
		return false
	}

	switch chainSel.Sel.Name {
	case "Schemas":
		return IsQualifierFor(gf, chainSel, "go-cqrs-lite/system")
	case "Event":
		switch recv := chainSel.X.(type) {
		case *ast.CallExpr:
			return isSchemaBuilderChainCall(ctx, gf, recv)
		case *ast.Ident:
			return ctx.Registry.schemaBuilderIdents[gf.Path][recv.Name]
		}
	}

	return false
}

// trackSchemaBuilderAssignments records `schemas := system.Schemas()` (and
// the rare re-assignment form) so later builder-method calls through the
// variable resolve to the schema declaration tier. Same-file only: the fluent
// builder is declared and consumed in one place; a cross-file builder var is
// a missed declaration (safe direction).
func trackSchemaBuilderAssignments(ctx *AnalysisContext, gf *GoFile, stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}

		sel, ok := SelectorFromExpr(call.Fun)
		if !ok || sel.Sel.Name != "Schemas" || !IsQualifierFor(gf, sel, "go-cqrs-lite/system") {
			continue
		}

		if i >= len(stmt.Lhs) {
			continue
		}

		if ident, ok := stmt.Lhs[i].(*ast.Ident); ok {
			if ctx.Registry.schemaBuilderIdents == nil {
				ctx.Registry.schemaBuilderIdents = make(map[string]map[string]bool)
			}

			if ctx.Registry.schemaBuilderIdents[gf.Path] == nil {
				ctx.Registry.schemaBuilderIdents[gf.Path] = make(map[string]bool)
			}

			ctx.Registry.schemaBuilderIdents[gf.Path][ident.Name] = true
		}
	}
}

// isSchemaOpName reports whether the function name is a schema migration-op
// constructor carrying (eventType, sourceVersion, ...) positionally.
// RenameType deliberately does not qualify: its second argument is the target
// TYPE, not a version, and type renames do not add a version rung.
func isSchemaOpName(name string) bool {
	switch name {
	case "RenameField", "AddField", "RemoveField", "Transform", "Split":
		return true
	}

	return false
}
