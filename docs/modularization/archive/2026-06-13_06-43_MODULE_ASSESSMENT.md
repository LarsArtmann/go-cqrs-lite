# Go Modularize Assessment — 2026-06-13

## Current State: Workspace Mode (go.work)

The project is already fully modularized with 27 sub-modules in `go.work`. This is NOT a greenfield modularization — the question is whether current boundaries are correct.

## Module Inventory

| #   | Module      | Layer | Int Deps | Ext Deps | Status              |
| --- | ----------- | ----- | :------: | :------: | ------------------- |
| 1   | id/         | 0     |    0     |    2     | ✅ Leaf             |
| 2   | dispatcher/ | 0     |    0     |    1     | ✅ Leaf             |
| 3   | codec/      | 0     |    0     |    1     | ✅ Leaf             |
| 4   | otel/       | 0     |    0     |    6     | ✅ Leaf             |
| 5   | catalog/    | 0     |    0     |    2     | ⚠️ God-package risk |
| 6   | snapshot/   | 1     |    3     |    0     | ✅ Clean            |
| 7   | signing/    | 1     |    2     |    0     | ✅ Clean            |
| 8   | encryption/ | 1     |    3     |    1     | ✅ Clean            |
| 9   | event/      | 1     |    3     |    3     | ✅ Core             |
| 10  | command/    | 2     |    3     |    2     | ✅ Clean            |
| 11  | query/      | 2     |    2     |    2     | ✅ Clean            |
| 12  | memory/     | 2     |    5     |    0     | ✅ Test impl        |
| 13  | listing/    | 2     |    3     |    0     | ✅ Clean            |
| 14  | pebble/     | 2     |    3     |    1     | ✅ Clean            |
| 15  | schema/     | 2     |    3     |    0     | ✅ Clean            |
| 16  | decider/    | 3     |    6     |    0     | ✅ Core             |
| 17  | projection/ | 3     |    5     |    1     | ✅ Clean            |
| 18  | storage/    | 3     |    7     |    2     | ✅ Clean            |
| 19  | middleware/ | 3     |    6     |    4     | ⚠️ Wide concern     |
| 20  | turso/      | 4     |    5     |    2     | ✅ Clean            |
| 21  | watermill/  | 4     |    3     |    1     | ✅ Clean            |
| 22  | testutil/   | —     |    2     |    0     | ✅ Test helper      |

## DAG Verification

```
No circular dependencies detected. ✅
Layer graph is strict DAG. ✅
Replace directives consistent across all modules. ✅
Go version consistent (1.26.3) across all modules. ✅
```

## Assessment

### Modules to Keep (20/22)

All core library modules have clear, single purposes. No action needed.

### Modules to Watch (2/22)

| Module      | Concern                                                                                            | Action                                                  |
| ----------- | -------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| catalog/    | 6 sub-packages could be separate modules                                                           | Monitor; split only if consumers need selective imports |
| middleware/ | 9 concerns in one module (retry, CB, SSE, health, tracing, metrics, logging, validation, recovery) | Monitor; split SSE and healthcheck if they grow         |

### Dead Code

| Path                  | Issue                          | Action                                       |
| --------------------- | ------------------------------ | -------------------------------------------- |
| pkg/config/           | Not in go.work, not referenced | Add to go.work or remove                     |
| pkg/gracefulshutdown/ | Not in go.work, not referenced | Add to go.work or remove                     |
| root go.mod           | Empty shell (no .go files)     | Add doc.go or accept as repo root convention |

## Verdict

**The current modularization is excellent.** No merges or splits needed. The boundaries are well-chosen, the dependency graph is clean, and there are no cycles. Score: 9/10.

The only improvements are:

1. Clean up `pkg/` directory (add to go.work or remove)
2. Monitor `catalog/` and `middleware/` for future split opportunities
3. Fix 2 catalog lint issues
