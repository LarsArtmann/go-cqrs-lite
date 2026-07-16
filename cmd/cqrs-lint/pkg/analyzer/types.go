// Package analyzer provides CQRS-specific analysis types and registry building
// for the cqrs-lint linter.
package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// CommandInfo describes a command type found in the analyzed code.
type CommandInfo struct {
	Name          string // struct type name
	Package       string // import path
	File          string // file path
	Pos           token.Position
	HasBasicCmd   bool     // embeds *command.BasicCommand
	ManualID      bool     // has manual ID() method
	ManualType    bool     // has manual Type() method
	ManualAggID   bool     // has manual AggregateID() method
	IDReturnsZero bool     // ID() returns zero-value composite literal
	Fields        []string // field names
}

// EventInfo describes an event type found in the analyzed code.
type EventInfo struct {
	Name    string // struct type name (payload type)
	Package string
	File    string
	Pos     token.Position
}

// FoldInfo describes a fold/apply function.
type FoldInfo struct {
	FuncName    string
	File        string
	Pos         token.Position
	StateType   string
	HasSwitch   bool     // has a switch on evt.Type()
	HasDefault  bool     // has a default case
	DefaultNil  bool     // default case returns nil error
	UnknownVars []string // variables for the event parameter
}

// DeciderInfo describes a decider construct.
type DeciderInfo struct {
	Package      string
	File         string
	Pos          token.Position
	StateType    string
	IsOO         bool // object-oriented aggregate (has uncommittedEvents)
	IsFunctional bool // functional decider (uses decider.Decider)
}

// ProjectionInfo describes a projection implementation.
type ProjectionInfo struct {
	Name       string
	Package    string
	File       string
	Pos        token.Position
	HasAsync   bool // Handle method launches goroutines or sends to channels
	EventTypes []string
}

// HandlerInfo describes a registered command/query handler.
type HandlerInfo struct {
	CommandType string
	FuncName    string
	File        string
	Pos         token.Position
}

// AnalysisContext provides shared state for all rule detectors.
type AnalysisContext struct {
	Fset        *token.FileSet
	Packages    []*packages.Package
	Registry    *CQRSRegistry
	ProjectRoot string
	ModulePath  string

	// AllGoFiles is a flat list of all Go source files for easy iteration.
	GoFiles []*GoFile
}

// GoFile wraps a parsed Go file with its package context.
type GoFile struct {
	Path   string
	Pkg    *packages.Package
	AST    *ast.File
	IsTest bool
}

// IsCQRSImport returns true if the package imports any go-cqrs-lite module.
func IsCQRSImport(p *packages.Package) bool {
	for _, imp := range p.Imports {
		if imp == nil {
			continue
		}

		if isCQRSModulePath(imp.PkgPath) {
			return true
		}
	}

	return false
}

func isCQRSModulePath(path string) bool {
	prefix := "github.com/larsartmann/go-cqrs-lite"
	if path == prefix {
		return true
	}

	if len(path) > len(prefix) && path[:len(prefix)+1] == prefix+"/" {
		return true
	}

	return false
}

// TypeResolver helps resolve AST identifiers to types.Package types.
type TypeResolver struct {
	Info *types.Info
}

// ResolveType resolves an ast.Expr to a types.Type if possible.
func (tr *TypeResolver) ResolveType(expr ast.Expr) types.Type {
	if tr.Info == nil {
		return nil
	}

	return tr.Info.TypeOf(expr)
}
