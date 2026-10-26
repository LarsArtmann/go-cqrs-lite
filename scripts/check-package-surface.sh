#!/usr/bin/env bash
# check-package-surface.sh — per-package API-surface ratchet.
#
# WHY: the 2026-10-24 architecture review's #1 finding was that ONE package
# (`metaengine`) held 829 exported symbols and 167 production files: 13% of the
# library API behind a single namespace. The 350-line file ratchet bounds FILES,
# never packages, so package-level concentration was invisible to every gate.
# This gate adds the missing dimension: exported symbols and production files
# per package.
#
# POLICY (ratchet, deliberately mirroring check-file-size.sh):
#   - a package NOT in the baseline may not exceed LIMIT_SYMBOLS exported
#     symbols nor LIMIT_FILES production files — new packages must be born
#     under both ceilings;
#   - a baselined package may not GROW beyond its recorded counts; shrinking is
#     always allowed (ratchet down), and an entry that falls under both limits
#     should be removed with --update-baseline;
#   - --update-baseline re-pins the snapshot. Structural shifts only, and the
#     resulting baseline diff is reviewed like a golden.
#
# SYMBOL SOURCE: docs/api_surface.txt — the cmd/api-stability golden for the
# modules it tracks. That file is itself gated (api-stability meta-tests,
# #check-api-stability, CI "API stability check"), so the counts cannot drift
# silently. Packages from untracked modules (some cmd/, example/, test-helper
# modules) contribute ZERO symbols here; their file counts are still ratcheted
# and the summary prints how many on-disk packages are untracked so the blind
# spot stays visible.
#
# Usage:
#   bash scripts/check-package-surface.sh                  # gate (default)
#   bash scripts/check-package-surface.sh --report         # top-N markdown table
#   bash scripts/check-package-surface.sh --tsv            # full "<dir> TAB <symbols> TAB <files>" map
#   bash scripts/check-package-surface.sh --json           # the same map as JSON
#   bash scripts/check-package-surface.sh --update-baseline
#   bash scripts/check-package-surface.sh --self-test      # mutation proof
#
# Env overrides (used by --self-test and by fixtures, never in normal runs):
#   PKGSURF_ROOT, PKGSURF_API_FILE, PKGSURF_BASELINE
set -euo pipefail

ROOT="${PKGSURF_ROOT:-$(git rev-parse --show-toplevel)}"
cd "$ROOT"

API_FILE="${PKGSURF_API_FILE:-docs/api_surface.txt}"
BASELINE="${PKGSURF_BASELINE:-scripts/package-surface-baseline.txt}"

# 150 symbols / 40 files: the review's proposed v5 ceiling was ~300 per package;
# 150 keeps new packages honest without baselining the six historical offenders
# twice over. Raise only with a review, never to green a diff.
LIMIT_SYMBOLS="${PKGSURF_LIMIT_SYMBOLS:-150}"
LIMIT_FILES="${PKGSURF_LIMIT_FILES:-40}"

# Single scratch directory, removed on exit. Globals (not function locals)
# because an EXIT trap cannot see a local that already went out of scope — the
# self-test hit exactly that as "tmp: unbound variable".
WORK=""
cleanup() {
	if [ -n "$WORK" ]; then rm -rf "$WORK"; fi
}
trap cleanup EXIT

find_go_files() {
	find . -name '*.go' -not -name '*_test.go' \
		-not -name '*.pb.go' \
		-not -name '*.gen.go' \
		-not -name '*_templ.go' \
		-not -path './vendor/*' \
		-not -path './.git/*' \
		-not -path '*/testdata/*' \
		-not -path '*/node_modules/*' \
		-not -path '*/internal/cattest/*'
}

# symbol_counts: "<pkgdir><TAB><exported-symbols>" for every package in the
# api-stability golden. The golden line shape is "<pkgdir>/<kind> <Name>".
symbol_counts() {
	awk '
		BEGIN { split("func method struct type const var interface", k, " "); for (i in k) valid[k[i]] = 1 }
		{
			sp = index($0, " ")
			if (sp == 0) next
			head = substr($0, 1, sp - 1)
			n = split(head, parts, "/")
			if (n < 2) next                      # "<kind> <Name>" at a module root
			kind = parts[n]
			if (!(kind in valid)) next           # defensive: unknown record shape
			pkg = parts[1]
			for (j = 2; j < n; j++) pkg = pkg "/" parts[j]
			seen[pkg]++
		}
		END { for (p in seen) printf "%s\t%d\n", p, seen[p] }
	' "$API_FILE" | sort
}

# file_counts: "<pkgdir><TAB><production .go files>" straight from the tree.
file_counts() {
	find_go_files | sed 's|^\./||' | awk -F/ '
		NF == 1 { next }                          # a .go file at the repo root
		{
			dir = $1
			for (i = 2; i < NF; i++) dir = dir "/" $i
			seen[dir]++
		}
		END { for (d in seen) printf "%s\t%d\n", d, seen[d] }
	' | sort
}

