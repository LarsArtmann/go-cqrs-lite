// Package analyzer provides CQRS-specific analysis types and registry building
// for the cqrs-lint linter.
package analyzer

import (
	"go/ast"
	"go/token"
	"os"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

// EventEmission records where an event type was emitted via event.New/NewEvent.
type EventEmission struct {
	File string
	Line int
}

// CommandInfo describes a command type found in the analyzed code.
type CommandInfo struct {
	Name          string // struct type name
	Package       string // import path
	File          string // file path
	Pos           token.Position
	HasBasicCmd   bool     // embeds *command.BasicCommand
	ManualID      bool     // has manual ID() method
	ManualType    bool     // has manual Type() or StreamID() method
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
	Package     string
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
	Package   string
	File      string
	Pos       token.Position
	StateType string
	IsOO      bool // object-oriented aggregate (has uncommittedEvents)
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

// AnalysisContext provides shared state for all rule detectors.
type AnalysisContext struct {
	Fset        *token.FileSet
	Packages    []*packages.Package
	Registry    *CQRSRegistry
	ProjectRoot string
	ModulePath  string

	// AllGoFiles is a flat list of all Go source files for easy iteration.
	GoFiles []*GoFile

	// FeatureProfile centralizes "what kind of system is this?" as feature flags.
	// Detectors consult this instead of independently re-deriving project context.
	// Computed once in BuildContext after the scan completes.
	FeatureProfile FeatureProfile

	// RulesConfig carries rule-specific overrides from .cqrs-lint.json (the
	// "rules" key). Detectors consult it to suppress documented false-positive
	// patterns. Zero value = no overrides = detectors behave as before.
	// Wired in main.go run(); zero in unit tests (BuildContextFromSource).
	RulesConfig RulesConfig

	// LoadErrors holds per-package errors encountered during BuildContext.
	// Non-empty means the analysis is partial; callers should warn the user.
	LoadErrors []PackageLoadError

	// lineCache caches file contents for SourceLine to avoid repeated disk reads.
	lineCache sync.Map // filename → []string
}

// PackageLoadError describes a package that failed to load during analysis.
// The analysis may still proceed with the remaining packages, but the caller
// should surface these errors so the user knows the result is partial.
type PackageLoadError struct {
	Module  string   // go.mod directory (filesystem path)
	PkgPath string   // offending package import path, empty if the whole module failed
	Errors  []string // human-readable error messages
}

// GoFile wraps a parsed Go file with its package context.
type GoFile struct {
	Path   string
	Pkg    *packages.Package
	AST    *ast.File
	IsTest bool
}

// SourceLine reads the source file at the given line number and returns the trimmed line.
// Returns empty string if the file cannot be read or the line is out of range.
// File contents are cached to avoid repeated disk reads for the same file.
func (ctx *AnalysisContext) SourceLine(filename string, line int) string {
	if filename == "" || line <= 0 {
		return ""
	}

	var lines []string

	if cached, ok := ctx.lineCache.Load(filename); ok {
		if l, ok := cached.([]string); ok {
			lines = l
		}
	} else {
		data, err := os.ReadFile(filename)
		if err != nil {
			return ""
		}

		lines = strings.Split(string(data), "\n")
		ctx.lineCache.Store(filename, lines)
	}

	if line > len(lines) {
		return ""
	}

	return strings.TrimSpace(lines[line-1])
}

// IsCQRSImport returns true if the package imports any go-cqrs-lite module.
func IsCQRSImport(p *packages.Package) bool {
	for _, imp := range p.Imports {
		if imp == nil {
			continue
		}

		if IsCQRSModulePath(imp.PkgPath) {
			return true
		}
	}

	return false
}

// IsCQRSModulePath returns true if the given import path is part of
// the go-cqrs-lite module ecosystem.
func IsCQRSModulePath(path string) bool {
	prefix := "github.com/larsartmann/go-cqrs-lite"
	if path == prefix {
		return true
	}

	if len(path) > len(prefix) && path[:len(prefix)+1] == prefix+"/" {
		return true
	}

	return false
}
