# Summary

<!-- One or two sentences: what does this PR change, and why? -->

## What

<!-- Bullet list of the changes. -->

## Why

<!-- The problem or motivation. Link the ADR / TODO_LIST / issue if one exists. -->

## Verification

<!-- Which gates did you run for THIS diff? -->

- [ ] `nix run .#verify` (or `#verify-fast`) green
- [ ] api-stability golden regenerated (if exported symbols changed): `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update`
- [ ] doc-check green (if skill docs / AGENTS.md changed)
- [ ] New tests cover the changed behavior

## Notes for reviewers

<!-- Anything surprising: tradeoffs, follow-ups, known gaps. -->
