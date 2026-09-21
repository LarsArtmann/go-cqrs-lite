#!/usr/bin/env bash
# quiet-window-run.sh — run a load-sensitive command only inside a quiet window.
#
# Waits until load1 AND load5 are both below --ceiling, then runs CMD. A
# non-zero CMD exit is retried up to --attempts times (each attempt re-waits
# for a fresh quiet window) unless --no-retry: a scarce quiet window should
# not be wasted on a single unlucky failure, but a deterministically failing
# command must not be retried forever. Per-attempt output is appended to
# --log. The wrapped command is expected to run its own quality gates
# (benchmark-regression.sh, calibration-gate.sh, ...); this tool only owns
# the window and the retry budget.
#
# Usage:
#   scripts/quiet-window-run.sh -- ./scripts/benchmark-regression.sh
#   scripts/quiet-window-run.sh --ceiling 3 --deadline 3600 -- ./cmd/x
#   scripts/quiet-window-run.sh --self-test
#
# Options:
#   --ceiling N      load1/load5 must both be < N (default: 5)
#   --deadline SECS  give up after this many seconds total (default: 21600)
#   --attempts N     max command attempts (default: 3)
#   --no-retry       run the command exactly once
#   --log FILE       append per-attempt output (default: temp file, kept and
#                    its path printed at the end)
#   --self-test      fault-injection suite: planted loadavg fixtures and
#                    counter-fixture commands (never touches /proc/loadavg)
#   --               the command to run (required except with --self-test)
#
# Env: QUIET_WINDOW_LOADAVG_FILE — point the load probe at a fixture file
# instead of /proc/loadavg. Internal hook for --self-test only — do not set
# in normal use.
#
# Exit codes: 0 = command passed inside a quiet window; 1 = command failed
# on every attempt (last attempt's output is at the end of the log); 2 =
# usage; 3 = deadline hit before any attempt ran.
set -uo pipefail

CEILING=5
DEADLINE_SECS=21600
ATTEMPTS=3
LOG=""
SELFTEST=0
LOADAVG_FILE="${QUIET_WINDOW_LOADAVG_FILE:-/proc/loadavg}"

usage() {
	sed -n '2,32p' "$0" | sed 's/^# \{0,1\}//'
	exit 2
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--ceiling)
		CEILING="$2"
		shift 2
		;;
	--deadline)
		DEADLINE_SECS="$2"
		shift 2
		;;
	--attempts)
		ATTEMPTS="$2"
		shift 2
		;;
	--no-retry)
		ATTEMPTS=1
		shift
		;;
	--log)
		LOG="$2"
		shift 2
		;;
	--self-test)
		SELFTEST=1
		shift
		;;
	--)
		shift
		break
		;;
	-h | --help)
		usage
		;;
	*)
		echo "quiet-window-run: unknown option: $1" >&2
		usage
		;;
	esac
done

stamp() { date -u +%Y-%m-%dT%H:%M:%SZ; }

window_open() {
	awk -v c="$CEILING" '{exit ($1 < c && $2 < c) ? 0 : 1}' "$LOADAVG_FILE"
}

