# Status: Value-Integrity Wave 1 — Byte Pins, Error Codes, and a Parallel Closure

> **Created:** 2026-10-25 10:46 CET (Sunday)
> **HEAD at writing:** `ce727efd1` ("chore: auto-commit 1 changed file(s)"), tree clean
> **Session window:** execution began 2026-10-24 12:34 CEST, was suspended, and
> resumed 2026-10-25 10:46 CET — a ~22 h gap during which **76 commits** from
> other sessions landed (`git rev-list --count 5bb89e622..HEAD` = 76;
> `git diff --stat 5bb89e622..HEAD` = 124 files, +22 361 / −185).
> **Plan driving this work:**
> [`docs/planning/2026-10-24_12-09_SUPERB-value-integrity-and-gate-unblock-pareto-plan.md`](../planning/2026-10-24_12-09_SUPERB-value-integrity-and-gate-unblock-pareto-plan.md)
> (Wave 1 = T1, T2, T5–T8).
> **Point-in-time snapshot.** Not a living document.

---

## 0. TL;DR

1. **T1 is done and mutation-proven** — three raw-bytes pins
   (`metaengine/sqliteengine/stored_bytes_pin_test.go`, 213 lines) read the
   actual SQLite columns for `StreamAppend`, `MapSet` and `MapSetAt` +
   version row, and two independent mutations turn them red.
2. **T2's code half is done, its proof half is not.** `assertRejection` now
   pins the error CODE as well as the family across nine rejection specs; the
   batch all-or-nothing mutation (T2.1) was never run.
3. **T5–T8 were closed by a parallel session while this session was
   suspended** — `keycodec.EncodeJSONErr` plus all 21 LSM value sites plus
   `adttest.AssertValueEncodeRejection`. I verified all five affected modules
   green at HEAD rather than trusting the commit message.
4. **The plan's own execution log is wrong about T1–T3** ("pre-shipped"): T1
   was mine, T2's mutation and T3's turso proof never happened.
5. **The COOKBOOK still described the LSM stringify gap as open** while the
   CHANGELOG claimed closure. Fixed in this session (two-paragraph rewrite,
   committed by the daemon as `ce727efd1`).

---

## a) FULLY DONE (this session, with evidence)

### a1. T1 — stored-bytes pins for the SQLite write paths

`metaengine/sqliteengine/stored_bytes_pin_test.go` (new, 213 lines, package
`sqliteengine_test`).

| Spec | What it pins | Literal expectation |
| ---- | ------------ | ------------------- |
| `TestStoredBytes_StreamAppend` | each value of a 3-value batch, in caller order, in its own `meta_stream_log` row | `{"title":"pin","n":7}`, `raw-string`, `42` |
| `TestStoredBytes_MapSet` | `meta_map` key column and value column for a struct value | key `k1`, value `{"title":"pin","n":7}` |
| `TestStoredBytes_MapSetAt` | latest view and `meta_cell_versions` row carry the **same bytes** | both `{"title":"pin","n":7}` |

Deliberately literal (not derived from the encoder) so an encoder that agrees
with itself cannot satisfy the pin. Raw reads go through a second `database/sql`
handle on the same shared-cache in-memory URI.

**Mutation proofs (both run, both red, both restored):**

| Mutation | Where | Result |
| -------- | ----- | ------ |
| Remove the string fast-path from `EncodeStreamValueErr` (strings get JSON-quoted) | `metaengine/stream_codec.go` | `stream row 1 = "\"raw-string\"", want "raw-string"` → FAIL |
| Wrap the value before marshalling (`json.Marshal(struct{ V any }{value})`) | `metaengine/sqliteengine/encoding.go` | `MapSet` and `MapSetAt` both `= "{\"V\":{…}}", want "{\"title\":…}"` → FAIL |

Evidence: `cd metaengine/sqliteengine && GOWORK=off go test ./... -count=1` →
`ok … 5.507s`; filtered run `ok … 0.169s`; `gofmt -l` empty; `go vet ./...`
clean.

### a2. T2 (code half) — error-code assertions on every rejection spec

`metaengine/sqliteengine/encoding_error_test.go`: `assertRejection` now takes a
`wantCode` and asserts `errorfamily.Classify(err) == Rejection` **and**
`errorfamily.Code(err)`:

```go
// skip-validate: fragment of the test file, not a standalone file
const (
	dbValueEncodeCode     = "sqliteengine.encode_value"
	streamValueEncodeCode = "metaengine.encode_stream_value"
)
```

