// cqrs-gen generates typed handler registration code from Go structs marked with
// cqrs annotations.
//
// Usage:
//
//	cqrs-gen -type=command -output=commands_gen.go ./...
//	cqrs-gen -type=query -output=queries_gen.go ./...
//	cqrs-gen -type=event -output=events_gen.go ./...
//
// Marker comments in source code (the identifier after the kind becomes the
// registered command/query/event type):
//
//	//cqrs:command CreateUser
//	//cqrs:query GetUser
//	//cqrs:event UserCreated
//	type CreateUserCmd struct {
//	    *command.BasicCommand
//	    Name string
//	}
package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/spf13/cobra"
)

const (
	genTypeCommand = "command"
	genTypeQuery   = "query"
	genTypeEvent   = "event"
)

type AppConfig struct {
	cmdguard.Config

	Type   string `default:"command"        flag:"type"   help:"handler type to generate: command, query, or event"`
	Output string `default:"handlers_gen.go" flag:"output" help:"output file path"`
	Pkg    string `default:""               flag:"pkg"     help:"package name for generated file (defaults to the scanned source package)"`
}

func main() {
	cli, err := cmdguard.NewCLI(
		"cqrs-gen",
		"Generate typed handler registration code from cqrs annotations",
		AppConfig{},
		cmdguard.WithCLILong(
			"cqrs-gen generates typed handler registration code from Go structs marked with cqrs annotations.\n\n"+
				"Marker comments in source code:\n"+
				"  //cqrs:command CreateUser\n"+
				"  //cqrs:query GetUser\n"+
				"  //cqrs:event UserCreated",
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CLI: %v\n", err)
		os.Exit(1)
	}

	rootCmd := cli.RootCommand()
	rootCmd.Use = "cqrs-gen [-type=command] [-output=handlers_gen.go] [paths...]"
	rootCmd.RunE = func(_ *cobra.Command, args []string) error {
		cfg := cli.Config()

		paths := args
		if len(paths) == 0 {
			paths = []string{"."}
		}

		return run(cfg.Type, cfg.Output, cfg.Pkg, paths)
	}

	cli.ExecuteAndExit(context.Background())
}

func run(handlerType, outputFile, pkg string, paths []string) error {
	if handlerType != genTypeCommand && handlerType != genTypeQuery && handlerType != genTypeEvent {
		return fmt.Errorf("invalid type %q: must be 'command', 'query', or 'event'", handlerType)
	}

	entries, err := scan(paths, handlerType)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "no cqrs markers found")
		return nil
	}

	entries = dedupEntries(os.Stderr, entries)

	if pkg == "" {
		pkg = entries[0].PackageName
	}

	code, err := generate(pkg, handlerType, entries)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	if err := os.WriteFile(outputFile, []byte(code), 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	fmt.Printf("generated %d handlers → %s\n", len(entries), outputFile)
	return nil
}

// Entry represents a single discovered cqrs marker.
type Entry struct {
	// TypeName is the marker value: the command/query/event type string passed
	// to RegisterTyped / Subscribe (e.g. "CreateUser", "GetUser", "UserCreated").
	TypeName string
	// StructName is the Go type name the marker was attached to.
	StructName string
	// PackageName is the Go package name of the source file (f.Name.Name), used
	// as the default package for the generated file.
	PackageName string
}

func scan(paths []string, handlerType string) ([]Entry, error) {
	var entries []Entry

	for _, path := range paths {
		found, err := scanPath(path, handlerType)
		if err != nil {
			return nil, fmt.Errorf("scan path %s: %w", path, err)
		}
		entries = append(entries, found...)
	}

	return entries, nil
}

func scanPath(root, handlerType string) ([]Entry, error) {
	var entries []Entry

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		found, err := scanFile(path, handlerType)
		if err != nil {
			return fmt.Errorf("scan %s: %w", path, err)
		}
		entries = append(entries, found...)
		return nil
	})

	return entries, err
}

// shouldSkipDir reports whether a directory tree should be excluded from the
// scan. Hidden directories (".git", ".idea"), dependency trees ("vendor",
// "node_modules"), and fixture dirs ("testdata") never carry cqrs markers and
// only slow the walk or risk false matches against third-party code.
func shouldSkipDir(name string) bool {
	if name == "vendor" || name == "node_modules" || name == "testdata" {
		return true
	}

	return strings.HasPrefix(name, ".")
}

// dedupEntries drops entries that share a StructName with an earlier entry.
// Two structs with the same name cannot coexist in one generated file
// (duplicate RegisterXxxHandler declarations), so the first wins and later
// collisions are reported so the caller can split the run if both are needed.
func dedupEntries(w io.Writer, entries []Entry) []Entry {
	seen := make(map[string]struct{}, len(entries))
	deduped := make([]Entry, 0, len(entries))

	for _, e := range entries {
		if _, ok := seen[e.StructName]; ok {
			_, _ = fmt.Fprintf(
				w,
				"warning: duplicate struct %q skipped (already registered)\n",
				e.StructName,
			)
			continue
		}
		seen[e.StructName] = struct{}{}
		deduped = append(deduped, e)
	}

	return deduped
}

func scanFile(path, handlerType string) ([]Entry, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var entries []Entry
	prefix := "//cqrs:" + handlerType + " "
	tagKey := handlerType

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
			markerValue := extractMarker(genDecl.Doc, prefix)
			if markerValue == "" {
				markerValue = extractMarker(ts.Doc, prefix)
			}

			// Fall back to struct tag
			if markerValue == "" {
				if structType, ok := ts.Type.(*ast.StructType); ok {
					markerValue = extractStructTag(structType, tagKey)
				}
			}

			if markerValue == "" {
				continue
			}

			entries = append(entries, Entry{
				TypeName:    markerValue,
				StructName:  ts.Name.Name,
				PackageName: f.Name.Name,
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

		// field.Tag.Value is the raw literal including backticks; strip exactly
		// one leading and one trailing backtick rather than treating "`" as a
		// trim cutset.
		raw := strings.TrimSuffix(strings.TrimPrefix(field.Tag.Value, "`"), "`")
		cqrsTag := reflect.StructTag(raw).Get("cqrs")
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

	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

`

	queryImports = `import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

`

	eventImports = `import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

`
)
