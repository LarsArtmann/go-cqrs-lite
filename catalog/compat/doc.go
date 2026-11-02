// Package compat makes the schema-compatibility policy in
// docs/SCHEMA-COMPATIBILITY.md executable.
//
// The repository already had three separate answers to "did the payload shape
// change?". A written policy (docs/SCHEMA-COMPATIBILITY.md), static ladder
// checks (cqrs-lint E021/E022), and drift detection
// ([schema.EventSchema.Fingerprint]). A fingerprint can only report that a
// shape changed; it cannot report whether that change was additive or
// breaking. This package answers that question by walking two
// [schema.Schema] trees and classifying every difference against the policy's
// own rows.
//
// # What it is not
//
// It is not a second source of truth: schemas stay Go types rendered by
// [catalog/schema], and this package only reads them. It does not replace
// [catalog.Catalog.Validate] or [catalog.Catalog.ValidateCoeffects]; it
// composes the latter and adds the three cross-resource rules nothing else
// owns. It does not change [catalog.Registry] merge semantics.
//
// # Why no CUE
//
// The design was originally to delegate subtyping to CUE's
// encoding/jsonschema.Extract plus cue.Value.Subsume. A fidelity spike over
// the policy's own twelve fixture rows classified at most 7 of 10 shape
// fixtures correctly in every option combination, and misclassified every
// additive case, including "add an optional field" (the policy's first row).
// The evidence and the reopen trigger live in
// docs/evidence/2026-11-02_cue-subsume-spike.md and ADR-0161. The structural
// walk below is that spike's recorded fallback.
package compat
