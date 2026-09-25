#!/usr/bin/env bash
# restore-depguard.sh, Self-heals the depguard allow-list block in .golangci.yml.
#
# Problem (six incidents, 2026-08-30 .. 2026-09-18): auto-commit waves keep
# deleting the whole `linters.settings.depguard` block (and re-adding gci to
# formatters, that twin is healed by check-formatters.sh). check-depguard.sh
# detects the deletion but recovery was manual every time; the block was gone
# for ~4 days twice because the gate did not run between waves. This script
# closes that gap: when the block is GONE it splices the pinned golden
# (scripts/depguard-block.golden.yml) back in place, the repair loop beats
# manual fixes against a machine-authored regression.
#
# Golden freshness rules (the golden is a restore source, never a pin):
#   - config list is a SUPERSET of golden  → golden is stale; refresh it from
#     the config silently (legitimate dependency additions evolve the config).
#   - config list is MISSING golden entries → partial deletion or a legitimate
#     removal; NOT auto-repaired. Fail loudly and tell the operator to either
#     restore the entries or update the golden if the removal was intentional.
#   - config block absent/empty → the corruption signature; restore from golden.
#
# Usage: bash scripts/restore-depguard.sh   (called from check-depguard.sh)
#        bash scripts/restore-depguard.sh --self-test
# Exit:  0 when the config is healthy or was repaired, 1 when a human must
#        decide (partial shrinkage) or the golden is missing and cannot be
#        bootstrapped.

set -euo pipefail

# RD_ROOT: internal hook for --self-test fixtures (same pattern as
# CALIB_GATE_LOADAVG_FILE — point the script at a planted tree).
if [ -n "${RD_ROOT:-}" ]; then
	cd "$RD_ROOT"
else
	cd "$(git rev-parse --show-toplevel)"
fi

CONFIG=.golangci.yml
GOLDEN=scripts/depguard-block.golden.yml

if [ "${1:-}" = "--self-test" ]; then
	RD_TMP="$(mktemp -d)"
	trap 'rm -rf "${RD_TMP:-}"' EXIT
	mkdir -p "$RD_TMP/scripts"

	cat >"$RD_TMP/.golangci.yml" <<'YML'
linters:
  settings:
    depguard:
      rules:
        main:
          deny:
            - pkg: github.com/larsartmann/go-cqrs-lite
          allow:
            - $gostd
            - example.com/alpha
            - example.com/beta
YML
	cat >"$RD_TMP/scripts/depguard-block.golden.yml" <<'YML'
    depguard:
      rules:
        main:
          deny:
            - pkg: github.com/larsartmann/go-cqrs-lite
          allow:
            - $gostd
            - example.com/alpha
            - example.com/beta
YML

	echo "━━━ restore-depguard self-test ━━━"
	failures=0

	# Leg 1: healthy config passes untouched.
	before=$(sha256sum "$RD_TMP/.golangci.yml" | cut -d' ' -f1)
	RD_ROOT="$RD_TMP" "$0" >/dev/null 2>&1 || failures=$((failures + 1))
	after=$(sha256sum "$RD_TMP/.golangci.yml" | cut -d' ' -f1)
	if [ "$before" = "$after" ] && [ "$failures" -eq 0 ]; then
		echo "  ✓ PASS: healthy config untouched"
	else
		echo "  ✗ FAIL: healthy config must pass unchanged"
		failures=$((failures + 1))
	fi

	# Leg 2: deleted block gets spliced back from the golden.
	grep -v -e 'depguard:' -e 'deny:' -e 'pkg:' -e 'allow:' -e 'gostd' -e 'example.com' \
		"$RD_TMP/.golangci.yml" >"$RD_TMP/.golangci.yml.m" && mv "$RD_TMP/.golangci.yml.m" "$RD_TMP/.golangci.yml"
	# Assert the MANGLE landed before repairing — a mutation that deletes
	# nothing would make the restore assertion below pass vacuously.
	if grep -q 'depguard:' "$RD_TMP/.golangci.yml"; then
		echo "  ✗ FAIL: fixture mutation did not delete the depguard block"
		failures=$((failures + 1))
	fi
	RD_ROOT="$RD_TMP" "$0" >/dev/null 2>&1 || failures=$((failures + 1))
	if grep -q 'example.com/alpha' "$RD_TMP/.golangci.yml"; then
		echo "  ✓ PASS: deleted depguard block restored from golden"
	else
		echo "  ✗ FAIL: block was not restored"
		failures=$((failures + 1))
	fi

	# Leg 3: partial shrinkage refuses loudly (no silent auto-repair).
	sed -i '/example.com\/beta/d' "$RD_TMP/.golangci.yml"
	if RD_ROOT="$RD_TMP" "$0" >/dev/null 2>&1; then
		echo "  ✗ FAIL: partial shrinkage must exit nonzero"
		failures=$((failures + 1))
	else
		echo "  ✓ PASS: partial shrinkage refused loudly"
	fi

	if [ "$failures" -gt 0 ]; then
		echo "self-test: ${failures} failure(s)"
		exit 1
	fi
	echo "self-test: all green"
	exit 0
