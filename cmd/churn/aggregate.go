package main

import (
	"os/exec"
	"strings"
)

// aggregateMode selects how per-file stats are grouped.
type aggregateMode int

const (
	aggFile aggregateMode = iota
	aggDir
	aggModule
)

// parseAggregateMode maps a flag string to an aggregateMode.
func parseAggregateMode(s string) aggregateMode {
	switch s {
	case "dir", "directory":
		return aggDir
	case "module", "mod":
		return aggModule
	default:
		return aggFile
	}
}

// aggregate merges per-file stats into the requested grouping.
func aggregate(stats map[string]*FileStat, mode aggregateMode) []*FileStat {
	if mode == aggFile {
		out := make([]*FileStat, 0, len(stats))
		for _, s := range stats {
			out = append(out, s)
		}
		return out
	}
	key := keyerFor(mode)
	merged := make(map[string]*FileStat)
	for _, s := range stats {
		k := key(s.Path)
		m := merged[k]
		if m == nil {
			m = &FileStat{Path: k, Authors: make(map[string]struct{})}
			merged[k] = m
		}
		mergeStat(m, s)
	}
	out := make([]*FileStat, 0, len(merged))
	for _, s := range merged {
		out = append(out, s)
	}
	return out
}

// mergeStat folds src into dst, unioning authors and summing numeric fields.
func mergeStat(dst, src *FileStat) {
	dst.Commits += src.Commits
	dst.Added += src.Added
	dst.Deleted += src.Deleted
	dst.Weight += src.Weight
	dst.Complexity += src.Complexity
	for a := range src.Authors {
		dst.Authors[a] = struct{}{}
	}
	if (dst.FirstTouch.IsZero() || src.FirstTouch.Before(dst.FirstTouch)) && !src.FirstTouch.IsZero() {
		dst.FirstTouch = src.FirstTouch
	}
	if src.LastTouch.After(dst.LastTouch) {
		dst.LastTouch = src.LastTouch
	}
}

// keyerFor returns a path-to-group-key function for the given mode.
func keyerFor(mode aggregateMode) func(string) string {
	switch mode {
	case aggDir:
		return dirKey
	case aggModule:
		roots := moduleRoots()
		return func(p string) string { return moduleKey(p, roots) }
	default:
		return func(p string) string { return p }
	}
}

// dirKey returns the immediate parent directory of a path.
func dirKey(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	return "."
}

// moduleRoots returns the set of directories containing a go.mod file.
func moduleRoots() map[string]bool {
	out, err := exec.Command("git", "ls-files", "*/go.mod", "go.mod").Output()
	if err != nil {
		return nil
	}
	roots := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		roots[dirKey(line)] = true
	}
	return roots
}

// moduleKey returns the nearest ancestor module root for a path, falling back
// to the directory if no enclosing go.mod is found.
func moduleKey(path string, roots map[string]bool) string {
	d := path
	for {
		d = dirKey(d)
		if roots[d] {
			return d
		}
		if d == "." {
			break
		}
	}
	return dirKey(path)
}
