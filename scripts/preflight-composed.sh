#!/usr/bin/env bash
# preflight-composed.sh — run the cheap verify phases BEFORE launching a
# composed gate. The S03 arc (2026-09-19/20) burned attempts 7-9 on failures
# a sub-5-minute phase check would have caught pre-launch: each composed
# attempt reaches the expensive phases (test/race ~25 min) only after the
# cheap gates — a stale api golden found by attempt 9 was knowable in ~40 s.
# This wrapper runs those phases standalone, stops at the first red, and
# prints the remedy — fix it, THEN launch #verify via the --wait-loop recipe.
#
# Phases (each <5 min on a warm cache):
#   lint-config   golangci config verify + hash golden + depguard + formatters
#   templ         templ codegen drift + FileName tripwire
#   bench-gate    benchmark regression gate fixture tests
#   coverage      coverage drift gate
#   api-stability golden vs source (the stale-golden class)
#   duplication   no-new-clones gate
#
# Usage:
#   bash scripts/preflight-composed.sh                     # all phases
#   PREFLIGHT_ONLY=templ,api-stability bash scripts/preflight-composed.sh
#   PREFLIGHT_SKIP=coverage,duplication bash scripts/preflight-composed.sh
#   bash scripts/preflight-composed.sh --self-test         # fixture suite
#   bash scripts/preflight-composed.sh --list              # phase names only
#
# Self-test fixture hook: PREFLIGHT_FAKE_RC_<PHASE-UPPERCASE> forces a phase's
# exit code without running its command (hermetic; never set in normal use).
#
# Exit codes: 0 = all phases green, 1 = a phase failed (remedy printed),
# 2 = usage (unknown phase in ONLY/SKIP).
set -uo pipefail

# Forced go env chain (T05): the phases shell out to `nix run .#check-*`
# apps that inherit THIS shell's env — under the ambient GOTOOLCHAIN=local
# + full /mnt/buildcache caches the api-stability watermill tidy check dies
# with ENOSPC and reads as a false "not tidy" failure (observed 2026-09-22).
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/go-env.sh"

PHASES=(
	"lint-config|nix run .#check-lint-config|intentional config change? scripts/check-golangci-hash.sh --update and commit BOTH .golangci.yml + the golden; unexpected? git diff .golangci.yml then git restore .golangci.yml"
	"templ|nix run .#check-templ|regenerate from the right cwd: (cd catalog/docserver && templ generate); never from the repo root (FileName metadata bakes the cwd)"
	"bench-gate|nix run .#check-bench-gate|a fixture failure means benchmark-regression.sh semantics drifted — fix the script or the fixture, never the expectation"
	"coverage|nix run .#check-coverage|coverage dropped below the floor — add tests in the named module(s) or run nix run .#check-coverage -- --update if the floor moved deliberately"
	"api-stability|nix run .#check-api-stability|API surface changed without a golden regen: cd cmd/api-stability && GOWORK=off go run . --update (same edit rule, AGENTS contract)"
	"duplication|nix run .#check-duplication|new clone group: consolidate, or annotate //art-dupl:accept <reason> on/above the region's first line, or re-pin baseline for structural shifts"
)

phase_names() {
	local p
	for p in "${PHASES[@]}"; do
		cut -d'|' -f1 <<<"$p"
	done
}

usage_fail() {
	echo "preflight-composed: $1" >&2
	exit 2
}

