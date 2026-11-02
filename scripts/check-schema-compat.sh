#!/usr/bin/env bash
# Schema-compatibility gate: turns docs/SCHEMA-COMPATIBILITY.md into a build
# failure. Two questions, both answered by catalog/cmd/catalog-compat:
#
#   check     — did a payload shape change breakingly without the message
#               version moving? (the policy's "write an op, bump the version"
#               rows; the ladder itself stays cqrs-lint E021/E022's job)
#   validate  — the cross-resource invariants no other function owns
#               (direction legality, channel and data-store resolution) plus
#               Catalog.ValidateCoeffects, composed verbatim
#
# A finding fails the gate only when it is BLOCKING. A breaking shape change
# WITH a version bump is reported but passes: whether the upcast op actually
# exists is a declaration question, and schema.EventSchema answers it.
#
# Usage:
#   bash scripts/check-schema-compat.sh --catalog FILE [--baseline FILE]
#   bash scripts/check-schema-compat.sh --update-baseline --catalog FILE [--baseline FILE]
#   bash scripts/check-schema-compat.sh --self-test
#
# Exit codes: 0 clean, 1 gate failure, 2 usage or I/O error.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
self_test_fixtures="$repo_root/scripts/testdata/schema-compat"

usage() {
	cat >&2 <<'USAGE'
usage: check-schema-compat.sh --catalog FILE [--baseline FILE]
       check-schema-compat.sh --update-baseline --catalog FILE [--baseline FILE]
       check-schema-compat.sh --self-test
USAGE
}

# build_cli <output-path> — one build per invocation; the CLI lives in the
# catalog module, so GOWORK=off keeps the resolution identical to a consumer's.
build_cli() {
	local out="$1"
	( cd "$repo_root/catalog" && GOWORK=off go build -o "$out" ./cmd/catalog-compat )
}

# self_test runs the mutation legs. Every leg asserts a DIFFERENT failure mode,
# so a gate that silently stops detecting anything cannot pass.
self_test() {
	local tmp fail=0 bin
	tmp="$(mktemp -d)"
	# Single quotes would expand only at exit, when the local is gone; embed
	# the path now instead.
	trap "rm -rf '$tmp'" EXIT

	bin="$tmp/catalog-compat"
	build_cli "$bin"

	"$bin" baseline --catalog "$self_test_fixtures/old.json" --out "$tmp/baseline.json" >/dev/null

	# Leg 1: a removed field with an unchanged version must fail, and the
	# stderr line must name the message so a reader knows what to fix.
	if "$bin" check --catalog "$self_test_fixtures/breaking.json" \
		--baseline "$tmp/baseline.json" >"$tmp/out1" 2>"$tmp/err1"; then
		echo "::error::self-test leg 1: a breaking change with no version bump passed" >&2
		fail=1
	elif ! grep -q '::error::order_placed' "$tmp/err1"; then
		echo "::error::self-test leg 1: the failure did not name the message" >&2
		fail=1
	fi

	# Leg 2: an added optional field must pass with the same baseline.
	if ! "$bin" check --catalog "$self_test_fixtures/additive.json" \
		--baseline "$tmp/baseline.json" >"$tmp/out2" 2>"$tmp/err2"; then
		echo "::error::self-test leg 2: an additive change failed: $(cat "$tmp/err2")" >&2
		fail=1
	fi

	# Leg 3: a missing baseline must fail as a usage error naming the file.
	rc=0
	"$bin" check --catalog "$self_test_fixtures/additive.json" \
		--baseline "$tmp/absent.json" >"$tmp/out3" 2>"$tmp/err3" || rc=$?
	if [ "$rc" -ne 2 ]; then
		echo "::error::self-test leg 3: a missing baseline exited $rc, want 2" >&2
		fail=1
	elif ! grep -q 'absent.json' "$tmp/err3"; then
		echo "::error::self-test leg 3: the error did not name the baseline file" >&2
		fail=1
	fi

	# Leg 4: --update-baseline must make the same comparison green. This is
	# the mutation-tested golden: leg 1 red, regenerate, green.
	"$bin" baseline --catalog "$self_test_fixtures/breaking.json" --out "$tmp/baseline.json" >/dev/null
	if ! "$bin" check --catalog "$self_test_fixtures/breaking.json" \
		--baseline "$tmp/baseline.json" >"$tmp/out4" 2>"$tmp/err4"; then
		echo "::error::self-test leg 4: a refreshed baseline did not pass" >&2
		fail=1
	fi

	# Leg 5: validate must accept a clean catalog and reject a dangling
	# channel reference, so the invariant half of the gate is proven too.
	if ! "$bin" validate --catalog "$self_test_fixtures/old.json" >"$tmp/out5" 2>"$tmp/err5"; then
		echo "::error::self-test leg 5: a clean catalog failed validation: $(cat "$tmp/err5")" >&2
		fail=1
	elif "$bin" validate --catalog "$self_test_fixtures/dangling-channel.json" >"$tmp/out6" 2>"$tmp/err6"; then
		echo "::error::self-test leg 5: a dangling channel reference passed validation" >&2
		fail=1
	elif ! grep -q 'ghost_channel' "$tmp/out6"; then
		echo "::error::self-test leg 5: the violation did not name the dangling channel" >&2
		fail=1
	fi

	if [ "$fail" -eq 0 ]; then
		echo "✓ schema-compat self-test: 6/6 mutation legs pass"
	fi

	return "$fail"
}

if [ "${1:-}" = "--self-test" ]; then
	self_test
	exit $?
fi

catalog_path=""
baseline_path="compat-baseline.json"
update_baseline=0

while [ $# -gt 0 ]; do
	case "$1" in
		--catalog) catalog_path="${2:-}"; shift 2 ;;
		--baseline) baseline_path="${2:-}"; shift 2 ;;
		--update-baseline) update_baseline=1; shift ;;
		-h|--help) usage; exit 0 ;;
		*) echo "::error::unknown argument: $1" >&2; usage; exit 2 ;;
	esac
done

if [ -z "$catalog_path" ]; then
	echo "::error::--catalog is required" >&2
	usage
	exit 2
fi

tmp_bin="$(mktemp)"
trap 'rm -f "$tmp_bin"' EXIT
build_cli "$tmp_bin"

if [ "$update_baseline" -eq 1 ]; then
	"$tmp_bin" baseline --catalog "$catalog_path" --out "$baseline_path"
	echo "✓ schema-compat: baseline refreshed at $baseline_path"
	exit 0
fi

status=0

"$tmp_bin" check --catalog "$catalog_path" --baseline "$baseline_path" || status=$?
"$tmp_bin" validate --catalog "$catalog_path" || status=$?

if [ "$status" -ne 0 ]; then
	echo "::error::schema-compatibility gate failed (see the lines above)" >&2
	exit 1
fi

echo "✓ schema-compat: policy and invariants hold"
