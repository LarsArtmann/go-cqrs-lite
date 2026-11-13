package analyzer

import (
	"errors"
	"fmt"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

// loadMu serializes packages.Load calls: x/tools' loader is not goroutine-safe
// against concurrent Load invocations (go/types state races under -race), and
// nothing here benefits from overlapping them — BuildContext is sequential.
var loadMu sync.Mutex //nolint:gochecknoglobals // serializes packages.Load; x/tools loader is not goroutine-safe

// errSilentEmptyLoad guards the silent-empty case: go/packages can return
// ZERO packages with NO error when the module graph is broken (stale go.sum,
// unresolvable requires — the 2026-10-02 fixture rot). An empty context here
// would render a clean bill of health for a project cqrs-lint never actually
// read. A repo with no Go sources at all is still a legitimate clean verdict.
var errSilentEmptyLoad = errors.New("analyzing nothing must not report clean")

func loadFromDir(dir string, fset *token.FileSet) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedSyntax | packages.NeedImports | packages.NeedFiles,
		Fset:  fset,
		Tests: false,
		Dir:   dir,
	}

	loadMu.Lock()
	pkgs, err := packages.Load(cfg, "./...")
	loadMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("load packages from %s: %w", dir, err)
	}

	return pkgs, nil
}

// analysisSkipDir reports whether a directory name is excluded from analysis
// walks (shared by findGoModDirs and hasGoSourceFiles).
func analysisSkipDir(base string) bool {
	return base == "vendor" || base == ".git" || base == "node_modules" ||
		base == "testdata" || base == "dist" || base == "build"
}

// findGoModDirs walks the directory tree under root and returns all directories
// containing a go.mod file. Skips vendor, .git, node_modules, and similar dirs.
func findGoModDirs(root string) ([]string, error) {
	var dirs []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip inaccessible paths, continue walking
		}

		if d.IsDir() {
			if analysisSkipDir(d.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		if d.Name() == "go.mod" {
			dir := filepath.Dir(path)
			dirs = append(dirs, dir)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}

	return dirs, nil
}

// hasGoSourceFiles reports whether any .go file exists under the given module
// directories (stopping at the first hit). Used by the silent-empty guard to
// tell "repo has no Go code" (legitimate clean verdict) from "Go code exists
// but go/packages returned nothing" (swallowed driver failure — loud error).
func hasGoSourceFiles(dirs []string) bool {
	for _, dir := range dirs {
		found := false

		err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
			if err != nil || found {
				return nil //nolint:nilerr // skip inaccessible paths, continue walking
			}

			if d.IsDir() {
				if analysisSkipDir(d.Name()) {
					return filepath.SkipDir
				}

				return nil
			}

			if strings.HasSuffix(d.Name(), ".go") {
				found = true

				return filepath.SkipAll
			}

			return nil
		})
		if err != nil || found {
			return found
		}
	}

	return false
}

