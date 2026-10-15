package eventcatalog

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// indexManifestFile is the machine-readable manifest written to the export
// root alongside the MDX tree. Federation hubs diff it between builds for
// cheap change detection (resource added/removed/version-bumped) and PR
// gates, without walking MDX frontmatter.
const indexManifestFile = "catalog.index.json"

// indexManifestSchemaVersion bumps whenever the manifest's field set changes
// incompatibly. Consumers should reject schema versions above their own.
// v2 (2026-10-15): optional errorSurface section (erraudit errorflow
// attachment — WithErrorflowFile); v1 readers see the bumped version and
// stop instead of silently ignoring error-surface changes.
const indexManifestSchemaVersion = 2

// manifestResource is one exported resource entry in catalog.index.json.
// Path is relative to the export root with forward slashes.
type manifestResource struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// indexManifest is the catalog.index.json document. Resources are ordered
// deterministically (canonical kind order, then ID) so re-exporting an
// unchanged catalog produces byte-identical output. ErrorSurface is emitted
// only when the exporter carries an errorflow attachment (omitempty: v1
// byte-compat for plain exports).
type indexManifest struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Resources     []manifestResource    `json:"resources"`
	ErrorSurface  *manifestErrorSurface `json:"errorSurface,omitempty"`
}

// manifestErrorSurface is the error-surface section of catalog.index.json
// (manifest schemaVersion 2): the attached erraudit errorflow export,
// name-sorted by erraudit, projected to the fields federation gates diff.
// WrappedBy/CheckedBy counts (not site lists) so gate diffs stay about the
// SURFACE, not call positions.
type manifestErrorSurface struct {
	Tool          string                  `json:"tool"`
	SchemaVersion int                     `json:"schemaVersion"`
	Scope         string                  `json:"scope"`
	Source        string                  `json:"source,omitempty"`
	FilesScanned  int                     `json:"filesScanned"`
	Errors        []manifestErrorSurfaceE `json:"errors"`
}

type manifestErrorSurfaceE struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Code      string `json:"code,omitempty"`
	Family    string `json:"family,omitempty"`
	Wraps     string `json:"wraps,omitempty"`
	Status    string `json:"status"`
	WrappedBy int    `json:"wrappedBy,omitempty"`
	CheckedBy int    `json:"checkedBy,omitempty"`
}

// manifestKindOrder is the canonical resource order of the manifest. It is
// stable by declaration: services first (the mesh's owning units), then
// their messages, then the supporting collections.
func manifestKindOrder() []string {
	return []string{
		"services", "commands", "events", "queries", "channels",
		"domains", "containers", "entities", "data-products", "flows",
		"agents", "teams", "users", "docs",
	}
}

