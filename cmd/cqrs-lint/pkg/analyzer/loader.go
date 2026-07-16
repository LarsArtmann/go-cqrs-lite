package analyzer

import (
	"fmt"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// LoadPackages loads Go packages from the given directory with full type info.
func LoadPackages(dir string) ([]*packages.Package, *token.FileSet, error) {
	fset := token.NewFileSet()
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
		return nil, nil, fmt.Errorf("load packages: %w", err)
	}

	return pkgs, fset, nil
}

// BuildContext loads packages, builds the CQRSRegistry, and creates an AnalysisContext.
func BuildContext(projectRoot string) (*AnalysisContext, error) {
	pkgs, fset, err := LoadPackages(projectRoot)
	if err != nil {
		return nil, err
	}

	ctx := &AnalysisContext{
		Fset:        fset,
		Packages:    pkgs,
		ProjectRoot: projectRoot,
		Registry:    NewCQRSRegistry(),
	}

	// Collect Go files and build registry.
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			continue
		}

		if !IsCQRSImport(pkg) {
			continue
		}

		ctx.ModulePath = pkg.PkgPath

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

	return ctx, nil
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
