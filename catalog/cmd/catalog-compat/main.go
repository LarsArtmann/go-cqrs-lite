// Command catalog-compat classifies payload-schema changes for a catalog.
//
// It turns docs/SCHEMA-COMPATIBILITY.md into a build gate:
//
//	catalog-compat baseline --catalog catalog.json --out compat-baseline.json
//	catalog-compat check    --catalog catalog.json --baseline compat-baseline.json
//	catalog-compat validate --catalog catalog.json
//	catalog-compat conflicts --left a.json --right b.json
//
// Exit codes: 0 clean, 1 gate failure, 2 usage or I/O error.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/compat"
)

const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

const defaultBaselineName = "compat-baseline.json"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)

		return exitUsage
	}

	switch args[0] {
	case "baseline":
		return runBaseline(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "conflicts":
		return runConflicts(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)

		return exitOK
	default:
		fmt.Fprintf(stderr, "catalog-compat: unknown command %q\n", args[0])
		usage(stderr)

		return exitUsage
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `catalog-compat — executable schema-compatibility policy

  baseline   --catalog FILE [--out FILE]           write the shipped-shape snapshot
  check      --catalog FILE [--baseline FILE]      fail on a breaking change with no version bump
  validate   --catalog FILE                        run the cross-resource catalog invariants
  conflicts  --left FILE --right FILE              report divergent declarations before a merge

Exit codes: 0 clean, 1 gate failure, 2 usage or I/O error.
`)
}

func runBaseline(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("baseline", flag.ContinueOnError)
	flags.SetOutput(stderr)

	catalogPath := flags.String("catalog", "", "catalog JSON file")
	outPath := flags.String("out", defaultBaselineName, "baseline file to write")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	cat, code := loadCatalog(*catalogPath, stderr)
	if code != exitOK {
		return code
	}

	baseline := compat.NewBaseline(cat)
	if err := baseline.WriteFile(*outPath); err != nil {
		fmt.Fprintf(stderr, "catalog-compat: %v\n", err)

		return exitUsage
	}

	fmt.Fprintf(stdout, "wrote %d message(s) to %s\n", len(baseline.Messages), *outPath)

	return exitOK
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(stderr)

	catalogPath := flags.String("catalog", "", "catalog JSON file")
	baselinePath := flags.String(
		"baseline",
		defaultBaselineName,
		"baseline file to compare against",
	)

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	baseline, err := compat.ParseBaseline(*baselinePath)
	if err != nil {
		fmt.Fprintf(stderr, "catalog-compat: %v\n", err)

		return exitUsage
	}

	cat, code := loadCatalog(*catalogPath, stderr)
	if code != exitOK {
		return code
	}

	report := compat.CompareBaseline(baseline, cat)

	return emitReport("check", report, stdout, stderr)
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)

	catalogPath := flags.String("catalog", "", "catalog JSON file")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	cat, code := loadCatalog(*catalogPath, stderr)
	if code != exitOK {
		return code
	}

	violations := compat.Validate(cat)
	for _, violation := range violations {
		fmt.Fprintf(stdout, "%s: %s\n", violation.Path, violation.Message)
	}

	if len(violations) > 0 {
		fmt.Fprintf(stderr, "catalog-compat: %d invariant violation(s)\n", len(violations))

		return exitFail
	}

	fmt.Fprintln(stdout, "catalog-compat: invariants ok")

	return exitOK
}

func runConflicts(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("conflicts", flag.ContinueOnError)
	flags.SetOutput(stderr)

	leftPath := flags.String("left", "", "left catalog JSON file")
	rightPath := flags.String("right", "", "right catalog JSON file")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	left, code := loadCatalog(*leftPath, stderr)
	if code != exitOK {
		return code
	}

	right, code := loadCatalog(*rightPath, stderr)
	if code != exitOK {
		return code
	}

	conflicts := compat.Conflicts(left, right)
	for _, conflict := range conflicts {
		fmt.Fprintln(stdout, conflict)
	}

	if len(conflicts) > 0 {
		fmt.Fprintf(stderr, "catalog-compat: %d conflicting declaration(s)\n", len(conflicts))

		return exitFail
	}

	fmt.Fprintln(stdout, "catalog-compat: no conflicts")

	return exitOK
}

func emitReport(command string, report compat.Report, stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, report)

	blocking := report.Blocking()
	if len(blocking) == 0 {
		return exitOK
	}

	for _, finding := range blocking {
		fmt.Fprintf(stderr, "::error::%s: %s\n", finding.Path, finding.Message)
	}

	fmt.Fprintf(
		stderr,
		"catalog-compat: %s failed with %d blocking finding(s)\n",
		command,
		len(blocking),
	)

	return exitFail
}

func loadCatalog(path string, stderr io.Writer) (*catalog.Catalog, int) {
	if path == "" {
		fmt.Fprintln(stderr, "catalog-compat: --catalog is required")
		return nil, exitUsage
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "catalog-compat: read catalog %s: %v\n", path, err)

		return nil, exitUsage
	}

	cat := &catalog.Catalog{}
	if err := json.Unmarshal(data, cat); err != nil {
		fmt.Fprintf(stderr, "catalog-compat: parse catalog %s: %v\n", path, err)

		return nil, exitUsage
	}

	if len(cat.Services) == 0 {
		fmt.Fprintf(stderr, "catalog-compat: warning: catalog %s declares no services\n", path)
	}

	return cat, exitOK
}