// buildIndexManifest derives the manifest from the same catalog the MDX
// writers consume. Message entries mirror writeAllMessages: every message
// once, under its canonical collection, deduplicated across services.
// Sidecar files (schemas, changelogs, examples) are not itemized — the
// manifest is a resource inventory, not a file listing.
func buildIndexManifest(cat *catalog.Catalog) indexManifest {
	byKind := make(map[string][]manifestResource)

	for _, svc := range cat.Services {
		byKind["services"] = append(byKind["services"], manifestResource{
			ID: string(svc.ID), Kind: "services", Version: string(svc.Version),
			Path: resourceIndexPath("services", string(svc.ID)),
		})
	}

	appendMessages := func(kind string, messages []catalog.Message) {
		seen := make(map[string]struct{}, len(messages))
		for _, msg := range messages {
			key := string(catalog.Key(msg))
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}

			byKind[kind] = append(byKind[kind], manifestResource{
				ID: key, Kind: kind, Version: string(msg.Version),
				Path: resourceIndexPath(kind, key),
			})
		}
	}
	appendMessages("commands", commandsOf(cat))
	appendMessages("events", eventsOf(cat))
	appendMessages("queries", queriesOf(cat))

	for _, ch := range cat.Channels {
		byKind["channels"] = append(byKind["channels"], manifestResource{
			ID: string(ch.ID), Kind: "channels", Version: string(ch.Version),
			Path: resourceIndexPath("channels", string(ch.ID)),
		})
	}
	for _, domain := range cat.Domains {
		byKind["domains"] = append(byKind["domains"], manifestResource{
			ID: string(domain.ID), Kind: "domains", Version: string(domain.Version),
			Path: resourceIndexPath("domains", string(domain.ID)),
		})
	}
	for _, ds := range cat.DataStores {
		byKind["containers"] = append(byKind["containers"], manifestResource{
			ID: string(ds.ID), Kind: "containers", Version: string(ds.Version),
			Path: resourceIndexPath("containers", string(ds.ID)),
		})
	}
	for _, entity := range cat.Entities {
		byKind["entities"] = append(byKind["entities"], manifestResource{
			ID: string(entity.ID), Kind: "entities", Version: string(entity.Version),
			Path: resourceIndexPath("entities", string(entity.ID)),
		})
	}
	for _, dp := range cat.DataProducts {
		byKind["data-products"] = append(byKind["data-products"], manifestResource{
			ID: string(dp.ID), Kind: "data-products", Version: string(dp.Version),
			Path: resourceIndexPath("data-products", string(dp.ID)),
		})
	}
	for _, flow := range cat.Flows {
		byKind["flows"] = append(byKind["flows"], manifestResource{
			ID: string(flow.ID), Kind: "flows", Version: string(flow.Version),
			Path: resourceIndexPath("flows", string(flow.ID)),
		})
	}
	for _, agent := range cat.Agents {
		byKind["agents"] = append(byKind["agents"], manifestResource{
			ID: string(agent.ID), Kind: "agents", Version: string(agent.Version),
			Path: resourceIndexPath("agents", string(agent.ID)),
		})
	}
	for _, team := range cat.Teams {
		byKind["teams"] = append(byKind["teams"], manifestResource{
			ID: string(team.ID), Kind: "teams",
			Path: "teams/" + string(team.ID) + ".mdx",
		})
	}
	for _, user := range cat.Users {
		byKind["users"] = append(byKind["users"], manifestResource{
			ID: string(user.ID), Kind: "users",
			Path: "users/" + string(user.ID) + ".mdx",
		})
	}
	for _, doc := range cat.CustomDocs {
		slug := doc.Slug
		if slug == "" {
			slug = string(doc.ID)
		}
		byKind["docs"] = append(byKind["docs"], manifestResource{
			ID: string(doc.ID), Kind: "docs",
			Path: resourceIndexPath("docs", slug),
		})
	}

	manifest := indexManifest{SchemaVersion: indexManifestSchemaVersion}
	for _, kind := range manifestKindOrder() {
		group := byKind[kind]
		slices.SortFunc(group, func(a, b manifestResource) int {
			return strings.Compare(a.ID, b.ID)
		})
		manifest.Resources = append(manifest.Resources, group...)
	}

	return manifest
}

// buildErrorSurface projects the errorflow attachment into the manifest's
// errorSurface section. Site lists become counts (gate diffs track the
// surface, not call positions); the error order is erraudit's name sort.
func buildErrorSurface(export *errorflowExport) *manifestErrorSurface {
	if export == nil {
		return nil
	}

	errors := make([]manifestErrorSurfaceE, 0, len(export.Errors))
	for _, entry := range export.Errors {
		errors = append(errors, manifestErrorSurfaceE{
			Name:      entry.Name,
			Kind:      entry.Kind,
			Code:      entry.Code,
			Family:    entry.Family,
			Wraps:     entry.Wraps,
			Status:    entry.Status,
			WrappedBy: len(entry.WrappedBy),
			CheckedBy: len(entry.CheckedBy),
		})
	}

	return &manifestErrorSurface{
		Tool:          "erraudit",
		SchemaVersion: export.SchemaVersion,
		Scope:         "module",
		Source:        export.Package,
		FilesScanned:  export.FilesScanned,
		Errors:        errors,
	}
}

// resourceIndexPath builds the canonical <collection>/<id>/index.mdx path
// with forward slashes (independent of the host OS separator).
func resourceIndexPath(collection, id string) string {
	return collection + "/" + id + "/" + indexFile
}

// writeIndexManifest writes catalog.index.json to the export root. It runs
// after every resource writer so the manifest only exists for fully
// successful exports. The errorSurface section rides along when the
// exporter carries an errorflow attachment (manifest schemaVersion 2).
func (e *Exporter) writeIndexManifest(cat *catalog.Catalog) error {
	manifest := buildIndexManifest(cat)
	manifest.ErrorSurface = buildErrorSurface(e.errorflow)

	data, err := json.Marshal(
		manifest,
		jsontext.WithIndentPrefix(""),
		jsontext.WithIndent("  "),
	)
	if err != nil {
		return errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.index_manifest.1",
			"marshal index manifest: %v",
			err,
		)
	}

	return os.WriteFile( //nolint:wrapcheck // direct passthrough
		filepath.Join(e.outputDir, indexManifestFile),
		data,
		filePerm,
	)
}
