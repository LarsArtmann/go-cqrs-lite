# Comprehensive Execution Plan — Post-Audit Fixes

> **Status: ✅ COMPLETED** · **Date completed:** 2026-06-14
>
> 2026-06-13 · Sorted by effort vs impact · Each step is a self-contained commit

## Priority Matrix

```
         HIGH IMPACT
              │
   P1─────────┼─────────P2
   (Quick     │  (Medium
    wins)     │   effort)
              │
LOW EFFORT────┼────HIGH EFFORT
              │
   P3─────────┼─────────P4
   (Low       │  (High
    value)    │   effort)
              │
         LOW IMPACT
```

## Tier 1: Quick Wins (P1 — Low Effort, High Impact)

| #   | Task                                            | Status  | Commit     | Notes                                                |
| --- | ----------------------------------------------- | ------- | ---------- | ---------------------------------------------------- |
| 1   | Fix 2 catalog lint issues (goconst, nolintlint) | ✅ DONE | `b7df8d75` | message_config.go goconst fix                        |
| 2   | Fix encryption/static_resolver.go mapsloop hint | ✅ DONE | `b7df8d75` | Uses `maps.Copy` now                                 |
| 3   | Fix nil context warnings in encryption tests    | ✅ DONE | `b7df8d75` | nil → context.Background() (4 sites)                 |
| 4   | Fix codec/TestGolden_JSONCodec_Encode golden    | ✅ DONE | `3d5ec978` | Golden file regenerated                              |
| 5   | Clean up pkg/ directory                         | ✅ DONE | `b7df8d75` | Orphaned pkg/config/ + pkg/gracefulshutdown/ removed |

## Tier 2: Medium Effort (P2 — Medium Effort, High Impact)

| #   | Task                                                  | Status  | Commit     | Notes                                                            |
| --- | ----------------------------------------------------- | ------- | ---------- | ---------------------------------------------------------------- |
| 6   | Update README.md with encryption + turso sections     | ✅ DONE | `654be757` | README updated with encryption/turso/example-encryption sections |
| 7   | Extract shared base64 decode helper (check existing!) | ✅ DONE | `b7df8d75` | event.DecodeBase64String + event.ExtractCustomBytes (`42e17f4f`) |
| 8   | Parameterize SQL load helpers in storage/             | ✅ DONE | `4002fa87` | storage/sql/helpers.go: SharedInsertEvents, SharedCheckpointLoad |

## Tier 3: High Value (P2 — Higher Effort, High Impact)

| #   | Task                       | Status  | Commit     | Notes                                                   |
| --- | -------------------------- | ------- | ---------- | ------------------------------------------------------- |
| 9   | Add BDD test for catalog/  | ✅ DONE | `b7df8d75` | catalog/catalog_bdd_suite_test.go + catalog_bdd_test.go |
| 10  | Add testutil/ to AGENTS.md | ✅ DONE | `654be757` | testutil documented in AGENTS.md module tree            |

## D2 Execution Graph

```d2
direction: right
fix_lint -> fix_mapsloop -> fix_nil_ctx -> fix_codec_golden -> cleanup_pkg
cleanup_pkg -> update_readme -> extract_base64 -> parameterize_sql
parameterize_sql -> add_catalog_bdd -> update_agents
```

## Research Notes (Before Implementing)

### Existing code to check before creating new abstractions:

- `event/codec.go` — does it have base64 helpers?
- `event/metadata_json.go` — does it handle encoding/decoding?
- `signing/payload.go` — does it have encoding logic?
- `storage/sql/helpers.go` — does it have query builders?

### Go libraries to consider:

- `maps.Copy` (stdlib Go 1.21+) for mapsloop fix
- Existing `samber/ro` for reactive bridge patterns
- Existing `go-error-family` for error classification
