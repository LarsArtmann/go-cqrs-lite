package main

import (
	"bufio"
	"os"
	"strings"
)

// generatedSuffixes are Go file suffixes produced by code generators; they are
// excluded from analysis by default because their churn reflects tooling, not
// hand-written maintenance cost.
var generatedSuffixes = []string{".gen.go", "_gen.go", ".pb.go", ".pb.gw.go", ".templ.go"}

// countSLOC returns the number of source lines: non-blank lines that are not
// comment-only. Files that no longer exist (deleted in history) yield 0.
func countSLOC(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	var n int
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if t == "" {
			continue
		}
		if isComment(t) {
			continue
		}
		n++
	}
	return n
}

// isComment reports whether a trimmed line is a comment-only line.
func isComment(t string) bool {
	return strings.HasPrefix(t, "//") || strings.HasPrefix(t, "/*") || strings.HasPrefix(t, "*")
}

// isGenerated reports whether path looks like a generated file.
func isGenerated(path string) bool {
	for _, s := range generatedSuffixes {
		if strings.HasSuffix(path, s) {
			return true
		}
	}
	return false
}