Nine specs now classify: map set, planned map set, map update, map set-at, log
append, multimap add (all `sqliteengine.encode_value`), stream append and
stream append-expected (`metaengine.encode_stream_value`).

**Plan premise corrected:** the code is not split "map vs stream" but
"sqlite-local value encoder vs shared stream codec". `LogAppend` and
`MultiAdd` go through the sqlite-local encoder, so they carry the *map* code.
My first attempt asserted `metaengine.encode_stream_value` for them and went red
with `error code = "sqliteengine.encode_value"` — the test was right, the
assumption was wrong.

### a3. Independent verification of the parallel session's Wave-1 closure

I re-ran every affected module at HEAD instead of accepting the commit message
`7396d262d` ("Close the LSM silent-stringify class…"):

| Module | Result |
| ------ | ------ |
| `metaengine/keycodec` | ok 0.007 s |
| `metaengine/adttest` | ok 0.015 s |
| `metaengine/pebbleengine` | ok 0.528 s |
| `metaengine/bboltengine` | ok **489.283 s** (soak included; see §d6) |
| `metaengine/badgerengine` | ok 0.716 s |
| `metaengine/sqliteengine` | ok 5.507 s |

Code inspection confirms the class is closed where claimed: all 21 LSM value
sites call `keycodec.EncodeJSONErr` (pebble 7, bbolt 7, badger 7) and the
`encodeJSON = keycodec.EncodeJSON` aliases are gone. `keycodec.encode_value`
is rowed in `docs/error-taxonomy.md`; the api golden carries both new symbols
(`docs/api_surface.txt`, 6309 lines).

### a4. COOKBOOK de-staling (fixed on sight, per the standing permission)

`metaengine/COOKBOOK.md` said the LSM engines "still encode values through
`keycodec.EncodeJSON`", quoted **22** write sites, and claimed closing it
"needs `keycodec.EncodeJSON` to return an error". All three statements were
false at HEAD. Replaced with the closure (21 sites, `EncodeJSONErr`,
`adttest` pin), the still-open graph-encoder tail, and an explicit note that
`sqliteengine.encodeKey` / pebble's `indexKeyValue` stringify **identity**, not
payload, on purpose.

### a5. Gate probes run in this session

| Gate | Result |
| ---- | ------ |
| `nix run .#check-md-go` | **GREEN** — validator rebuilt (`3f466de`), 1519 code blocks valid, 83 skipped, 105 baselined archived errors |
| `bash scripts/check-error-taxonomy.sh` | **GREEN** — 544 codes / 20 modules, 574 doc claims |
| `nix run .#check-file-size` | **RED** — 3 offenders, all foreign (see §d/§e) |
| `art-dupl check . --threshold 3` | **RED** — 1 new group (was 8 on 2026-10-24) |

So T15 (md-go-validator derivation) is, contrary to the 12:00 report, **done** —
the vendorHash re-pin from the 19:29 session holds and the gate runs.

---

## b) PARTIALLY DONE

| Item | Done | Missing |
| ---- | ---- | ------- |
| **T2** batch atomicity | pinned by `TestSQLite_StreamAppend_RejectsBatchAtomically` + `…Expected…`, both now code-asserted | **T2.1 mutation never run** — no revert of the batch pre-encode was ever executed, so "the test can fail" is asserted, not proven |
| **T1** coverage | unplanned `MapSet`, `MapSetAt` + version row, `StreamAppend` | no pin for planned-table writes (`execPlannedSet` → different table/column), `MapUpdate`'s re-encode, `StreamAppendExpected`, and **no byte pin at all for the 3 LSM engines** |
| **T11** docs | CHANGELOG rows accurate, TODO_LIST row accurate ("3/4 fixed"), COOKBOOK now accurate (this session) | no COOKBOOK numbers for the LSM change; FAQ and `metaengine/README.md` have no statement to fix (verified by grep) |
| **T4** conformance | `adttest.AssertValueEncodeRejection` + wired into pebble/bbolt/badger | sqlite deliberately **not** wired (keeps its stronger dedicated specs with `sqliteengine.encode_value`) — defensible, but the harness therefore does not cover the engine that started the class |
| **Commit hygiene** | all work is committed, tree clean | every phase landed inside `chore: auto-commit` commits; the authored commit failed (see §d2) |

---

## c) NOT STARTED (verified by inspection, not by assumption)

