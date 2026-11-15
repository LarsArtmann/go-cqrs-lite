package metaengine

// Canonical turso-go version range for the grouped-materialized-view upstream
// defect (ADR-0135): grouped views silently lose cross-transaction deltas and
// collapse past ~27k rows on every driver version from TursoGoIVMVerifiedFrom
// through TursoGoIVMVerifiedThrough, last live-checked on TursoGoIVMLastVerified.
// These constants are the SINGLE SOURCE of truth: the Doctor WARN
// (materialized_view_doctor.go) renders them, and
// scripts/check-turso-version.sh (nix run .#check-turso-version) fails when
// any live doc/test citation names a different version. Re-verify live with
// the -tags ivmrepro repro suite (metaengine/tursoengine) before bumping;
// the full flip procedure is docs/turso-go-ivm-fix-flip-runbook.md.
//
// 2026-11-15 MF2.6 finding: upstream FIXED all three defects (A delta loss,
// B 27k collapse, C COMMIT abort) between v0.8.3 and v0.8.4-pre.2 — verified
// live on v0.8.4-pre.2, v0.8.4-pre.3, and v0.9.0-pre.2 (repro suite "did not
// reproduce" on all three defects, envelope guard exact through the real
// engine in enforce mode, full tursoengine suite green on v0.9.0-pre.2).
// TursoGoIVMFixedFromPre records the first fixed version; the production pin
// deliberately stays on the newest STABLE (v0.8.3, still defective) until a
// stable fixed release ships — then the flip runbook executes end-to-end in
// one change. Full evidence: docs/research/
// 2026-09-07_turso-go-ivm-commit-failure-issue-draft.md addendum.
const (
	TursoGoIVMVerifiedFrom    = "v0.7.2"
	TursoGoIVMVerifiedThrough = "v0.8.3"
	TursoGoIVMLastVerified    = "2026-11-15"

	// TursoGoIVMFixedFromPre is the first driver version verified to have
	// fixed the grouped-view defects (a PRE-RELEASE — not pinned in
	// production until it or a successor goes stable).
	TursoGoIVMFixedFromPre = "v0.8.4-pre.2"
)
