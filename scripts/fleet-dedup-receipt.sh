#!/usr/bin/env bash
# fleet-dedup-receipt.sh — per-repo-cwd idempotency receipts for the dedup fleet.
#
# The 2026-11-20 fleet-baseline lesson (docs/evidence/2026-11-20_fleet-idempotency-baseline.md §1.1):
# .cqrs-lint.json binds from the PROCESS CWD. A receipt captured from any other directory
# silently applies that directory's preset (the library root's `preset: library` disables
# F030/V007) and the numbers lie. This script is the repeatable honest shape: for each fleet
# repo it cds INTO the repo, then captures the scorecard idempotency row and the F030 count.
#
# Output per repo (one line each):
#   <repo>\tscorecard: <evidence | MISSING>\tF030: <n>
# Full scorecard JSON and F030 text findings are written to build/fleet-receipts/<ts>/<repo>/.
#
# Usage:
#   scripts/fleet-dedup-receipt.sh [repo-dir ...]     # default: the nine surveyed repos
#   CQRS_LINT=/path/to/binary scripts/fleet-dedup-receipt.sh
#
# Environment: GOWORK is forced off per repo run (per-module standalone loads, the same
# shape the baseline used). Exits non-zero if a repo directory does not exist.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/go-env.sh
source "${REPO_ROOT}/scripts/go-env.sh"

DEFAULT_REPOS=(
	"$HOME/projects/InboxClean"
	"$HOME/projects/browser-history"
	"$HOME/projects/browser-history-precheck"
	"$HOME/projects/games/KeyCountdown"
	"$HOME/projects/DiscordSync"
	"$HOME/projects/games/SEC"
	"$HOME/projects/github-local-sync"
	"$HOME/projects/Zlota44"
	"$HOME/projects/games/SwettySwipperWeb"
)

REPOS=("$@")
if [ "${#REPOS[@]}" -eq 0 ]; then
	REPOS=("${DEFAULT_REPOS[@]}")
fi

if [ -z "${CQRS_LINT:-}" ]; then
	echo "building cqrs-lint from the working tree..." >&2
	CQRS_LINT="$(mktemp -d)/cqrs-lint"
	(cd "${REPO_ROOT}/cmd/cqrs-lint/v4" && GOWORK=off go build -o "${CQRS_LINT}" .)
fi

TS="$(date +%Y%m%d-%H%M%S)"
OUT_DIR="${REPO_ROOT}/build/fleet-receipts/${TS}"
mkdir -p "${OUT_DIR}"

for repo in "${REPOS[@]}"; do
	name="$(basename "${repo}")"
	if [ ! -d "${repo}" ]; then
		echo "ERROR: ${repo} is not a directory" >&2
		exit 1
	fi

	receipt_dir="${OUT_DIR}/${name}"
	mkdir -p "${receipt_dir}"

	# PRESET BINDS FROM CWD — every command below must run from inside the repo.
	(
		cd "${repo}"
		GOWORK=off "${CQRS_LINT}" scorecard --path . --format json \
			>"${receipt_dir}/scorecard.json" 2>"${receipt_dir}/scorecard.stderr"
		GOWORK=off "${CQRS_LINT}" --path . --format text \
			>"${receipt_dir}/lint.txt" 2>"${receipt_dir}/lint.stderr"
	)

	evidence="$(jq -r '[.used[] | select(.key == "idempotency")][0].evidence // "MISSING"' \
		"${receipt_dir}/scorecard.json")"
	f030="$(grep -c '\[F030\]' "${receipt_dir}/lint.txt" || true)"

	head="$(git -C "${repo}" rev-parse --short HEAD)"
	printf '%s\tHEAD %s\tscorecard: %s\tF030: %s\n' "${name}" "${head}" "${evidence}" "${f030}"
done

echo "receipts: ${OUT_DIR}" >&2
