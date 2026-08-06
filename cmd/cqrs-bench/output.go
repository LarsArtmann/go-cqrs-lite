package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

func writeResult(format, output string, config benchkit.Config, result *benchkit.Result) {
	withOutput(output, func(w *os.File) {
		renderRunResult(w, resolveFormat(format), config, result)
	})
}

func writeComparison(
	format, output string,
	results map[string]*benchkit.Result,
) {
	withOutput(output, func(w *os.File) {
		renderComparison(w, resolveFormat(format), results)
	})
}

func writeSweep(format, output string, results []benchkit.SweepResult) {
	withOutput(output, func(w *os.File) {
		renderSweep(w, resolveFormat(format), results)
	})
}

func writeSoakResult(format, output string, result *benchkit.SoakResult) {
	withOutput(output, func(w *os.File) {
		renderSoakResult(w, resolveFormat(format), result)
	})
}

// withOutput opens the output destination and calls fn with the writer,
// ensuring it is closed when fn returns. Collapses the repeated
// "w := openOutput(output); defer closeOutput(w)" prologue shared by every
// write* entry point so the open/close lifecycle lives in one place.
func withOutput(output string, fn func(w *os.File)) {
	w := openOutput(output)
	defer closeOutput(w)

	fn(w)
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
		//cqrs-lint:ignore(C015,C023) file close helper, error not actionable
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
