package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

func collectDirExports(dir string) (string, []string, error) {
	fset := token.NewFileSet()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"api_stability.read_dir",
			"read dir %s",
			dir,
		)
	}

	pkgFiles := make(map[string][]*ast.File)

	pkgName := ""

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			return "", nil, errorfamily.Wrapf(
				err,
				errorfamily.Infrastructure,
				"api_stability.parse_file",
				"parse file %s",
				entry.Name(),
			)
		}

		if pkgName == "" {
			pkgName = file.Name.Name
		}

		pkgFiles[file.Name.Name] = append(pkgFiles[file.Name.Name], file)
	}

	var exports []string

	for _, files := range pkgFiles {
		for _, file := range files {
			exports = append(exports, collectFileExports(file)...)
		}
	}

	sort.Strings(exports)

	return pkgName, exports, nil
}

// excludedSubdir reports whether a sub-directory must be skipped when
// sweeping a module for public API: vendored, hidden, testdata, internal,
// and underscore/dot-prefixed directories are never consumer-facing imports.
func excludedSubdir(name string) bool {
	return name == "vendor" || name == "internal" || name == "testdata" ||
		strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// collectModuleExports collects the root package plus every importable
// sub-package of a module. Sub-package symbols are prefixed with their path
// relative to the module root, so the golden distinguishes e.g.
// catalog/docserver from the catalog root package. `main` sub-packages are
// skipped: their identifiers are not importable API.
func collectModuleExports(modPath string) ([]string, error) {
	var exports []string

	_, exps, err := collectDirExports(modPath)
	if err != nil {
		return nil, err
	}

	exports = append(exports, exps...)

	err = filepath.WalkDir(modPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() || path == modPath {
			return nil
		}

		if excludedSubdir(d.Name()) {
			return fs.SkipDir
		}

		pkgName, subExports, subErr := collectDirExports(path)
		if subErr != nil {
			return subErr
		}

		if pkgName == "main" {
			return nil
		}

		rel, relErr := filepath.Rel(modPath, path)
		if relErr != nil {
			return errorfamily.Wrapf(
				relErr,
				errorfamily.Infrastructure,
				"api_stability.rel_path",
				"rel %s",
				path,
			)
		}

		for _, e := range subExports {
			exports = append(exports, rel+"/"+e)
		}

		return nil
	})
	if err != nil {
		return nil, errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"api_stability.walk_module",
			"walk module %s",
			modPath,
		)
	}

	return exports, nil
}

func collectFileExports(file *ast.File) []string {
	var exports []string

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			exports = append(exports, collectGenDeclExports(d)...)
		case *ast.FuncDecl:
			if d.Name.IsExported() {
				if d.Recv == nil {
					exports = append(exports, "func "+d.Name.Name)
				} else {
					exports = append(exports, "method "+d.Name.Name)
				}
			}
		}
	}

	return exports
}

func collectGenDeclExports(genDecl *ast.GenDecl) []string {
	var exports []string

	for _, spec := range genDecl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if !s.Name.IsExported() {
				continue
			}

			exports = append(exports, typeExportName(s))
		case *ast.ValueSpec:
			prefix := strings.ToLower(genDecl.Tok.String())

			for _, name := range s.Names {
				if name.IsExported() {
					exports = append(exports, prefix+" "+name.Name)
				}
			}
		}
	}

	return exports
}

func typeExportName(ts *ast.TypeSpec) string {
	switch ts.Type.(type) {
	case *ast.InterfaceType:
		return "interface " + ts.Name.Name
	case *ast.StructType:
		return "struct " + ts.Name.Name
	default:
		return "type " + ts.Name.Name
	}
}