fi

# Extract the full depguard block (header through last allow item), tolerating
# indentation; empty output means the block is gone.
extract_block() {
	awk '
		/^[ \t]+depguard:/ { in_block = 1 }
		in_block && started && /^[ \t]{2}[^ \t]/ { exit }
		in_block && started && /^[ \t]{4}[^ \t]+:/ && $0 !~ /depguard/ { exit }
		in_block {
			if (!started) {
				started = 1
			}

			print
		}
	' "$CONFIG"
}

# Extract only the allow entries (prefix list), skipping $gostd and comments,
# the same set the freshness comparison works on. The trailing || true keeps
# an empty/absent block from killing the script via pipefail before the
# repair branch can run (mutation-tested: the grep-exit-1 silent death).
extract_allow_items() {
	local gostd_pattern='^[$]gostd$'

	extract_block | sed -n 's/^[[:space:]]*-[[:space:]]*//p' | grep -v "$gostd_pattern" | sort -u || true
}

golden_items() {
	local gostd_pattern='^[$]gostd$'

	sed -n 's/^[[:space:]]*-[[:space:]]*//p' "$GOLDEN" | grep -v "$gostd_pattern" | sort -u || true
}

current_items=$(extract_allow_items)
spliced=0

# ── Corruption signature: block gone → restore from golden ──────────────────
if [ -z "$current_items" ]; then
	if [ ! -s "$GOLDEN" ]; then
		echo "FICTION: $CONFIG lost the depguard block and $GOLDEN is missing, cannot self-repair." >&2
		exit 1
	fi

	echo "REPAIR: depguard block missing from $CONFIG, splicing pinned golden back in"
	echo "        (the auto-commit corruption class, sixth incident 2026-09-18)."

	if grep -Eq '^[ \t]*disable:' "$CONFIG" && grep -Eq '^[ \t]*-[ \t]*depguard[ \t]*$' "$CONFIG"; then
		echo "NOTICE: depguard is explicitly disabled in $CONFIG, leaving the disable decision intact." >&2
		exit 0
	fi

	awk -v golden="$GOLDEN" '
		BEGIN {
			while ((getline line < golden) > 0) {
				n++
				g[n] = line
			}
		}
		/^linters:/ { in_linters = 1 }
		in_linters && /^  settings:/ && !done {
			print
			for (i = 1; i <= n; i++) {
				print g[i]
			}

			done = 1
			next
		}
		{ print }
	' "$CONFIG" >"$CONFIG.tmp" && mv "$CONFIG.tmp" "$CONFIG"

	current_items=$(extract_allow_items)

	if [ -z "$current_items" ]; then
		echo "FICTION: depguard block still missing after repair attempt." >&2
		exit 1
	fi

	golden_count=$(grep -c '^[[:space:]]*-' "$GOLDEN" || true)
	echo "REPAIR: restored $golden_count-entry allow list; golangci config verify runs next gate pass."
	spliced=1
fi

# ── Golden freshness ─────────────────────────────────────────────────────────
if [ ! -s "$GOLDEN" ]; then
	extract_block >"$GOLDEN"
	echo "BOOTSTRAP: created $GOLDEN from the current $CONFIG depguard block."

	exit 0
fi

missing=$(comm -23 <(golden_items) <(printf '%s\n' "$current_items" | sort -u))

if [ -n "$missing" ]; then
	echo "FICTION: depguard allow list is missing entries pinned in $GOLDEN (partial deletion or intentional removal?):" >&2
	echo "$missing" | awk '{ print "  MISSING: " $0 }' >&2
	echo "Fix: restore them in $CONFIG, or update $GOLDEN if the removal was intentional." >&2

	exit 1
fi

stale=$(comm -13 <(golden_items) <(printf '%s\n' "$current_items" | sort -u))

if [ -n "$stale" ] && [ "$spliced" -eq 0 ]; then
	extract_block >"$GOLDEN"
	echo "GOLDEN: refreshed $GOLDEN from config (allow list legitimately grew by $(echo "$stale" | wc -l) entries)."
fi
