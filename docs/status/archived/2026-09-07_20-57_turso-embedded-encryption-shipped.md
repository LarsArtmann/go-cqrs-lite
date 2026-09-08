# Status Report — Turso Blog Research → Embedded Encryption Shipped

> **RESOLVED + ARCHIVED (docs-health pass 2026-09-08).** The feature shipped
> and all its own gates were green. Resolved follow-ups struck below:
> f19 (modules.md tursoengine row — added 2026-09-08), b2/f20 (libSQL→Turso
> terminology sweep — done 2026-09-08), f1's doc half (WithEncryption
> reachability note in README — done 2026-09-08), f36 (HARVEST — done
> 2026-09-08 docs pass → TODO_LIST), f37 (annotate the 20:18 predecessor —
> done 2026-09-08 docs pass). The open remainder (f2 strict-vs-lenient DSN
> policy, f3 `file:` DSN live test, f4 second-ADT round-trip, upstream
> issues, DriverConfig.Encryption ADR, sync decision, redaction audit) is
> in TODO_LIST → "Metaengine — follow-ups" / v5 Unification.

**Date:** 2026-09-07 20:57 CEST
**Session scope:** Read the 5 Turso blog posts the user supplied (0.7.0 release, 3-part encryption series, Turso Sync launch), mapped them to `metaengine/tursoengine`, and executed the resulting work: fixed a latent DSN-flag bug, shipped first-class embedded encryption support, hardened redaction pins, updated all docs, and ran every applicable gate. This report covers ONLY this session. (Predecessor: `docs/status/2026-09-07_20-18_turso-byok-encryption-assessment.md`.)

---

## Session Summary

**"What is libSQL?" answered:** Turso replaced libSQL (the C fork) with Turso Database, a Rust SQLite rewrite (beta dropped at v0.7). The `tursogo` driver we pin opens EMBEDDED Turso Database engines only — no remote Hrana client exists in `database/sql`, which reshapes the whole Cloud-BYOK story: pure-remote Go connections to an encrypted cloud database are unreachable today; the sync engine's struct config is the only path.

**Shipped:** `tursoengine.WithEncryption(cipher, hexKey)` + typed `Cipher` constants — opt-in experimental page-level AEAD encryption for embedded Turso databases, verified live against the pinned driver (round-trip + wrong-key explicit rejection on 3 ciphers). Plus the matview `experimental=` silent-skip bug fix, redaction pin tests, and full doc coverage.

---

## a) FULLY DONE

1. **READ + UNDERSTAND: all 5 blog posts digested with driver-source cross-checks.**
   - Evidence: local engine ciphers (aegis256/aegis128l ± x2/x4, aes128gcm/aes256gcm; NO chacha20poly1305 locally), hex-vs-base64 key-format split, 0.5–2.8% mixed overhead, no native rekeying anywhere, sync = push-logical/pull-physical with Last-Push-Wins + `transform` hook, sync engine present in our pinned driver (`bindings_sync.go`), pluggable I/O backends at 0.7.
   - Scope: research only.

2. **Live capability probe BEFORE designing.**
   - Evidence: temp probe test proved driver v0.7.2 does full encrypted round-trips — open with `experimental=encryption`, `MapSet`, close, reopen with key, wrong-key reopen fails with `turso: error: Decryption failed for page=1`. Probe file deleted after promoting its logic into the permanent test.
   - Scope: verified the design assumption before committing to API shape.

3. **EXECUTE: matview DSN-flag silent-skip bug fixed (previous report item f5).**
   - Evidence: new `dsn.go` — `withExperimentalToken` merges experimental flags into comma-lists (`experimental=encryption` → `experimental=views%2Cencryption`), other params stay byte-identical, unparseable queries pass through, `notexperimental=1` doesn't false-positive. 9-case table in `dsn_internal_test.go` + live `TestTursoEncryption_WithMaterializedViews` coexistence test.
   - Committed: daemon waves ending `7a674904b`/`7f112a4ea` region.

