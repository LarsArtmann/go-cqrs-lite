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
	//
	// In a multi-module workspace this holds the PRIMARY module's profile (the
	// go.mod at the project root, or the shallowest module). It is NOT a cross-
	// module merge, so an example app's ListenAndServe no longer flips
	// server=true for the library module. Per-module refinement is available
	// via FeatureProfiles / ProfileForFile.
	FeatureProfile FeatureProfile

	// FeatureProfiles holds a per-module FeatureProfile keyed by the module's
	// go.mod directory. Populated in BuildContext by DetectFeaturesPerModule.
	// Per-file detectors resolve the correct profile for each finding's file via
	// ProfileForFile. Empty for single-module projects (use FeatureProfile).
	FeatureProfiles map[string]FeatureProfile

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
	Path string
	Pkg  *packages.Package
	AST  *ast.File
	IsTest bool
	// ModuleDir is the filesystem path of the go.mod directory this file belongs
	// to. Set in BuildContext so feature detection can be partitioned per module
	// (a multi-module workspace gets one FeatureProfile per module instead of a
	// single cross-module merge). Empty when the file's module is unknown.
	ModuleDir string
}

// ProfileForFile returns the FeatureProfile for the module that owns the given
// file path. This lets per-file detectors evaluate each finding against its own
// module's features rather than a workspace-wide merge (so a server call in an
// examples/ module does not enable server-only rules for the library module).
// Falls back to the primary FeatureProfile when no per-module profile matches
// (e.g. single-module projects, files outside any known module).
func (ctx *AnalysisContext) ProfileForFile(path string) FeatureProfile {
	if len(ctx.FeatureProfiles) == 0 {
		return ctx.FeatureProfile
	}

	// A file belongs to the module whose dir is the longest matching prefix of
	// the file path (handles nested modules: /repo/examples/basic owns files
	// under /repo/examples/basic/...).
	var bestDir string
	for dir := range ctx.FeatureProfiles {
		if !strings.HasPrefix(path, dir+string(os.PathSeparator)) && path != dir {
			continue
		}
		if len(dir) > len(bestDir) {
			bestDir = dir
		}
	}

	if bestDir != "" {
		return ctx.FeatureProfiles[bestDir]
	}

	return ctx.FeatureProfile
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

// IsLibrarySelfLint reports whether the analyzed code IS the go-cqrs-lite
// library itself (not a consumer importing it). This is used to auto-suppress
// consumer-coaching rules (A001/A008/A020/A021/A023/E005/E007) that are
// meaningless when linting the library's own source — the library cannot
// coach itself to "adopt" its own features.
//
// Detection checks the module path and package import paths. When any
// analyzed package's import path starts with the go-cqrs-lite prefix, the
// code is the library itself.
func (ctx *AnalysisContext) IsLibrarySelfLint() bool {
	if IsCQRSModulePath(ctx.ModulePath) {
		return true
	}

	for _, gf := range ctx.GoFiles {
		if gf.Pkg != nil && IsCQRSModulePath(gf.Pkg.PkgPath) {
			return true
		}
	}

	return false
}
