package version

import (
	"os"
	"strings"
)

// cqrsRequire represents a single go-cqrs-lite dependency line in go.mod.
type cqrsRequire struct {
	Path     string // full module path, e.g. "github.com/larsartmann/go-cqrs-lite/event/v4"
	Version  string // version string, e.g. "v4.2.0"
	Line     int    // 1-based line number in go.mod
	Indirect bool   // true if the require has an // indirect comment
}

// parseGoModCQRSRequires reads a go.mod file and returns all go-cqrs-lite
// require directives (skipping replace directives). It is a lightweight
// text-based parser — no dependency on golang.org/x/mod.
func parseGoModCQRSRequires(path string) []cqrsRequire {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var requires []cqrsRequire

	lines := strings.Split(string(data), "\n")
	inRequireBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "replace ") || strings.HasPrefix(trimmed, "replace\t") {
			inRequireBlock = false
			continue
		}

		if trimmed == "require (" {
			inRequireBlock = true
			continue
		}

		if trimmed == ")" && inRequireBlock {
			inRequireBlock = false
			continue
		}

		var req string
		if inRequireBlock {
			req = trimmed
		} else if rest, ok := strings.CutPrefix(trimmed, "require "); ok {
			req = rest
		} else {
			continue
		}

		cr := parseRequireLine(req, i+1)
		if cr != nil {
			requires = append(requires, *cr)
		}
	}

	return requires
}

// shortModuleName extracts the module suffix from a full import path.
// e.g. "github.com/larsartmann/go-cqrs-lite/event/v4" → "event/v4".
func shortModuleName(path string) string {
	const prefix = "github.com/larsartmann/go-cqrs-lite/"
	if strings.HasPrefix(path, prefix) {
		return path[len(prefix):]
	}
	return path
}

// majorMinorVersion extracts the major and minor numbers from a semver string.
// e.g. "v4.2.0" → (4, 2, true). Returns (0, 0, false) on parse failure.
func majorMinorVersion(v string) (major, minor int, ok bool) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}

	major, err1 := atoi(parts[0])
	minor, err2 := atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}

	return major, minor, true
}

// atoi is a stdlib-free integer parser for small non-negative numbers.
func atoi(s string) (int, error) {
	if s == "" {
		return 0, errEmptyString
	}

	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errNotDigit
		}

		n = n*10 + int(c-'0')
	}

	return n, nil
}

var (
	errEmptyString = &parseError{msg: "empty string"}
	errNotDigit    = &parseError{msg: "non-digit character"}
)

type parseError struct{ msg string }

func (e *parseError) Error() string { return e.msg }

// parseRequireLine extracts a cqrsRequire from a single require line like
// "github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0 // indirect".
// Returns nil for non-cqrs lines.
func parseRequireLine(line string, lineNum int) *cqrsRequire {
	if !strings.Contains(line, "go-cqrs-lite") {
		return nil
	}

	indirect := strings.Contains(line, "// indirect")
	codePart := line
	if idx := strings.Index(codePart, "//"); idx >= 0 {
		codePart = strings.TrimSpace(codePart[:idx])
	}

	parts := strings.Fields(codePart)
	if len(parts) < 2 {
		return nil
	}

	return &cqrsRequire{
		Path:     parts[0],
		Version:  parts[1],
		Line:     lineNum,
		Indirect: indirect,
	}
}
