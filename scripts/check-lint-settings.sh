#!/usr/bin/env bash
# check-lint-settings.sh — settings-canaries for .golangci.yml: pin the
# linter-settings KEY-SET and assert critical settings survive any reformat
# or re-pin, so a settings DROP cannot hide behind a legitimate-looking
# config-verify + re-pinned hash.
#
# Problem (2026-10-08..10, incident f5 of the 13-50-10-10 report): the whole
# .golangci.yml hash tripwire BLESSED the cyclop/errcheck settings drop — its
# own --update instructions re-pinned the regressed config, and config-verify
# passed because the file remained schema-valid. A key-set pin plus
# value-level canaries fail on the DROP itself, even after a re-pin.
#
# Usage: bash scripts/check-lint-settings.sh
# Exit:  0 all canaries present, 1 any canary missing/changed

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

CONFIG=.golangci.yml

fail=0

# --- Canary 1: the linter-settings key set (a dropped settings block is the
# regression class; new keys are ALLOWED — additions-only pin).
settings_keys() {
	awk '/^  settings:$/ { inside=1; next } inside && /^  [a-z][a-z0-9_-]*:$/ { inside=0 } inside && /^ {4}[a-z][a-z0-9_-]*:/ { print }' "$CONFIG" |
		sed 's/^ *//;s/://'
}

for want in cyclop errcheck; do
	if ! settings_keys | grep -qx "$want"; then
		echo "✗ lint-settings: settings.$want block missing (dropped settings class)" >&2
		fail=1
	fi
done

# --- Canary 2: value-level pins (the two settings whose loss motivated this).
pin_value() { # $1 = yq-style path expression via awk on the block, $2 = want, $3 = label
	local block="$1" want="$2" label="$3" got
	got="$(awk -v key="$block" '
		$0 ~ "^    " key ":" { inside=1; next }
		inside && /^    [a-z]/ { inside=0 }
		inside && $1 == "max-complexity:" { print $2; exit }
	' "$CONFIG")"
	if [ "$got" != "$want" ]; then
		echo "✗ lint-settings: $label = '${got:-<missing>}', want '$want'" >&2
		fail=1
	fi
}

pin_value cyclop 25 "cyclop.max-complexity"

if ! awk '
	/^    errcheck:$/ { inside=1; next }
	inside && /^    [a-z]/ { inside=0 }
	inside && $1 == "exclude-functions:" { found=1 }
	END { exit !found }
' "$CONFIG"; then
	echo "✗ lint-settings: errcheck.exclude-functions list missing" >&2
	fail=1
fi

if [ "$fail" -eq 0 ]; then
	echo "✓ lint-settings: settings key-set + value canaries intact (cyclop.max-complexity=25, errcheck.exclude-functions present)"
fi

exit "$fail"
