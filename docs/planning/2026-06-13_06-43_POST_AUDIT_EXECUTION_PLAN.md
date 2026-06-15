# Comprehensive Execution Plan — Post-Audit Fixes

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

| #   | Task                                            | Effort | Impact                       | Commit? |
| --- | ----------------------------------------------- | ------ | ---------------------------- | ------- |
| 1   | Fix 2 catalog lint issues (goconst, nolintlint) | 10min  | Zero lint across ALL modules | YES     |
| 2   | Fix encryption/static_resolver.go mapsloop hint | 5min   | Zero gopls hints             | YES     |
| 3   | Fix nil context warnings in encryption tests    | 10min  | Zero gopls warnings          | YES     |
| 4   | Fix codec/TestGolden_JSONCodec_Encode golden    | 5min   | 40/40 test packages pass     | YES     |
| 5   | Clean up pkg/ directory                         | 10min  | Clean module inventory       | YES     |

## Tier 2: Medium Effort (P2 — Medium Effort, High Impact)

| #   | Task                                                  | Effort | Impact                        | Commit? |
| --- | ----------------------------------------------------- | ------ | ----------------------------- | ------- |
| 6   | Update README.md with encryption + turso sections     | 30min  | Complete consumer-facing docs | YES     |
| 7   | Extract shared base64 decode helper (check existing!) | 30min  | Eliminate 2 clone groups      | YES     |
| 8   | Parameterize SQL load helpers in storage/             | 1hr    | Eliminate 2 clone groups      | YES     |

## Tier 3: High Value (P2 — Higher Effort, High Impact)

| #   | Task                       | Effort | Impact                                   | Commit? |
| --- | -------------------------- | ------ | ---------------------------------------- | ------- |
| 9   | Add BDD test for catalog/  | 1hr    | BDD coverage for largest untested module | YES     |
| 10  | Add testutil/ to AGENTS.md | 5min   | Accurate docs                            | YES     |

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
