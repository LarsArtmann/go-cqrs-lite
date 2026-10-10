#!/usr/bin/env bash
# dupe-signal — branching-flow `dupe` minus the ADR-0152 mirror noise.
#
# Why this exists: the dupe analyzer has no working suppression surface for
# duplicate-type findings — `//nolint:branching-flow` directives are ignored
# by dupe (verified 2026-10-10 against the installed 0.6.4 binary AND the
# tool repo's local master), and the v4↔core/v5 dual-support mirrors
# (ADR-0152) dominate its output with by-design duplication (~120 rows).
# Until the tool grows suppression for dupe, this wrapper drops exactly two
# classes of groups:
#   1. v4-mirror — group members split between <mod>/... and core/v5/<mod>/...
#      (the mirror key derives from the paths themselves, so newly registered
#      mirror pairs are covered without editing this script)
#   2. intra-v5  — every member under core/v5/ (the single-module topology
#      makes cross-package twins inside v5 the v5 module's own concern)
# Everything else passes through untouched, and suppressed groups are listed
# after the report — filtering is visible, never silent.
#
# Usage: bash scripts/dupe-signal.sh [-- PATH...]   (extra branching-flow args)
# Requires: branching-flow, jq
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARGS=()
if [[ "${1:-}" == "--" ]]; then
	shift
	ARGS=("$@")
fi

RAW="$(branching-flow dupe "${ARGS[@]:-.}" --format json --exclude-generated 2>/dev/null)"

classify='
def v5key(f): f | sub("^core/v5/"; "") | split("/")[0];
group_by(.Group)
| map(. as $g | {
    id: $g[0].Group,
    rows: $g,
    n: ($g | length),
    n_v5: ([$g[] | select(.File | startswith("core/v5/"))] | length),
    keys: ([$g[] | select(.File | startswith("core/v5/")) | v5key(.File)] | unique),
    v4firsts: ([$g[] | select((.File | startswith("core/v5/")) | not) | (.File | split("/")[0])] | unique)
  })
| map(. as $g |
    if $g.n_v5 == $g.n then $g + {class: "intra-v5"}
    elif $g.n_v5 > 0 and ($g.keys | length) == 1
      and ($g.v4firsts | length) == 1 and $g.v4firsts[0] == $g.keys[0]
    then $g + {class: "v4-mirror"}
    else $g + {class: "signal"}
    end)
'

echo "# dupe-signal: duplicate-type findings minus ADR-0152 mirrors"
echo
echo "$RAW" | jq -r "$classify | [.[] | select(.class == \"signal\")] | sort_by(.id)
  | if length == 0 then \"(no non-mirror duplicate-type findings)\" else
    .[] | \"group \\(.id) [\\(.n) sites] \\(.rows[0].Kind) \\(.rows[0].Type) — \\(.rows[0].Verdict)\",
      (.rows[] | \"  - \\(.File):\\(.Line)\")
  end"

echo
echo "$RAW" | jq -r "$classify | [.[] | select(.class != \"signal\")] as \$supp
  | ([.[] | select(.class == \"signal\")] | length) as \$kept
  | \"kept \\(\$kept) signal group(s); suppressed \\(\$supp | length) mirror group(s):\" + (
    if (\$supp | length) == 0 then \"\\n  (none)\" else
      \"\\n\" + ([\$supp | sort_by(.id) | .[] | \"  group \\(.id) [\\(.class), \\(.n) sites] \\(.rows[0].Type): \" + ([.rows[].File] | join(\", \"))] | join(\"\\n\"))
    end)"
