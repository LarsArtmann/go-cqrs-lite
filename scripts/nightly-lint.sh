#!/usr/bin/env bash
# nightly-lint.sh — scheduled local lint gate: catches the rot the auto-commit
# daemon absorbs into master WITHOUT ever running a linter (the
# daemon-bypasses-lint hole, 2026-10-03).
#
# Why nightly and NOT a pre-commit hook: the daemon commits continuously,
# including mid-edit states of concurrent agents; a pre-commit gate would
# stall or fail-loop the daemon on every transiently broken tree (cascade
# risk), and hooking #verify has no natural trigger point between daemon
# commits. A nightly timer matches the existing nightly-bench pattern: zero
# daemon interference, rot is caught within 24h, logs are decision-grade.
#
# Driver: systemd user timer (scripts/nightly/go-cqrs-nightly-lint.timer) or
# manual: scripts/nightly-lint.sh
# Logs: /var/tmp/cqrs-nightly/<UTC-date>-lint.log (LINT-ROT marker = triage)
#
# --self-test: composition checks + fault injection via NIGHTLY_LINT_CMD
# (a failing stub command must make the gate report failure).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null || echo "$SCRIPT_DIR/..")"

if [ "${1:-}" = "--self-test" ]; then
	failures=0

	bash -n "$SCRIPT_DIR/nightly-lint.sh" || failures=$((failures + 1))

	for pattern in 'NIGHTLY_LINT_CMD' 'nix run .#lint' 'LINT-ROT'; do
		if ! grep -qF -- "$pattern" "$SCRIPT_DIR/nightly-lint.sh"; then
			echo "  ✗ FAIL: nightly-lint lost contract line: $pattern" >&2
			failures=$((failures + 1))
		fi
	done

	for unit in go-cqrs-nightly-lint.service go-cqrs-nightly-lint.timer; do
		if ! [ -f "$SCRIPT_DIR/nightly/$unit" ]; then
			echo "  ✗ FAIL: missing timer unit scripts/nightly/$unit" >&2
			failures=$((failures + 1))
		fi
	done
	if ! grep -q 'nightly-lint.sh' "$SCRIPT_DIR/nightly/go-cqrs-nightly-lint.service" 2>/dev/null; then
		echo "  ✗ FAIL: timer service does not invoke nightly-lint.sh" >&2
		failures=$((failures + 1))
	fi

	# Fault injection: a failing lint command must FAIL the gate.
	tmp="$(mktemp -d)"
	printf '#!/usr/bin/env bash\necho "planted lint failure" >&2\nexit 1\n' >"$tmp/stub-lint"
	chmod +x "$tmp/stub-lint"
	if NIGHTLY_LINT_CMD="$tmp/stub-lint" NIGHTLY_LINT_LOG="$tmp/nightly.log" \
		bash "$SCRIPT_DIR/nightly-lint.sh" >/dev/null 2>&1; then
		echo "  ✗ FAIL: gate must fail when lint fails" >&2
		failures=$((failures + 1))
	fi
	if ! grep -q 'LINT-ROT' "$tmp/nightly.log" 2>/dev/null; then
		echo "  ✗ FAIL: failing lint must leave a LINT-ROT marker in the log" >&2
		failures=$((failures + 1))
	fi

	# Positive control: a passing stub must PASS the gate.
	printf '#!/usr/bin/env bash\nexit 0\n' >"$tmp/stub-lint"
	if ! NIGHTLY_LINT_CMD="$tmp/stub-lint" NIGHTLY_LINT_LOG="$tmp/ok.log" \
		bash "$SCRIPT_DIR/nightly-lint.sh" >/dev/null 2>&1; then
		echo "  ✗ FAIL: gate must pass when lint passes" >&2
		failures=$((failures + 1))
	fi

	rm -rf "$tmp"
	if [ "$failures" -gt 0 ]; then
		echo "nightly-lint self-test: ${failures} failure(s)" >&2
		exit 1
	fi
	echo "nightly-lint self-test: all green"
	exit 0
fi

LOGDIR=/var/tmp/cqrs-nightly
LOG="${NIGHTLY_LINT_LOG:-$LOGDIR/$(date -u +%Y-%m-%d)-lint.log}"
mkdir -p "$(dirname "$LOG")"
cd "$REPO" || exit 2

# The lint command this gate runs. Default: the repo's canonical lint gate
# (nix run .#lint per-module golangci-lint sweep). Overridable for self-test.
run_lint() {
	if [ -n "${NIGHTLY_LINT_CMD:-}" ]; then
		${NIGHTLY_LINT_CMD}
	elif command -v nix >/dev/null 2>&1; then
		nix run .#lint --no-write-lock-file
	else
		echo "nightly-lint: nix not on PATH and no NIGHTLY_LINT_CMD override" >&2
		return 3
	fi
}

echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] nightly lint start (repo=$REPO)" >>"$LOG"
run_lint >>"$LOG" 2>&1
rc=$?
if [ "$rc" -eq 0 ]; then
	echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] nightly lint exit=0 (clean)" >>"$LOG"
	exit 0
fi
echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] nightly lint exit=$rc" >>"$LOG"
echo "LINT-ROT: the daemon absorbed lint-dirty code into master — triage the log above" >>"$LOG"
exit "$rc"
