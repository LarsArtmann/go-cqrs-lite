# Code Quality Scan Report — 2026-06-13

## Executive Summary

| Metric       | Result                                               |
| ------------ | ---------------------------------------------------- |
| Build        | PASS                                                 |
| Lint         | 2 issues in 1 module (catalog)                       |
| Tests        | 1 golden file mismatch (fixed)                       |
| Clone Groups | 8 (threshold ≥80 tokens)                             |
| Modules      | 27 (22 library + 2 cmd + 1 integration + 2 examples) |

## Build Result

All 27 modules build cleanly via `nix run .#build`.

## Lint Issues

| #   | Module  | File              | Line | Linter     | Issue                                                                    |
| --- | ------- | ----------------- | ---- | ---------- | ------------------------------------------------------------------------ |
| 1   | catalog | message_config.go | 192  | goconst    | String `Cmd` has 3 occurrences — extract to constant                     |
| 2   | catalog | message_config.go | 190  | nolintlint | `//nolint:goconst` directive is unused (nolint doesn't suppress goconst) |

**25/27 modules are lint-clean.**

## Test Results

- 26/27 module test suites pass
- 1 golden file update: `middleware/testdata/golden/health-check-response.json` (formatting diff: compact→expanded JSON array) — FIXED

## Code Duplication (dupl ≥80 tokens)

| #   | Files                                                   | Lines | Severity | Description                                                                      |
| --- | ------------------------------------------------------- | ----- | -------- | -------------------------------------------------------------------------------- |
| 1   | encryption/ciphertext.go, signing/signature.go          | 31    | Medium   | Base64 UnmarshalJSON with URL-safe→Std fallback — identical pattern              |
| 2   | encryption/event.go, signing/event.go                   | 25    | Medium   | Extract bytes from event metadata with base64 decode — identical pattern         |
| 3   | storage/event_store_load.go (internal)                  | 13    | Low      | LoadFromVersion vs LoadToVersion differ only in operator (`>` vs `<=`)           |
| 4   | middleware/circuit_breaker.go, middleware/middleware.go | 17    | Low      | Config Validate() pattern — field range checks                                   |
| 5   | command/dispatcher.go, query/dispatcher.go              | 22    | Low      | Dispatcher Dispatch() with handler lookup — similar but different types          |
| 6   | catalog/validate.go (internal)                          | 24    | Low      | Two validate functions for different message types                               |
| 7   | encryption/aesgcm.go, encryption/xchacha20.go           | 45    | Medium   | AES-GCM vs XChaCha20 encrypt/decrypt — algorithm differs but structure identical |
| 8   | storage/command_store_load.go (internal)                | 15    | Low      | LoadFromVersion vs LoadAll differ only in WHERE clause                           |

### Recommendations

1. **(Medium) Extract shared base64 decoder** — encryption + signing both use `[]byte` types with identical JSON unmarshal logic. Create a shared `internal/encoding` helper or a generic `DecodeBase64Raw(data []byte) ([]byte, error)`.
2. **(Medium) Extract metadata extractor** — `ExtractCiphertext` and `ExtractSignature` are structurally identical. A generic `ExtractBytesFromMetadata(evt, key) ([]byte, error)` would eliminate both.
3. **(Low) Parameterize SQL load helpers** — storage `LoadFromVersion`/`LoadToVersion` and `command_store_load` pairs differ only in operators; could use a parameterized query builder.
4. **(Low) Config validation DSL** — middleware has repeated field-range-check patterns; a small validation helper could reduce boilerplate.
5. **(Low) Dispatcher generic pattern** — command/query dispatchers are similar by design (same pattern, different types); duplication is acceptable.
