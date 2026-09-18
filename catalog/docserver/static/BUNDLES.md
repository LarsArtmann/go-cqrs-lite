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
| `fonts/*.woff2` (14)| `https://fonts.scalar.com/<name>.woff2` (the URLs embedded in scalar.js) | 263,900 total | Inter (8 subsets) + Inter mono (6 subsets) for the OpenAPI UI |

The woff2 files are Scalar's official CDN font binaries, byte-identical to
what scalar.js requests. `docserver.go` serves scalar.js with its hardcoded
`https://fonts.scalar.com/` URLs rewritten to `<DocsPath>/static/fonts/`, so
the OpenAPI UI loads fonts from this server (`font-src 'self'`) instead of
being denied with 26 console refusals per visit. `fonts.scalar.com` must NOT
be re-added to `deliberatelyDeniedOrigins` in `csp_browser_test.go`: after
the rewrite, a refusal naming it means the rewrite regressed.

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
   - Fonts: if the new bundle changes the font URL set, re-download the
     referenced woff2 files into `fonts/`, and update the rewrite constant
     (`scalarFontCDN` in `docserver.go`) if the origin moved.
   - The AsyncAPI bundle phones home to api.scalar.com; the CSP deliberately
     blocks that origin — `csp_browser_test.go` whitelists exactly that
     refusal.
4. Run the full gate: `nix run .#check-csp` (headless Chromium asserts every
   page renders, mounts, and emits no unexpected CSP refusals).