write_baseline() {
	WORK="$(mktemp -d)"
	local syms="$WORK/syms" files="$WORK/files"
	symbol_counts >"$syms"
	file_counts >"$files"
	{
		echo "# Package-surface ratchet baseline (see scripts/check-package-surface.sh)."
		echo "# Format: <exported-symbols><TAB><production-files><TAB><package-dir>, symbols desc."
		echo "# Ceilings for NEW packages: $LIMIT_SYMBOLS symbols / $LIMIT_FILES files."
		echo "# Remove an entry once the package sits under BOTH ceilings."
		echo "# Regenerate: bash scripts/check-package-surface.sh --update-baseline"
		awk -F'\t' -v OFS='\t' -v ls="$LIMIT_SYMBOLS" -v lf="$LIMIT_FILES" '
			FILENAME == ARGV[1] { s[$1] = $2; next }
			{ f[$1] = $2; all[$1] = 1 }
			END {
				for (p in s) all[p] = 1
				for (p in all) {
					ns = (p in s) ? s[p] : 0
					nf = (p in f) ? f[p] : 0
					if (ns > ls || nf > lf) print ns, nf, p
				}
			}
		' "$syms" "$files" | sort -t$'\t' -k1,1nr -k3,3
	} >"$BASELINE"
	echo "Baseline written: $(grep -cv '^#' "$BASELINE") offender(s)."
}

tsv_dump() {
	WORK="$(mktemp -d)"
	local syms="$WORK/syms" files="$WORK/files"
	symbol_counts >"$syms"
	file_counts >"$files"
	join -t$'\t' -a1 -a2 -e0 -o '0,1.2,2.2' \
		<(sort -t$'\t' -k1,1 "$syms") \
		<(sort -t$'\t' -k1,1 "$files") |
		sort -t$'\t' -k1,1
}

json_dump() {
	WORK="$(mktemp -d)"
	local syms="$WORK/syms" files="$WORK/files"
	symbol_counts >"$syms"
	file_counts >"$files"
	join -t$'\t' -a1 -a2 -e0 -o '0,1.2,2.2' \
		<(sort -t$'\t' -k1,1 "$syms") \
		<(sort -t$'\t' -k1,1 "$files") |
		sort -t$'\t' -k1,1 |
		awk -F'\t' 'BEGIN { printf "{\"packages\":{" } {
			if (NR > 1) printf ","
			printf "\"%s\":{\"symbols\":%d,\"files\":%d}", $1, $2, $3
		} END { printf "}}\n" }'
}

report() {
	WORK="$(mktemp -d)"
	local syms="$WORK/syms" files="$WORK/files"
	symbol_counts >"$syms"
	file_counts >"$files"
	echo "| Package | Exported symbols | Production files |"
	echo "| ------- | ---------------: | ---------------: |"
	join -t$'\t' -a1 -e0 -o '0,1.2,2.2' \
		<(sort -t$'\t' -k1,1 "$syms") \
		<(sort -t$'\t' -k1,1 "$files") |
		sort -t$'\t' -k2,2nr |
		awk -F'\t' -v OFS=' | ' 'NR <= 20 { print "| " $1, $2, $3 " |" }'
	echo ""
	echo "Ceilings for new packages: $LIMIT_SYMBOLS symbols / $LIMIT_FILES files."
}

