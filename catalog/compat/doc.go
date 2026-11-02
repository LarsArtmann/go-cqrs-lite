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
// encoding/jsonschema.Extract plus cue.Value.Subsume, whose own documentation
// says it "tests whether v is a backwards compatible (newer) API version of
// w". A fidelity spike over 24 fixtures taken from the policy's own rows
// scored 22/24 in the best configuration (Config{} plus cue.Raw()), which is
// not good enough because both residual misses report "compatible" for a
// change the policy calls breaking: a format swap (date-time to uuid) and a
// newly constrained array item list. CUE cannot see either, since format is
// not a constraint and an open list subsumes a constrained one. The
// plausible-sounding cue.Final() option is worse still (16/24, turning
// remove-field, rename-field and add-required-field into "compatible").
//
// A false "compatible" licenses shipping a replay-breaking change, so the
// error is asymmetric and the tolerance is zero. The evidence and the reopen
// triggers live in docs/evidence/2026-11-02_cue-subsume-spike.md and
// ADR-0163. The structural walk below is that spike's recorded fallback: it
// sees Format, Items, Pattern and Enum as first-class fields because it walks
// this repository's own schema model.
package compat
