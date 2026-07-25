package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

func writeResult(format, output string, config benchkit.Config, result *benchkit.Result) {
	w := openOutput(output)
	defer closeOutput(w)

	switch format {
	case "json":
		if err := benchkit.WriteJSON(w, result); err != nil {
			fatalf("write JSON: %v", err)
		}
	case "benchstat":
		benchkit.WriteBenchstat(w, result)
	case "manifest":
		if err := benchkit.WriteManifest(w, config, result); err != nil {
			fatalf("write manifest: %v", err)
		}
	default:
		benchkit.PrintReport(w, result)
	}
}

func writeComparison(
	format, output string,
	results map[string]*benchkit.Result,
) {
	w := openOutput(output)
	defer closeOutput(w)

	switch format {
	case "json":
		if err := benchkit.WriteComparisonJSON(w, results); err != nil {
			fatalf("write JSON: %v", err)
		}
	case "markdown":
		benchkit.PrintMarkdown(w, results)
	default:
		benchkit.PrintComparison(w, results)
	}
}

func writeSweep(format, output string, results []benchkit.SweepResult) {
	w := openOutput(output)
	defer closeOutput(w)

	switch format {
	case "json":
		if err := benchkit.WriteSweepJSON(w, results); err != nil {
			fatalf("write JSON: %v", err)
		}
	default:
		benchkit.PrintSweep(w, results)
	}
}

func openOutput(path string) *os.File {
	if path == "" || path == "-" {
		return os.Stdout
	}

	f, err := os.Create(path)
	if err != nil {
		fatalf("create output file: %v", err)
	}

	return f
}

func closeOutput(f *os.File) {
	if f != os.Stdout {
		_ = f.Close()
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cqrs-bench: "+format+"\n", args...)
	os.Exit(1)
}

func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}

	v := info.Main.Version
	if v != "" && v != "(devel)" {
		return v
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
			return "(devel, " + setting.Value[:7] + ")"
		}
	}

	return "(devel)"
}