| Task | Evidence it is untouched |
| ---- | ------------------------ |
| **T3** turso delegation proof | no `*encode*`/`*rejection*` test in `metaengine/tursoengine/`; no `encode_value` string in that module |
| **T9** graph encoders | `mysqlengine/graph.go:235`, `duckdbengine/graph.go:122`, `pgengine/graph.go:126`, `badgerengine/graph.go:52` still `fmt.Sprintf("%v", v)` |
| **T10** bigtable | `bigtableengine/engine.go:321` still `[]byte(fmt.Sprintf("%v", v))` |
| **T12** file-size | 3 offenders: `scheduling/sqlstore/store.go` **382** (was 363 — grew while deprecated), `system/adapter_event.go` 352, `metaengine/layout.go` 360 |
| **T13** clones | 1 new group: identical signatures at `catalog/docserver/layout.templ:15` + `:31` and `metaengine/pebbleengine/sort_index.go:26` |
| **T14** repo lint | not re-run this session (a full `#lint` is ~98 modules; the 2026-10-24 matrix had a dozen red modules) |
| **T16** taxonomy generator | not fixed; the parallel session *accepted* the generator's sentinel-attribution drift as "the source of truth" instead |
| **T17** `#verify-fast` | never completed (T12–T14 still red) |
| **T18–T24** | untouched (envelope owner packet, KV tag measurement, backfill design, preflight script, batch benchmark, harvest) |

---

## d) TOTALLY FUCKED UP (honest list)

1. **The plan's execution log claims "T1–T3 pre-shipped per §1.2".** §1.2 is a
   table of what shipped in waves 1–4; it does not contain T2 or T3. T2's
   mutation was never run and T3's turso proof does not exist. A future reader
   would believe Wave 1 is fully proven. This is the same failure class the
   12:00 report was written to kill (claim reflects intent, not evidence) —
   committed again 8 hours later.
2. **My authored commit lost the race and my work became `chore:` history.**
   `git commit` exited 128: `cannot lock ref 'HEAD': is at 79e968998… but
   expected 619854859…` — the auto-commit daemon committed between `git add`
   and `git commit`. T1 + T2 landed inside `79e968998 chore: auto-commit 2
   changed file(s)`. The plan's guard #8 ("commit at each phase boundary")
   is not executable while a daemon commits every few seconds.
3. **I left the tree RED for ~2 minutes on 2026-10-24 12:41.** My T2 code
   became live as soon as it was written, and the daemon committed it *before*
   I ran the test that proved my LogAppend/MultiAdd codes were wrong. Another
   session then fixed it at 12:43 (`90e320753`, which also renamed the
   constants to `dbValueEncodeCode`/`streamValueEncodeCode`). I discovered this
   only when reconstructing the timeline for this report.
4. **My 2026-10-25 "fix" run was a no-op.** I re-applied the rename and the
   code swap by string replacement; the file had already been fixed, so the
   script changed nothing and the green run proved only that someone else's
   fix worked. I reported it to myself as my own fix at the time.
5. **Docs contradicted each other for ~14 hours.** `CHANGELOG.md` (19:1x)
   claimed the LSM class closed; `metaengine/COOKBOOK.md` still described it as
   an open gap with a stale site count. Nobody reads both files in the same
   minute, so nothing caught it. Fixed in this session.
6. **`bboltengine`'s default suite is 8 minutes** (489 s) because the soak test
   runs unconditionally; the parallel session used `SOAK_SKIP_BOLT=1` (6.3 s)
   and recorded that, so the fast path exists but is not the default.
7. **I nearly shipped a second overclaim.** My first COOKBOOK rewrite grouped
   `pebbleengine`'s `layout_planner.go` `%v` sites with the graph encoders as
   "value loss". They are index-key stringification (identity) — reading the
   function before committing caught it. This is the third time in two sessions
   that the overclaim came from assuming instead of reading.

---

## e) WHAT WE SHOULD IMPROVE

1. **Byte pins belong inside the migration task, not a later one.** T6–T8
   changed the encoder on 21 sites; not one of those engines has a
   stored-bytes pin. The SQLite pin took ~25 minutes and would have caught a
   silent form change in any LSM engine too.
2. **Doc closure must ride the same commit as the code.** T11 was a Wave-2
   task. Had it been merged into T6–T8, the COOKBOOK would never have lagged.
3. **A contradiction check between CHANGELOG and COOKBOOK/README claims is
   cheap** — even a grep for "Known gap" plus the symbol it names would have
   flagged this one.
4. **Parallel sessions need claim markers.** Two agents executed the same plan
   within 8 hours; §11 of the plan has no "claimed by / at" field, so both
   assumed the other had covered T1–T3.
