#!/usr/bin/env python3
"""Citation sweep for docs/review/libs/ audit reports.

Extracts every ``path:line`` citation from the reports and verifies file
existence + line range against the consumer repo AND the library checkouts
under ~/projects. Resolution order: absolute path, repo-relative,
``<lib>/``-qualified cross-repo, bare-basename fallback.

Exit 0 = all citations resolve; exit 1 = any stale citation, any report with
zero citations, or a self-test failure. Re-run after ANY report edit.

Usage:
  scripts/citation-sweep.py [--root REPO] [REPORT ...]   # default: docs/review/libs/*.md
  scripts/citation-sweep.py --self-test
"""

from __future__ import annotations

import argparse
import re
import shutil
import sys
import tempfile
from pathlib import Path

PROJECTS = Path.home() / "projects"

LIB_DIRS = [
    "go-error-family", "go-idempotency", "go-retry", "go-business-rules",
    "go-etag", "httputil", "go-codec", "go-sse", "go-datastar",
    "go-atomic-write", "go-filewatcher", "go-daemon", "go-output",
    "go-health", "go-health-dashboard", "go-credentials-store", "cmdguard",
    "templ-components", "go-appkit", "go-cqrs-lite", "cqrs-htmx",
]

EXTS = (".go", ".rs", ".ts", ".tsx", ".py", ".sh", ".md", ".json", ".yml",
        ".yaml", ".html", ".templ", ".css", ".toml", ".nix", ".txt")

# path:line, path:line-line, path:Lline (godoc style). En-dashes normalized.
CITE_RE = re.compile(
    r"`?(?P<path>(?:[A-Za-z0-9_.\-]+/)*[A-Za-z0-9_.\-]+\.[A-Za-z0-9]+)"
    r"(?::|\s*line\s+|#L)(?P<start>\d+)"
    r"(?:[–—-]+(?P<end>\d+))?" + "`?"
)

SKIP_PREFIXES = ("http://", "https://", "v4.", "v0.", "v1.", "v2.", "v5.")


def normalize(text: str) -> str:
    return text.replace("–", "-").replace("—", "-")


def count_lines(path: Path) -> int:
    try:
        with path.open(encoding="utf-8", errors="replace") as fh:
            return sum(1 for _ in fh)
    except OSError:
        return -1


def resolve(path_str: str, root: Path) -> Path | None:
    """Absolute -> repo-relative -> <lib>/-qualified -> basename fallback."""
    p = Path(path_str)
    if p.is_absolute():
        return p if p.is_file() else None
    cand = root / path_str
    if cand.is_file():
        return cand
    parts = p.parts
    if parts and parts[0] in LIB_DIRS:
        rest = Path(*parts[1:])
        cand = PROJECTS / parts[0] / rest
        if cand.is_file():
            return cand
    # Bare basename: search consumer repo first, then lib checkouts (bounded).
    name = p.name
    for base in [root] + [PROJECTS / d for d in LIB_DIRS]:
        if not base.is_dir():
            continue
        for hit in base.rglob(name):
            s = str(hit)
            if "/vendor/" in s or "/.git/" in s or "/node_modules/" in s:
                continue
            return hit
    return None


def sweep(report: Path, root: Path) -> tuple[list[str], list[str]]:
    text = normalize(report.read_text(encoding="utf-8", errors="replace"))
    ok, bad = [], []
    for m in CITE_RE.finditer(text):
        path_str = m.group("path")
        if path_str.startswith(SKIP_PREFIXES):
            continue
        start = int(m.group("start"))
        end = int(m.group("end") or start)
        cite = f"{path_str}:{start}" + (f"-{end}" if end != start else "")
        resolved = resolve(path_str, root)
        if resolved is None:
            bad.append(f"{report.name}: UNRESOLVED FILE {cite}")
            continue
        total = count_lines(resolved)
        if total < 0:
            bad.append(f"{report.name}: UNREADABLE {cite} -> {resolved}")
        elif start < 1 or end < start or end > total:
            bad.append(
                f"{report.name}: LINE RANGE {cite} outside "
                f"{resolved} (1..{total})")
        else:
            ok.append(f"{cite} -> {resolved}")
    return ok, bad


def self_test() -> int:
    tmp = Path(tempfile.mkdtemp(prefix="cite-sweep-"))
    try:
        good = tmp / "good.md"
        good.write_text("See event/event.go:1 and go-health/README.md:1\n")
        badf = tmp / "bad.md"
        badf.write_text("See event/event.go:999999 and nope/missing.go:7\n")
        empty = tmp / "empty.md"
        empty.write_text("No citations here.\n")
        root = Path(__file__).resolve().parent.parent
        ok, bad = sweep(good, root)
        if bad or len(ok) != 2:
            print(f"SELF-TEST FAIL (good): ok={ok} bad={bad}")
            return 1
        ok2, bad2 = sweep(badf, root)
        if len(bad2) != 2 or ok2:
            print(f"SELF-TEST FAIL (bad): ok={ok2} bad={bad2}")
            return 1
        ok3, bad3 = sweep(empty, root)
        if ok3 or bad3:
            print(f"SELF-TEST FAIL (empty should be flagged by caller): {ok3} {bad3}")
            return 1
        print("self-test: OK (2 resolved, 2 flagged, 0-cite report detected)")
        return 0
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("reports", nargs="*")
    ap.add_argument("--root", default=str(Path(__file__).resolve().parent.parent))
    ap.add_argument("--self-test", action="store_true")
    args = ap.parse_args()

    if args.self_test:
        return self_test()

    root = Path(args.root).resolve()
    reports = [Path(r) for r in args.reports]
    if not reports:
        reports = sorted((root / "docs/review/libs").glob("*.md"))
    if not reports:
        print("citation-sweep: no reports found")
        return 1

    total_ok, all_bad, zero_cite = 0, [], []
    for report in reports:
        ok, bad = sweep(report, root)
        total_ok += len(ok)
        all_bad.extend(bad)
        if not ok:
            zero_cite.append(report.name)

    for b in all_bad:
        print(f"FAIL {b}")
    for z in zero_cite:
        print(f"FAIL {z}: zero citations found (evidence-less report?)")
    print(f"citation-sweep: {total_ok} ok, {len(all_bad)} bad, "
          f"{len(reports)} reports")
    return 1 if (all_bad or zero_cite) else 0


if __name__ == "__main__":
    sys.exit(main())
