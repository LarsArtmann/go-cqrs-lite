# Status: KV/LSM Layout Recalibration from Size-Stable Benches — 2026-08-16 14:09

Task: TODO_LIST §"Layout roles" — "Re-derive KV/LSM layout constants from
size-stable benches": make the pre-2026-08-15 calibration benches
(`metaengine/layout_calibration_bench_test.go`,
`metaengine/bench/bench_layout_calibration_disk_test.go` EmbedWrite)
size-stable (replace-only mutation, as the Row/Columnar benches do) and
re-measure. Session interrupted mid-derivation by a storage-model challenge;
numeric constant edits and all downstream steps are still pending.

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Root-cause analysis — TWO flaws found, one deeper than the TODO described. (1) Memory EmbedWrite: typed assertion passes → append grows values unboundedly (the documented drift). (2) Disk EmbedWrite: `prev.(diskCalibOrder)` SILENTLY NEVER MATCHES on Pebble/bbolt — `MapUpdate` decodes prev into `map[string]any`, so the mutation was a **no-op**: the bench measured read + re-encode + write of an unchanged value. The prior session's premise ("LSM bench appends a child every iteration → grows unboundedly") was factually wrong for the disk bench. |
| 2  | Verified (sub-agent code audit) `MultiAdd` is O(1) on memory (slice append), pebble (seq-keyed `db.Set`), bbolt (seq-keyed `bucket.Put`) → NormalizeWrite benches were already size-stable; no changes needed there.                                                                                                                                                                                                                                                                                                                                                              |
| 3  | Memory bench fixed: replace-only mutation via package-level `calibFixedChildren` (fixed 4-item slice); header rewritten (run command with jsonv2 tag, size-stability contract, cross-ref to the disk bench, benchtime 2s).                                                                                                                                                                                                                                                                                                                                                          |
| 4  | Disk bench fixed: EmbedWrite now uses the shared `rmwReplaceChildren` map-form helper from the Row bench file (same package); latent no-op mutation bug documented in the header; run guidance updated to 2s.                                                                                                                                                                                                                                                                                                                                                                      |
| 5  | `go vet` clean on `metaengine` and `metaengine/bench` (GOWORK=off, `-tags "goexperiment.jsonv2"`).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| 6  | Re-measurement, three rounds: (a) concurrent run — INVALID, outliers from self-inflicted load contention; (b) sequential 5-count — still noisy (bbolt EmbedRead trending 3255→7543ns, bimodal bbolt NormalizeWrite); (c) final exclusive sequential run, `-benchtime=1s -count=10`, medians computed by a throwaway Go script (`/tmp/analyze_calib.go`, logs `/tmp/kv_calib3.log`, `/tmp/lsm_calib3.log`). Allocs constant per op across all counts — size-stability empirically confirmed.                                                                                  |
| 7  | Final measured medians: KV memory EmbedRead 70.6ns / EmbedWrite 145.7ns / NormRead 126.8ns / NormWrite 123.0ns → **read 1.796x, write 0.844x**, storage 0.485x (JSON 3-projection model, unchanged). Pebble (2811/9798/4676/3093 ns) → read 1.664x, write 0.316x. bbolt (2990/15468/4535/15491 ns) → read 1.516x, **write 1.001x** (single-writer model fully neutralizes normalize's write advantage — confirms the 2026-08-11 finding). **LSM geomean: read 1.59x, write 0.56x**.                                                                                       |
| 8  | `layout_scoring.go`: the big provenance comment above `scoreEmbed` rewritten — two conventions documented (KV/LSM anchor convention vs Row/Columnar geomean-centering), honest measured ratios, explicit disclosure that READ constants are floor-adjusted (honest KV 0.90 / LSM 1.18 would flip cells), removed the inaccurate "All calibrated cells are geometric-mean centered" claim.                                                                                                                                                                                         |
| 9  | Derivation table computed and hand-verified against all 16 matrix cells (embed rows unchanged as anchors): **KV normalize 1.8 / 0.48→0.84 / 0.63; LSM normalize 1.45→1.67 / 0.75→0.62 / 0.80**. All 8 KV/LSM winners preserved; the fragile LSM × Balanced margin improves **0.01 → 0.10**. Verified only `layout_scoring.go` holds the numbers — `layout_matrix_test.go` pins winners, not values; no other test references the constants.                                                                                                                                     |

## b) PARTIALLY DONE

| # | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| - | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Constant re-derivation: ONLY the `scoreEmbed` doc comment landed. The numeric constants in `scoreNormalize` (KV WriteCost 0.48, LSM ReadCost 1.45, WriteCost 0.75) and their inline comments are NOT yet edited — the file still carries the old values. Known wording issue to fix while editing: the new comment says read constants are "pinned at the minimum"; that is true for LSM (floor 1.67) but KV stays at 1.8 vs its floor 1.43 — either lower KV read to ~1.45 or reword to "at or above the floor".                                                                   |
| 2 | LSM storage pair (1.15/0.80): investigation concluded it inherits the memory-bench JSON 3-projection model ("storage ratios are engine-independent and measured once", 2026-08-11 convention), while the 2026-08-15 Row/Columnar calibration set the precedent of measuring REAL on-disk bytes per engine. Engine-specific effects the JSON model cannot see: bbolt stores every multimap child under its own seq-suffixed key (`mm|col|key|%020d`, ~40 bytes) inside B+Tree pages; Pebble compresses SSTable blocks (repeated field names inside one embed blob compress better than many small child values). Disk storage bench NOT yet written — blocked on Question 1. |

## c) NOT STARTED

1. Numeric constant edits in `scoreNormalize` + inline comments (the actual
   `0.48→0.84`, `1.45→1.67`, `0.75→0.62` changes).
2. `layout_matrix_test.go` run after the constant edits (all 16 cells must
   keep their winners).
3. Full `metaengine` module test run.
4. Disk storage-size bench (real Pebble/bbolt bytes) — pending Question 1.
5. Docs: TODO_LIST check-off, CHANGELOG entry, ADR-0124 addendum table
   (0.5/1.0/1.3 vs 1.8/0.48/0.63 and 0.74/1.10/1.15 vs 1.45/0.75/0.80 rows +
   "0.48x (KV) and 0.75x (LSM)" conclusion bullet),
   `docs/planning/METAENGINE-LAYOUT-PLANNING-MODEL.md` priority table
   (quotes "KV read 1.8 vs 0.5; LSM read 1.45 vs 0.74", "KV write 0.48 vs
   1.0; LSM write 0.75 vs 1.10").
6. Gates: gofumpt/nix fmt on touched files, doc-check (AGENTS.md/skill refs
   untouched so far), bench-module lint.
7. Bench header: document the derivation protocol (exclusive sequential run,
   `-count=10`, median) so future recalibrations are reproducible.

## d) TOTALLY FUCKED UP (honest list)

| #  | What                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Ran the KV and LSM benches **concurrently** (two background shells at once) — the exact load-sensitivity class AGENTS.md warns about. Produced outliers (memory EmbedWrite 244/252ns vs 145ns clean; pebble EmbedRead 5.6µs spike). Self-caught from the data, re-ran exclusively. Cost one wasted measurement round. Lesson already in AGENTS.md; I failed to apply it to *benches* specifically.                                                                                                                                                                                                                                            |
| 2  | A multiedit dropped `b.ReportAllocs()` from the memory EmbedWrite (old_string contained it, new_string didn't). Caught in post-edit file review and re-added. Lesson: always diff-review the file after multiedit, not just trust "Applied N edits".                                                                                                                                                                                                                                                                                                                                                                                          |
| 3  | First derivation table quoted LSM storage as "0.485x (model, unchanged)" — i.e., silently inherited the memory-engine JSON model for the disk family. Did not notice the inconsistency until challenged. The Row/Columnar precedent (real per-engine bytes) makes the LSM storage pair the weakest number in the table; it is now Question 1.                                                                                                                                                                                                                                    |
| 4  | Record correction (prior session's fuckup, not mine): the 2026-08-15 report item 1 claimed the LSM bench "appends a child on every iteration → the value grows unboundedly". False — the typed assertion meant the mutation never applied. The re-measure goal was right; the flaw taxonomy was wrong. Fixed bench now asserts nothing about the shape but actually mutates, and the header documents why.                                                                                                                                                                                                                                       |

## e) WHAT WE SHOULD IMPROVE IN THIS PROJECT

1. Storage parity: add a real disk-bytes storage bench for Pebble/bbolt
   (same shape as `BenchmarkRowLayoutCalibration_Storage` — separate dirs per
   layout, close-before-stat, `reportStorageRatio` convention) so the LSM
   storage pair stops inheriting a JSON model.
2. The memory `StorageOverhead` bench prints to stderr while the Row storage
   bench uses `b.ReportMetric` — unify on `ReportMetric`.
3. Check in the median/ratio derivation script (currently throwaway in /tmp)
   under `scripts/` or document the protocol in the bench header.
4. No-op mutations must fail loudly: bench mutation callbacks that silently
   return `prev` unchanged (the disk bench bug) should be structurally
   impossible — e.g. a mutation that returns the input unchanged should
   `b.Fatal`, or the helper should count applied mutations.
5. Per-engine write spread: pebble 0.32x vs bbolt 1.00x is a 3x range hidden
   in one family geomean. At minimum document the spread next to the
   constant; structurally, per-engine overrides would be a design change.
6. The old constants were derived from single 60s runs (count=1); the
   protocol should be median-of-N with an exclusive machine — encode it in
   the bench headers.

## f) NEXT

1. Apply the numeric constants (KV norm write 0.84; LSM norm read 1.67, write
   0.62) + inline comments; resolve the "minimum vs at-or-above floor" wording
   (likely: lower KV read 1.8→1.45 for consistency, or reword).
2. Run `layout_matrix_test` (16 cells) + full `metaengine` test suite.
3. Decide Question 1 → optionally write the disk storage bench and re-derive
   the LSM storage pair from real bytes.
4. Update docs: TODO_LIST check-off, CHANGELOG, ADR-0124 addendum,
   METAENGINE-LAYOUT-PLANNING-MODEL.md.
5. Format gates (gofumpt on touched files) + doc-check + bench lint.
6. Encode the derivation protocol in bench headers (see e.3/e.6).
7. If Question 2 answer is "fully honest ratios": flip the matrix expectations
   (honest values make KV Normalize in all four cells and LSM Normalize in
   three of four — only LSM ReadSpeed stays Embed, margin 0.07) and update
   every doc that quotes the matrix.
8. If Question 3 answer is "split per-engine": design per-engine layout cost
   overrides in `layout_scoring.go` (structural change, needs its own session).
9. Unify the memory StorageOverhead bench on `b.ReportMetric`.
10. Make bench mutations self-verifying (e.4).

## g) QUESTIONS

1. **LSM storage pair**: measure real Pebble/bbolt on-disk bytes now (new
   bench, effort S/M, matches the Row/Columnar precedent and likely moves the
   1.15/0.80 pair), or keep the engine-independent JSON 3-projection model for
   KV/LSM and re-derive only read/write from the size-stable benches?
2. **Lever-pinning vs full honesty**: the 2026-08-11 session deferred going
   fully data-driven ("would require changing the design invariant per user
   approval"). With honest read ratios (KV 0.90, LSM 1.18), KV flips to
   Normalize in ALL four priority cells and LSM in three of four (only LSM
   ReadSpeed stays Embed, margin 0.07). Keep lever-pinning (current choice,
   documented in the constants), or go fully honest and update the matrix?
3. **bbolt vs pebble write spread** (1.00x vs 0.32x): keep the family geomean
   (current) or introduce per-engine layout cost overrides?

— next step (unless redirected): item 1 (apply the numeric constants), then 2
(tests), then 3/4 pending the answers above.