5. **`SOAK_SKIP_BOLT=1` should be the documented iteration default** for
   `bboltengine`, with the full soak named as the pre-merge run.
6. **Mutation proofs should be scripted, not hand-patched.** My two mutations
   were applied and reverted with ad-hoc Python; a
   `scripts/mutation-check.sh` with `trap`-based restore would remove the risk
   of leaving a tree mutated (and would make the proof reproducible).
7. **`art-dupl`'s remaining "new group" is a signature-only clone** (two templ
   one-liners and a pebble index helper sharing a parameter list). A
   signature-only class, or a higher token threshold for parameter lists,
   would stop this false-positive tail.
8. **The daemon should either be pausable or adopt authored staging**: the
   current behaviour destroys phase boundaries, and the plan's mitigation
   (path-staged commits) is not executable in practice.
9. **Deprecated modules should be frozen, not grown.** `scheduling/sqlstore` is
   deprecated-until-v5.0.0 yet `store.go` went 363 → 382 in a day.
10. **Rejection codes should be named by *layer*, not by data shape.**
    `sqliteengine.encode_value` covering map, log and multimap writes is
    accurate but reads like a map-only code. `turso` inherits the same code by
    delegation, which will surprise anyone grepping for `tursoengine.*`.

---

## f) Up to 50 things to do next

Ordered by leverage. Items marked *(plan Tn)* are already scoped there.

1. **T2.1** revert `StreamAppend`'s batch pre-encode (encode inside the loop),
   prove `…RejectsBatchAtomically` goes red, restore *(plan T2)*.
2. **T3** write a turso delegation rejection test and record whether the
   libSQL path runs or skips under `-short`.
3. **T1-ext** add a planned-table byte pin (`execPlannedSet` path).
4. **T1-ext** add a `MapUpdate` re-encode byte pin.
5. **T1-ext** add a `StreamAppendExpected` byte pin (same helper, atomic path).
6. **T1-ext** add a **pebble** byte pin (single `encodeJSONErr` site first,
   then the batch).
7. **T1-ext** add a **bbolt** byte pin.
8. **T1-ext** add a **badger** byte pin.
9. Run all four new pins under `-race`.
10. **Fix the plan log's "T1–T3 pre-shipped" line** and add a per-task
    "verified by / at" column to §11 *(plan T24)*.
11. **T9** decide the graph rule (keys stringified, values JSON) and write it
    in the COOKBOOK before touching code.
12. Migrate `mysqlengine/graph.go`.
13. Migrate `duckdbengine/graph.go`.
14. Migrate `pgengine/graph.go`.
15. Migrate `badgerengine/graph.go` (the fourth engine the plan forgot).
16. Add one graph rejection spec per engine (or one skip-gated shared spec).
17. **T10** migrate `bigtableengine/engine.go:321` or record an explicit,
    reasoned exception.
18. **T12** split `scheduling/sqlstore/store.go` (382 → ≤350).
19. Freeze `scheduling/sqlstore` against growth (lint or a size pin).
20. **T12** split `system/adapter_event.go` (352).
21. **T12** split `metaengine/layout.go` (360).
22. Re-run `#check-file-size` to green.
23. **T13** classify the single remaining clone group (annotate vs split).
24. Raise the `art-dupl` threshold or add a signature-only class.
25. **T14** capture the failing-module list from `#lint` verbatim.
26. Fix `keycodec` godox/gosmopolitan if unowned.
27. Classify the remaining lint reds by owner.
28. **T17** run `#verify-fast` end-to-end after 18–27.
29. Record the composed receipt in the plan (§11).
30. **T16** add a sentinel-verification step to `generate-error-taxonomy.sh`.
31. Add a mutation-tested golden to `check-error-taxonomy.sh --self-test`.
32. Re-generate and diff: only intended rows change.
33. **T4-ext** wire `adttest.AssertValueEncodeRejection` into `sqliteengine`
    behind a code-parameter so both codes are covered.
34. **T4-ext** add turso to the same conformance row once T3 exists.
35. Document `SOAK_SKIP_BOLT=1` as the iteration default in
    `docs/agents/gotchas-testing.md`.
36. Add a `scripts/wave-preflight.sh` (pgrep buildflow, daemon check, load
    gate, tree state) *(plan T22)*.
