package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

func collectExports(dir string) ([]string, error) {
	fset := token.NewFileSet()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"api_stability.read_dir",
			"read dir %s",
			dir,
		)
	}

	pkgFiles := make(map[string][]*ast.File)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Infrastructure,
				"api_stability.parse_file",
				"parse file %s",
				entry.Name(),
			)
		}

		pkgName := file.Name.Name
		pkgFiles[pkgName] = append(pkgFiles[pkgName], file)
	}

	var exports []string

	for _, files := range pkgFiles {
		for _, file := range files {
			exports = append(exports, collectFileExports(file)...)
		}
	}

	sort.Strings(exports)

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
