package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// navIssue is one broken navigation reference (TOC anchor or § cross-ref).
type navIssue struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Msg  string `json:"msg"`
}

// secToken matches one § cross-reference: "§2.13", "§2.21b", "§2.27/2.28",
// "§6.15–6.16" (slash or dash lists expand to one check per endpoint).
var secToken = regexp.MustCompile(
	`§[0-9][0-9a-zA-Z]*(?:\.[0-9]+)*[a-z]?(?:[ \t]*[/–—-][ \t]*§?[0-9][0-9a-zA-Z]*(?:\.[0-9]+)*[a-z]?)*`,
)

var secNumber = regexp.MustCompile(`[0-9][0-9a-zA-Z]*(?:\.[0-9]+)*[a-z]?`)

var (
	adrSuffix    = regexp.MustCompile(`ADR-[0-9]+$`)
	formerSuffix = regexp.MustCompile(`\bformer\s+[A-Za-z0-9_-]+(?:\.md)?$`)
	nameSuffix   = regexp.MustCompile(`([A-Za-z][A-Za-z0-9_-]*\.md|[A-Za-z][A-Za-z0-9_-]*)$`)
	linkSuffix   = regexp.MustCompile(`\]\(([^)]+)\)$`)
)

// docNav is the parsed navigation surface of one markdown document.
type docNav struct {
	headings []heading
	slugs    map[string]int
	numbers  map[string]bool
}

// navChecker validates TOC anchors and § cross-refs across the checked file
// set. Bare § refs resolve against the referencing file first, then — only
// when exactly one checked document has that section number — against the
// pool; zero or multiple matches are reported so the ref gets a doc prefix.
type navChecker struct {
	repoRoot string
	pool     map[string]*docNav // keyed by absolute real path
	issues   []navIssue
}

func newNavChecker(repoRoot string) *navChecker {
	return &navChecker{repoRoot: repoRoot, pool: make(map[string]*docNav)}
}

// checkFiles validates every checked file and returns the issues found.
func checkFiles(files []string, repoRoot string) []navIssue {
	nc := newNavChecker(repoRoot)

	// Preload the § pool: only skill-scope docs follow the § convention.
	for _, f := range files {
		if !secScoped(f, repoRoot) {
			continue
		}

		if dn := nc.load(f); dn != nil {
			nc.pool[nc.realPath(f)] = dn
		}
	}

	for _, f := range files {
		nc.checkFile(f)
	}

	return nc.issues
}

// secScoped reports whether a file follows the skill-docs § cross-ref
// convention (doc-check's default scan set). Project prose (README,
// TODO_LIST, ROADMAP, ...) uses § too freely-formatted, so § refs there are
// not validated; TOC anchors are universal and checked everywhere.
func secScoped(path, repoRoot string) bool {
	real := path

	if abs, err := filepath.Abs(path); err == nil {
		real = abs
	}

	if rel, err := filepath.Rel(repoRoot, real); err == nil {
		if rel == "AGENTS.md" ||
			rel == "docs/DOMAIN_LANGUAGE.md" ||
			rel == "docs/METAENGINE_DOMAIN_LANGUAGE.md" {
			return true
		}
	}

	return strings.Contains(real, "/.agents/skills/")
}

// load parses one markdown file's navigation surface (nil when unreadable —
// plain file existence is check-doc-links.sh's domain, not ours).
func (nc *navChecker) load(path string) *docNav {
	real := nc.realPath(path)

	if dn, ok := nc.pool[real]; ok {
		return dn
	}

	data, err := os.ReadFile(real)
	if err != nil {
		return nil
	}

	dn := &docNav{
		headings: parseHeadings(string(data)),
		slugs:    map[string]int{},
		numbers:  map[string]bool{},
	}
	dn.slugs = slugCounts(dn.headings)
	dn.numbers = numberSet(dn.headings)

	nc.pool[real] = dn

	seen := map[string]int{}
	for _, h := range dn.headings {
		if h.number == "" {
			continue
		}

		if first, dup := seen[h.number]; dup {
			nc.issue(path, h.line, fmt.Sprintf(
				"duplicate section number §%s (first at line %d)", h.number, first))

			continue
		}

		seen[h.number] = h.line
	}

	return dn
}

// realPath resolves symlinks so relative targets resolve against the file's
// real directory (SKILL.md is a symlink into .agents/skills/...).
func (nc *navChecker) realPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}

	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}

	return path
}

