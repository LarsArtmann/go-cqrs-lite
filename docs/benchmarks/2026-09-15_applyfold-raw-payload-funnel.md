# applyFold raw-payload funnel — struct hot-path defense — 2026-09-15

**Provenance:** quick micro-bench on the dev laptop (AMD Ryzen AI MAX+ 395,
32 threads, `-benchtime 1s`, single run, GOWORK=off module mode). NOT a
quiet-window calibration run; `benchmark-baseline.txt` was NOT re-pinned and
no dgraph constants were re-anchored from these numbers.

**Question it answers** (05-38 §b2): the encoded-apply fix added
`decodeRawFoldPayload` to the single `applyFold` funnel every dispatch path
shares. What does the struct hot path pay for it?

## Numbers (`metaengine/encoded_bench_test.go`)

| Benchmark                        | ns/op   | B/op | allocs/op |
| -------------------------------- | ------- | ---- | --------- |
| `DecodeRawFoldPayloadStruct`     | 1.9     | 0    | 0         |
| `ApplyFoldStructPayload` (total) | 377.5   | 80   | 3         |
| `ApplyFoldEncodedPayload` (total)| 809.5   | 168  | 6         |

**Reading:** the struct hot path pays **~1.9 ns and zero allocations** — one
failed `jsontext.Value` type assertion, then pass-through. The ~432 ns delta
between the struct and encoded `applyFold` runs is the per-fold JSON decode,
paid only when the payload actually arrives undecoded (`ApplyEncoded` /
`ApplyEncodedRecord` / their EventLog replays). The funnel does not tax the
struct path.
