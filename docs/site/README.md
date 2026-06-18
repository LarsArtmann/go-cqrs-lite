# Documentation Site

This directory contains the source for the go-cqrs-lite documentation site,
built with [MkDocs](https://www.mkdocs.org/) and the
[Material for MkDocs](https://squidfunk.github.io/mkdocs-material/) theme.

## Local Development

```bash
# Install MkDocs and Material theme
pip install mkdocs mkdocs-material

# Serve locally at http://localhost:8000
mkdocs serve

# Build static site to site/
mkdocs build
```

## Structure

```
docs/
├── index.md          # Homepage
├── getting-started/  # Quick start guides
├── modules/          # Per-module documentation
├── architecture/     # Architecture patterns and decisions
├── adr/              # Architecture Decision Records
└── api/              # API reference (generated)
```

## Deployment

The site is deployed to GitHub Pages automatically when changes are pushed
to the `docs/` directory. The deployment workflow is defined in
`.github/workflows/docs.yml`.

## Why MkDocs?

- **Markdown-native** — all existing docs are already in Markdown
- **Material theme** — fast, searchable, beautiful out of the box
- **Python toolchain** — doesn't pollute the Go module graph
- **No JavaScript build step** — simpler than Docusaurus/Hugo for this use case
