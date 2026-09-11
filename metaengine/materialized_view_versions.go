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
const (
	TursoGoIVMVerifiedFrom    = "v0.7.2"
	TursoGoIVMVerifiedThrough = "v0.8.0-pre.10"
	TursoGoIVMLastVerified    = "2026-09-11"
)
