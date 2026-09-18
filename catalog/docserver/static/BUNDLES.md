# Vendored docserver bundles

Self-hosted browser bundles for the embedded docs UIs. Nothing here is
written by hand except `docs-ui.src.css` (compiled to `docs-ui.css` via
`nix run .#build-docserver-css`); the `.js`/`.css` files below are vendored
npm artifacts, upgraded deliberately, never at runtime from a CDN.

| File               | Source artifact (npm)                                   | Bytes     | Exposes                        |
| ------------------ | ------------------------------------------------------- | --------- | ------------------------------ |
| `scalar.js`        | `@scalar/api-reference@1.69.0` `dist/browser/standalone.js` | 3,776,829 | `window.Scalar.createApiReference` |
| `asyncapi-react.js`| `@asyncapi/react-component@3.2.1` `browser/standalone/index.js` | 3,042,659 | `window.AsyncApiStandalone.render` |
| `asyncapi-react.css`| `@asyncapi/react-component@3.2.1` `styles/default.min.css` | 24,647    | `.aui-root`-scoped styles      |

## Upgrade procedure

1. Download the exact artifact from the npm package tarball
   (`https://registry.npmjs.org/<pkg>/<version>` → `npm pack <pkg>@<version`
   also works) and replace the file.
2. Verify the contract the pages depend on **in the actual bytes**, not the
   docs: `grep` for the global/entry above (`createApiReference`,
   `AsyncApiStandalone`), and check the stylesheet's scope class
   (`.aui-root`) still matches what the component mounts.
3. Re-check CSP interactions:
   - Scalar injects runtime styles → `style-src 'unsafe-inline'` must stay.
   - The AsyncAPI component's ajv compiler uses `new Function` → the
     `/docs/asyncapi` page keeps its scoped `script-src 'unsafe-eval'`
     (`csp.go`). Scalar needs no eval.
   - Both bundles phone home (fonts.scalar.com, api.scalar.com); the CSP
     deliberately blocks those origins — `csp_browser_test.go` whitelists
     exactly those refusals.
4. Run the full gate: `nix run .#check-csp` (headless Chromium asserts every
   page renders, mounts, and emits no unexpected CSP refusals).
