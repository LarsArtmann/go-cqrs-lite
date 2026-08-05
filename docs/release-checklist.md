# Release Checklist

> Step-by-step release process for go-cqrs-lite.
> See also: `CONTRIBUTING.md` → Release Process, `scripts/tag-release.sh`.

## Pre-release verification

1. **Full verify gate GREEN:**

   ```bash
   nix run .#verify
   ```

   Must exit 0. Includes build + vet + test + race + lint + api-stability + doc-check.

2. **Coverage drift check:**

   ```bash
   nix run .#check-coverage
   ```

   If any module drifted beyond ±2%, update AGENTS.md and the `EXPECTED` map.

3. **Duplication check:**

   ```bash
   nix run .#check-duplication
   ```

   If new clones are flagged, either consolidate or update the baseline:
   `art-dupl baseline . --threshold 3 --semantic`.

4. **Dependency layers:**

   ```bash
   nix run .#check-layers
   ```

5. **Vulnerability scan (post-cut):**
   ```bash
   nix run .#vulncheck
   ```

## Tagging

1. **Verify all modules are tagged.** 67 of 68 modules should have tags reachable
   from HEAD:

   ```bash
   git tag --merged HEAD | grep '/v' | sort
   ```

   If any module is orphaned (tag points to a commit not in HEAD), re-tag before
   cutting the release.

2. **Tag all modules** via the release script (annotated tags, never lightweight):

   ```bash
   bash scripts/tag-release.sh v4.2.0
   ```

   The script tags every module listed in `cmd/api-stability/main.go` `modules`.

3. **Push tags** (requires user approval):
   ```bash
   git push origin --tags
   ```

## CHANGELOG

1. Move `[Unreleased]` entries to a versioned section:
   ```markdown
   ## [v4.2.0] — 2026-07-27
   ```
2. Create a fresh empty `[Unreleased]` section above it.
3. Verify the doc-assertion gate: `[Unreleased]` must appear exactly once.

## Post-release

1. **Verify `go get` resolves** for a sample module:
   ```bash
   GOWORK=off go get github.com/larsartmann/go-cqrs-lite/event/v4@v4.2.0
   ```
2. **Run vulncheck** across all module deps.
3. **Update ROADMAP** — move shipped items from Themes to Release History.

## Semver rules

- **Patch (v4.x.Y):** bug fixes, no new public API.
- **Minor (v4.Y.0):** new public API (new exported symbols). This is the default
  for any change that adds exports.
- **Major (v5.0.0):** breaking changes to existing public API. Requires ADR.

> **Lesson learned (codec/v4.1.1):** new public API (`TranscodeToJSON`) was
> shipped under a patch tag. New API always requires a minor bump. When in
> doubt, bump minor.
