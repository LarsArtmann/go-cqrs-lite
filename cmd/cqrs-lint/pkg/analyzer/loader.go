package analyzer

import (
	"fmt"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// LoadPackages loads Go packages from the given directory with full type info.
func LoadPackages(dir string) ([]*packages.Package, *token.FileSet, error) {
	fset := token.NewFileSet()

	pkgs, err := loadFromDir(dir, fset)
	if err != nil {
		return nil, nil, fmt.Errorf("load packages: %w", err)
	}

	return pkgs, fset, nil
}

func loadFromDir(dir string, fset *token.FileSet) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedSyntax | packages.NeedImports | packages.NeedFiles,
		Fset:       fset,
		Tests:      false,
		Dir:        dir,
		BuildFlags: []string{"-tags=goexperiment.jsonv2"},
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("load packages from %s: %w", dir, err)
	}

	return pkgs, nil
}

// findGoModDirs walks the directory tree under root and returns all directories
// containing a go.mod file. Skips vendor, .git, node_modules, and similar dirs.
func findGoModDirs(root string) ([]string, error) {
	var dirs []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == ".git" || base == "node_modules" ||
				base == "testdata" || base == "dist" || base == "build" {
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

// BuildContext loads packages, builds the CQRSRegistry, and creates an AnalysisContext.
// Supports monorepos: recursively discovers all go.mod files under projectRoot
// and merges packages from every module into a single context.
func BuildContext(projectRoot string) (*AnalysisContext, error) {
	modDirs, err := findGoModDirs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("find modules: %w", err)
	}

	if len(modDirs) == 0 {
		modDirs = []string{projectRoot}
	}

	fset := token.NewFileSet()

	ctx := &AnalysisContext{
		Fset:        fset,
		ProjectRoot: projectRoot,
		Registry:    NewCQRSRegistry(),
	}

	for _, dir := range modDirs {
		pkgs, err := loadFromDir(dir, fset)
		if err != nil {
			continue
		}

		ctx.Packages = append(ctx.Packages, pkgs...)

		for _, pkg := range pkgs {
			if len(pkg.Errors) > 0 {
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

				path := pkg.GoFiles[min(i, len(pkg.GoFiles)-1)]
				goFile := &GoFile{
					Path:   path,
					Pkg:    pkg,
					AST:    file,
					IsTest: strings.HasSuffix(path, "_test.go"),
				}
				ctx.GoFiles = append(ctx.GoFiles, goFile)

				if !goFile.IsTest {
					scanFile(ctx, goFile)
				}
			}
		}
	}

	filterEventPayloads(ctx)

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

// BuildContextFromPackages creates an AnalysisContext from pre-loaded packages.
// This is used by tests that parse inline source code.
func BuildContextFromPackages(fset *token.FileSet, pkgs []*packages.Package) *AnalysisContext {
	ctx := &AnalysisContext{
		Fset:     fset,
		Packages: pkgs,
		Registry: NewCQRSRegistry(),
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			continue
		}

		for i, file := range pkg.Syntax {
			if file == nil {
				continue
			}

			path := ""
			if i < len(pkg.GoFiles) {
				path = pkg.GoFiles[i]
			}

			if path == "" && file.Name != nil {
				path = filepath.Join(pkg.PkgPath, file.Name.Name+".go")
			}

			goFile := &GoFile{
				Path:   path,
				Pkg:    pkg,
				AST:    file,
				IsTest: strings.HasSuffix(path, "_test.go"),
			}
			ctx.GoFiles = append(ctx.GoFiles, goFile)

			if !goFile.IsTest {
				scanFile(ctx, goFile)
			}
		}
	}

	return ctx
}
