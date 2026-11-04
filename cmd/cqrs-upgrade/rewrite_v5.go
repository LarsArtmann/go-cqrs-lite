package main

// rewrite_v5.go implements the --to-v5 pass (T08 slice 1, ADR-0152): an
// AST-discovered, byte-exact import rewrite from the folded v4 trains to
// their core/v5 twins. Slice 1 deliberately does NOT edit go.mod — the core
// module is untagged, so an applied require would not resolve; the pass
// reports the require changes instead. Import literals are replaced only
// inside real import-spec ranges discovered by the parser, so occurrences in
// strings, comments, or elsewhere are never touched.

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// v5ImportTable maps every folded v4 import path to its core/v5 twin. The
// fold set is the set of Go package directories under core/v5 (testdata
// excluded); TestV5TableMatchesCoreTree pins the table to that tree, so
// extending the fold forces a deliberate table update here.
var v5ImportTable = map[string]string{
	"github.com/larsartmann/go-cqrs-lite/command/v4":            "github.com/larsartmann/go-cqrs-lite/core/v5/command",
	"github.com/larsartmann/go-cqrs-lite/command/v4/commandtest": "github.com/larsartmann/go-cqrs-lite/core/v5/command/commandtest",
	"github.com/larsartmann/go-cqrs-lite/dedup/v4":              "github.com/larsartmann/go-cqrs-lite/core/v5/dedup",
	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4":         "github.com/larsartmann/go-cqrs-lite/core/v5/dispatcher",
	"github.com/larsartmann/go-cqrs-lite/event/v4":              "github.com/larsartmann/go-cqrs-lite/core/v5/event",
	"github.com/larsartmann/go-cqrs-lite/id/v4":                 "github.com/larsartmann/go-cqrs-lite/core/v5/id",
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest":          "github.com/larsartmann/go-cqrs-lite/core/v5/id/idtest",
	"github.com/larsartmann/go-cqrs-lite/kv/v4":                 "github.com/larsartmann/go-cqrs-lite/core/v5/kv",
	"github.com/larsartmann/go-cqrs-lite/kv/v4/viewstoretest":   "github.com/larsartmann/go-cqrs-lite/core/v5/kv/viewstoretest",
	"github.com/larsartmann/go-cqrs-lite/metadata/v4":           "github.com/larsartmann/go-cqrs-lite/core/v5/metadata",
	"github.com/larsartmann/go-cqrs-lite/query/v4":              "github.com/larsartmann/go-cqrs-lite/core/v5/query",
	"github.com/larsartmann/go-cqrs-lite/query/v4/querytest":    "github.com/larsartmann/go-cqrs-lite/core/v5/query/querytest",
	"github.com/larsartmann/go-cqrs-lite/record/v4":             "github.com/larsartmann/go-cqrs-lite/core/v5/record",
}

// v4CqrsPrefix is the shared prefix of every go-cqrs-lite v4 import path.
const v4CqrsPrefix = "github.com/larsartmann/go-cqrs-lite/"

// isCqrsV4Import reports whether an import path belongs to a go-cqrs-lite
// v4 train (the module root or any train module path).
func isCqrsV4Import(path string) bool {
	if !strings.HasPrefix(path, v4CqrsPrefix) {
		return false
	}
	rest := strings.TrimPrefix(path, v4CqrsPrefix)

	return rest == "v4" || strings.Contains(rest, "/v4") || strings.HasPrefix(rest, "v4/")
}

// v5ModuleReport is the --to-v5 result for one module directory.
type v5ModuleReport struct {
	Dir          string
	FilesScanned int
	Rewrites     []v5FileRewrite
	RemainingV4  []string // unique not-yet-folded cqrs v4 imports (sorted)
	Error        string
}

// v5FileRewrite records one file's applied (or planned) import changes.
type v5FileRewrite struct {
	File    string
	Changes []v5ImportChange
}

// v5ImportChange is one v4 -> v5 import literal replacement.
type v5ImportChange struct {
	From string
	To   string
}

// runV5Pass walks dir's Go files (vendor/, testdata/, .git/ excluded) and
// rewrites folded v4 imports to their core/v5 twins. With dryRun set nothing
// is written; the report is identical either way.
func runV5Pass(dir string, dryRun bool) v5ModuleReport {
	rep := v5ModuleReport{Dir: dir}

	remaining := map[string]bool{}

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()

			if path != dir && (name == "vendor" || name == "testdata" || name == ".git" || name == "node_modules") {
				return filepath.SkipDir
			}

			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		changes, remains, err := rewriteV5ImportsInFile(path, dryRun)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		rep.FilesScanned++
		for _, r := range remains {
			remaining[r] = true
		}
		if len(changes) > 0 {
			rep.Rewrites = append(rep.Rewrites, v5FileRewrite{File: path, Changes: changes})
		}

		return nil
	})
	if err != nil {
		rep.Error = err.Error()

		return rep
	}

	rep.RemainingV4 = sortedKeys(remaining)

	return rep
}

