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
#   4. REDACTION (ADR-0160): the partner slice the fixture also renders
#      carries no trace of the excluded hostile-named store — as text, link
#      or section — while out-of-scope names still degrade to plain text
#
# --self-test plants corrupted copies of a rendered fixture and verifies the
# assertions actually fail on them (the calibration-gate fault-injection
# pattern, now four corruptions). New goldens/assertions must be
# mutation-verified before they count as pins.
#
# Workdir override: CHECK_ARCHITECTURE_REFERENCE_DIR=/path (kept on failure).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="${CHECK_ARCHITECTURE_REFERENCE_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/arch-reference-validate.XXXXXX")}"
out="$work/architecture.html"
slice_out="$work/partner-slice.html"
hostile="https://internal.example.com/secret/quarterly-revenue"

cleanup_ok() { [[ -n "${CHECK_ARCHITECTURE_REFERENCE_DIR:-}" ]] || rm -rf "$work"; }

self_test() {
	local bad="$work/bad.html" bad_slice="$work/bad-slice.html"
	(cd "$root/catalog" && GOWORK=off go run ./cmd/architecture-fixture "$bad" "$bad_slice")

	local failed=0
	mutate_and_expect_fail() { # $1 = sed program, $2 = label
		sed -i "$1" "$bad"
		if assert_all "$bad" >/dev/null 2>&1; then
			echo "SELF-TEST FAIL: corruption was NOT caught: $2" >&2
			failed=1
		fi
		(cd "$root/catalog" && GOWORK=off go run ./cmd/architecture-fixture "$bad")
	}
	mutate_slice_and_expect_fail() { # $1 = sed program, $2 = label
		sed -i "$1" "$bad_slice"
		if assert_slice "$bad_slice" "$bad" >/dev/null 2>&1; then
			echo "SELF-TEST FAIL: slice corruption was NOT caught: $2" >&2
			failed=1
		fi
		(cd "$root/catalog" && GOWORK=off go run ./cmd/architecture-fixture "$bad" "$bad_slice")
	}

	# Corrupt 1: a section id renamed — anchor integrity must catch it.
	mutate_and_expect_fail 's|id="service-order-service"|id="service-order-service-x"|' "renamed section id"
	# Corrupt 2: external stylesheet injected — self-containment must catch it.
	mutate_and_expect_fail 's|</head>|<link rel="stylesheet" href="https://evil.example/x.css"></head>|' "external stylesheet"
	# Corrupt 3: a section deleted — coverage must catch it.
	mutate_and_expect_fail '/id="flow-place-order"/,/<\/section>/d' "missing flow section"
	# Corrupt 4: the excluded store's name injected into the slice — the
	# redaction assertion must catch it.
	mutate_slice_and_expect_fail "s|partner slice|partner slice $hostile|" "excluded name leaked into slice"

	if [[ "$failed" -ne 0 ]]; then
		exit 1
	fi

	echo "✓ check-architecture-reference self-test: all 4 corruptions caught"
}

# assert_all runs every assertion against $1; prints failures, returns nonzero.
assert_all() {
	local doc="$1" rc=0

	grep -q '<!doctype html>' "$doc" || {
		echo "FAIL: no doctype"
		rc=1
	}

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
		channel-orders data-store-order-db data-store-secret-store flow-place-order \
		entity-order adr-0001-events team-ordering doc-getting-started; do
		grep -q "id=\"$anchor\"" "$doc" || {
			echo "FAIL: missing section: $anchor"
			rc=1
		}
	done

	# grep -c counts LINES; the document is minified — count occurrences.
	local sections links
	sections="$(grep -o '<section class="section" id="' "$doc" | wc -l)"
	links="$(grep -o '<a class="item" href="#' "$doc" | wc -l)"
	if [[ "$sections" -lt 14 || "$links" -lt 14 ]]; then
		echo "FAIL: expected ≥14 sections and sidebar entries, got $sections/$links"
		rc=1
	fi

	return "$rc"
}

# assert_slice runs the shared artifact assertions plus the REDACTION
# contract (ADR-0160) against the partner slice: $1 = slice, $2 = whole
# reference (for the fewer-sections comparison). The slice excludes the
# hostile-named store, so its name must appear nowhere — not as text, not as
# a link, not as a section, not in the search index — while OTHER
# out-of-scope names still degrade to plain text (name-only mode).
assert_slice() {
	local doc="$1" whole="$2" rc=0

	assert_all "$whole" >/dev/null 2>&1 || {
		echo "FAIL: the whole reference regressed; the slice comparison is void"
		return 1
	}

	# doctype + self-containment + anchor integrity run via the shared
	# machinery: the cheapest honest route is to reuse assert_all's loops on
	# the slice for anchors only, inline here.
	grep -q '<!doctype html>' "$doc" || {
		echo "FAIL: slice has no doctype"
		rc=1
	}

	for forbidden in 'src="http' 'href="http' '<link ' '@import'; do
		if grep -qF "$forbidden" "$doc"; then
			echo "FAIL: slice not self-contained: found $forbidden"
			rc=1
		fi
	done

	while IFS= read -r target; do
		[[ -z "$target" ]] && continue
		if ! grep -q "id=\"$target\"" "$doc"; then
			echo "FAIL: slice link points at missing anchor: $target"
			rc=1
		fi
	done < <(grep -o 'href="#[^"]*"' "$doc" | sed 's/href="#//; s/"$//' | sort -u)

	if grep -qF "$hostile" "$doc"; then
		echo "FAIL: the excluded store's name leaked into the slice"
		rc=1
	fi

	if grep -q 'id="data-store-secret-store"' "$doc"; then
		echo "FAIL: the excluded store has a section in the slice"
		rc=1
	fi

	# Name-only liveness: another out-of-scope name still renders as plain
	# text, so the hostile absence is the exclusion working, not omit mode.
	if ! grep -qF 'Order Database' "$doc"; then
		echo "FAIL: slice stopped rendering out-of-scope names (the leak check guards nothing)"
		rc=1
	fi

	local slice_sections whole_sections
	slice_sections="$(grep -o '<section class="section" id="' "$doc" | wc -l)"
	whole_sections="$(grep -o '<section class="section" id="' "$whole" | wc -l)"
	if [[ "$slice_sections" -ge "$whole_sections" || "$slice_sections" -lt 5 ]]; then
		echo "FAIL: slice should have fewer sections than the whole (and ≥5), got $slice_sections vs $whole_sections"
		rc=1
	fi

	return "$rc"
}

if [[ "${1:-}" == "--self-test" ]]; then
	self_test
	exit 0
fi

echo "==> 1/2 rendering fixture catalog (whole + partner slice)"
(cd "$root/catalog" && GOWORK=off go run ./cmd/architecture-fixture "$out" "$slice_out")

echo "==> 2/2 asserting the artifacts ($(wc -c <"$out") + $(wc -c <"$slice_out") bytes)"
if assert_all "$out" && assert_slice "$slice_out" "$out"; then
	cleanup_ok
	echo "✓ check-architecture-reference: self-contained, anchors resolve, 14 sections, slice redacts"
else
	echo "FAIL: architecture reference failed validation (artifacts kept: $out $slice_out)" >&2
	exit 1
fi
