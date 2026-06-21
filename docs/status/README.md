# Status Reports — Historical Snapshots

> **⚠️ These reports are point-in-time snapshots, not living documents.**

Each file captures the project status at a specific timestamp. They are
preserved for audit trail and progress tracking.

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
