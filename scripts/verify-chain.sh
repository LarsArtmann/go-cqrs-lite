#!/usr/bin/env bash
# verify-chain.sh — resilient composed-verify runner: the load-storm survivor.
#
# Problem (every wave session since 2026-10-11): the composed proof chain
# (exclusive #verify behind the composed gate) must run on a shared host where
# evening load storms refuse nix runs for HOURS (load5 30-90, IO PSI 80%+,
# memory-emergency guards). Sessions armed ad-hoc /tmp scripts (retry 38/40,
# exhaustion, re-arming by hand each session); this script is that runner,
# durable and self-tested.
#
# Chain: go-env → can-run-composed-gate --wait-loop (blocks until quiet,
# max-wait 4h) → #check-coverage → #verify. Retry rule: ONLY exit code 12
# (the no-heavy-builds guard refusing under load — transient by definition)
# is retried; any other failure is a REAL gate result and stops the chain
# immediately (retrying a red #verify would just burn an hour).
#
# Usage:
#   scripts/verify-chain.sh              # run the chain (arm in background
#                                        # for storm-riding: it sleeps 300s
#                                        # between guard-refusal retries)
#   scripts/verify-chain.sh --self-test  # fault-injection suite
#
# Env knobs (self-test only):
#   VERIFY_CHAIN_GATE_CMD       overrides the wait-loop gate command string
#   VERIFY_CHAIN_COVERAGE_CMD   overrides the #check-coverage command string
#   VERIFY_CHAIN_VERIFY_CMD     overrides the #verify command string
#   VERIFY_CHAIN_MAX_RETRIES    per-step guard-refusal retries (default 40)
#   VERIFY_CHAIN_RETRY_SLEEP    seconds between refusals (default 300)
#   VERIFY_CHAIN_LOG            log file (default: build/verify-chain-<ts>.log
#                               — durable per the 2026-10-01 silent-death
#                               forensics lesson; /tmp logs are tmpfiles-bait)
#
# Exit: 0 chain green; 1 any step failed (real gate failure or retry
# exhaustion); 2 usage.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null || echo "$SCRIPT_DIR/..")"

MAX_RETRIES="${VERIFY_CHAIN_MAX_RETRIES:-40}"
RETRY_SLEEP="${VERIFY_CHAIN_RETRY_SLEEP:-300}"
GATE_CMD="${VERIFY_CHAIN_GATE_CMD:-}"
COVERAGE_CMD="${VERIFY_CHAIN_COVERAGE_CMD:-nix run .#check-coverage}"
VERIFY_CMD="${VERIFY_CHAIN_VERIFY_CMD:-nix run .#verify}"

usage() {
	sed -n '2,8p' "${BASH_SOURCE[0]}"
	exit 2
}

run_with_retry() {
	# $@ = command; retries ONLY rc=12 (load-guard refusal), stop on real red.
	local rc i
	for i in $(seq 1 "$MAX_RETRIES"); do
		"$@"
		rc=$?
		[ "$rc" -eq 0 ] && return 0
		if [ "$rc" -ne 12 ]; then
			echo "[chain] step FAILED for real (rc=$rc), not retrying" >&2
			return "$rc"
		fi
		echo "[chain] step refused by load guard (rc=12), retry $i/$MAX_RETRIES in ${RETRY_SLEEP}s"
		sleep "$RETRY_SLEEP"
	done
	echo "[chain] retry budget exhausted ($MAX_RETRIES guard refusals)" >&2
	return 1
}

chain_main() {
	# Mandatory env chain (docs/agents/gowork-modes.md): the ambient session
	# can carry GOTOOLCHAIN=local and broken cache paths that poison builds.
	source "$SCRIPT_DIR/go-env.sh" 2>/dev/null
	cd "$REPO" || return 1

	if [ -n "$GATE_CMD" ]; then
		bash -c "$GATE_CMD" || {
			echo "[chain] composed gate did not go green" >&2
			return 1
		}
	else
		VERIFY_MAX_WAIT=14400 bash "$SCRIPT_DIR/can-run-composed-gate.sh" --wait-loop || {
			echo "[chain] composed gate did not go green" >&2
			return 1
		}
	fi

	run_with_retry bash -c "$COVERAGE_CMD" || return 1
	run_with_retry bash -c "$VERIFY_CMD"
}

