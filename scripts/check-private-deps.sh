#!/usr/bin/env bash
# check-private-deps.sh — no workspace go.mod may require a dependency the
# Go module proxy cannot serve.
#
# A require on a PRIVATE repository breaks every workspace-wide go command
# on machines without credentials (CI module loading exits 128 at the auth
# prompt) and breaks consumers who tidy against our examples. The taskmanager
# class: go-must is private past v0.1.0, so examples carry local helper
# copies instead of requiring it.
#
# Checks (fast, offline):
#   1. No go.mod requires a repo on the KNOWN-PRIVATE blocklist (go-must).
#   2. example/* go.mods require only AUDITED-PUBLIC larsartmann repos
#      (or non-larsartmann deps). The audit was run live 2026-09-13 via
#      `gh repo view <repo> --json visibility`: cmdguard, go-atomic-write,
#      go-branded-id, go-codec, go-error-family, go-finding,
#      go-flightrecorder, go-idempotency, go-ndjson, go-output, go-retry,
#      go-sse, samber-do-auditlog, templ-components — all PUBLIC.
#   3. No go.mod requires ANY larsartmann repo absent from the audited set
#      (a NEW sibling must be audited before it enters a go.mod — run
#      --audit to check its visibility, then extend the allowlist).
#
#   --audit  re-runs the live GitHub visibility audit for every required
#            repo (needs gh + network; not part of the fast gate).
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

KNOWN_PRIVATE=(
	github.com/larsartmann/go-must
)

AUDITED_PUBLIC=(
	github.com/larsartmann/cmdguard
	github.com/larsartmann/go-atomic-write
	github.com/larsartmann/go-branded-id
	github.com/larsartmann/go-codec
	github.com/larsartmann/go-cqrs-lite
	github.com/larsartmann/go-error-family
	github.com/larsartmann/go-finding
	github.com/larsartmann/go-flightrecorder
	github.com/larsartmann/go-idempotency
	github.com/larsartmann/go-ndjson
	github.com/larsartmann/go-output
	github.com/larsartmann/go-retry
	github.com/larsartmann/go-sse
	github.com/larsartmann/samber-do-auditlog
	github.com/larsartmann/templ-components
)

violations=0
checked=0

audit_live() {
	local repos
	repos="$(grep -rh --include=go.mod -o 'github.com/larsartmann/[a-zA-Z0-9_-]*' . |
		sort -u | grep -v 'larsartmann/go-cqrs-lite$')"
	for repo in $repos; do
		local short="${repo#github.com/larsartmann/}"
		local vis
		vis="$(gh repo view "larsartmann/${short}" --json visibility --jq .visibility 2>/dev/null || echo UNREACHABLE)"
		printf '%-24s %s\n' "$short" "$vis"
		if [ "$vis" != "PUBLIC" ]; then
			echo "  ^ NOT public — extend KNOWN_PRIVATE or make the repo public"
			violations=$((violations + 1))
		fi
	done
}

if [ "${1:-}" = "--audit" ]; then
	audit_live
	if [ "$violations" -gt 0 ]; then
		exit 1
	fi
	echo "All required sibling repos are public."
	exit 0
fi

while IFS= read -r gomod; do
	while IFS= read -r repo; do
		repo="${repo%% *}"
		[ -z "$repo" ] && continue
		checked=$((checked + 1))

		for private in "${KNOWN_PRIVATE[@]}"; do
			if [ "$repo" = "$private" ]; then
				echo "FAIL  ${gomod}: requires known-private ${repo}"
				echo "      (proxy-invisible past its last public tag; copy the helpers locally)"
				violations=$((violations + 1))
			fi
		done

		case "$repo" in
		github.com/larsartmann/*)
			audited=0
			for public in "${AUDITED_PUBLIC[@]}"; do
				if [ "$repo" = "$public" ]; then
					audited=1

					break
				fi
			done
			if [ "$audited" -eq 0 ]; then
				echo "FAIL  ${gomod}: requires unaudited sibling ${repo}"
				echo "      run '$0 --audit' to check its visibility, then extend AUDITED_PUBLIC"
				violations=$((violations + 1))
			fi
			;;
		esac

		case "$gomod" in
		example/*)
			case "$repo" in
			github.com/larsartmann/*)
				audited=0
				for public in "${AUDITED_PUBLIC[@]}"; do
					if [ "$repo" = "$public" ]; then
						audited=1

						break
					fi
				done
				if [ "$audited" -eq 0 ]; then
					echo "FAIL  ${gomod}: example requires non-public ${repo}"
					violations=$((violations + 1))
				fi
				;;
			esac
			;;
		esac
	done < <(grep -o 'github.com/larsartmann/[a-zA-Z0-9._-]*' "$gomod" | sort -u)
done < <(find . -name go.mod -not -path './vendor/*' -not -path './.git/*')

echo ""
echo "check-private-deps: ${checked} larsartmann require(s) across the workspace."
if [ "$violations" -gt 0 ]; then
	echo "${violations} private/unaudited dependency violation(s). A private require"
	echo "breaks anonymous consumers and credential-less CI module loading."
	exit 1
fi