// rewriteV5ImportsInFile rewrites the folded v4 import literals of one file.
// It returns the changes made (or planned, in dry-run mode) plus every cqrs
// v4 import that has no v5 twin yet. Byte-exact: only the quoted literals
// inside real import specs are replaced, in descending offset order.
func rewriteV5ImportsInFile(path string, dryRun bool) ([]v5ImportChange, []string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
	if err != nil {
		return nil, nil, fmt.Errorf("parse: %w", err)
	}

	type span struct {
		start, end int // byte offsets of the quoted import literal
		from, to   string
	}
	var spans []span
	remaining := map[string]bool{}

	for _, imp := range f.Imports {
		path2, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, nil, fmt.Errorf("unquote %s: %w", imp.Path.Value, err)
		}
		to, folded := v5ImportTable[path2]
		switch {
		case folded:
			spans = append(spans, span{start: imp.Path.Pos() - 1, end: imp.Path.End() - 1, from: path2, to: to})
		case isCqrsV4Import(path2):
			remaining[path2] = true
		}
	}

	if len(spans) == 0 {
		return nil, sortedKeys(remaining), nil
	}

	changes := make([]v5ImportChange, 0, len(spans))
	out := string(src)
	for i := len(spans) - 1; i >= 0; i-- {
		s := spans[i]
		out = out[:s.start] + `"` + s.to + `"` + out[s.end:]
		changes = append([]v5ImportChange{{From: s.from, To: s.to}}, changes...)
	}

	if !dryRun {
		info, err := os.Stat(path)
		if err != nil {
			return nil, nil, err
		}
		if err := os.WriteFile(path, []byte(out), info.Mode().Perm()); err != nil {
			return nil, nil, err
		}
	}

	return changes, sortedKeys(remaining), nil
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}
// runV5Mode executes --to-v5 over dirs and prints the human report; with
// --strict it fails on errored modules or not-yet-folded v4 imports.
func runV5Mode(cfg config, dirs []string) error {
	reports := make([]v5ModuleReport, 0, len(dirs))
	for _, dir := range dirs {
		reports = append(reports, runV5Pass(dir, cfg.dryRun))
	}

	for i, rep := range reports {
		if cfg.workspace {
			fmt.Printf("== %s ==\n", rep.Dir)
		}

		printV5Report(rep)

		if i < len(reports)-1 {
			fmt.Println()
		}
	}

	if cfg.dryRun {
		fmt.Println("dry run: no changes written")
	}

	return strictV5GateError(cfg.strict, reports)
}

// printV5Report renders one module's v5 pass result.
func printV5Report(rep v5ModuleReport) {
	if rep.Error != "" {
		fmt.Printf("v5 pass errored: %s\n", rep.Error)

		return
	}

	n := 0
	for _, fw := range rep.Rewrites {
		n += len(fw.Changes)
	}
	fmt.Printf("scanned %d Go files: %d import rewrite(s) in %d file(s)\n",
		rep.FilesScanned, n, len(rep.Rewrites))
	for _, fw := range rep.Rewrites {
		for _, c := range fw.Changes {
			fmt.Printf("  %s\n    %s\n    %s\n", fw.File, c.From, c.To)
		}
	}
	if len(rep.RemainingV4) > 0 {
		fmt.Printf("  not-yet-folded v4 imports (%d): %s\n",
			len(rep.RemainingV4), strings.Join(rep.RemainingV4, ", "))
	}
	fmt.Println("  go.mod: report-only (core/v5 untagged) — add the require once it tags")
}

// strictV5GateError is the --strict verdict for --to-v5: an errored module
// fails first (unscanned is unproven), then any not-yet-folded v4 import.
func strictV5GateError(strict bool, reports []v5ModuleReport) error {
	if !strict {
		return nil
	}
	for _, rep := range reports {
		if rep.Error != "" {
			return fmt.Errorf("%w in %s — see report", errStrictV5ModuleFailed, rep.Dir)
		}
	}
	for _, rep := range reports {
		if len(rep.RemainingV4) > 0 {
			return fmt.Errorf("%w in %s (%d) — see report",
				errStrictV5Remaining, rep.Dir, len(rep.RemainingV4))
		}
	}

	return nil
}
