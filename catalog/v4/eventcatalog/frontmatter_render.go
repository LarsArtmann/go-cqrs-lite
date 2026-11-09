package eventcatalog

import (
	"fmt"
	"time"

	yaml "github.com/go-faster/yaml"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// renderMDX marshals frontmatter to YAML and wraps it with the MDX body.
func renderMDX(fm any, title, summary string, includeGraph bool) (string, error) {
	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", errorfamily.WrapCorruption(err, "catalog.marshal_frontmatter",
			"marshal frontmatter to YAML")
	}

	body := fmt.Sprintf("---\n%s---\n\n# %s\n", string(data), title)

	if summary != "" {
		body += "\n" + summary + "\n"
	}

	if includeGraph {
		body += "\n<NodeGraph />\n"
	}

	return body, nil
}

// rejectUnsupportedBaseConfig fails loudly when BaseConfig carries fields the
// EventCatalog format cannot represent. EventCatalog resourceGroups items
// must be typed pointers ({id, version, type: <resource-kind>}) — the
// catalog's plain string items cannot be mapped, and emitting them makes the
// downstream eventcatalog build fail schema validation.
func rejectUnsupportedBaseConfig(resourceKind, resourceID string, cfg catalog.BaseConfig) error {
	if len(cfg.ResourceGroups) == 0 {
		return nil
	}

	return errorfamily.Newf(errorfamily.Rejection, "catalog.exporter.unsupported.1",
		"%s %s: EventCatalog resourceGroups require typed resource pointers, which the catalog's "+
			"string-based ResourceGroup.Items cannot express; remove ResourceGroups before exporting "+
			"to EventCatalog", resourceKind, resourceID)
}

func toBaseConfig(b catalog.BaseConfig) baseConfigFM {
	var fm baseConfigFM
	if b.Sidebar != nil {
		fm.Sidebar = &sidebarFM{Badge: b.Sidebar.Badge, Label: b.Sidebar.Label}
	}
	if b.Styles != nil {
		// EventCatalog styles nest node color/label under styles.node — flat
		// nodeColor/nodeLabel keys are silently stripped by zod.
		fm.Styles = &stylesFM{
			Icon: b.Styles.Icon,
			Node: &nodeStylesFM{
				Color: b.Styles.NodeColor,
				Label: b.Styles.NodeLabel,
			},
		}
	}
	fm.EditUrl = b.EditUrl
	if b.Draft != nil {
		fm.Draft = &draftFM{Title: b.Draft.Title, Message: b.Draft.Message}
	}
	fm.Visualiser = b.Visualiser

	// DetailsPanel has no EventCatalog equivalent (its per-resource
	// detailsPanel maps section keys to {visible} flags, not a section list)
	// and is intentionally not emitted.

	return fm
}

func toSource(s *catalog.Source) *sourceFM {
	if s == nil {
		return nil
	}

	return &sourceFM{
		Provider: s.Provider,
		ID:       s.ID,
		URL:      string(s.URL),
	}
}

func toDeprecated(deprecated bool, info *catalog.DeprecationInfo) any {
	if info != nil {
		fm := deprecationFM{Message: info.Message}
		if info.Date != nil {
			fm.Date = info.Date.Format(time.DateOnly)
		}

		return fm
	}

	if deprecated {
		return true
	}

	return nil
}

func toDataProductOutputs(outputs []catalog.DataProductOutput) []dataProductOutputFM {
	if len(outputs) == 0 {
		return nil
	}

	out := make([]dataProductOutputFM, len(outputs))
	for i, o := range outputs {
		fm := dataProductOutputFM{
			ID:      string(o.ID),
			Version: string(o.Version),
		}

		if o.Contract != nil {
			fm.Contract = &dataContractFM{
				Path: o.Contract.Path,
				Name: string(o.Contract.Name),
				Type: o.Contract.Type,
			}
		}

		out[i] = fm
	}

	return out
}

func toSchemas(schemas []catalog.SchemaPointer) []schemaPointerFM {
	if len(schemas) == 0 {
		return nil
	}

	out := make([]schemaPointerFM, len(schemas))
	for i, s := range schemas {
		out[i] = schemaPointerFM{
			ID: s.ID, Ref: s.Ref, File: s.File, Path: s.Path,
			Name: string(s.Name), Format: s.Format, Default: s.Default,
		}
	}

	return out
}

func toEntityProperties(props []catalog.EntityProperty) []entityPropertyFM {
	if len(props) == 0 {
		return nil
	}

	out := make([]entityPropertyFM, len(props))
	for i, p := range props {
		out[i] = entityPropertyFM{
			Name:                 string(p.Name),
			Type:                 p.Type,
			Required:             p.Required,
			Description:          p.Description,
			References:           string(p.References),
			ReferencesIdentifier: p.ReferencesIdentifier,
			RelationType:         p.RelationType,
		}
	}

	return out
}

// --- Message ID collection (shared by service and agent) ---

func collectSendsReceives(svc catalog.Service) ([]pointer, []pointer) {
	sends := make([]pointer, 0, len(svc.Events))
	receives := make([]pointer, 0, len(svc.Events)+len(svc.Commands)+len(svc.Queries))

	for _, msg := range svc.Events {
		id := catalog.Key(msg)
		if msg.IsSend() {
			sends = append(sends, pointer{ID: string(id), Version: string(msg.Version)})
		} else {
			receives = append(receives, pointer{ID: string(id), Version: string(msg.Version)})
		}
	}

	for _, cmd := range svc.Commands {
		receives = append(
			receives,
			pointer{ID: string(catalog.Key(cmd)), Version: string(cmd.Version)},
		)
	}

	for _, q := range svc.Queries {
		receives = append(receives, pointer{ID: string(catalog.Key(q)), Version: string(q.Version)})
	}

	return sends, receives
}
