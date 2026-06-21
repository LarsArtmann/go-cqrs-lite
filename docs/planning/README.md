# Planning Docs — Historical Snapshots

> **⚠️ These documents are point-in-time snapshots, not living documents.**

Each file in this directory captures the state of thinking at a specific
timestamp. They are preserved for historical context and audit trail.

## What this means

- **APIs referenced may no longer exist.** Modules were renamed, merged, or
  deleted (e.g., `readmodel/` → `kv/`, `projection/` dissolved, `memory/` →
  `storage/memory/`, ghost bus code removed).
- **Status claims may be stale.** Items marked "TODO" or "BROKEN" in older
  snapshots are frequently **already resolved** in later snapshots or in the
  current codebase.
- **Always verify against the current code.** The authoritative sources are:
  - [`TODO_LIST.md`](../../TODO_LIST.md) — current actionable tasks
  - [`ROADMAP.md`](../../ROADMAP.md) — direction and vision
  - [`FEATURES.md`](../../FEATURES.md) — feature inventory by status
  - The code itself (`go test ./... -count=1`)

## How to use these docs

1. Read the **newest** planning doc first — it reflects the most recent thinking.
2. Use older docs only to understand _why_ a decision was made, not _what_ the
   current state is.
3. If a doc claims something is broken, run the tests to verify before acting.
