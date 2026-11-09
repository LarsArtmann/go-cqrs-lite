#!/usr/bin/env bash
# check-echo-gates.sh — detect "echo gates": shell variables whose check
# result is computed but only ever echoed, never branched on.
#
# The class (2026-11-08 audit, verify-docs.sh module-count gate): a script
# computes drift into a variable, prints it, and exits 0 either way — a gate
# that LOOKS like verification but cannot fail. This detector flags
# `name=$(...)` assignments whose every subsequent reference lives inside
# echo/printf lines (or comments), with zero references in tests, ifs,
# arithmetic, or exit paths.
#
# Usage:
#   scripts/check-echo-gates.sh [--self-test]
#
# --self-test: run against planted fixtures (one healthy gate, one echo
#   gate) and verify the detector flags exactly the echo gate.
#
# Known limits (deliberate, to keep false positives at zero):
#   - Line-based: a var referenced inside a multi-line string that only
#     happens to start with echo is treated as an echo use. Conservative.
#   - Variables whose every use is inside $(... ) substitution on an echo
#     line count as echo uses.
#   - A variable reused in another script is invisible here (per-file scan).
#
# Exemptions: an assignment line carrying a trailing
#   # echo-gate:accept <reason>
# is skipped — intentional informational displays inside otherwise-real
# gates (counts in OK summaries, elapsed-time banners). The reason is
# mandatory and must say WHY the value may never branch.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_DIR="${REPO_ROOT}/scripts"

self_test() {
	SELFTEST_DIR="$(mktemp -d)"
	trap 'rm -rf "$SELFTEST_DIR"' EXIT
	local fixture_dir="$SELFTEST_DIR"

	cat >"$fixture_dir/healthy.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
drift=$(grep -c stale docs.txt || true)
if [ "$drift" -ne 0 ]; then
	echo "FAIL: $drift stale rows"
	exit 1
fi
EOF

	cat >"$fixture_dir/echo-gate.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
drift=$(grep -c stale docs.txt || true)
echo "found $drift stale rows"
EOF

	cat >"$fixture_dir/exempted.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
rows=$(wc -l <docs.txt) # echo-gate:accept informational — banner count; the gate below owns the verdict
echo "scanned $rows rows"
if grep -q stale docs.txt; then
	exit 1
fi
EOF

	local out rc=0
	out="$(scan_dir "$fixture_dir")" || rc=$?

	if [ "$rc" -eq 0 ]; then
		echo "SELF-TEST FAIL: planted echo gate was not flagged"
		return 1
	fi
	if ! printf '%s\n' "$out" | grep -q "echo-gate.sh"; then
		echo "SELF-TEST FAIL: output does not name the echo-gate fixture:"
		echo "$out"
		return 1
	fi
	if printf '%s\n' "$out" | grep -q "healthy.sh"; then
		echo "SELF-TEST FAIL: healthy fixture falsely flagged:"
		echo "$out"
		return 1
	fi
	if printf '%s\n' "$out" | grep -q "exempted.sh"; then
		echo "SELF-TEST FAIL: exemption marker did not suppress the fixture:"
		echo "$out"
		return 1
	fi
	echo "SELF-TEST OK: echo gate flagged, healthy + exempted fixtures passed"
	return 0
}

# scan_dir <dir> — prints one finding line per echo-gate variable, rc 1 if any.
scan_dir() {
	local dir="$1"
	local findings=0

	local f
	while IFS= read -r f; do
		local line var
		while IFS= read -r line; do
			# Assignment from command/v4 substitution at statement start;
			# echo-gate:accept lines are deliberate informational displays.
			case "$line" in
			*'# echo-gate:accept '*) continue ;;
			esac
			[ "${line#[[:space:]]local }" != "$line" ] && line="${line#*local }"
			var="$(printf '%s\n' "$line" | sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=\$[(].*.*/\1/p')"
			[ -z "$var" ] && continue

			# Collect every later line mentioning the variable; classify uses.
			local uses non_echo=0 echo_uses=0
			uses="$(awk -v v="$var" '
				index($0, "$" v) > 0 || index($0, "${" v "}") > 0 {print}
			' "$f")"

			local u
			while IFS= read -r u; do
				[ -z "$u" ] && continue
				# Skip the assignment line itself and pure comments.
				case "$u" in
				"$var"=*) continue ;;
				*"="#*) ;;
				esac
				case "$u" in
				*"="*"]"*) ;; # fallthrough handled below
				esac
				# Strip leading whitespace; echo-use if first word is echo/printf.
				local stripped="${u#"${u%%[![:space:]]*}"}"
				case "$stripped" in
				echo[\ [:space:]]* | printf[\ [:space:]]*) echo_uses=$((echo_uses + 1)) ;;
				\#*) ;; # comment mention
				*) non_echo=$((non_echo + 1)) ;;
				esac
			done <<<"$uses"

			if [ "$non_echo" -eq 0 ] && [ "$echo_uses" -gt 0 ]; then
				echo "ECHO-GATE: $f: variable '\$$var' is computed but only echoed — branch on it or delete the check"
				findings=$((findings + 1))
			fi
		done < <(grep -n '=[[:space:]]*\$[(]' "$f" | sed 's/^[0-9]*://' || true)
	done < <(find "$dir" -maxdepth 1 -name '*.sh' -type f | sort)

	if [ "$findings" -gt 0 ]; then
		echo "check-echo-gates: $findings echo-gate variable(s) found"
		return 1
	fi
	return 0
}

if [ "${1:-}" = "--self-test" ]; then
	self_test
	exit $?
fi

scan_dir "$TARGET_DIR"