run_command() {
	if [[ $SELFTEST == 1 ]]; then
		echo "quiet-window-run: --self-test runs its own scenarios; do not pass a command"
		usage
	fi
	if [[ $# -eq 0 ]]; then
		echo "quiet-window-run: no command given (use -- CMD)" >&2
		usage
	fi
	[[ -n "$LOG" ]] || LOG="$(mktemp /tmp/quiet-window-run.XXXXXX.log)"
	echo "[$(stamp)] === quiet-window-run: ceiling=$CEILING deadline=${DEADLINE_SECS}s attempts=$ATTEMPTS cmd=$*"

	local deadline=$((SECONDS + DEADLINE_SECS))
	local attempt
	for ((attempt = 1; attempt <= ATTEMPTS; attempt++)); do
		while ! window_open; do
			if ((SECONDS > deadline)); then
				echo "[$(stamp)] DEADLINE: no quiet window (ceiling=$CEILING) within ${DEADLINE_SECS}s; command NOT run. Log: $LOG"
				exit 3
			fi
			sleep 30
		done
		echo "[$(stamp)] attempt $attempt/$ATTEMPTS: window open ($(tr '\n' ' ' <"$LOADAVG_FILE")); running: $*"
		"$@" >>"$LOG" 2>&1
		local rc=$?
		echo "[$(stamp)] attempt $attempt exit=$rc"
		if [[ $rc == 0 ]]; then
			echo "[$(stamp)] === PASS (log: $LOG)"
			exit 0
		fi
	done
	echo "[$(stamp)] === FAIL: command failed on all $ATTEMPTS attempt(s) (log: $LOG)"
	exit 1
}

self_test() {
	local dir
	dir="$(mktemp -d)"
	local rc_total=0

	local quiet_file="$dir/quiet.loadavg"
	printf '2.0 3.0 4.0 1/100 1\n' >"$quiet_file"
	local loud_file="$dir/loud.loadavg"
	printf '20.0 30.0 40.0 1/100 1\n' >"$loud_file"

	local counter="$dir/counter"
	echo 0 >"$counter"
	local fail_once
	fail_once="$dir/fail-once.sh"
	cat >"$fail_once" <<EOF
#!/usr/bin/env bash
n=\$(cat "$counter")
echo "attempt \$n"
if ((n == 0)); then echo 1 | tee "$counter" >/dev/null; exit 7; fi
exit 0
EOF
	chmod +x "$fail_once"

	check() {
		local name="$1" want="$2" got="$3"
		if [[ "$got" == "$want" ]]; then
			echo "ok   $name (exit $got)"
		else
			echo "FAIL $name: want exit $want, got $got"
			rc_total=1
		fi
	}

	# 1. quiet fixture + passing command -> 0
	QUIET_WINDOW_LOADAVG_FILE="$quiet_file" "$0" --no-retry --log "$dir/1.log" -- true >/dev/null 2>&1
	check "quiet+pass" 0 $?

	# 2. loud fixture forever + 3s deadline -> 3, command never ran
	QUIET_WINDOW_LOADAVG_FILE="$loud_file" "$0" --deadline 3 --log "$dir/2.log" -- true >/dev/null 2>&1
	check "never-quiet deadline" 3 $?
	if grep -q "attempt" "$dir/2.log" 2>/dev/null; then
		echo "FAIL never-quiet deadline: command ran despite deadline"
		rc_total=1
	fi

	# 3. fails once then passes, attempts 3 -> 0, both attempts logged
	QUIET_WINDOW_LOADAVG_FILE="$quiet_file" "$0" --attempts 3 --log "$dir/3.log" -- "$fail_once" >/dev/null 2>&1
	check "retry-then-pass" 0 $?
	grep -q "attempt 0" "$dir/3.log" && grep -q "attempt 1" "$dir/3.log" \
		&& echo "ok   retry logged both attempts" \
		|| {
			echo "FAIL retry-then-pass: attempts missing from log"
			rc_total=1
		}

	# 4. always-failing command, attempts 2 -> 1
	QUIET_WINDOW_LOADAVG_FILE="$quiet_file" "$0" --attempts 2 --log "$dir/4.log" -- false >/dev/null 2>&1
	check "always-fail exhausted" 1 $?

	# 5. no command -> usage exit 2
	QUIET_WINDOW_LOADAVG_FILE="$quiet_file" "$0" >/dev/null 2>&1
	check "usage" 2 $?

	rm -rf "$dir"
	if [[ $rc_total == 0 ]]; then
		echo "PASS: quiet-window-run self-test"
	else
		echo "FAIL: quiet-window-run self-test"
	fi
	exit "$rc_total"
}

if [[ $SELFTEST == 1 ]]; then
	self_test
fi

run_command "$@"
