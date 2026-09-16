package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	linkTarget  = regexp.MustCompile(`\]\(([^)]+)\)`)
	dedupSuffix = regexp.MustCompile(`^(.+)-([0-9]+)$`)
)

var movedBulletRe = regexp.MustCompile(`^\s*>?\s*[-*]\s*§[0-9].*moved to\b`)

// checkAnchors validates every markdown link that carries a #fragment.
func (nc *navChecker) checkAnchors(path string, vis visibleLine, dir string, self *docNav) {
	for _, m := range linkTarget.FindAllStringSubmatchIndex(vis.text, -1) {
		target := vis.text[m[2]:m[3]]

		if !anchorWorthy(target) {
			continue
		}

		fileSlugs, ok := nc.targetSlugs(target, dir, self)
		if !ok {
			continue // plain file target (or unreadable): not our gate
		}

		frag := target[strings.Index(target, "#")+1:]
		if fileSlugs[frag] > 0 {
			continue
		}

		if base, n, ok := splitDedup(frag); ok && fileSlugs[base] > n {
			continue // GitHub's duplicate-heading -1/-2 suffix
		}

		nc.issue(path, vis.num, fmt.Sprintf("broken anchor %q -> no heading slugs to it", target))
	}
}

// anchorWorthy filters targets that carry a fragment we should validate.
func anchorWorthy(target string) bool {
	if strings.ContainsAny(target, " \t") || !strings.Contains(target, "#") {
		return false
	}

	for _, scheme := range []string{"http://", "https://", "mailto:", "data:"} {
		if strings.HasPrefix(target, scheme) {
			return false
		}
	}

	if idx := strings.Index(target, "#"); idx == 0 {
		return true
	} else if matched, _ := regexp.MatchString(`\.go:[0-9]+$`, target[:idx]); matched {
		return false
	}

	return true
}

// targetSlugs returns the slug set of the link's target document. Same-file
// fragments (#foo) resolve to the referencing file; file#frag targets are
// resolved relative to dir and silently skipped when the file is missing
// (broken file paths are check-doc-links.sh's domain).
func (nc *navChecker) targetSlugs(target, dir string, self *docNav) (map[string]int, bool) {
	idx := strings.Index(target, "#")

	if idx == 0 {
		return self.slugs, true
	}

	resolved := filepath.Join(dir, target[:idx])
	if doc := nc.load(resolved); doc != nil {
		return doc.slugs, true
	}

	return nil, false
}

// splitDedup parses GitHub's duplicate-heading anchor suffix: "#slug-2" ->
// ("slug", 2, true).
func splitDedup(frag string) (string, int, bool) {
	m := dedupSuffix.FindStringSubmatch(frag)
	if m == nil {
		return "", 0, false
	}

	n, err := strconv.Atoi(m[2])
	if err != nil || n < 1 {
		return "", 0, false
	}

	return m[1], n, true
}
