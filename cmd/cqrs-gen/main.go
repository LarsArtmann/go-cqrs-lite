// cqrs-gen generates typed handler registration code from Go structs marked with
// cqrs annotations.
//
// Usage:
//
//	cqrs-gen -type=command -output=commands_gen.go ./...
//	cqrs-gen -type=query -output=queries_gen.go ./...
//
// Marker comments in source code:
//
//	//cqrs:command CreateUser
//	type CreateUserCmd struct {
//	    *command.BasicCommand
//	    Name string
//	}
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	genTypeCommand = "command"
	genTypeQuery   = "query"
)

//nolint:gochecknoglobals // CLI flags
var (
	genType    = flag.String("type", genTypeCommand, "handler type to generate: command or query")
	outputFile = flag.String("output", "handlers_gen.go", "output file path")
	pkgName    = flag.String(
		"pkg",
		"",
		"package name for generated file (defaults to current directory)",
	)
)

func main() {
	flag.Parse()
	os.Exit(run(*genType, *outputFile, *pkgName, flag.Args()))
}

func run(genType, outputFile, pkg string, paths []string) int {
	if genType != genTypeCommand && genType != genTypeQuery {
		fmt.Fprintf(os.Stderr, "invalid type %q: must be 'command' or 'query'\n", genType)
		return 1
	}

	if len(paths) == 0 {
		paths = []string{"."}
	}

	entries, err := scan(paths, genType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
		return 1
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "no cqrs markers found")
		return 0
	}

	if pkg == "" {
		pkg = filepath.Base(mustAbs(paths[0]))
	}

	code := generate(pkg, genType, entries)

	if err := os.WriteFile(outputFile, []byte(code), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		return 1
	}

	fmt.Printf("generated %d handlers → %s\n", len(entries), outputFile)
	return 0
}

func mustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// Entry represents a single discovered cqrs marker.
type Entry struct {
	CommandType string
	StructName  string
	PackagePath string
}

func scan(paths []string, genType string) ([]Entry, error) {
	var entries []Entry

	for _, path := range paths {
		found, err := scanPath(path, genType)
		if err != nil {
			return nil, err
		}
		entries = append(entries, found...)
	}

	return entries, nil
}

func scanPath(root, genType string) ([]Entry, error) {
	var entries []Entry

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		found, err := scanFile(path, genType)
		if err != nil {
			return errorfamily.Wrapf(
				err,
				errorfamily.Infrastructure,
				"cqrs_gen.scan",
				"scan %s",
				path,
			)
		}
		entries = append(entries, found...)
		return nil
	})

	return entries, err
}

func scanFile(path, genType string) ([]Entry, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	prefix := "//cqrs:" + genType + " "
	tagKey := genType

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Check comment marker first (backward compat)
			cmdType := extractMarker(genDecl.Doc, prefix)
			if cmdType == "" {
				cmdType = extractMarker(ts.Doc, prefix)
			}

			// Fall back to struct tag
			if cmdType == "" {
				if structType, ok := ts.Type.(*ast.StructType); ok {
					cmdType = extractStructTag(structType, tagKey)
				}
			}

			if cmdType == "" {
				continue
			}

			entries = append(entries, Entry{
				CommandType: cmdType,
				StructName:  ts.Name.Name,
				PackagePath: f.Name.Name,
			})
		}
	}

	return entries, nil
}

func extractMarker(doc *ast.CommentGroup, prefix string) string {
	if doc == nil {
		return ""
	}

	for _, c := range doc.List {
		if after, ok := strings.CutPrefix(c.Text, prefix); ok {
			return after
		}
	}

	return ""
}

// extractStructTag scans a struct's fields for a `cqrs:"command:CreateUser"` or
// `cqrs:"query:GetUser"` tag on a special _ (underscore) field used as a
// type-level marker. This provides a cleaner alternative to comment markers:
//
//	type CreateUserCmd struct {
//	    _ struct{} `cqrs:"command:CreateUser"`
//	    Name string
//	}
func extractStructTag(st *ast.StructType, key string) string {
	if st.Fields == nil {
		return ""
	}

	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}

		tagValue := strings.Trim(field.Tag.Value, "`")
		cqrsTag := reflect.StructTag(tagValue).Get("cqrs")
		if cqrsTag == "" {
			continue
		}

		// Tag format: "command:CreateUser" or "query:GetUser"
		parts := strings.SplitN(cqrsTag, ":", 2)
		if len(parts) == 2 && parts[0] == key {
			return parts[1]
		}
	}

	return ""
}

const (
	commandImports = `import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
)

`

	queryImports = `import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

`
)

type genSpec struct {
	imports    string
	writeEntry func(b *strings.Builder, e Entry)
}

//nolint:gochecknoglobals // static code-generation dispatch table; data, not state
var genSpecs = map[string]genSpec{
	genTypeCommand: {
		imports:    commandImports,
		writeEntry: writeCommandHandler,
	},
	genTypeQuery: {
		imports:    queryImports,
		writeEntry: writeQueryHandler,
	},
}

func writeCommandHandler(b *strings.Builder, e Entry) {
	fmt.Fprintf(
		b,
		"// Register%sHandler registers a typed handler for %s commands.\n",
		e.StructName,
		e.CommandType,
	)
	fmt.Fprintf(
		b,
		"func Register%sHandler(d *command.Dispatcher, handler func(context.Context, *%s) error) error {\n",
		e.StructName,
		e.StructName,
	)
	fmt.Fprintf(b, "\treturn command.RegisterTyped(d, %q, handler)\n", e.CommandType)
	b.WriteString("}\n\n")
}

func writeQueryHandler(b *strings.Builder, e Entry) {
	fmt.Fprintf(
		b,
		"// Register%sHandler registers a typed handler for %s queries.\n",
		e.StructName,
		e.CommandType,
	)
	fmt.Fprintf(
		b,
		"func Register%sHandler[R any](d *query.Dispatcher, handler func(context.Context, *%s) (R, error)) error {\n",
		e.StructName,
		e.StructName,
	)
	fmt.Fprintf(
		b,
		"\treturn query.RegisterTyped[*%s, R](d, %q, handler)\n",
		e.StructName,
		e.CommandType,
	)
	b.WriteString("}\n\n")
}

func generate(pkg, genType string, entries []Entry) string {
	spec := genSpecs[genType]

	var b strings.Builder

	b.WriteString("// Code generated by cqrs-gen. DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	b.WriteString(spec.imports)

	for _, e := range entries {
		spec.writeEntry(&b, e)
	}

	return b.String()
}