# ── self-test: stub commands + fault injection ──────────────────────────────
self_test() {
	local rc=0 tmp
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' RETURN

	check() { # $1=desc, $2=test command result (0 = pass)
		if [ "$2" -ne 0 ]; then
			echo "  ✗ FAIL: $1" >&2
			rc=1
		fi
	}

	check_eq() { # $1=desc $2=actual $3=expected
		if [ "$2" != "$3" ]; then
			echo "  ✗ FAIL: $1 (got '$2' want '$3')" >&2
			rc=1
		fi
	}

	bash -n "$SCRIPT_DIR/verify-chain.sh"
	check "syntax" $?

	for pattern in 'rc=12' 'can-run-composed-gate.sh' '#verify'; do
		grep -qF -- "$pattern" "$SCRIPT_DIR/verify-chain.sh"
		check "contract line present: $pattern" $?
	done

	# Counting stub: succeeds once its counter reaches N, exits CODE before.
	counter_stub() { # $1=counter-file $2=success-at-N $3=refusal-code
		printf '#!/usr/bin/env bash\nf=%s\n[ -f "$f" ] || echo 0 >"$f"\nc=$(cat "$f")\necho $((c+1)) >"$f"\nif [ "$c" -ge %s ]; then exit 0; fi\nexit %s\n' "$1" "$2" "$3" >"$1.stub"
		chmod +x "$1.stub"
	}

	# T1: green stubs everywhere -> chain exits 0, each step ran once.
	printf '#!/usr/bin/env bash\nexit 0\n' >"$tmp/pass"
	chmod +x "$tmp/pass"
	counter_stub "$tmp/cov" 0 0
	counter_stub "$tmp/ver" 0 0
	VERIFY_CHAIN_GATE_CMD="$tmp/pass" VERIFY_CHAIN_COVERAGE_CMD="$tmp/cov.stub" \
		VERIFY_CHAIN_VERIFY_CMD="$tmp/ver.stub" VERIFY_CHAIN_RETRY_SLEEP=0 \
		bash "$SCRIPT_DIR/verify-chain.sh" >/dev/null 2>&1
	check "all-green chain exits 0" $?
	check_eq "coverage step ran exactly once" "$(cat "$tmp/cov" 2>/dev/null || echo 0)" "1"
	check_eq "verify step ran exactly once" "$(cat "$tmp/ver" 2>/dev/null || echo 0)" "1"

	# T2: two guard refusals then success -> chain exits 0 after retries.
	counter_stub "$tmp/cov2" 2 12
	VERIFY_CHAIN_GATE_CMD="$tmp/pass" VERIFY_CHAIN_COVERAGE_CMD="$tmp/cov2.stub" \
		VERIFY_CHAIN_VERIFY_CMD="$tmp/pass" VERIFY_CHAIN_RETRY_SLEEP=0 \
		bash "$SCRIPT_DIR/verify-chain.sh" >/dev/null 2>&1
	check "refused-then-green step exits 0" $?
	check_eq "step attempted 3 times (2 refusals + 1 green)" "$(cat "$tmp/cov2")" "3"

	# T3: REAL failure (rc=1) -> immediate stop, exactly one attempt.
	counter_stub "$tmp/cov3" 99 1
	VERIFY_CHAIN_GATE_CMD="$tmp/pass" VERIFY_CHAIN_COVERAGE_CMD="$tmp/cov3.stub" \
		VERIFY_CHAIN_VERIFY_CMD="$tmp/pass" VERIFY_CHAIN_RETRY_SLEEP=0 \
		bash "$SCRIPT_DIR/verify-chain.sh" >/dev/null 2>&1
	want_fail=$? # expect nonzero
	if [ "$want_fail" -eq 0 ]; then
		echo "  ✗ FAIL: real gate failure fails the chain" >&2
		rc=1
	fi
	check_eq "real failure not retried (1 attempt)" "$(cat "$tmp/cov3")" "1"

	# T4: permanent refusal exhausts the retry budget -> chain fails.
	counter_stub "$tmp/cov4" 99 12
	VERIFY_CHAIN_GATE_CMD="$tmp/pass" VERIFY_CHAIN_COVERAGE_CMD="$tmp/cov4.stub" \
		VERIFY_CHAIN_VERIFY_CMD="$tmp/pass" VERIFY_CHAIN_RETRY_SLEEP=0 \
		VERIFY_CHAIN_MAX_RETRIES=2 \
		bash "$SCRIPT_DIR/verify-chain.sh" >/dev/null 2>&1
	want_fail=$?
	if [ "$want_fail" -eq 0 ]; then
		echo "  ✗ FAIL: retry exhaustion fails the chain" >&2
		rc=1
	fi
	check_eq "budget respected (2 attempts, then exhaustion)" "$(cat "$tmp/cov4")" "2"

	# T5: gate never green -> chain fails BEFORE coverage runs.
	printf '#!/usr/bin/env bash\nexit 1\n' >"$tmp/failgate"
	chmod +x "$tmp/failgate"
	counter_stub "$tmp/cov5" 0 0
	VERIFY_CHAIN_GATE_CMD="$tmp/failgate" VERIFY_CHAIN_COVERAGE_CMD="$tmp/cov5.stub" \
		VERIFY_CHAIN_VERIFY_CMD="$tmp/pass" VERIFY_CHAIN_RETRY_SLEEP=0 \
		bash "$SCRIPT_DIR/verify-chain.sh" >/dev/null 2>&1
	want_fail=$?
	if [ "$want_fail" -eq 0 ]; then
		echo "  ✗ FAIL: red gate fails the chain" >&2
		rc=1
	fi
	[ ! -f "$tmp/cov5" ]
	check "coverage never ran when gate red" $?

	if [ "$rc" -eq 0 ]; then
		echo "  ✓ verify-chain self-test passed"
	else
		echo "  self-test FAILED" >&2
	fi
	return "$rc"
}

case "${1:-}" in
"")
	# Durable log (2026-10-01 lesson: chains die silently and /tmp logs vanish;
	# build/ is gitignored and survives). pipefail keeps the chain's own rc.
	LOG="${VERIFY_CHAIN_LOG:-$REPO/build/verify-chain-$(date +%Y%m%d-%H%M%S).log}"
	mkdir -p "$(dirname "$LOG")" 2>/dev/null || true
	chain_main "$@" 2>&1 | tee "$LOG"
	;;
--self-test) self_test ;;
*) usage ;;
esac