4. **EXECUTE: typed embedded encryption shipped.**
   - Evidence: `encryption.go` — `Cipher` type + 8 constants, `cipherKeyLen` (16/32B by cipher), `WithEncryption` option, `applyEncryption` validation (unknown cipher lists valid ones; non-hex and wrong-length key errors quote `openssl rand -hex N` and NEVER the key material), `withEncryptionParams` (merges `experimental=encryption`, appends `encryption_cipher`/`encryption_hexkey`; rejects DSNs already carrying encryption params — one key source only), remote-DSN rejection with actionable guidance. Wired in `register.go` `New()`.
   - Tests: internal validation table + no-key-leak assertions; external live round-trips for `aes256gcm`/`aegis256`/`aegis128l` (write → close → reopen correct key → wrong-key rejection).

5. **EXECUTE: redaction pin tests (previous report items f12/f13).**
   - Evidence: `:memory:?encryption_hexkey=…` → `:memory:` (parse-failure fallback drops the query); `/data/%zz.db?encryption_hexkey=…` → path-only; both shapes added to the never-leak test list.

6. **EXECUTE: docs + memory updated.**
   - Evidence: `metaengine/tursoengine/README.md` encryption section (ciphers, semantics, redaction guarantee, Cloud-BYOK boundary, no-rekeying note); `faq.md` "Does the Turso engine support encryption at rest?" entry; `CHANGELOG.md` three entries (Added WithEncryption + matview merge fix; Fixed redaction leak); `AGENTS.md` Turso terminology + embedded-only-driver gotcha.
   - Gates: `doc-check` ✓ (1012 refs, 45 packages, zero warnings); `check-changelog-symbols.sh` ✓ (170 citations honest); api golden regenerated (`docs/api_surface.txt`, 6723 exports) + `TestEvery*` meta-tests ✓.

7. **VERIFY: full gate run on my diff.**
   - Evidence: full module suite ×2 green (3.7s / 2.9s); `-race` full suite green (17.4s); `go vet` clean; `gofmt -l` clean on all 8 touched files; golangci — my files contribute ZERO findings (only the pre-existing foreign `maintidx` on `matview_bench_test.go` remains).
   - Scope: `metaengine/tursoengine` (7 files) + root docs.

---

## b) PARTIALLY DONE

1. **Encryption reachability through the composition layers.**
   - Works: direct `tursoengine.New(..., WithEncryption(...))`.
   - Open: the `RegisterDriver` factory path (`init()` in `register.go`) forwards only `MaterializedViews` — `DriverConfig` has no encryption slot, so consumers composing via `metaengine` driver dispatch or `system` EngineConfig YAML CANNOT reach `WithEncryption`. **I implemented this scoping deliberately but forgot to DOCUMENT it** in README/FAQ — a consumer composing through `system` will look for the option, not find it, and get no pointer. Effort to fix: S (docs) + M (v5 `DriverConfig.Encryption`).
2. **Turso terminology cleanup.**
   - Works: ~~AGENTS.md records the Turso-Database-vs-libSQL split; new code/docs use precise terms.~~
   - ~~Open: README title still says "Turso/libSQL-Backed Engine" and several body lines still say "libSQL" loosely; `register.go` package comment says "libSQL" in places. Effort: S.~~ done 2026-09-08 (docs-health pass: README title + 5 body mentions swept).
