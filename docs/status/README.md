# Status Reports — Historical Snapshots

> **⚠️ These reports are point-in-time snapshots, not living documents.**

Each file captures the project status at a specific timestamp. They are
preserved for audit trail and progress tracking.

**Fully-resolved reports live in [`archived/`](archived/)** (consolidated
2026-08-29 from the older `archive/` + `archived/` split). A report is moved
there once every item it raised is verified resolved, tracked in
[TODO_LIST.md](../../TODO_LIST.md), or superseded — the 2026-08-29 docs-health
audit classified and annotated ~450 August reports and archived ~380 of them,
with 28 stale claims corrected inline. The July archive pass (2026-08-29,
same session) moved all 2026-07 status (240 files) and planning (52 files)
snapshots to `archived/` — July work is shipped or superseded by the August
waves; inbound references from active docs were repointed.

**2026-09-06 passes (two):** the morning pass archived 83 files (25 status
reports 08-27→09-06, 8 plans + 16 artifacts, 5 reviews, 23 feedback) and
rebuilt TODO_LIST (817→452 lines). The evening pass harvested + inline-
annotated all ten 2026-09-06 session reports (02:40→15:09) plus the cqrs-lint
pareto plan (79 table rows struck), archived them (11 files), added the
missing CHANGELOG `[Unreleased]` wave entries, and extended FEATURES/ROADMAP.
New reports land here unarchived; the next docs-health pass harvests their
forward-looking sections into TODO_LIST/ROADMAP, then archives them.

## What this means

- **Claims of "broken" or "failing" may be resolved.** The codebase evolves
  rapidly. Many items flagged as broken in older reports are fixed in later
  reports or in the current codebase.
- **Module references may be outdated.** Several modules were renamed, merged,
  or deleted between reports (e.g., `readmodel/`, `projection/`, `memory/bus.go`).
- **Coverage numbers and export counts are frozen in time** and will not match
  the current state.

## How to use these reports

1. Read the **newest** report first for the most accurate picture.
2. For the current ground truth, run: `go build ./... && go test ./... -count=1`
3. For current tasks, see [`TODO_LIST.md`](../../TODO_LIST.md).
4. For current features, see [`FEATURES.md`](../../FEATURES.md).

## Quick verification commands

```bash
go build ./...                    # Build health
go test ./... -count=1            # Test health
go vet ./...                      # Static analysis
find . -name "*.go" -not -name "*_test.go" -exec wc -l {} + | sort -rn | head  # Largest files
```

## Link hygiene

Relative markdown links across living docs are checked by
[`scripts/check-doc-links.sh`](../../scripts/check-doc-links.sh) (resolves
symlinked docs like `SKILL.md`, skips fenced code and archived history).
Run it after any doc move; CI-truth for doc references remains `cmd/doc-check`.
