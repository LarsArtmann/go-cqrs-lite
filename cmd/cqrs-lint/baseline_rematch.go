package main

import "strconv"

// rematchOrphanedReasons re-attaches prior reasons whose exact key vanished
// (the finding moved lines) to reason-less current entries of the same
// (rule, file). Mutates entries in place and reports every reason that could
// not be re-attached unambiguously. Returns the number of reasons dropped.
func rematchOrphanedReasons(
	entries []baselineEntry,
	prior *baselineDocument,
	warn func(format string, args ...any),
) int {
	if prior == nil {
		return 0
	}

	if warn == nil {
		warn = func(string, ...any) {}
	}

	currentKeys := make(map[baselineKey]bool, len(entries))
	for _, entry := range entries {
		currentKeys[entry.key()] = true
	}

	type groupKey struct {
		rule string
		file string
	}

	orphaned := make(map[groupKey][]baselineEntry)

	for _, entry := range prior.Entries {
		if entry.Reason == "" || currentKeys[entry.key()] {
			continue
		}

		group := groupKey{rule: entry.Rule, file: entry.File}
		orphaned[group] = append(orphaned[group], entry)
	}

	dropped := 0

	for group, orphans := range orphaned {
		candidates := make([]*baselineEntry, 0)

		for i := range entries {
			e := &entries[i]
			if e.Rule == group.rule && e.File == group.file && e.Reason == "" {
				candidates = append(candidates, e)
			}
		}

		allSameReason := true

		for _, o := range orphans[1:] {
			if o.Reason != orphans[0].Reason {
				allSameReason = false

				break
			}
		}

		switch {
		case len(candidates) == 0:
			dropped += len(orphans)

			for _, o := range orphans {
				loc := o.Rule + " " + o.File + ":" + strconv.Itoa(o.Line) + ":" + strconv.Itoa(o.Column)
				warn("cqrs-lint: --update-baseline drops reason "+strconv.Quote(o.Reason)+" on "+loc+" — the finding vanished\n")
			}
		case allSameReason:
			reason := orphans[0].Reason

			for _, c := range candidates {
				c.Reason = reason
			}
		case len(orphans) == 1 && len(candidates) == 1:
			candidates[0].Reason = orphans[0].Reason
		default:
			dropped += len(orphans)

			for _, o := range orphans {
				loc := o.Rule + " " + o.File + ":" + strconv.Itoa(o.Line) + ":" + strconv.Itoa(o.Column)
				warn("cqrs-lint: --update-baseline drops reason " + strconv.Quote(o.Reason) + " on " + loc +
					" — "+strconv.Itoa(len(orphans))+
					" moved findings with differing reasons share this file; re-annotate manually\n")
			}
		}
	}

	return dropped
}
