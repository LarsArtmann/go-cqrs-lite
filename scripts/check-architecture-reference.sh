#!/usr/bin/env bash
# check-architecture-reference: render-validate the catalog/architecture HTML
# export outside the go test process.
#
# Renders a small but complete fixture catalog via
# catalog/cmd/architecture-fixture, then asserts the properties the package
# promises its consumers:
#   1. self-containment (no external stylesheet/script/font references)
#   2. anchor integrity (every href="#x" and search-index hit has an id="x")
#   3. expected section coverage (one section per fixture resource kind)
#
# --self-test plants corrupted copies of a rendered fixture and verifies the
# assertions actually fail on them (the calibration-gate fault-injection
# pattern). New goldens/assertions must be mutation-verified before they
# count as pins.
#
# Workdir override: CHECK_ARCHITECTURE_REFERENCE_DIR=/path (kept on failure).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="${CHECK_ARCHITECTURE_REFERENCE_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/arch-reference-validate.XXXXXX")}"
out="$work/architecture.html"

cleanup_ok() { [[ -n "${CHECK_ARCHITECTURE_REFERENCE_DIR:-}" ]] || rm -rf "$work"; }

self_test() {
	local bad="$work/bad.html"
	(cd "$root/catalog" && GOWORK=off go run ./cmd/architecture-fixture "$bad")

	local failed=0
	mutate_and_expect_fail() { # $1 = sed program, $2 = label
		sed -i "$1" "$bad"
		if assert_all "$bad" >/dev/null 2>&1; then
			echo "SELF-TEST FAIL: corruption was NOT caught: $2" >&2
			failed=1
		fi
		(cd "$root/catalog" && GOWORK=off go run ./cmd/architecture-fixture "$bad")
	}

	# Corrupt 1: a section id renamed — anchor integrity must catch it.
	mutate_and_expect_fail 's|id="service-order-service"|id="service-order-service-x"|' "renamed section id"
	# Corrupt 2: external stylesheet injected — self-containment must catch it.
	mutate_and_expect_fail 's|</head>|<link rel="stylesheet" href="https://evil.example/x.css"></head>|' "external stylesheet"
	# Corrupt 3: a section deleted — coverage must catch it.
	mutate_and_expect_fail '/id="flow-place-order"/,/<\/section>/d' "missing flow section"

	if [[ "$failed" -ne 0 ]]; then
		exit 1
	fi

	echo "✓ check-architecture-reference self-test: all corruptions caught"
}

# assert_all runs every assertion against $1; prints failures, returns nonzero.
assert_all() {
	local doc="$1" rc=0

	grep -q '<!doctype html>' "$doc" || { echo "FAIL: no doctype"; rc=1; }

	for forbidden in 'src="http' 'href="http' '<link ' '@import'; do
		if grep -qF "$forbidden" "$doc"; then
			echo "FAIL: not self-contained: found $forbidden"
			rc=1
		fi
	done

	# Anchor integrity: every href="#x" target must exist as an id="x".
	while IFS= read -r target; do
		[[ -z "$target" ]] && continue
		if ! grep -q "id=\"$target\"" "$doc"; then
			echo "FAIL: link points at missing anchor: $target"
			rc=1
		fi
	done < <(grep -o 'href="#[^"]*"' "$doc" | sed 's/href="#//; s/"$//' | sort -u)

	# Search-index anchors must resolve too.
	while IFS= read -r target; do
		if ! grep -q "id=\"$target\"" "$doc"; then
			echo "FAIL: search hit points at missing anchor: $target"
			rc=1
		fi
	done < <(grep -o '"h":"[^"]*"' "$doc" | sed 's/"h":"//; s/"$//' | sort -u)

	# Section coverage: one section per fixture resource.
	for anchor in overview domain-ordering service-order-service \
		command-create-order event-order-created query-get-order \
		channel-orders data-store-order-db flow-place-order \
		entity-order adr-0001-events team-ordering doc-getting-started; do
		grep -q "id=\"$anchor\"" "$doc" || { echo "FAIL: missing section: $anchor"; rc=1; }
	done

	# grep -c counts LINES; the document is minified — count occurrences.
	local sections links
	sections="$(grep -o '<section class="section" id="' "$doc" | wc -l)"
	links="$(grep -o '<a class="item" href="#' "$doc" | wc -l)"
	if [[ "$sections" -lt 13 || "$links" -lt 13 ]]; then
		echo "FAIL: expected ≥13 sections and sidebar entries, got $sections/$links"
		rc=1
	fi

	return "$rc"
}

if [[ "${1:-}" == "--self-test" ]]; then
	self_test
	exit 0
fi

echo "==> 1/2 rendering fixture catalog"
(cd "$root/catalog" && GOWORK=off go run ./cmd/architecture-fixture "$out")

echo "==> 2/2 asserting the artifact ($(wc -c <"$out") bytes)"
if assert_all "$out"; then
	cleanup_ok
	echo "✓ check-architecture-reference: self-contained, anchors resolve, 13 sections"
else
	echo "FAIL: architecture reference failed validation (artifact kept: $out)" >&2
	exit 1
fi
