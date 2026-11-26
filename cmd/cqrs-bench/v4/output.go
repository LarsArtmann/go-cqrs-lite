package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"

	atomicwrite "github.com/larsartmann/go-atomic-write"
	"github.com/larsartmann/go-output"
)

// reportTarget is where a report goes: stdout, or a file that is replaced
// atomically once the whole report is rendered.
type reportTarget struct {
	// path is the report file; empty or "-" means stdout.
	path string
	// terminal says the destination renders like an interactive terminal:
	// tables may use color and --format auto picks the table layout.
	terminal bool
}

// stdoutTarget reports to stdout. The decision looks at stdout itself, not
// at whether go-output would color it: NO_COLOR or CI=true used to downgrade
// a real terminal to the plain-text report.
func stdoutTarget() reportTarget {
	return reportTarget{terminal: isCharDevice(os.Stdout) || output.ColorModeAuto.ShouldColor()}
}

// fileTarget reports to the file at path; "" and "-" mean stdout.
func fileTarget(path string) reportTarget {
	if path == "" || path == "-" {
		return stdoutTarget()
	}

	return reportTarget{path: path}
}

// isCharDevice reports whether f is a character device, which is what a
// terminal is (/dev/null qualifies too: a table rendered into it is lost
// either way).
func isCharDevice(f *os.File) bool {
	info, err := f.Stat()

	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// color is the color mode for reports written to the target: a file never
// contains color codes.
func (t reportTarget) color() output.ColorMode {
	if t.terminal {
		return output.ColorModeAuto
	}

	return output.ColorModeNever
}

// resolveFormat picks the default layout for the target when format is auto:
// a table for a terminal, text for anything else. An explicit format passes
// through.
func (t reportTarget) resolveFormat(format string) string {
	if format != "" && format != formatAuto {
		return format
	}

	if t.terminal {
		return formatTable
	}

	return formatText
}

// write renders the report into the target. A file is staged and renamed,
// so an interrupted or failed render never leaves a truncated report behind,
// and a failed write is an error rather than a silent empty file.
func (t reportTarget) write(render func(w io.Writer) error) error {
	if t.path == "" {
		return render(os.Stdout)
	}

	if err := atomicwrite.WriteFunc(t.path, render); err != nil {
		return fmt.Errorf("%w %s: %w", errWriteReport, t.path, err)
	}

	return nil
}

// checkWritable fails before the benchmark starts when its report could not
// be written afterwards (missing directory, read-only location). The probe
// file is created and removed next to the target; the target itself is not
// touched.
func checkWritable(path string) error {
	target := fileTarget(path)
	if target.path == "" {
		return nil
	}

	probe, err := os.CreateTemp(filepath.Dir(target.path), ".cqrs-bench-*")
	if err != nil {
		return fmt.Errorf("%w: --output %s cannot be written: %w", errFlagValue, target.path, err)
	}

	probePath := probe.Name()

	_ = probe.Close()
	_ = os.Remove(probePath)

	return nil
}

// warnf prints a warning to stderr. Warnings are not progress output: they
// report ignored flags and failed cleanup, so --quiet keeps them.
func warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cqrs-bench: warning: "+format+"\n", args...)
}

// reportClose closes every binding and reports the ones it could not clean
// up, naming the directory so the operator can remove it.
func reportClose(bindings []binding) {
	for _, bound := range bindings {
		if err := bound.close(); err != nil {
			warnf("%v", err)
		}
	}
}

// warnBindings prints the backend warnings gathered while opening.
func warnBindings(bindings []binding) {
	for _, bound := range bindings {
		for _, warning := range bound.warnings {
			warnf("%s", warning)
		}
	}
}

// fatalf ends the process before any command has run. Command handlers
// return errors instead, so their deferred cleanup (temporary stores,
// profiles) runs.
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
