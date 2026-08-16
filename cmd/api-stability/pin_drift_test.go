package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Sibling-module requires: path + version, with optional trailing comment.
var siblingRequireRe = regexp.MustCompile(
	`^\s*(github\.com/larsartmann/go-cqrs-lite/[^\s]+)\s+(v[0-9][^\s]+)`,
)

var versionSuffixRe = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)

// TestSiblingModulePinsResolve guards the standalone-build (GOWORK=off) graph:
// every require of a sibling module that is NOT governed by a replace
// directive must resolve to a real, published tag.
//
// Two failure classes, from real incidents:
//   - A pinned version that has no tag (commit 169b5d42: v4.x.y assumed but
//     never created) — standalone builds fail to resolve.
//   - A pseudo-version pin (v4.2.1-0.2026...) — resolves only while the
//     commit is reachable; the tag-release gate exists to prevent these.
//
// Staleness (pin older than the latest tag) is reported as a WARNING, not a
// failure: the repo-wide sweep policy is a pending user decision (ROADMAP
// Open Questions), and every module currently lags somewhere. Flip
// enforceStaleness to true once the sweep lands.
func TestSiblingModulePinsResolve(t *testing.T) {
	root := repoRoot(t)

	if !gitAvailable(t, root) {
		t.Skip(
			"git tags unavailable (hermetic/nix source) — pin drift check requires a git checkout",
		)
	}

	allTags := listTags(t, root)

	enforceStaleness := false

	var broken, stale []string

	for _, mod := range modules {
		goModPath := filepath.Join(root, mod, "go.mod")

		if _, err := os.Stat(goModPath); os.IsNotExist(err) {
			// Tracked package without its own module (e.g. id/idtest) — no
			// pin graph to check.
			continue
		}

		requireLines, replacePaths := parseGoMod(t, goModPath)

		for dep, version := range requireLines {
			if _, replaced := replacePaths[dep]; replaced {
				// A replace directive governs resolution for this dep; the pin
				// is advisory until the replace-drop sweep removes it.
				continue
			}

			depDir := strings.TrimPrefix(dep, "github.com/larsartmann/go-cqrs-lite/")

			if latest := latestTagFor(allTags, depDir); latest != "" {
				if version == latest {
					continue
				}

				switch {
				case !tagExists(allTags, depDir, version) && isPseudoVersion(version):
					broken = append(
						broken,
						fmt.Sprintf(
							"%s requires %s %s — pseudo-version pin; replace with a real tag",
							mod,
							depDir,
							version,
						),
					)
				case !tagExists(allTags, depDir, version):
					broken = append(broken, fmt.Sprintf("%s requires %s %s — no such tag exists",
						mod, depDir, version))
				case compareVersions(version, latest) < 0:
					msg := fmt.Sprintf(
						"%s requires %s %s (latest tag: %s)",
						mod,
						depDir,
						version,
						latest,
					)
					if enforceStaleness {
						broken = append(broken, msg+" — stale pin")
					} else {
						stale = append(stale, msg)
					}
				}
			}
		}
	}

	sort.Strings(broken)
	sort.Strings(stale)

	for _, s := range stale {
		t.Logf("STALE PIN (sweep pending, see ROADMAP Open Questions): %s", s)
	}

	if len(stale) > 0 {
		t.Logf("%d stale pin(s) — informational until the pin-sweep policy is decided", len(stale))
	}

	for _, b := range broken {
		t.Errorf("BROKEN PIN: %s", b)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	return root
}

func gitAvailable(t *testing.T, root string) bool {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		return false
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = root

	return cmd.Run() == nil
}

func listTags(t *testing.T, root string) []string {
	t.Helper()

	out, err := exec.Command("git", "tag", "-l").Output()
	if err != nil {
		t.Fatalf("git tag -l: %v", err)
	}

	tags := strings.Fields(string(out))
	if len(tags) == 0 {
		t.Log("no tags in checkout — nothing to compare")
	}

	return tags
}

// parseGoMod extracts sibling-module requires (path → version) and the set of
// replaced module paths from a go.mod file. Line-based on purpose: the pin
// graph needs only require/replace structure.
func parseGoMod(t *testing.T, path string) (map[string]string, map[string]struct{}) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	requires := make(map[string]string)
	replaces := make(map[string]struct{})

	inRequireBlock, inReplaceBlock := false, false

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)

		switch {
		case line == "" || strings.HasPrefix(line, "//"):
			continue
		case strings.HasPrefix(line, "require ("):
			inRequireBlock = true

			continue
		case strings.HasPrefix(line, "replace ("):
			inReplaceBlock = true

			continue
		case line == ")":
			inRequireBlock, inReplaceBlock = false, false

			continue
		case strings.HasPrefix(line, "require "):
			collectRequire(requires, strings.TrimPrefix(line, "require "))
		case strings.HasPrefix(line, "replace "):
			collectReplace(replaces, strings.TrimPrefix(line, "replace "))
		case inRequireBlock:
			collectRequire(requires, line)
		case inReplaceBlock:
			collectReplace(replaces, line)
		}
	}

	return requires, replaces
}

func collectRequire(into map[string]string, line string) {
	m := siblingRequireRe.FindStringSubmatch(line)
	if m == nil {
		return
	}

	into[m[1]] = m[2]
}

func collectReplace(into map[string]struct{}, line string) {
	fields := strings.Fields(line)
	if len(fields) >= 3 && strings.HasPrefix(fields[0], "github.com/larsartmann/go-cqrs-lite/") {
		into[fields[0]] = struct{}{}
	}
}

// latestTagFor returns the newest version tag belonging to module dir D.
// A tag belongs to D iff it is exactly "D/<version>" — nested-module tags
// (e.g. event/v4/eventtest/v4.2.0 when D == "event") are excluded.
func latestTagFor(tags []string, dir string) string {
	best := ""

	for _, tag := range tags {
		if !strings.HasPrefix(tag, dir+"/") {
			continue
		}

		version := strings.TrimPrefix(tag, dir+"/")
		if !versionSuffixRe.MatchString(version) || isPseudoVersion(version) {
			continue
		}

		if best == "" || compareVersions(version, best) > 0 {
			best = version
		}
	}

	return best
}

func tagExists(tags []string, dir, version string) bool {
	for _, tag := range tags {
		if tag == dir+"/"+version {
			return true
		}
	}

	return false
}

func isPseudoVersion(version string) bool {
	return strings.Contains(version, "-0.20") || len(strings.SplitN(version, "-", 2)) > 1
}

// compareVersions orders two vMAJOR.MINOR.PATCH strings (prereleases are
// ignored beyond a simple "shorter wins" rule — the repo uses plain tags).
func compareVersions(a, b string) int {
	as := strings.Split(strings.TrimPrefix(a, "v"), ".")
	bs := strings.Split(strings.TrimPrefix(b, "v"), ".")

	for i := range 3 {
		ai, bi := part(as, i), part(bs, i)
		if ai != bi {
			if ai < bi {
				return -1
			}

			return 1
		}
	}

	return 0
}

func part(v []string, i int) int {
	if i >= len(v) {
		return 0
	}

	n, err := strconv.Atoi(strings.SplitN(v[i], "-", 2)[0])
	if err != nil {
		return 0
	}

	return n
}