3. **Encryption test coverage breadth.**
   - Works: MapBackend round-trips + reopen semantics on 3 ciphers, race-clean.
   - Open: only the Map ADT is exercised live (journal/counter/graph paths all ride the same encrypted page layer, but are unproven live); `file:`-prefixed DSNs (the blog's own syntax) never verified live — my probe tested plain paths only. Effort: S–M.
4. **Release readiness of the encryption work.**
   - Works: everything committed, gates green, golden current.
   - Open: no tag exists for tursoengine carrying `WithEncryption`; consumer pin sweep not run (repo rule: codec-adjacent API in a tagged module requires same-wave pin bumps). Blocked on release-timing decision (g)1. Effort: S within a wave.

---

## c) NOT STARTED

All session-derived, none begun:

1. Upstream issues to tursodatabase/turso-go: missing `DriverContext`/`OpenConnector` (struct-level config without DSN stringification), missing pure-remote BYOK key support, and the silent-ignore of mistyped DSN params (a typo'd `encryption_hexkkey=` opens the DB UNENCRYPTED with no warning — see (e)2). Each needs verify-before-filing depth.
2. `recipes.md` §turso encryption recipe (FAQ+README done; recipes skipped).
3. `modules.md` tursoengine row (no row exists today).
4. Example snippet in `example/` (examples are the copy-paste surface per repo policy).
5. Sibling-driver secret-redaction audit (pg/mysql DSNs with embedded passwords in error paths) — same class as the turso leak, unstarted.
6. Sync/embedded-replica first-class support decision + design (only Go path to Cloud BYOK).
7. v5 ADR: `DriverConfig.Encryption` + `KeyProvider` (waiting on (g)2).
8. README/libSQL terminology sweep (b2).
9. Defensive hardening against near-miss encryption param typos (see (e)2 and (g)3).

---

## d) TOTALLY FUCKED UP

Nothing shipped is broken — every gate is green and all claims are test-backed. Radical-honesty near-misses:

1. **First probe test was garbage and I shipped it to disk anyway.**
   - What happened: the initial `encryption_probe_test.go` draft contained a leftover meaningless interface stub (`MapSet(b interface{ Done() bool }...)`) — unidiomatic scaffolding I caught only on immediate reread, rewrote, then deleted outright.
   - Severity: none (never ran, never committed). Cost: one wasted write cycle.
2. **golines whack-a-mole: 3 lint cycles to fix what one scoped format pass would have caught.**
   - Each long-line fix revealed the next one (`encryption.go` ×2, test ×1, then one more). I ran lint-first and formatted piecemeal instead of running gofumpt/golines over my new files ONCE before the first lint. Cost: ~3 extra tool cycles.
3. **Wrong lint scope amplified foreign noise.**
   - First lint ran `./...`, surfacing the parallel session's untracked `ivmrepro/` findings (errcheck/forbidigo/noctx…) as if they were mine to read. Scoping to `.` (the package) was the correct first move given known concurrent-session work.
4. **CHANGELOG wording bug caught by self-review pre-commit**: "`redactDSN` dropped no encryption keys" (double-negative confusion) → fixed to "redacted no encryption keys" before the daemon wave landed.

---

## e) WHAT WE SHOULD IMPROVE

1. **Probe-then-design worked; make it the rule.** The live capability probe BEFORE writing the API killed all design guesswork (no speculative abstractions for a driver feature that might not work). Formalize: any feature built atop an external dependency's experimental surface starts with a throwaway live probe.
2. **The silent-unencrypted edge needs a decision.** The driver silently IGNORES unknown/mistyped DSN params: `encryption_hexkkey=` (typo) or case variants open the database UNENCRYPTED with zero warning. My conflict check only catches exactly-spelled params. Options: tursoengine rejects any DSN param containing "encrypt"/"key" that the driver wouldn't recognize (strict), or document-only (lenient). This is an API-philosophy call (g)3, but the audit itself is cheap and worth doing regardless.
3. **Scoped format pass before lint, not after.** Run gofumpt + goimports (-local) + golines on NEW files immediately after writing them; lint findings should never be the formatter's discovery mechanism (cost this session: 3 cycles).
4. **Document scoping decisions WHERE the feature is documented.** The `WithEncryption`-is-not-reachable-via-DriverConfig decision lived only in my head; a consumer reading the README has no way to know. Rule: every "not wired through X" decision gets one sentence in the feature's README section.
5. **Foreign-noise isolation during gates.** With a known-active parallel session (ivmrepro at 20:43, `record_context_test.go` dirty, AGENTS.md IVM entry evolving mid-session), lint/test scope should default to the packages I own, widening only for the final cross-check.
6. **Pre-edit foreign-file re-check.** The parallel session created files DURING my session; I checked ownership at session start but not immediately before each edit batch. No collision occurred — luck, not process.
7. **Version-pinned dependency claims → extract-and-read was right.** Reading the exact pinned driver version (v0.7.2 zip extraction) rather than the stale cached v0.6.1 prevented wrong claims about sync/encryption APIs. Keep as standing practice.

---

## f) Top 50 Things To Get Done Next

> Session-scoped brainstorm (HARVEST fuel — most beyond the top ~10 are ROADMAP material). Impact/Effort/Category per item.

| #  | Task                                                                                                                                                                                                    | Impact | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Document the DriverConfig/`system` reachability gap for `WithEncryption` in README + FAQ (one paragraph: direct `New()` only today)                                                                     | High   | S      | Documentation |
| 2  | Add defensive tursoengine check: reject DSN params that LOOK like encryption params (contain "encrypt", case-insensitive) but aren't exactly the driver's two — closes the silent-unencrypted typo edge | High   | S      | Quality       |
| 3  | Verify `file:`-prefixed DSNs live with `WithEncryption` (the blog's own syntax; parseDSN cuts at `?` but the Rust core must accept the scheme)                                                          | High   | S      | Quality       |
| 4  | Live round-trip test through a second ADT (Counter or journal append) under encryption — Map-only coverage today                                                                                        | Medium | S      | Quality       |
| 5  | Live test: matview aggregation actually SERVES from views on an encrypted engine (coexistence test constructs but doesn't query a view)                                                                 | Medium | S      | Quality       |
| 6  | Add `TestApplyEncryption_CaseVariantParams` pin: `Encryption_HEXKEY=` is not the driver's param — assert current behavior and document it                                                               | Medium | S      | Quality       |
| 7  | Key-zeroization note in README (Go strings are immutable; keys persist in memory until GC — matches Turso's own threat model notes)                                                                     | Low    | S      | Documentation |
| 8  | Decide + implement strict-vs-lenient DSN param policy for tursoengine (needs (g)3)                                                                                                                      | Medium | M      | Feature       |
| 9  | Draft `metaengine.DriverConfig.Encryption` + `KeyProvider func(ctx) ([]byte, error)` (v5 ADR groundwork)                                                                                                | Medium | M      | Feature       |
| 10 | Write the v5 ADR: encryption-at-rest configuration, operator-owned keys, engines fail loudly when unable to honor                                                                                       | Medium | L      | Feature       |
| 11 | Decide sync/embedded-replica first-class support (only Go path to Cloud BYOK; needs (g)2 demand answer)                                                                                                 | High   | M      | Feature       |
| 12 | Prototype sync support: `TursoSyncDatabaseConfig` → tursoengine option surface (`WithSync(remote, authToken, …)`), incl. `ReservedBytes` for encrypted remotes                                          | Medium | L      | Feature       |
| 13 | File upstream: `DriverContext`/`OpenConnector` for struct-level TursoDatabaseConfig (kills DSN stringification entirely)                                                                                | Medium | S      | Cleanup       |
| 14 | File upstream: pure-remote connections cannot present a BYOK encryption key                                                                                                                             | Medium | S      | Cleanup       |
| 15 | File upstream: mistyped DSN encryption params are silently ignored → silently-unencrypted DBs                                                                                                           | High   | S      | Cleanup       |
| 16 | Verify-before-filing pass on #13–15 against latest turso-go main (not just v0.7.2)                                                                                                                      | Medium | M      | Quality       |
| 17 | Tag tursoengine with `WithEncryption` + redaction fix; consumer pin sweep in the same wave (repo rule)                                                                                                  | High   | S      | Cleanup       |
| 18 | `recipes.md` §turso encryption recipe (cipher table, hex-vs-base64, key-from-env, Cloud-BYOK boundary)                                                                                                  | Medium | M      | Documentation |
| ~~19~~ | ~~`modules.md` row for `metaengine/tursoengine` (module table lacks the engine's own row)~~ done 2026-09-08 (docs-health pass)                                                                                                  | Low    | S      | Documentation |
| ~~20~~ | ~~libSQL→Turso-Database terminology sweep in tursoengine README title/body + `register.go` comments~~ done 2026-09-08 for README (register.go comments remain — folded into the propagation wave)                                                  | Low    | S      | Cleanup       |
| 21 | Example snippet in `example/` (encrypted embedded engine, key from env) — examples are the consumer copy-paste surface                                                                                  | Medium | S      | Documentation |
| 22 | Audit pg/mysql/bbolt/pebble engines for the same error-path DSN-secret leak class (passwords in `postgres://user:pass@…`)                                                                               | High   | M      | Quality       |
| 23 | Extract a shared redaction helper if #22 finds 3+ drivers reimplementing it (per-module budget rules apply; possibly `storage/sql` or a Tier-0 home)                                                    | Medium | M      | Cleanup       |
| 24 | Encryption + `storage/turso` connector interplay check (modules.md says the connector delegates to storage; does IT have an encryption story?)                                                          | Medium | M      | Quality       |
| 25 | Add encryption matrix entry to `enginetest`/`adttest` harness docs (harness DSN plumbing must carry query params for engines that need them)                                                            | Low    | M      | Documentation |
| 26 | Watch turso-go releases: `encryption` leaving the experimental list → bump pin, re-audit, re-tag                                                                                                        | Low    | S      | Quality       |
| 27 | Track upstream "What's Coming Next" (native rekeying, in-place encrypt-existing, KDF passphrases, ATTACH with per-DB keys) → each changes our API once landed                                           | Low    | S      | Documentation |
| 28 | Cipher-size validation UX: on wrong-length key, error could suggest the matching `openssl rand -hex N` per CIPHER (already does) — pin with a test asserting the exact hint                             | Low    | S      | Quality       |
| 29 | Property test: `redactDSN` output never contains any value of a param whose name contains "key", for generated DSN shapes (remote/local/memory/malformed)                                               | Low    | M      | Quality       |
| 30 | Confirm tursoengine redaction+encryption tests are exercised in CI's per-module matrix (`testModules` coupling meta-test should enforce — verify once)                                                  | Low    | S      | Quality       |
| 31 | Capture the two-model encryption decision table (local-experimental vs Cloud-BYOK-production) into `docs/DOMAIN_LANGUAGE.md` if encryption becomes a domain concept                                     | Low    | S      | Documentation |
| 32 | Explore `turso-go`'s exported `NewConnection(TursoConnection, extraIo)` as a TODAY escape hatch for struct-level config (would de-scope #13)                                                            | Medium | M      | Feature       |
| 33 | Interplay doc: `encryption/` module (payload AEAD) + at-rest encryption = defense in depth; neither substitutes the other                                                                               | Medium | S      | Documentation |
| 34 | Key-management doc: rotation via export/reimport (downtime!), per-tenant keys via per-database provisioning, secret-manager pattern — Turso Part 3 mapped to our stack                                  | Medium | M      | Documentation |
| 35 | Consider `TURSO_ENCRYPTION_KEY`-style env-var fallback in tursoengine (opt-in, explicit; never a silent default)                                                                                        | Low    | S      | Feature       |
| ~~36~~ | ~~HARVEST this report: items 1–3, 9–11, 17–22 into TODO_LIST/ROADMAP per docs-health~~ done 2026-09-08 docs pass                                                                                                                                                          | High   | S      | Documentation |
| ~~37~~ | ~~Annotate the 20:18 predecessor report: matview bug (f5) FIXED this session, redaction pins (f12/13) DONE (docs-health ANNOTATE mode)~~ done 2026-09-08 docs pass                                                                    | Medium | S      | Documentation |
| 38 | `check-duplication` run scoped to tursoengine (param-walk idiom appears twice — under threshold 3, but verify the tool agrees)                                                                          | Low    | S      | Quality       |
| 39 | Bench: encryption overhead on OUR workload (Turso claims 0.5–2.8% mixed; one `benchkit`/`matview_bench` cell with `WithEncryption` on/off would make it ours)                                           | Low    | M      | Quality       |
| 40 | Verify `WithEncryption` + `WithMaterializedViews` + `Priority` (DriverConfig path) don't fight over the DSN query namespace as more options land (namespace discipline doc)                             | Low    | S      | Documentation |
| 41 | Error-message audit: every `fmt.Errorf` in tursoengine that could carry a DSN goes through `redactDSN` (grep-audit; `withEncryptionParams` errors don't include DSNs — confirm none others do)          | Medium | S      | Quality       |
| 42 | Consider exposing `redactDSN` (or a `RedactDSN`) publicly for consumers logging DSNs in THEIR code — API-surface decision                                                                               | Low    | S      | Feature       |
| 43 | Session-process: scoped lint/test from the first run when concurrent sessions are active (this session paid 1 wasted cycle)                                                                             | Low    | S      | Quality       |
| 44 | Session-process: scoped formatter pass on new files before first lint (paid 3 cycles this session)                                                                                                      | Low    | S      | Quality       |
| 45 | Session-process: re-check `git status` for foreign files immediately before each edit batch, not only at session start                                                                                  | Low    | S      | Quality       |
| 46 | Track the parallel session's IVM COMMIT-wall finding (~27k rows, AGENTS.md line 233) — if upstream-fixed, re-evaluate `seedInTxE` chunk guards + my matview tests' assumptions                          | Medium | S      | Quality       |
| 47 | Decide whether `Cipher` should validate at OPTION-CONSTRUCTION time (fail fast in `WithEncryption`) vs at `New()` (current) — current allows option reuse; document the choice                          | Low    | S      | Quality       |
| 48 | Write the missing `modules.md`/README cross-link: tursoengine README ↔ faq entry ↔ CHANGELOG (single entry point per feature)                                                                           | Low    | S      | Documentation |
| 49 | Upstream-watch: Turso 0.8 — encrypted MVCC + pluggable storage (0.7 groundwork) may add local rekeying; re-assess rotation guidance then                                                                | Low    | S      | Documentation |
| 50 | Re-verify this report's live-test claims after the next driver bump (tests are the pins, but the README prose cites upstream behavior that can drift)                                                   | Low    | S      | Documentation |

---

## g) Questions I Cannot Answer Myself

**Q1 (release timing):** Should tursoengine be tagged NOW to ship `WithEncryption` + the redaction fix (solo patch/minor tag + consumer pin sweep per repo rules), or do these ride the next scheduled tag wave? I know the mechanical cost of solo tags (clean-tree check, pin sweeps, version-sequence verification) but not the consumer demand side — whether anyone is waiting on encrypted embedded Turso.

**Q2 (composition surface):** Is exposing encryption through the `DriverConfig`/`system` YAML layer (operator-composed deployments) a real near-term need, or is the direct `tursoengine.New(...)` option sufficient for v4.x? This decides whether `DriverConfig.Encryption` is a v5-ADR-only topic or needs an interim v4 backport — and I can't see consumer composition patterns from inside this repo.

**Q3 (API philosophy):** For DSN params tursoengine doesn't recognize that LOOK security-relevant (e.g. a typo'd `encryption_hexkkey=` that the driver silently ignores, yielding an unencrypted database): should tursoengine adopt a STRICT posture (reject unknown `*encrypt*`/`*key*`-ish params at construction) or stay LENIENT (document the edge, fix upstream)? My lean is strict — a silent-unencrypted database violates "make impossible states unrepresentable" — but strictness changes behavior for existing DSNs, which is your call, not mine.

---

## Evidence Index

- Daemon commits carrying this session: `344d0e760`, `7f112a4ea`, `89f249de5`, `83d813fc9`, `6e49616fb`, `7a674904b`, `b9b6e42a6`, `27dd7ae86` (heuristic auto-commits; my paths verified via `git log -- metaengine/tursoengine/`).
- Files: `metaengine/tursoengine/{encryption.go, dsn.go, matview.go, register.go}` + 4 test files + README; root `CHANGELOG.md`, `AGENTS.md`, `docs/api_surface.txt`; `.agents/skills/go-cqrs-lite/references/faq.md`.
- Gates: full module suite ×2 ✓; `-race` ✓ (17.4s); vet ✓; gofmt ✓; golangci — my files zero findings (1 pre-existing foreign `maintidx`); doc-check ✓ (1012 refs); changelog-symbols ✓ (170 citations); api golden + `TestEvery*` ✓; `check-formatters.sh` self-heal executed (daemon had re-added `gci` again).
- Foreign state observed, untouched: untracked `metaengine/tursoengine/ivmrepro/` (parallel IVM repro, created 20:43), dirty `metaengine/record_context_test.go`, `.golangci.yml` re-poisoning, pre-existing `maintidx` finding on `matview_bench_test.go`.
