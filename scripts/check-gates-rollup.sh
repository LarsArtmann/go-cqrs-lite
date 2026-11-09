#!/usr/bin/env bash
# check-gates-rollup.sh — ONE aggregate red/green signal across the scheduled
# and push gate workflows, so green Autoretry runs can never again mask red
# nightlies (the 2026-11-08 brutal-audit finding: three red nightly classes
# hidden behind green masking runs for days).
#
# Required workflows (edit the list in ONE place below): the latest COMPLETED
# run of each must not have conclusion=failure. In-progress/queued runs are
# ignored (they will be judged on their own schedule); cancelled/skipped runs
# are ignored (supersession is normal with cancel-in-progress).
#
# Real mode queries `gh run list`; self-test mode reads a fixture JSON from
# ROLLUP_FIXTURE_JSON so the aggregation logic is pinned without touching the
# API (planted: all-green passes; one-failure fails NAMING the owning
# workflow — the mutation proof).
#
# Usage: check-gates-rollup.sh [--self-test]
set -uo pipefail

REQUIRED_WORKFLOWS=(
	"CI"
	"Nightly Gates"
	"Fuzz Nightly"
	"Benchmarks"
	"Soaks"
)

SELF_TEST=0
[ "${1:-}" = "--self-test" ] && SELF_TEST=1

aggregate() { # <runs-json> → prints "workflow|conclusion|createdAt|url" rows, returns red count
	local json="$1" wf rc=0 row
	for wf in "${REQUIRED_WORKFLOWS[@]}"; do
		row="$(jq -r --arg w "$wf" '
			map(select(.workflowName == $w and .status == "completed"))
			| sort_by(.createdAt) | reverse
			| .[0]
			| if . == null then
				"\($w)|NO_COMPLETED_RUN||"
			  else
				"\($w)|\(.conclusion // "?")|\(.createdAt)|\(.url // "")"
			  end' <<<"$json")"
		conclusion="${row#*|}"
		conclusion="${conclusion%%|*}"
		echo "$row"
		[ "$conclusion" = "failure" ] && rc=$((rc + 1))
	done
	return "$rc"
}

if [ "$SELF_TEST" -eq 1 ]; then
	green='[{"workflowName":"CI","status":"completed","conclusion":"success","createdAt":"2026-11-08T10:00:00Z","url":"u1"},
	        {"workflowName":"Nightly Gates","status":"completed","conclusion":"success","createdAt":"2026-11-08T10:00:00Z","url":"u2"},
	        {"workflowName":"Fuzz Nightly","status":"completed","conclusion":"success","createdAt":"2026-11-08T10:00:00Z","url":"u3"},
	        {"workflowName":"Benchmarks","status":"completed","conclusion":"success","createdAt":"2026-11-08T10:00:00Z","url":"u4"},
	        {"workflowName":"Soaks","status":"completed","conclusion":"success","createdAt":"2026-11-08T10:00:00Z","url":"u5"}]'
	if ! aggregate "$green" >/dev/null; then
		echo "SELF-TEST FAIL: all-green fixture must pass"
		exit 1
	fi

	red="${green/Fuzz Nightly\",\"status\":\"completed\",\"conclusion\":\"success\"/Fuzz Nightly\",\"status\":\"completed\",\"conclusion\":\"failure\"}"
	if aggregate "$red" >/dev/null; then
		echo "SELF-TEST FAIL: red Fuzz Nightly must fail the rollup"
		exit 1
	fi
	# Capture BEFORE grepping: aggregate exits non-zero on red, and a direct
	# pipe into `grep -q` races SIGPIPE under pipefail (the 2026-10-01 lesson).
	red_rows="$(aggregate "$red")"
	grep -q '^Fuzz Nightly|failure' <<<"$red_rows" || {
		echo "SELF-TEST FAIL: rollup must NAME the owning workflow"
		exit 1
	}

	inprogress="${green/Soaks\",\"status\":\"completed\"/Soaks\",\"status\":\"in_progress\"}"
	if ! aggregate "$inprogress" >/dev/null; then
		echo "SELF-TEST FAIL: an in-progress run must not count as red"
		exit 1
	fi

	echo "check-gates-rollup self-test: OK (green passes, red fails naming the owner, in-progress ignored)"
	exit 0
fi

command -v gh >/dev/null 2>&1 || {
	echo "check-gates-rollup: gh CLI required"
	exit 1
}
command -v jq >/dev/null 2>&1 || {
	echo "check-gates-rollup: jq required"
	exit 1
}

json="$(gh run list --limit 100 --json workflowName,status,conclusion,createdAt,url 2>/dev/null)"
if [ -z "$json" ] || [ "$json" = "[]" ]; then
	echo "check-gates-rollup: no runs returned by gh — refusing to pass on missing data"
	exit 1
fi

red=0
aggregate "$json" | while IFS='|' read -r wf conclusion createdAt url; do
	if [ "$conclusion" = "failure" ]; then
		echo "RED   $wf (failed at $createdAt) $url"
	elif [ "$conclusion" = "NO_COMPLETED_RUN" ]; then
		echo "NONE  $wf — no completed run in the last 100"
	else
		echo "green $wf ($conclusion at $createdAt)"
	fi
done

# aggregate again for the exit code (the pipe above consumed it)
tmp="$(mktemp)" && trap 'rm -f "$tmp"' EXIT
aggregate "$json" >"$tmp" || red=1

if [ "$red" -ne 0 ]; then
	echo "check-gates-rollup: at least one required workflow is RED — see above; fix the OWNER, not this rollup"
	exit 1
fi
echo "check-gates-rollup: all required workflows green (or pending)"