// BuildContext loads packages, builds the CQRSRegistry, and creates an AnalysisContext.
// Supports monorepos: recursively discovers all go.mod files under projectRoot
// and merges packages from every module into a single context.
func BuildContext(projectRoot string) (*AnalysisContext, error) {
	// Canonicalize the root to an absolute path: every downstream path form
	// (module dirs, per-module feature-profile keys, go.mod locations) derives
	// from it, while finding positions from go/packages are always absolute.
	// With a relative root (the doctor command passes the config/flag value
	// verbatim, e.g. "."), profile keys came out relative and ProfileForFile's
	// longest-prefix match silently missed for EVERY file — detectors gated on
	// per-module features then evaluated against the primary module's profile
	// (the doctor audit on browser-history lost the api module's HasServer,
	// classifying two working B029/B030 suppressions stale; fixed 2026-10-12).
	// The SECOND layer of the same failure class (issue #126, InboxClean
	// 2026-11-10): a root INSIDE a module found no go.mod at all, degraded to
	// partial-module feature detection, and misclassified working B029/B030/
	// B014 directives stale — fixed by findEnclosingGoModDir resolving the
	// owning module upward (see the subtree note below).
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}
	projectRoot = absRoot

	modDirs, err := findGoModDirs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("find modules: %w", err)
	}

	// Subtree-of-a-module runs (issue #126): when no go.mod sits at or under
	// the requested root, the root is INSIDE a module whose go.mod lives
	// above. Feature flags are module-scoped facts (HasServer lives in the
	// module's cmd/, not in the internal/ package being linted), so feature
	// detection must see the whole ENCLOSING module while findings and
	// scanning stay scoped to the requested subtree (the Go ./... subtree
	// convention). Without this, a doctor --audit-suppressions run from a
	// subdirectory misprofiled every per-file-gated rule and classified
	// working B029/B030/B014 suppressions stale — and --prune-suppressions
	// would have deleted live directives on that verdict.
	featureScopeRoot := ""
	if len(modDirs) == 0 {
		if enclosing, ok := findEnclosingGoModDir(projectRoot); ok {
			modDirs = []string{enclosing}
			featureScopeRoot = projectRoot
		} else {
			modDirs = []string{projectRoot}
		}
	}

	fset := token.NewFileSet()

	ctx := &AnalysisContext{
		Fset:        fset,
		ProjectRoot: projectRoot,
		Registry:    NewCQRSRegistry(),
	}

	// packagesByModule records which packages were loaded from each go.mod dir.
	// *packages.Package does not carry its module dir, so we capture it here for
	// per-module feature detection (DetectFeaturesPerModule).
	packagesByModule := map[string][]*packages.Package{}

	// featureFiles feeds feature detection. In subtree runs it carries the
	// whole enclosing module's files even though only the subtree is
	// scanned into the registry.
	var featureFiles []*GoFile

	for _, dir := range modDirs {
		pkgs, err := loadFromDir(dir, fset)
		if err != nil {
			ctx.LoadErrors = append(ctx.LoadErrors, PackageLoadError{
				Module: dir,
				Errors: []string{err.Error()},
			})
			continue
		}

		ctx.Packages = append(ctx.Packages, pkgs...)
		packagesByModule[dir] = append(packagesByModule[dir], pkgs...)

		for _, pkg := range pkgs {
			if len(pkg.Errors) > 0 {
				msgs := make([]string, 0, len(pkg.Errors))
				for _, e := range pkg.Errors {
					msgs = append(msgs, e.Error())
				}
				ctx.LoadErrors = append(ctx.LoadErrors, PackageLoadError{
					Module:  dir,
					PkgPath: pkg.PkgPath,
					Errors:  msgs,
				})
				continue
			}

			if !IsCQRSImport(pkg) {
				continue
			}

			if ctx.ModulePath == "" {
				ctx.ModulePath = pkg.PkgPath
			}

			for i, file := range pkg.Syntax {
				if file == nil {
					continue
				}

				// Guard against missing/short GoFiles. The syntax file at index i
				// maps to pkg.GoFiles[i]; when they mismatch (cgo/processed files,
				// or a package with no files) we can't get a reliable path, so skip
				// rather than panic or silently misattribute locations.
				if len(pkg.GoFiles) == 0 || i >= len(pkg.GoFiles) {
					continue
				}

				path := pkg.GoFiles[i]
				goFile := &GoFile{
					Path:      path,
					Pkg:       pkg,
					AST:       file,
					IsTest:    strings.HasSuffix(path, "_test.go"),
					ModuleDir: dir,
				}
				featureFiles = append(featureFiles, goFile)

				if featureScopeRoot != "" && !isUnderDir(path, featureScopeRoot) {
					// Outside the requested subtree: feature detection sees it,
					// scanning and findings do not.
					continue
				}

				ctx.GoFiles = append(ctx.GoFiles, goFile)

				if !goFile.IsTest {
					scanFile(ctx, goFile)
				}
			}
		}
	}

	if len(ctx.Packages) == 0 && len(ctx.LoadErrors) == 0 && hasGoSourceFiles(modDirs) {
		return nil, fmt.Errorf(
			"%w: no packages loaded from %d module dir(s) under %s with zero load errors — "+
				"the module graph is likely broken (stale go.sum or missing deps; try `go mod tidy` per module)",
			errSilentEmptyLoad, len(modDirs), projectRoot,
		)
	}

	filterEventPayloads(ctx)
	ResolveRegisteredTypeConsts(ctx.Registry)
	ResolveConverterRoutedEventIDs(ctx)
	ResolveHelperEmitCalls(ctx)
	ResolveCatalogHelperCalls(ctx)
	ResolveEmittedEventTypeConsts(ctx.Registry)
	ResolveFoldTombstoneCases(ctx.Registry)
	ResolveHandlerMethods(ctx)
	ResolveTransportAdapters(ctx)

	// Per-module feature detection. For a single-module project this produces
	// one profile identical to the old merged detection. For a multi-module
	// workspace each module gets its own profile (so an examples/ app's
	// ListenAndServe no longer flips server=true for the library module), and
	// the PRIMARY module's profile (shallowest dir, typically the project root)
	// is exposed via FeatureProfile for global detectors + doctor output.
	switch {
	case featureScopeRoot != "":
		// Subtree run: one profile for the whole enclosing module, keyed by
		// its dir so ProfileForFile's longest-prefix match resolves every
		// subtree file to it.
		full := detectFeatureSignals(ctx.Packages, featureFiles, ctx.Registry)
		ctx.FeatureProfiles = map[string]FeatureProfile{modDirs[0]: full}
		ctx.FeatureProfile = full
	case len(modDirs) > 1:
		ctx.FeatureProfiles = DetectFeaturesPerModule(ctx, packagesByModule)
		ctx.FeatureProfile = primaryModuleProfile(ctx.FeatureProfiles, projectRoot, modDirs)
	default:
		ctx.FeatureProfile = DetectFeatures(ctx)
	}

	return ctx, nil
}

// filterEventPayloads removes structs from Registry.Events that are not
// actual event payloads (i.e., not used as arguments to event.New/NewEvent).
// scanGenDecl registers ALL structs; this post-pass keeps only real event payloads.
func filterEventPayloads(ctx *AnalysisContext) {
	if len(ctx.Registry.EventPayloadTypes) == 0 {
		ctx.Registry.Events = nil
		return
	}

	filtered := ctx.Registry.Events[:0]
	for _, evt := range ctx.Registry.Events {
		if ctx.Registry.EventPayloadTypes[evt.Name] {
			filtered = append(filtered, evt)
		}
	}
	ctx.Registry.Events = filtered
}