func (nc *navChecker) issue(file string, line int, msg string) {
	nc.issues = append(nc.issues, navIssue{File: file, Line: line, Msg: msg})
}

// checkFile validates anchors and § refs of one file against visible text
// (fenced blocks skipped, inline-code spans blanked at identical offsets).
func (nc *navChecker) checkFile(path string) {
	self := nc.load(path)
	if self == nil {
		return
	}

	real := nc.realPath(path)
	dir := filepath.Dir(real)

	data, err := os.ReadFile(real)
	if err != nil {
		return
	}

	for _, vl := range visibleLines(string(data)) {
		if movedBulletRe.MatchString(vl.text) {
			continue // TOC bullet pointing at a section that moved away
		}

		nc.checkAnchors(path, vl, dir, self)

		if secScoped(path, nc.repoRoot) {
			nc.checkSecRefs(path, vl, dir, self)
		}
	}
}

// checkSecRefs validates every § cross-reference on one visible line.
func (nc *navChecker) checkSecRefs(path string, vl visibleLine, dir string, self *docNav) {
	for _, loc := range secToken.FindAllStringIndex(vl.text, -1) {
		token := vl.text[loc[0]:loc[1]]

		target, ok := nc.secTarget(vl.text[:loc[0]], dir, path)
		if !ok {
			continue
		}

		for _, num := range secNumbers(token) {
			nc.checkSecNumber(path, vl.num, token, num, target, self)
		}
	}
}

// secTarget resolves the § ref's target document from its immediate prefix.
// Filters keep deliberate shapes working: ADR-relative refs (ADR-0124 §7),
// historical moved-pointers ("the former recipes §2.3"), and refs whose doc
// name cannot be resolved fall back to bare resolution.
func (nc *navChecker) secTarget(before, dir, path string) (*docNav, bool) {
	tail := strings.TrimRight(strings.ReplaceAll(before, "`", ""), " \t")

	switch {
	case adrSuffix.MatchString(tail), formerSuffix.MatchString(tail):
		return nil, false
	}

	if m := linkSuffix.FindStringSubmatch(tail); m != nil {
		if dn := nc.load(filepath.Join(dir, m[1])); dn != nil {
			return dn, true
		}
	}

	if m := nameSuffix.FindStringSubmatch(tail); m != nil {
		if dn := nc.loadDocName(m[1], dir); dn != nil {
			return dn, true
		}
	}

	return nc.pool[nc.realPath(path)], true // bare: resolve against self
}

// loadDocName resolves a short doc name ("recipes", "core.md") to a checked
// document: same dir, its references/ subdir, or the repo skill references.
func (nc *navChecker) loadDocName(name, dir string) *docNav {
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}

	for _, cand := range []string{
		filepath.Join(dir, name),
		filepath.Join(dir, "references", name),
		filepath.Join(dir, "..", "references", name),
		filepath.Join(nc.repoRoot, ".agents", "skills", "go-cqrs-lite", "references", name),
		filepath.Join(nc.repoRoot, ".agents", "skills", "go-cqrs-lite", name),
	} {
		if dn := nc.load(cand); dn != nil {
			return dn
		}
	}

	return nil
}

// checkSecNumber validates one § endpoint against its target document. Bare
// refs fall back to the checked pool when the referencing doc lacks the
// number: exactly one matching document is accepted, anything else is an
// issue so the ref gains an explicit doc prefix.
func (nc *navChecker) checkSecNumber(
	path string, line int, token, num string, target *docNav, self *docNav,
) {
	if target == nil {
		target = self
	}

	if target.numbers[num] {
		return
	}

	if target == self {
		var hits []*docNav

		for _, dn := range nc.pool {
			if dn.numbers[num] {
				hits = append(hits, dn)
			}
		}

		switch {
		case len(hits) == 1:
			return // unique across the checked docs: accept bare ref
		case len(hits) > 1:
			nc.issue(path, line, fmt.Sprintf(
				"ambiguous %s: §%s exists in %d checked docs — name the doc",
				token,
				num,
				len(hits),
			))

			return
		}
	}

	nc.issue(path, line, fmt.Sprintf(
		"broken § cross-ref %s: no section §%s in target doc", token, num))
}

// secNumbers splits one § token into its endpoint numbers.
func secNumbers(token string) []string {
	var nums []string

	for _, m := range secNumber.FindAllString(token, -1) {
		nums = append(nums, m)
	}

	return nums
}
