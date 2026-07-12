# CBOR Codec Final Polish Plan

**Date:** 2026-06-11 11:47 UTC
**Status:** CBORCodec is implemented and tested. This plan covers the remaining improvements from self-review.

## What's Actually Left to Improve

### Research findings:

1. **`CanonicalEncOptions()` is RFC 7049**, NOT RFC 8949. We claim "IETF STD 94 (RFC 8949)" in the doc comment. This is a doc lie. `CoreDetEncOptions()` is the actual RFC 8949 preset. However, Canonical is a SUBSET of CoreDet — Canonical options (sorted keys, shortest floats) are included in CoreDet. For our signing use case, Canonical is sufficient. The fix is to correct the doc comment.
2. **Encoding `cbor` tag usage**: CBOR uses `json` struct tags by default (fxamacker/cbor reads them). Our test structs use `json:"name"` tags — correct, no change needed.
3. **No split brains found**: `Encoding` type used consistently. No duplicate definitions.
4. **No ghost systems**: CBORCodec is wired into Codec interface, used by all stores.

### Real issues to fix (ordered by impact/effort):

| #   | Task                                                    | Impact | Effort | Why                    |
| --- | ------------------------------------------------------- | ------ | ------ | ---------------------- |
| 1   | Fix doc comment: claim RFC 7049 Canonical, not RFC 8949 | High   | 2min   | Doc currently lies     |
| 2   | Add `cbor:"name"` tag test (verify dual-tag compat)     | Medium | 3min   | Consumers need to know |
| 3   | Add CBOR size comparison test (vs JSON)                 | High   | 3min   | Proves CBOR value      |
| 4   | Add signing determinism integration test                | High   | 8min   | Core use case          |
| 5   | Add `TestCBORCodec_RoundTrip_Slice`                     | Low    | 2min   | Coverage gap           |
| 6   | Add `TestCBORCodec_RoundTrip_NestedStruct`              | Low    | 3min   | Coverage gap           |
| 7   | Run full test+lint, commit, push                        | -      | 5min   | Gate                   |

### What we should NOT do (YAGNI):

- DecMode configuration — consumers can wrap CBORCodec if they need strict decode
- CoreDet vs Canonical switch — Canonical is the right default for signing, and we don't need CoreDet's additional rules
- Property-based tests — our fuzz tests already cover this
- CBOR streaming encoder — premature for a library
- Module-level README — doc.go + examples are sufficient