37. Write `scripts/mutation-check.sh` with trap-based restore.
38. **T18** send the envelope one-pager (option C+D vs E) to the owner.
39. Fold the measured LSM cost into it *(plan T18)*.
40. **T23** benchmark the batch encode (1 and 100 values, allocs/op).
41. Record the numbers in COOKBOOK (no gate).
42. Add a CHANGELOG↔COOKBOOK contradiction grep to a docs gate.
43. Add the 21-site count to `FEATURES.md`'s metaengine row (currently only
    CHANGELOG/TODO carry it).
44. Re-run `#check-api-stability` to confirm the 6309-line golden is current.
45. Re-run `#check-modsums` (76 commits of dependency churn since the last run).
46. Re-run `#check-coverage` (new `adttest`/`keycodec` code shifts the
    metaengine floor).
47. **T24** harvest §f into `TODO_LIST.md`.
48. Update the 12:09 plan header status to "Wave 1 closed except T2.1/T3".
49. Index this report in `docs/status/README.md`.
50. Re-run `scripts/verify-chain.sh` once 18–28 are green.

---

## g) Three questions I cannot answer myself

1. **Concurrent ownership.** Another session executed T4–T8 of *this* plan
   inside the 22 h this session was suspended. Do you want me to keep executing
   the remainder (T2.1, T3, T9, T10, then the gates), or is that session's
   claim still live? The plan has no claim markers, so I cannot tell from the
   tree.
2. **Graph identity rule.** Should graph node/edge **keys** keep the `%v`
   stringification (identity, like `sqliteengine.encodeKey`) while graph
   **values** must be JSON — or is a graph value allowed to stay lenient so
   existing rows keep round-tripping? This decides T9 and T10 and I cannot
   infer it from the code: `mysqlengine`/`duckdbengine`/`pgengine`/`badgerengine`
   graph.go currently stringify both.
3. **Authored history vs the daemon.** The auto-commit daemon committed my work
   mid-flight and cost me the commit (exit 128, HEAD ref race), and the plan's
   phase-boundary rule is therefore unenforceable. Is a `chore:` commit
   acceptable as the permanent record (with the plan log as the narrative), or
   should I invest in a coordination mechanism (staging area lease, daemon
   pause flag) before the next wave?

---

## Appendix — evidence index

| Claim | Command / file | Result |
| ----- | -------------- | ------ |
| Byte pins green | `cd metaengine/sqliteengine && GOWORK=off go test ./... -count=1` | ok 5.507 s |
| Byte pins red under mutation A | drop string fast-path, same command | `stream row 1 = "\"raw-string\""` FAIL |
| Byte pins red under mutation B | wrap value, same command | `MapSet` + `MapSetAt` value FAIL |
| LSM closure real | `grep -n "EncodeJSONErr" metaengine/{pebble,bbolt,badger}engine/*.go` | 21 hits, 0 `EncodeJSON` aliases |
| LSM suites green | per-module `GOWORK=off go test ./... -count=1` | keycodec 0.007 s, adttest 0.015 s, pebble 0.528 s, bbolt 489.283 s, badger 0.716 s |
| md-go gate now works | `nix run .#check-md-go` | 1519 blocks valid, 105 baselined archived errors |
| taxonomy gate green | `bash scripts/check-error-taxonomy.sh` | 544 codes / 20 modules / 574 claims |
| api golden carries both | `docs/api_surface.txt` | `metaengine/keycodec/func EncodeJSONErr`, `metaengine/adttest/func AssertValueEncodeRejection` |
| file-size still red | `nix run .#check-file-size` | sqlstore 382, adapter_event 352, layout 360 |
| clones nearly clean | `art-dupl check . --threshold 3` | 1 new group (was 8) |
| 76 commits of parallel work | `git rev-list --count 5bb89e622..HEAD` | 76 |
| daemon won the commit race | `git commit` output on 2026-10-24 12:40 | `cannot lock ref 'HEAD'` exit 128 |
| wrong-code assumption | test output 2026-10-24 12:41 | `error code = "sqliteengine.encode_value", want "metaengine.encode_stream_value"` |

**Files created this session:** `metaengine/sqliteengine/stored_bytes_pin_test.go`.
**Files modified this session:** `metaengine/sqliteengine/encoding_error_test.go`,
`metaengine/COOKBOOK.md`; transient mutations in `metaengine/stream_codec.go`
and `metaengine/sqliteengine/encoding.go` (both restored byte-identically —
`git diff` empty afterwards).
**Files this session must not touch:** anything from the concurrent session
touching `scenario/`, `systemscenario/`, `catalog/docserver/`,
`scheduling/sqlstore/`, `versions.json`.