selected_phases() {
	local -a out=()
	local want p name
	declare -A only=()
	if [[ -n "${PREFLIGHT_ONLY:-}" ]]; then
		IFS=',' read -ra want <<<"$PREFLIGHT_ONLY"
		for w in "${want[@]}"; do
			only["$w"]=1
		done
	fi
	for p in "${PHASES[@]}"; do
		name="${p%%|*}"
		if [[ ${#only[@]} -gt 0 ]]; then
			[[ -n "${only[$name]:-}" ]] || continue
		fi
		if [[ -n "${PREFLIGHT_SKIP:-}" ]] && grep -qF "$name" <<<",${PREFLIGHT_SKIP},"; then
			continue
		fi
		out+=("$p")
	done
	if [[ ${#only[@]} -gt 0 ]]; then
		for w in "${!only[@]}"; do
			grep -qxF "$w" <(phase_names) || usage_fail "unknown phase in PREFLIGHT_ONLY: $w (valid: $(phase_names | tr '\n' ' '))"
		done
	fi
	if [[ -n "${PREFLIGHT_SKIP:-}" ]]; then
		local IFS=','
		for w in ${PREFLIGHT_SKIP}; do
			grep -qxF "$w" <(phase_names) || usage_fail "unknown phase in PREFLIGHT_SKIP: $w (valid: $(phase_names | tr '\n' ' '))"
		done
	fi
	printf '%s\n' "${out[@]}"
}

run_phase() {
	local spec="$1" name cmd remedy fake_var start
	IFS='|' read -r name cmd remedy <<<"$spec"
	fake_var="PREFLIGHT_FAKE_RC_$(echo "$name" | tr '[:lower:]-' '[:upper:]_')"
	start=$SECONDS
	if [[ -n "${!fake_var:-}" ]]; then
		rc="${!fake_var}"
	else
		bash -c "$cmd" >/tmp/preflight-"$name".log 2>&1
		rc=$?
	fi
	if ((rc == 0)); then
		echo "  ✓ PASS: $name ($((SECONDS - start))s)"
		return 0
	fi
	echo "  ✗ FAIL: $name (rc=$rc, log: /tmp/preflight-$name.log)"
	echo "      remedy: $remedy"
	return 1
}

case "${1:-}" in
--list)
	phase_names
	exit 0
	;;
--self-test)
	echo "━━━ preflight-composed self-test ━━━"
	failures=0
	check() {
		if [[ "$2" == "$3" ]]; then
			echo "  ✓ PASS: $1"
		else
			echo "  ✗ FAIL: $1 (want $2, got $3)"
			failures=$((failures + 1))
		fi
	}

	PREFLIGHT_FAKE_RC_LINT_CONFIG=0 PREFLIGHT_FAKE_RC_TEMPL=0 \
		PREFLIGHT_FAKE_RC_BENCH_GATE=0 PREFLIGHT_FAKE_RC_COVERAGE=0 \
		PREFLIGHT_FAKE_RC_API_STABILITY=0 PREFLIGHT_FAKE_RC_DUPLICATION=0 \
		bash "$0" >/tmp/preflight-selftest.log 2>&1
	check "all-fake-green passes" 0 "$?"

	pout=$(PREFLIGHT_FAKE_RC_TEMPL=1 \
		PREFLIGHT_FAKE_RC_LINT_CONFIG=0 PREFLIGHT_FAKE_RC_BENCH_GATE=0 \
		PREFLIGHT_FAKE_RC_COVERAGE=0 PREFLIGHT_FAKE_RC_API_STABILITY=0 \
		PREFLIGHT_FAKE_RC_DUPLICATION=0 bash "$0" 2>&1)
	check "failing phase stops the run" 1 "$?"
	grep -q "remedy: regenerate from the right cwd" <<<"$pout" ||
		{
			check "failed phase prints its remedy" 0 1
		}

	out2=$(PREFLIGHT_ONLY=templ PREFLIGHT_FAKE_RC_TEMPL=0 bash "$0" 2>&1)
	[[ $(grep -c '✓ PASS' <<<"$out2") == 1 ]] && grep -q 'templ' <<<"$out2"
	check "PREFLIGHT_ONLY runs exactly the named phase" 0 "$?"

	PREFLIGHT_SKIP=coverage,duplication PREFLIGHT_FAKE_RC_LINT_CONFIG=0 \
		PREFLIGHT_FAKE_RC_TEMPL=0 PREFLIGHT_FAKE_RC_BENCH_GATE=0 \
		PREFLIGHT_FAKE_RC_API_STABILITY=0 \
		bash "$0" >/tmp/preflight-selftest.log 2>&1
	check "PREFLIGHT_SKIP drops the named phases" 0 "$?"

	PREFLIGHT_ONLY=nonsense bash "$0" >/dev/null 2>&1
	check "unknown ONLY phase is a usage error" 2 "$?"

	if ((failures == 0)); then
		echo "self-test: all green"
		exit 0
	fi
	echo "self-test: ${failures} failure(s)"
	exit 1
	;;
-h | --help)
	sed -n '2,33p' "$0"
	exit 0
	;;
"")
	:
	;;
*)
	usage_fail "unknown flag: $1 (see --help)"
	;;
esac

failures=0
selected_output=$(selected_phases) || exit "$?"
mapfile -t SELECTED <<<"$selected_output"
echo "━━━ preflight-composed: ${#SELECTED[@]} phase(s) before any #verify launch ━━━"
for spec in "${SELECTED[@]}"; do
	run_phase "$spec" || failures=$((failures + 1))
done

if ((failures > 0)); then
	echo "✗ preflight RED (${failures} phase(s)) — fix the remedies above BEFORE nix run .#verify"
	exit 1
fi
echo "✓ preflight GREEN — launch with: nix run .#quiet-window-run -- nix run .#verify"
