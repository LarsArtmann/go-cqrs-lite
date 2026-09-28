#!/usr/bin/env bash
# quiet-campaign.sh — batch orchestrator for load-sensitive campaign legs
# (publish-integrity plan M25). Each leg runs inside its OWN quiet window via
# scripts/quiet-window-run.sh (which owns window-wait, retries, and the log);
# this script only sequences legs, keeps one leg's failure from aborting the
# rest, and prints a receipt summary. A leg's own quality gates
# (benchmark-regression.sh, calibration-gate.sh) run INSIDE the leg command.
#
# Usage:
#   scripts/quiet-campaign.sh                          # default legs: mysql,dgraph
#   scripts/quiet-campaign.sh --legs mysql             # one leg
#   scripts/quiet-campaign.sh --legs dgraph -- <cmd>   # custom leg: cmd runs in
#                                                      # a quiet window instead of
#                                                      # the built-in dgraph bench
#   scripts/quiet-campaign.sh --dry-run                # print, do not execute
#   scripts/quiet-campaign.sh --self-test              # offline dispatch test
#
# Built-in legs (each a documented receipt source):
#   mysql   nix run .#integration-mysql-nspawn (G-T13 ADTSet leg; on failure
#           falls back to nix run .#integration-mysql-vm — the ~131s QEMU leg)
#   dgraph  ephemeral Dgraph + BenchmarkCalibration_DgraphScaled
#           (constants re-anchor campaign, calibration-2026-08-30.md §G1)
#
# Exit codes: 0 = every leg passed in a quiet window; 1 = ≥1 leg failed or
# exhausted its attempts; 2 = usage; 3 = deadline hit before a leg ran.
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
QWR="$REPO_ROOT/scripts/quiet-window-run.sh"
LEGS="mysql,dgraph"
DRY_RUN=0
SELFTEST=0
CUSTOM_CMD=()

usage() {
	sed -n '2,25p' "$0" | sed 's/^# \{0,1\}//'
	exit 2
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--legs)
		LEGS="$2"
		shift 2
		;;
	--dry-run)
		DRY_RUN=1
		shift
		;;
	--self-test)
		SELFTEST=1
		shift
		;;
	--)
		shift
		CUSTOM_CMD=("$@")
		break
		;;
	-h | --help)
		usage
		;;
	*)
		echo "quiet-campaign: unknown option: $1" >&2
		usage
		;;
	esac
done

DGRAPH_BENCH='cd metaengine/dgraphengine && GOWORK=off go test -run "^$" -bench BenchmarkCalibration_DgraphScaled -benchtime 20x -count 3 .'

leg_commands() {
	# prints: <leg-name>\t<command...>
	local leg="$1"
	case "$leg" in
	mysql)
		printf 'mysql\tnix run .#integration-mysql-nspawn || nix run .#integration-mysql-vm'
		;;
	dgraph)
		if [[ ${#CUSTOM_CMD[@]} -gt 0 ]]; then
			printf 'dgraph\t%s' "${CUSTOM_CMD[*]}"
		else
			printf 'dgraph\tnix run .#ephemeral-dgraph -- bash -c '"'"'%s'"'"'' "$DGRAPH_BENCH"
		fi
		;;
	*)
		echo "quiet-campaign: unknown leg: $leg (known: mysql, dgraph)" >&2
		return 2
		;;
	esac
}

run_campaign() {
	local failures=0 leg spec name cmd
	local IFS=','
	read -ra leg_array <<<"$LEGS"
	local logdir
	logdir="$(mktemp -d /tmp/quiet-campaign.XXXXXX)"
	echo "quiet-campaign: legs=[$LEGS] logs under $logdir"

	for leg in "${leg_array[@]}"; do
		spec="$(leg_commands "$leg")" || exit 2
		name="${spec%%$'\t'*}"
		cmd="${spec#*$'\t'}"
		echo "--- leg $name: $cmd"
		if [[ $DRY_RUN == 1 ]]; then
			echo "(dry-run: skipped)"
			continue
		fi
		if bash "$QWR" --log "$logdir/$name.log" -- bash -c "cd '$REPO_ROOT' && $cmd"; then
			echo "LEG $name: PASS (log: $logdir/$name.log)"
		else
			local rc=$?
			if [[ $rc == 3 ]]; then
				echo "LEG $name: DEADLINE — no quiet window; later legs still attempted" >&2
			else
				echo "LEG $name: FAIL (rc=$rc; log: $logdir/$name.log)" >&2
			fi
			failures=$((failures + 1))
		fi
	done

	if [[ $DRY_RUN == 1 ]]; then
		echo "quiet-campaign: dry-run complete"
		return 0
	fi
	echo "=== quiet-campaign summary: $failures failed leg(s); logs: $logdir"
	[[ $failures == 0 ]]
}

self_test() {
	local dir rc_total=0
	dir="$(mktemp -d)"
	local quiet_file="$dir/quiet.loadavg"
	printf '1.0 1.0 1.0 1/100 1\n' >"$quiet_file"

	# NOTE: capture-then-grep, never `| grep -q` — under `set -o pipefail`
	# grep -q's early exit SIGPIPEs the writer and fails the pipeline.

	# 1. dry-run dispatches both legs and executes nothing
	QUIET_WINDOW_LOADAVG_FILE="$quiet_file" "$0" --dry-run >"$dir/1.out" 2>&1
	if grep -q 'integration-mysql-nspawn' "$dir/1.out" &&
		grep -q 'BenchmarkCalibration_DgraphScaled' "$dir/1.out"; then
		echo "ok   dry-run dispatches mysql + dgraph legs"
	else
		echo "FAIL dry-run leg dispatch"
		rc_total=1
	fi

	# 2. custom -- cmd replaces the dgraph leg command
	QUIET_WINDOW_LOADAVG_FILE="$quiet_file" "$0" --dry-run --legs dgraph -- echo-marker >"$dir/2.out" 2>&1
	if grep -q 'echo-marker' "$dir/2.out"; then
		echo "ok   custom cmd overrides dgraph leg"
	else
		echo "FAIL custom cmd override"
		rc_total=1
	fi

	# 3. one real leg through quiet-window-run with a fixture load file: pass-through works
	QUIET_WINDOW_LOADAVG_FILE="$quiet_file" "$0" --legs dgraph -- true >"$dir/3.out" 2>&1
	if grep -q 'LEG dgraph: PASS' "$dir/3.out"; then
		echo "ok   quiet-window passthrough leg PASS"
	else
		echo "FAIL quiet-window passthrough"
		rc_total=1
	fi

	# 4. failing leg marks campaign failed but exits cleanly through the wrapper
	if QUIET_WINDOW_LOADAVG_FILE="$quiet_file" "$0" --legs dgraph -- false >/dev/null 2>&1; then
		echo "FAIL failing leg must fail the campaign"
		rc_total=1
	else
		echo "ok   failing leg fails campaign exit"
	fi

	# 5. unknown leg -> usage
	"$0" --legs bogus >/dev/null 2>&1
	if [[ $? == 2 ]]; then
		echo "ok   unknown leg usage"
	else
		echo "FAIL unknown leg should exit 2"
		rc_total=1
	fi

	rm -rf "$dir"
	if [[ $rc_total == 0 ]]; then
		echo "PASS: quiet-campaign self-test"
	else
		echo "FAIL: quiet-campaign self-test"
	fi
	exit "$rc_total"
}

if [[ $SELFTEST == 1 ]]; then
	self_test
fi
run_campaign
