#!/usr/bin/env bash
# check-gosec.sh — security scan over the core CQRS modules (CI parity).
#
# WHY: the CI "Security Scan (gosec)" leg ran a floating `nix-shell -p gosec`,
# which broke twice at once (2026-11-04): (1) 45 longstanding findings in
# legacy SQL-building code kept the leg permanently red, and (2) nixpkgs'
# stock gosec binary is compiled with an older Go toolchain than the workspace
# builds with, so its importer cannot read go1.27 export data — every
# generics-heavy dependency (otter/v2, failsafe-go) then failed type-checking
# and drove the exit code non-zero even with zero findings. This script is the
# one canonical invocation: the flake app pins a gosec rebuilt with the
# go1.27 builder plus go_1_27 on PATH, and GOTOOLCHAIN=local keeps the
# export-data writer on the same toolchain line as the reader.
#
# WHAT IS CHECKED: gosec over the core module trees (the same set the CI leg
# has scanned since 2026-06). Findings must be zero; suppressions are trailing
# `// #nosec GXXX -- reason` comments — the raw gosec binary ignores
# //nolint:gosec (that is the golangci dialect, a 2026-11-04 lesson: two such
# comments had been silently ineffective for months).
#
# Output: gosec.sarif in the working directory (the CI leg uploads it).
#
# Usage:
#   bash scripts/check-gosec.sh              # gate: exit non-zero on any finding
#   bash scripts/check-gosec.sh --self-test  # planted-fixture mutation proof
#
# Env overrides:
#   GOSEC_BIN  gosec binary to drive (default: `gosec` from PATH)
#   GOSEC_OUT  SARIF output path (default: ./gosec.sarif)
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

GOSEC_BIN="${GOSEC_BIN:-gosec}"
GOSEC_OUT="${GOSEC_OUT:-gosec.sarif}"

# The core-module scan set, identical to the CI leg's scope since 2026-06.
GOSEC_MODULES=(
	./event/v4/...
	./command/v4/...
	./query/v4/...
	./decider/v4/...
	./storage/v4/...
	./middleware/v4/...
	./signing/v4/...
	./listing/v4/...
	./transport/http/...
)

# The runtime `go` must stay on the same toolchain line as the gosec binary's
# importer (see WHY): forbid the auto-download that would out-write the reader.
export GOTOOLCHAIN=local

# One scratch directory removed on exit. The if-form matters: an && chain
# with a false left side returns 1, and a failing EXIT trap overrides the
# script's own success exit (proven 2026-11-05: gate mode was red-by-
# construction, "0 findings" with rc=1).
WORK=""
cleanup() {
	if [[ -n "$WORK" && -d "$WORK" ]]; then
		rm -rf "$WORK"
	fi
}
trap cleanup EXIT

self_test() {
	WORK="$(mktemp -d)"
	export GOCACHE="$WORK/gocache"
	export GOFLAGS=-mod=mod

	# Fixture 1 — clean module: the gate machinery must pass it.
	mkdir -p "$WORK/clean"
	cat >"$WORK/clean/go.mod" <<'EOF'
module selftest.example/gosec-clean

go 1.27
EOF
	cat >"$WORK/clean/main.go" <<'EOF'
package main

import "fmt"

func main() { fmt.Println("clean") }
EOF

	# Fixture 2 — planted G402 (InsecureSkipVerify): the gate MUST flag it.
	mkdir -p "$WORK/vuln"
	cat >"$WORK/vuln/go.mod" <<'EOF'
module selftest.example/gosec-vuln

go 1.27
EOF
	cat >"$WORK/vuln/main.go" <<'EOF'
package main

import (
  "crypto/tls"
  "fmt"
)

func main() {
  _ = &tls.Config{InsecureSkipVerify: true} // planted G402
  fmt.Println("vulnerable")
}
EOF

	local clean_rc=0 vuln_rc=0 vuln_hits=0
	(cd "$WORK/clean" && GOWORK=off "$GOSEC_BIN" -fmt json -out "$WORK/clean.json" ./...) ||
		clean_rc=$?
	if [[ "$clean_rc" -ne 0 ]]; then
		echo "self-test FAIL: clean fixture reported rc=$clean_rc (expected 0)"
		return 1
	fi

	(cd "$WORK/vuln" && GOWORK=off "$GOSEC_BIN" -fmt json -out "$WORK/vuln.json" ./...) ||
		vuln_rc=$?
	if [[ "$vuln_rc" -eq 0 ]]; then
		echo "self-test FAIL: planted G402 fixture passed (gate cannot fail — useless)"
		return 1
	fi
	vuln_hits="$(python3 -c "
import json
print(sum(1 for i in json.load(open('$WORK/vuln.json'))['Issues'] if i['rule_id'] == 'G402'))")"
	if [[ "$vuln_hits" -lt 1 ]]; then
		echo "self-test FAIL: non-zero rc but no G402 finding named (rc=$vuln_rc)"
		return 1
	fi

	echo "self-test OK: clean fixture green, planted G402 flagged (rc=$vuln_rc, hits=$vuln_hits)"
}

if [[ "${1:-}" == "--self-test" ]]; then
	self_test
	exit 0
fi

"$GOSEC_BIN" -fmt sarif -out "$GOSEC_OUT" "${GOSEC_MODULES[@]}"
echo "check-gosec: 0 findings across ${#GOSEC_MODULES[@]} module trees ($GOSEC_OUT)"
