#!/usr/bin/env bash
# exhaustruct_v5 canary: prove every ignore-patterns entry in .golangci.yml
# still matches its target type under the INSTALLED golangci-lint's
# exhaustruct semantics. `config verify` catches schema drift, and
# check-linter-names.sh catches renames — but a SEMANTICS change (the v4
# short-name -> v5 full-type-name switch that created exhaustruct_v5) can
# silently turn every ignore-pattern into a no-op: the linter starts
# flagging the "ignored" types, or worse, ignores nothing at all.
#
# Two layers:
#   1. Static: each ignore-pattern entry is present in .golangci.yml and its
#      target type still exists (stack in-repo; bbolt via the module cache
#      when available — WARN-only so nix sandboxes without a Go module cache
#      still pass).
#   2. Behavioral (hermetic stdlib fixture, offline): a temp module
#      constructs os/exec.Cmd (ignored) and a local control struct (not
#      ignored) unexhausted. With patterns: ONLY the control is flagged.
#      Without patterns: BOTH are flagged. Together these prove the
#      linter is active AND the full-name `os/exec.Cmd` pattern actually
#      matches under current semantics — the canary fails if either
#      assumption breaks.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
config="$repo_root/.golangci.yml"
golangci_bin="${GOLANGCI_LINT:-golangci-lint}"
status=0

# --- Layer 1: pattern presence + target existence -------------------------
for pattern in "os/exec.Cmd" "go.etcd.io/bbolt.Options" \
	"github.com/larsartmann/go-cqrs-lite/stack/v4.Capabilities"; do
	if grep -qF -- "- $pattern" "$config"; then
		echo "  pattern present: $pattern"
	else
		echo "FAIL: ignore-pattern \"$pattern\" missing from .golangci.yml" >&2
		status=1
	fi
done

if grep -q "^type Capabilities struct" "$repo_root/stack/capabilities.go"; then
	echo "  target exists: stack/v4.Capabilities (stack/capabilities.go)"
else
	echo "FAIL: stack/v4.Capabilities no longer declared in stack/capabilities.go — update the ignore-pattern" >&2
	status=1
fi

gomodcache="$(go env GOMODCACHE 2>/dev/null || true)"
bbolt_src="$(find "${gomodcache:-/nonexistent}/go.etcd.io" -maxdepth 1 -type d -name 'bbolt@*' 2>/dev/null | sort -V | tail -1 || true)"
if [ -n "$bbolt_src" ] && grep -rq "^type Options struct" "$bbolt_src"; then
	echo "  target exists: go.etcd.io/bbolt.Options ($(basename "$bbolt_src") in module cache)"
else
	echo "WARN: bbolt not in module cache — skipping existence check (pattern-presence still enforced)" >&2
fi

# --- Layer 2: behavioral fixture ------------------------------------------
if ! command -v "$golangci_bin" >/dev/null 2>&1; then
	echo "FAIL: golangci-lint ($golangci_bin) not found — cannot run behavioral canary" >&2
	exit 1
fi

tmpdir="$(mktemp -d)"
trap 'chmod -R u+w "$tmpdir" 2>/dev/null; rm -rf "$tmpdir"' EXIT

cat >"$tmpdir/go.mod" <<'EOF'
module exhaustructcanary

go 1.25
EOF

cat >"$tmpdir/main.go" <<'EOF'
package main

import "os/exec"

type control struct {
	A int
	B int
}

func main() {
	_ = exec.Cmd{}
	_ = control{A: 1}
}
EOF

make_config() {
	if [ "$1" = "patterns" ]; then
		cat >"$tmpdir/.golangci.yml" <<'EOF'
version: "2"
linters:
  enable:
    - exhaustruct_v5
  settings:
    exhaustruct_v5:
      ignore-patterns:
        - os/exec.Cmd
        - go.etcd.io/bbolt.Options
        - github.com/larsartmann/go-cqrs-lite/stack/v4.Capabilities
EOF
	else
		cat >"$tmpdir/.golangci.yml" <<'EOF'
version: "2"
linters:
  enable:
    - exhaustruct_v5
EOF
	fi
}

run_lint() {
	(
		cd "$tmpdir" &&
			export GOWORK=off GOTOOLCHAIN=local GOFLAGS=-mod=mod \
				GOPROXY=off GOCACHE="$tmpdir/gocache" GOMODCACHE="$tmpdir/gomod" &&
			"$golangci_bin" run --timeout 2m ./... 2>&1
	)
}

echo "==> behavioral canary: with ignore-patterns (expect ONLY control flagged)"
make_config patterns
with_out="$(run_lint)" && with_status=0 || with_status=$?

if [ "$with_status" -eq 0 ]; then
	echo "FAIL: exhaustruct_v5 flagged nothing — the linter is NOT active in the fixture (canary cannot work)" >&2
	echo "$with_out" >&2
	status=1
elif ! grep -q "control" <<<"$with_out"; then
	echo "FAIL: expected the un-ignored control struct to be flagged; exhaustruct output was:" >&2
	echo "$with_out" >&2
	status=1
elif grep -q "exec.Cmd" <<<"$with_out"; then
	echo "FAIL: os/exec.Cmd was flagged despite the ignore-pattern — pattern no longer matches (v5 semantics drift?)" >&2
	echo "$with_out" >&2
	status=1
else
	echo "  control flagged, os/exec.Cmd ignored — pattern matching works"
fi

echo "==> behavioral canary: WITHOUT ignore-patterns (expect BOTH flagged)"
make_config none
without_out="$(run_lint)" && without_status=0 || without_status=$?

if [ "$without_status" -eq 0 ]; then
	echo "FAIL: no findings without ignore-patterns — linter inactive, canary is blind" >&2
	status=1
elif ! grep -q "control" <<<"$without_out" || ! grep -q "exec.Cmd" <<<"$without_out"; then
	echo "FAIL: expected BOTH control and exec.Cmd flagged without ignore-patterns; output was:" >&2
	echo "$without_out" >&2
	status=1
else
	echo "  both flagged without patterns — the os/exec.Cmd entry does the ignoring"
fi

if [ "$status" -eq 0 ]; then
	echo "✓ exhaustruct_v5 canary: every ignore-pattern target verified under $(golangci-lint version --short 2>/dev/null || echo 'installed') semantics"
fi

exit "$status"
