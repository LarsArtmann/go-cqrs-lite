package main

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// commitPrefix marks the start of a commit record line emitted by git.
const commitPrefix = "@@@"

// FileStat aggregates churn and complexity for a single file path.
type FileStat struct {
	Path       string
	Commits    int
	Added      int
	Deleted    int
	Weight     float64 // recency-weighted churn (equals Churn when recency is disabled)
	Authors    map[string]struct{}
	Complexity int
	FirstTouch time.Time
	LastTouch  time.Time
}

// Churn returns raw lines added plus lines deleted.
func (f FileStat) Churn() int { return f.Added + f.Deleted }

// Hotspot returns the churn-times-complexity risk score.
// Weight already encodes recency decay when enabled, so this naturally
// produces a recency-weighted hotspot when a half-life is configured.
func (f FileStat) Hotspot() float64 { return f.Weight * float64(f.Complexity) }

// AuthorCount returns the number of distinct authors.
func (f FileStat) AuthorCount() int { return len(f.Authors) }

// history bundles the raw per-file stats with window metadata.
type history struct {
	Files        map[string]*FileStat
	TotalCommits int
	FirstCommit  time.Time
	LastCommit   time.Time
}

// analyzeOptions controls the git history window and recency decay.
type analyzeOptions struct {
	since       string
	until       string
	branch      string
	halfLifeDay float64 // 0 disables recency weighting
}

// collectStats runs git log over the configured window and aggregates per-file stats.
func collectStats(opts analyzeOptions, now time.Time) (*history, error) {
	args := []string{"log", "--numstat", "--no-merges", "-M",
		"--pretty=tformat:" + commitPrefix + "%H|%aI|%an"}
	if opts.since != "" {
		args = append(args, "--since="+opts.since)
	}
	if opts.until != "" {
		args = append(args, "--until="+opts.until)
	}
	if opts.branch != "" {
		args = append(args, opts.branch)
	}

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("git log: %w (%s)", err, stderr.String())
	}

	h := &history{Files: make(map[string]*FileStat)}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 0, 1<<16), 1<<20)

	var curAuthor string
	var curDate time.Time
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, commitPrefix):
			h.TotalCommits++
			curAuthor, curDate = parseCommitMarker(line)
			h.extendWindow(curDate)
		case line == "":
			continue
		default:
			applyNumStat(line, h.Files, curAuthor, curDate, opts.halfLifeDay, now)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("git log: %w (%s)", err, stderr.String())
	}
	return h, nil
}

// extendWindow widens the first/last commit timestamps.
func (h *history) extendWindow(t time.Time) {
	if t.IsZero() {
		return
	}
	if h.FirstCommit.IsZero() || t.Before(h.FirstCommit) {
		h.FirstCommit = t
	}
	if t.After(h.LastCommit) {
		h.LastCommit = t
	}
}

// parseCommitMarker extracts author and date from a "@@@hash|date|author" line.
func parseCommitMarker(line string) (author string, date time.Time) {
	body := strings.TrimPrefix(line, commitPrefix)
	parts := strings.SplitN(body, "|", 3)
	if len(parts) < 3 {
		return "", time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, parts[1])
	return parts[2], t
}

// applyNumStat parses an "added\tdeleted\tpath" line and folds it into stats.
func applyNumStat(line string, stats map[string]*FileStat, author string, date time.Time, halfLife float64, now time.Time) {
	add, del, file, ok := splitNumStat(line)
	if !ok {
		return
	}
	file = normalizeRename(file)
	s := stats[file]
	if s == nil {
		s = &FileStat{Path: file, Authors: make(map[string]struct{})}
		stats[file] = s
	}
	s.Commits++
	s.Added += add
	s.Deleted += del
	s.Weight += weighted(float64(add+del), date, halfLife, now)
	if author != "" {
		s.Authors[author] = struct{}{}
	}
	if (s.FirstTouch.IsZero() || date.Before(s.FirstTouch)) && !date.IsZero() {
		s.FirstTouch = date
	}
	if date.After(s.LastTouch) {
		s.LastTouch = date
	}
}

// weighted applies exponential recency decay to a churn delta.
func weighted(raw float64, date time.Time, halfLifeDays float64, now time.Time) float64 {
	if halfLifeDays <= 0 {
		return raw
	}
	ageDays := now.Sub(date).Hours() / 24
	if ageDays < 0 {
		ageDays = 0
	}
	return raw * math.Pow(0.5, ageDays/halfLifeDays)
}

// splitNumStat parses an "added\tdeleted\tpath" line.
func splitNumStat(line string) (add, del int, file string, ok bool) {
	tabs := strings.Split(line, "\t")
	if len(tabs) != 3 {
		return 0, 0, "", false
	}
	a, errA := strconv.Atoi(tabs[0])
	d, errD := strconv.Atoi(tabs[1])
	if errA != nil || errD != nil {
		return 0, 0, "", false
	}
	return a, d, tabs[2], true
}

// normalizeRename resolves git's rename notation to the current file path.
func normalizeRename(path string) string {
	if !strings.Contains(path, "=>") {
		return path
	}
	// Brace form: prefix/{old => new}suffix
	if o := strings.Index(path, "{"); o >= 0 {
		rest := path[o:]
		if c := strings.Index(rest, "}"); c > 0 {
			inner := rest[1:c]
			if a := strings.Index(inner, "=>"); a >= 0 {
				newPart := strings.TrimSpace(inner[a+2:])
				return path[:o] + newPart + path[o+c+1:]
			}
		}
	}
	// Simple form: old=>new
	if i := strings.Index(path, "=>"); i >= 0 {
		return strings.TrimSpace(path[i+2:])
	}
	return path
}