gate() {
	if [ ! -f "$API_FILE" ]; then
		echo "::error::$API_FILE missing (regenerate: cd cmd/api-stability && GOWORK=off go run . --update)."
		exit 1
	fi
	if [ ! -f "$BASELINE" ]; then
		echo "::error::$BASELINE missing — run 'bash scripts/check-package-surface.sh --update-baseline' and commit it."
		exit 1
	fi

	declare -A base_sym=() base_fil=() sym=() fil=()
	local ns nf dir
	while IFS=$'\t' read -r ns nf dir; do
		case "${ns:-}" in '' | \#*) continue ;; esac
		base_sym["$dir"]="$ns"
		base_fil["$dir"]="$nf"
	done <"$BASELINE"
	while IFS=$'\t' read -r dir ns; do
		[ -n "${dir:-}" ] && sym["$dir"]="$ns"
	done < <(symbol_counts)
	while IFS=$'\t' read -r dir nf; do
		[ -n "${dir:-}" ] && fil["$dir"]="$nf"
	done < <(file_counts)

	local failed=false stale=()
	while IFS= read -r dir; do
		ns="${sym[$dir]:-0}"
		nf="${fil[$dir]:-0}"
		if [ -z "${base_sym[$dir]:-}" ]; then
			if [ "$ns" -gt "$LIMIT_SYMBOLS" ]; then
				echo "::error::${dir}: $ns exported symbols (max $LIMIT_SYMBOLS) — a NEW oversized package; split it."
				failed=true
			fi
			if [ "$nf" -gt "$LIMIT_FILES" ]; then
				echo "::error::${dir}: $nf production files (max $LIMIT_FILES) — a NEW oversized package; split it."
				failed=true
			fi
			continue
		fi
		if [ "$ns" -gt "${base_sym[$dir]}" ]; then
			echo "::error::${dir}: exported symbols grew ${base_sym[$dir]} → $ns (ratchet: baselined packages may only shrink; split the package, or re-pin with --update-baseline after review)."
			failed=true
		fi
		if [ "$nf" -gt "${base_fil[$dir]}" ]; then
			echo "::error::${dir}: production files grew ${base_fil[$dir]} → $nf (ratchet: baselined packages may only shrink; split the package, or re-pin with --update-baseline after review)."
			failed=true
		fi
		if [ "$ns" -le "$LIMIT_SYMBOLS" ] && [ "$nf" -le "$LIMIT_FILES" ]; then
			stale+=("$dir")
		fi
	done < <(printf '%s\n' "${!sym[@]}" "${!fil[@]}" | sort -u)

	for dir in "${!base_sym[@]}"; do
		if [ -z "${sym[$dir]:-}" ] && [ -z "${fil[$dir]:-}" ]; then
			stale+=("$dir (gone)")
		fi
	done

	if [ "${#stale[@]}" -gt 0 ]; then
		echo "note: ${#stale[@]} baselined package(s) now fit under the ceilings — drop them with --update-baseline:"
		printf '  %s\n' "${stale[@]}"
	fi

	local tracked
	tracked="$(printf '%s\n' "${!sym[@]}" | wc -l)"
	local ondisk
	ondisk="$(printf '%s\n' "${!fil[@]}" | wc -l)"
	echo "coverage: ${tracked} package(s) have an api-stability golden row; ${ondisk} package(s) exist on disk."

	if [ "$failed" = true ]; then
		echo "::error::Package-surface ratchet violations detected (see above)."
		exit 1
	fi
	echo "✓ package-surface ratchet: no new oversized package, no growth ($(grep -cv '^#' "$BASELINE") baselined historical offender(s), new-package ceilings ≤ $LIMIT_SYMBOLS symbols / $LIMIT_FILES files)."
}

self_test() {
	local fixture="scripts/testdata/package-surface"
	local rc=0
	WORK="$(mktemp -d)"
	local tmp="$WORK"

	run_case() { # name expected_rc
		local name="$1" want="$2" got=0
		PKGSURF_ROOT="$tmp/$name" PKGSURF_API_FILE=api_surface.txt \
			PKGSURF_BASELINE=baseline.txt \
			bash "${BASH_SOURCE[0]}" >"$tmp/$name.log" 2>&1 || got=$?
		if [ "$got" -ne "$want" ]; then
			echo "FAIL: $name — expected rc=$want, got rc=$got"
			cat "$tmp/$name.log"
			rc=1
		else
			echo "ok: $name (rc=$got)"
		fi
	}

	cp -a "$fixture" "$tmp/clean"
	cp -a "$fixture" "$tmp/newgod"
	cp -a "$fixture" "$tmp/growth"
	cp -a "$fixture" "$tmp/shrink"

	# A brand-new package far over the ceilings must fail.
	for _ in $(seq 1 400); do echo "newgod/func Exported$_"; done >>"$tmp/newgod/api_surface.txt"
	# One extra symbol on a baselined package must fail.
	echo "big/func ExtraSymbol" >>"$tmp/growth/api_surface.txt"
	# Shrinking a baselined package is allowed.
	sed -i '/^big\/func Symbol/d' "$tmp/shrink/api_surface.txt"

	run_case clean 0
	run_case newgod 1
	run_case growth 1
	run_case shrink 0

	if ! grep -q 'NEW oversized package' "$tmp/newgod.log"; then
		echo "FAIL: newgod did not report a NEW oversized package"
		cat "$tmp/newgod.log"
		rc=1
	fi
	if ! grep -q 'may only shrink' "$tmp/growth.log"; then
		echo "FAIL: growth did not report the ratchet"
		cat "$tmp/growth.log"
		rc=1
	fi

	if [ "$rc" -eq 0 ]; then
		echo "✓ package-surface self-test: 4/4 mutation cases behaved as specified"
	fi
	return "$rc"
}

case "${1:-}" in
--update-baseline)
	write_baseline
	;;
--report)
	report
	;;
--json)
	json_dump
	;;
--tsv)
	tsv_dump
	;;
--self-test)
	self_test
	;;
"")
	gate
	;;
*)
	echo "unknown argument: $1 (expected --report, --tsv, --json, --update-baseline or --self-test)" >&2
	exit 2
	;;
esac
