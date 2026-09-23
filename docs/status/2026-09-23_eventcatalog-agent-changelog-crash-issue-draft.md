## Problem

With `changelog: { enabled: true }` in `eventcatalog.config.js`, any catalog that contains an agent fails `eventcatalog build` with exit code 1:

```
[ERROR] [build] Caught error rendering /docs/agents/IntegrationAgent/1.0.0/changelog:
TypeError: Cannot read properties of undefined (reading 'url')
    at getBadgeHref (.../badge-styles_BH5CZeRp.mjs:197:13)
    at AstroComponentInstance.Badge (.../astro-component_xbwE6Mqy.mjs:15:10)
```

The changelog page route generates pages for `agents` (and `systems`), but `getBadge()` on that page has no branch for either collection:

```ts
// src/pages/docs/[type]/[id]/[version]/changelog/_index.data.ts:17
const itemTypes: PageTypes[] = ['agents', 'events', 'commands', 'queries',
  'services', 'domains', 'systems', 'flows', 'containers'];

// src/pages/docs/[type]/[id]/[version]/changelog/index.astro:77-101
const getBadge = () => {
  if (props.collection === 'services') { ... }
  // ...events, commands, queries, domains, containers, flows
  // no 'agents' and no 'systems' branch -> falls through, returns undefined
};
const badges = [getBadge()]; // [undefined]

// src/components/Badge.astro:25 -> src/utils/badge-styles.ts:223-224
const href = getBadgeHref(badge);   // Badge.astro
if (!badge.url) return undefined;   // badge-styles.ts -> throws on undefined
```

No `changelog.mdx` is needed for the resource — the page renders for every agent as soon as the feature is on.

## Reproducer

```bash
npm init -y && npm i @eventcatalog/core
mkdir -p agents/IntegrationAgent
cat > agents/IntegrationAgent/index.mdx <<'EOF'
---
id: IntegrationAgent
name: Integration Agent
summary: Probe agent.
version: 1.0.0
---

Body.
EOF
cat > eventcatalog.config.js <<'EOF'
export default { changelog: { enabled: true } };
EOF
npx eventcatalog build
```

The same fixture with `enabled: false` builds clean, so the trigger is the combination (agent in catalog + changelog enabled), not the config alone.

Verified on @eventcatalog/core 4.6.3 and 4.11.2 (latest) — fails identically on both, so not a recent regression. Did not find an existing report for this.

## Fix

Any of these closes it; the first is probably the intended parity:

1. Add `agents` and `systems` branches to `getBadge()`.
2. `const badges = getBadge() ? [getBadge()] : [];`
3. Null-guard the badge in `Badge.astro` / `getBadgeHref`.

Related, same feature area, happy to split into its own issue: with changelog enabled the sidebar also links `/docs/{channels,data-products,entities}/<id>/<version>/changelog` (`sidebar-store/state.ts:794`, `builders/entity.ts`, `builders/data-product.ts`), but `itemTypes` does not include those three collections, so the build logs 3 broken sidebar links.

💘 Generated with Crush
